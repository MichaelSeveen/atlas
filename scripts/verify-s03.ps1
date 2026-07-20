[CmdletBinding()]
param()

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
    if ($LASTEXITCODE -ne 0) {
        $baseRevision = 'UNBORN'
    }
    $worktreeChanges = (& git status --porcelain=v1 | Out-String).Trim()
    if ($worktreeChanges.Length -eq 0) {
        $sourceRevision = $baseRevision
    }
    else {
        $sourceRevision = "UNCOMMITTED_WORKTREE(base=$baseRevision)"
    }

    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'verify-s02.ps1'))
    Invoke-NativeChecked -Command 'go' -Arguments @('build', './cmd/api', './cmd/worker', './cmd/simulator')
    Invoke-NativeChecked -Command 'go' -Arguments @('vet', './...')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './cmd/api/...', './tests/contract', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test',
        './cmd/api/internal/server',
        '-run',
        'TestFoundationEndpointContract|TestLiveServerSmokeHealthyAndMigrationBehind|TestMigrationLagFailsReadinessOnly|TestGoldenSyntheticTraceAndBoundedMetrics|TestInvalidRequestMetadataIsReplacedNotReflected|TestIdentifierAndTelemetryDegradationRemainSafe|TestReadinessCheckerReceivesBoundedDeadline|TestSecureCORSMatrix|TestResourceLimitsRouteInventoryAndSafeProblems|TestBuildAndHTTPConfigurationFailClosed|TestSlowHeaderIsBoundedByServerDeadline',
        '-count=1',
        '-v'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './tests/contract', '-count=1', '-v')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './cmd/api/internal/server', '-run=^$', '-fuzz=^FuzzUntrustedRequestMetadata$', '-fuzztime=100x')
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s03-contract-canary.ps1'))

    $evidenceReport = Join-Path $repositoryRoot 'evidence/phase-00/http/S03-http-foundation-report.md'
    $evidenceSidecar = Join-Path $repositoryRoot 'evidence/phase-00/http/S03-http-foundation-report.sha256'
    $expectedEvidenceDigest = ((Get-Content -LiteralPath $evidenceSidecar -Raw).Trim() -split '\s+')[0]
    $actualEvidenceDigest = (Get-FileHash -Algorithm SHA256 -LiteralPath $evidenceReport).Hash.ToLowerInvariant()
    if ($expectedEvidenceDigest -ne $actualEvidenceDigest) {
        throw "S03 evidence digest does not match its sidecar"
    }

    Write-Output 'active_openapi=docs/atlas-prd/03-contracts/openapi.yaml'
    Write-Output 'foundation_routes=/health/live,/health/ready,/version'
    Write-Output 'default_readiness=NOT_READY_UNTIL_REAL_PROBES_EXIST'
    Write-Output 'runtime_trace_exporter=DEFERRED'
    Write-Output 's03_fuzz_campaign=FuzzUntrustedRequestMetadata:100x'
    Write-Output "s03_evidence_digest=$actualEvidenceDigest"
    Write-Output "source_revision=$sourceRevision"
    Write-Output 's03_verification=PASS'
}
finally {
    Pop-Location
}
