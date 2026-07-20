[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$temporaryParent = [IO.Path]::GetFullPath((Join-Path $repositoryRoot '.tmp'))
$canaryRoot = [IO.Path]::GetFullPath((Join-Path $temporaryParent 's03-contract-canary'))
$requiredPrefix = $temporaryParent.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if (-not $canaryRoot.StartsWith($requiredPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing contract-canary work outside repository temporary directory: $canaryRoot"
}

$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $temporaryParent 'go-build'
$env:GOMODCACHE = Join-Path $temporaryParent 'go-mod'
$previousContractPath = $env:ATLAS_OPENAPI_PATH

if (Test-Path -LiteralPath $canaryRoot) {
    Remove-Item -LiteralPath $canaryRoot -Recurse -Force
}

try {
    New-Item -ItemType Directory -Path $canaryRoot -Force | Out-Null
    $canonicalPath = Join-Path $repositoryRoot 'docs/atlas-prd/03-contracts/openapi.yaml'
    $mutatedPath = Join-Path $canaryRoot 'openapi.yaml'
    $source = Get-Content -LiteralPath $canonicalPath -Raw
    $target = '  /health/ready:'
    if (-not $source.Contains($target)) {
        throw 'The expected readiness path was not found; the contract canary drifted.'
    }
    $mutated = $source.Replace($target, '  /health/ready-removed:')
    if ($mutated.Contains($target)) {
        throw 'More than one readiness path matched; refusing an ambiguous contract mutation.'
    }
    Set-Content -LiteralPath $mutatedPath -Value $mutated -NoNewline
    $env:ATLAS_OPENAPI_PATH = $mutatedPath

    Push-Location -LiteralPath $repositoryRoot
    try {
        $canaryOutput = (& go test ./tests/contract -run TestOpenAPIFoundationOperations -count=1 2>&1 | Out-String).Trim()
        $canaryExit = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }

    if ($canaryExit -eq 0) {
        throw 'Removed /health/ready contract path survived the focused contract test.'
    }
    if ($canaryOutput -notmatch '/health/ready') {
        throw "Contract canary failed for an unexpected reason:`n$canaryOutput"
    }

    Write-Output 'contract_canary=REMOVE_HEALTH_READY_PATH'
    Write-Output 'contract_canary_expected=go_test_failure'
    Write-Output 'contract_canary_observed=go_test_failure'
    Write-Output 'contract_ready_path_canary=KILLED'
}
finally {
    if ($null -eq $previousContractPath) {
        Remove-Item Env:ATLAS_OPENAPI_PATH -ErrorAction SilentlyContinue
    }
    else {
        $env:ATLAS_OPENAPI_PATH = $previousContractPath
    }
    if (Test-Path -LiteralPath $canaryRoot) {
        Remove-Item -LiteralPath $canaryRoot -Recurse -Force
    }
}
