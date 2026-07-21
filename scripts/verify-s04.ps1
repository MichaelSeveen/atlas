[CmdletBinding()]
param(
    [Parameter()]
    [switch]$Live
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'

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

Push-Location -LiteralPath $repositoryRoot
try {
    $baseRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { $baseRevision = 'UNBORN' }
    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) { $baseRevision } else { "UNCOMMITTED_WORKTREE(base=$baseRevision)" }

    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'verify-s03.ps1'))
    $unformatted = (& gofmt -l ./cmd ./internal ./tests | Out-String).Trim()
    if ($unformatted.Length -ne 0) { throw "Unformatted Go source:`n$unformatted" }
    Invoke-NativeChecked -Command 'go' -Arguments @('build', './cmd/api', './cmd/worker', './cmd/simulator', './cmd/envctl')
    Invoke-NativeChecked -Command 'go' -Arguments @('vet', './...')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './...', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './internal/platform/environment', '-run', 'TestMostAgentsSkip05|TestLocalConfigurationRejectsEndpointOutsideFixedComposeTopology|TestPrepareIsIdempotentAndResetRequiresExactEnvironmentConfirmation|TestCredentialFingerprintsAreUniqueAcrossPreparedEnvironments|TestFeatureFlagEvaluationIsConcurrentAndDefaultsOnSourceOutage', '-count=1', '-v')
    Invoke-NativeChecked -Command 'bun' -Arguments @('install', '--cwd', 'apps/web', '--frozen-lockfile')
    Invoke-NativeChecked -Command 'bun' -Arguments @('run', '--cwd', 'apps/web', 'typecheck')
    Invoke-NativeChecked -Command 'bun' -Arguments @('run', '--cwd', 'apps/web', 'test')
    Invoke-NativeChecked -Command 'bun' -Arguments @('run', '--cwd', 'apps/web', 'build')
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'validate', '--config-dir', 'deploy/environments')
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'seed-checksum', '--manifest', 'deploy/seeds/foundation.json')
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s04-config-canary.ps1'))
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s04-reset-canary.ps1'))
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s04-seed-canary.ps1'))

    $evidenceReport = Join-Path $repositoryRoot 'evidence/phase-00/environment/S04-environment-report.md'
    $evidenceSidecar = Join-Path $repositoryRoot 'evidence/phase-00/environment/S04-environment-report.sha256'
    $expectedEvidenceDigest = ((Get-Content -LiteralPath $evidenceSidecar -Raw).Trim() -split '\s+')[0]
    $actualEvidenceDigest = (Get-FileHash -Algorithm SHA256 -LiteralPath $evidenceReport).Hash.ToLowerInvariant()
    if ($expectedEvidenceDigest -ne $actualEvidenceDigest) {
        throw 'S04 evidence digest does not match its sidecar'
    }

    if ($Live) {
        & (Join-Path $PSScriptRoot 'test-s04-live.ps1')
        Write-Output 's04_live_verification=PASS'
    }
    else {
        Write-Output 's04_live_verification=NOT_REQUESTED'
    }
    Write-Output 'frontend_toolchain=bun@1.3.0'
    Write-Output 'frontend_framework=react@19.2.7'
    Write-Output 's04_named_skipped_tests=5,6'
    Write-Output "s04_evidence_digest=$actualEvidenceDigest"
    Write-Output "source_revision=$sourceRevision"
    Write-Output 's04_verification=PASS'
}
finally {
    Pop-Location
}
