[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidateSet('Up', 'Down', 'Restart', 'Status', 'Smoke', 'Reset')]
    [string]$Action,

    [Parameter()]
    [string]$Confirmation = '',

    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repositoryRoot 'deploy/local/compose.yaml'
$configDirectory = Join-Path $repositoryRoot 'deploy/environments'
$stateRoot = Join-Path $repositoryRoot '.tmp/environments'
$runtimeFile = Join-Path $stateRoot 'local/runtime.env'

function Invoke-NativeChecked {
    param(
        [Parameter(Mandatory)]
        [string]$Command,

        [Parameter()]
        [string[]]$Arguments = @()
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

function Initialize-LocalEnvironment {
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'validate', '--config-dir', $configDirectory)
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'seed-checksum', '--manifest', (Join-Path $repositoryRoot 'deploy/seeds/foundation.json'))
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'prepare', '--environment', 'local', '--config-dir', $configDirectory, '--state-root', $stateRoot)

    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    if ($changes.Length -eq 0) {
        $env:ATLAS_SOURCE_REVISION = (& git rev-parse HEAD | Out-String).Trim()
        $env:ATLAS_BUILD_TIME = (& git show -s --format=%cI HEAD | Out-String).Trim()
    }
    else {
        $env:ATLAS_SOURCE_REVISION = 'development'
        $env:ATLAS_BUILD_TIME = '1970-01-01T00:00:00Z'
    }
}

function Invoke-Compose {
    param([string[]]$Arguments)

    Invoke-NativeChecked -Command $ContainerRuntime -Arguments (@('compose', '--env-file', $runtimeFile, '--file', $composeFile) + $Arguments)
}

function Wait-ForFoundation {
    $deadline = [DateTimeOffset]::UtcNow.AddMinutes(5)
    do {
        try {
            $ready = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -TimeoutSec 3 -Uri 'http://127.0.0.1:18080/health/ready'
            $web = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -TimeoutSec 3 -Uri 'http://127.0.0.1:13000/runtime-config.json'
            $identity = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -TimeoutSec 3 -Uri 'http://127.0.0.1:18081/realms/atlas-customer-local/.well-known/openid-configuration'
            if ($ready.StatusCode -eq 200 -and $web.StatusCode -eq 200 -and $identity.StatusCode -eq 200) {
                return
            }
        }
        catch {
            # Services are still starting; do not emit connection details.
        }
        Start-Sleep -Seconds 2
    } while ([DateTimeOffset]::UtcNow -lt $deadline)

    throw 'Atlas local foundation did not become ready before the bounded deadline'
}

Push-Location -LiteralPath $repositoryRoot
try {
    switch ($Action) {
        'Up' {
            Initialize-LocalEnvironment
            Invoke-Compose -Arguments @('up', '--detach', '--build', '--remove-orphans')
            Wait-ForFoundation
            & (Join-Path $PSScriptRoot 'test-s04-live.ps1')
            Write-Output 's04_environment_up=PASS'
        }
        'Down' {
            if (Test-Path -LiteralPath $runtimeFile) {
                Invoke-Compose -Arguments @('down', '--remove-orphans')
            }
            Write-Output 's04_environment_down=PASS'
        }
        'Restart' {
            Initialize-LocalEnvironment
            Invoke-Compose -Arguments @('down', '--remove-orphans')
            Invoke-Compose -Arguments @('up', '--detach', '--remove-orphans')
            Wait-ForFoundation
            & (Join-Path $PSScriptRoot 'test-s04-live.ps1')
            Write-Output 's04_environment_restart=PASS'
        }
        'Status' {
            Initialize-LocalEnvironment
            Invoke-Compose -Arguments @('ps')
        }
        'Smoke' {
            & (Join-Path $PSScriptRoot 'test-s04-live.ps1')
        }
        'Reset' {
            if ($Confirmation -ne 'RESET ATLAS LOCAL') {
                throw 'Reset requires the exact confirmation RESET ATLAS LOCAL'
            }
            Initialize-LocalEnvironment
            Invoke-Compose -Arguments @('down', '--volumes', '--remove-orphans')
            Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'reset', '--environment', 'local', '--confirm', $Confirmation, '--state-root', $stateRoot)
            Write-Output 's04_environment_reset=PASS'
        }
    }
}
finally {
    Pop-Location
}
