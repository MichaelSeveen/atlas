[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'

function Invoke-NativeChecked([string]$Command, [string[]]$Arguments = @()) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Command failed: $Command $($Arguments -join ' ')" }
}

Push-Location -LiteralPath $repositoryRoot
try {
    $unformatted = (& gofmt -l ./internal/architecture ./tests/contract | Out-String).Trim()
    if ($unformatted.Length -ne 0) { throw "Unformatted Go source:`n$unformatted" }

    Invoke-NativeChecked 'go' @('run', './cmd/contractctl', 'lint', 'docs/atlas-prd/03-contracts/openapi.yaml', 'docs/atlas-prd/03-contracts/asyncapi.yaml')
    Invoke-NativeChecked 'go' @('test', './tests/contract', '-count=1')
    Invoke-NativeChecked 'go' @('test', './internal/architecture', '-count=1')

    & (Join-Path $PSScriptRoot 'test-p01-s02-contract-canary.ps1')
    if (-not $?) { throw 'Phase 01 S02 contract canaries failed.' }
    & (Join-Path $PSScriptRoot 'test-p01-evidence-integrity.ps1')
    if (-not $?) { throw 'Phase 01 evidence integrity failed.' }

    Write-Output 'p01_s02_runtime_routes=NOT_IMPLEMENTED(contract-and-decision-slice-only)'
    Write-Output 'p01_s02_traceability=PLANNED(runtime-evidence-pending)'
    Write-Output 'p01_s02_verification=PASS'
}
finally {
    Pop-Location
}
