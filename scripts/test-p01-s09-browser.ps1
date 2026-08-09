[CmdletBinding()]
param(
    [Parameter()]
    [switch]$InstallBrowser,

    [Parameter()]
    [ValidateSet('chrome', 'msedge', 'firefox', 'webkit')]
    [string]$Browser = 'msedge'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$webRoot = Join-Path $repositoryRoot 'apps/web'
$outputRoot = Join-Path $repositoryRoot 'output/playwright'
$session = 'atlas-p01-s09'
$port = 14173
$nodeExecutable = (Get-Command node -ErrorAction Stop).Source
$cliScript = Join-Path $webRoot 'node_modules/@playwright/cli/playwright-cli.js'

function Invoke-CLI {
    param([Parameter(ValueFromRemainingArguments)][string[]]$Arguments)
    & $nodeExecutable $cliScript "--session=$session" @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Playwright CLI failed: $($Arguments -join ' ')"
    }
}

New-Item -ItemType Directory -Path $outputRoot -Force | Out-Null

Push-Location -LiteralPath $webRoot
try {
    if ($InstallBrowser) {
        & $nodeExecutable $cliScript install-browser chromium
        if ($LASTEXITCODE -ne 0) { throw 'Pinned Playwright Chromium installation failed' }
    }

    $serverEnvironment = @{
        ATLAS_WEB_PORT = $port.ToString()
        ATLAS_ENVIRONMENT = 'local'
        ATLAS_ENVIRONMENT_BANNER = 'LOCAL - SYNTHETIC DATA ONLY'
        ATLAS_API_ORIGIN = "http://127.0.0.1:$port"
    }
    foreach ($entry in $serverEnvironment.GetEnumerator()) {
        [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
    }
    $bunExecutable = (Get-Command bun -ErrorAction Stop).Source
    $server = Start-Process -FilePath $bunExecutable -ArgumentList @('run', 'start') -WorkingDirectory $webRoot -PassThru -WindowStyle Hidden
    try {
        $ready = $false
        for ($attempt = 0; $attempt -lt 40; $attempt++) {
            $server.Refresh()
            if ($server.HasExited) {
                throw "Bounded Atlas web server exited before readiness with code $($server.ExitCode)"
            }
            try {
                $response = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:$port/runtime-config.json" -TimeoutSec 1
                if ($response.StatusCode -eq 200) { $ready = $true; break }
            }
            catch {
                Start-Sleep -Milliseconds 250
            }
        }
        if (-not $ready) { throw 'Bounded Atlas web server did not become ready' }

        $env:PLAYWRIGHT_MCP_OUTPUT_DIR = $outputRoot
        $env:PLAYWRIGHT_MCP_BROWSER = $Browser
        $env:PLAYWRIGHT_MCP_HEADLESS = 'true'
        $env:PLAYWRIGHT_MCP_ISOLATED = 'true'
        $env:PLAYWRIGHT_MCP_ALLOWED_ORIGINS = "http://127.0.0.1:$port"
        $env:PLAYWRIGHT_MCP_SAVE_TRACE = 'true'
        Invoke-CLI open
        Invoke-CLI run-code '--filename=browser/phase01-acceptance.ts'
        Invoke-CLI screenshot
        $console = (& $nodeExecutable $cliScript "--session=$session" console warning | Out-String)
        if ($LASTEXITCODE -ne 0) { throw 'Playwright console inspection failed' }
        if ($console -match '(?i)uncaught|unhandled|error:') {
            throw 'Browser console contains an unexpected error'
        }
        Write-Output 'p01_s09_browser_acceptance=PASS'
        Write-Output 'p01_s09_browser_storage=EMPTY'
        Write-Output 'p01_s09_browser_secret=MEMORY_ONLY_CLEARED'
        Write-Output 'p01_s09_browser_cross_tab_logout=PASS'
        Write-Output 'p01_s09_browser_bfcache_back_navigation=PASS'
    }
    finally {
        try { Invoke-CLI close } catch { }
        if ($server -and -not $server.HasExited) {
            Stop-Process -Id $server.Id -Force
            $server.WaitForExit()
        }
    }
}
finally {
    foreach ($name in @(
        'ATLAS_WEB_PORT', 'ATLAS_ENVIRONMENT', 'ATLAS_ENVIRONMENT_BANNER', 'ATLAS_API_ORIGIN',
        'PLAYWRIGHT_MCP_OUTPUT_DIR', 'PLAYWRIGHT_MCP_HEADLESS', 'PLAYWRIGHT_MCP_ISOLATED',
        'PLAYWRIGHT_MCP_ALLOWED_ORIGINS', 'PLAYWRIGHT_MCP_SAVE_TRACE', 'PLAYWRIGHT_MCP_BROWSER'
    )) {
        Remove-Item "Env:$name" -ErrorAction SilentlyContinue
    }
    Pop-Location
}
