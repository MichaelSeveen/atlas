[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$stateRoot = Join-Path $repositoryRoot '.tmp/s04-reset-canary'
$configDirectory = Join-Path $repositoryRoot 'deploy/environments'
$repositoryPrefix = [IO.Path]::GetFullPath($repositoryRoot).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$resolvedStateRoot = [IO.Path]::GetFullPath($stateRoot)
if (-not $resolvedStateRoot.StartsWith($repositoryPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'S04 reset canary target escapes the repository'
}

function Invoke-Go {
    param([string[]]$Arguments)
    & go @Arguments | Out-Host
    $exitCode = $LASTEXITCODE
    return $exitCode
}

Push-Location -LiteralPath $repositoryRoot
try {
    if (Test-Path -LiteralPath $stateRoot) {
        throw 'S04 reset canary target already exists; refusing an ambiguous cleanup'
    }
    if ((Invoke-Go @('run', './cmd/envctl', 'prepare', '--environment', 'local', '--config-dir', $configDirectory, '--state-root', $stateRoot)) -ne 0) {
        throw 'S04 reset canary setup failed'
    }
    $runtimePath = Join-Path $stateRoot 'local/runtime.env'
    if ((Invoke-Go @('run', './cmd/envctl', 'reset', '--environment', 'local', '--confirm', 'RESET ATLAS TEST', '--state-root', $stateRoot)) -eq 0) {
        throw 'Wrong-environment reset confirmation was accepted'
    }
    if (-not (Test-Path -LiteralPath $runtimePath -PathType Leaf)) {
        throw 'Rejected reset changed the contained local target'
    }
    if ((Invoke-Go @('run', './cmd/envctl', 'reset', '--environment', 'production-reference', '--confirm', 'RESET ATLAS PRODUCTION-REFERENCE', '--state-root', $stateRoot)) -eq 0) {
        throw 'Production-reference reset was accepted'
    }
    if ((Invoke-Go @('run', './cmd/envctl', 'reset', '--environment', 'local', '--confirm', 'RESET ATLAS LOCAL', '--state-root', $stateRoot)) -ne 0) {
        throw 'Exact contained local reset failed'
    }
    Write-Output 'reset_wrong_environment=REJECTED'
    Write-Output 'reset_production_reference=REJECTED'
    Write-Output 'reset_contained_local=PASS'
    Write-Output 'reset_canary=KILLED'
}
finally {
    if (Test-Path -LiteralPath $stateRoot) {
        Remove-Item -LiteralPath $stateRoot -Recurse -Force
    }
    Pop-Location
}
