[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('chrome', 'msedge', 'firefox', 'webkit')]
    [string]$Browser = 'msedge'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$webRoot = Join-Path $repositoryRoot 'apps/web'
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
$outputRoot = Join-Path $repositoryRoot 'output/playwright'
$session = 'atlas-p01-s09-keycloak'
$nodeExecutable = (Get-Command node -ErrorAction Stop).Source
$cliScript = Join-Path $webRoot 'node_modules/@playwright/cli/playwright-cli.js'

if (-not (Test-Path -LiteralPath $runtimeFile -PathType Leaf)) {
    throw 'Prepared local runtime environment is absent'
}

function Invoke-CLI {
    param([Parameter(ValueFromRemainingArguments)][string[]]$Arguments)
    & $nodeExecutable $cliScript "--session=$session" @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Playwright CLI failed: $($Arguments -join ' ')" }
}

New-Item -ItemType Directory -Path $outputRoot -Force | Out-Null
$env:PLAYWRIGHT_MCP_OUTPUT_DIR = $outputRoot
$env:PLAYWRIGHT_MCP_BROWSER = $Browser
$env:PLAYWRIGHT_MCP_HEADLESS = 'true'
$env:PLAYWRIGHT_MCP_ISOLATED = 'true'
$env:PLAYWRIGHT_MCP_ALLOWED_ORIGINS = 'http://127.0.0.1:13000;http://127.0.0.1:18080;http://127.0.0.1:18081'
$env:PLAYWRIGHT_MCP_SECRETS_FILE = $runtimeFile
$env:PLAYWRIGHT_MCP_SAVE_TRACE = 'true'

Push-Location -LiteralPath $webRoot
try {
    foreach ($uri in @(
        'http://127.0.0.1:13000/runtime-config.json',
        'http://127.0.0.1:18080/health/ready',
        'http://127.0.0.1:18081/realms/atlas-customer-local/.well-known/openid-configuration'
    )) {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $uri -TimeoutSec 5
        if ($response.StatusCode -ne 200) { throw "Synthetic browser dependency is not ready: $uri" }
    }

    Invoke-CLI open 'http://127.0.0.1:13000/customer'
    Invoke-CLI click "getByRole('link', { name: 'Continue to secure sign-in' })"
    Invoke-CLI fill '#username' 'synthetic-customer'
    Invoke-CLI fill '#password' 'ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'
    Invoke-CLI click '#kc-login'
    Invoke-CLI run-code '--filename=browser/live-keycloak-assertions.ts'
    Invoke-CLI screenshot
    Write-Output 'p01_s09_keycloak_browser=PASS(customer_login=true,opaque_session=true,logout_back_navigation=denied)'
}
finally {
    try { Invoke-CLI close } catch { }
    foreach ($name in @(
        'PLAYWRIGHT_MCP_OUTPUT_DIR', 'PLAYWRIGHT_MCP_BROWSER', 'PLAYWRIGHT_MCP_HEADLESS',
        'PLAYWRIGHT_MCP_ISOLATED', 'PLAYWRIGHT_MCP_ALLOWED_ORIGINS',
        'PLAYWRIGHT_MCP_SECRETS_FILE', 'PLAYWRIGHT_MCP_SAVE_TRACE'
    )) {
        Remove-Item "Env:$name" -ErrorAction SilentlyContinue
    }
    Pop-Location
}
