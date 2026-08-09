[CmdletBinding()]
param(
    [Parameter()]
    [switch]$Live,

    [Parameter()]
    [switch]$History,

    [Parameter()]
    [switch]$SupplyChain,

    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman',

    [Parameter()]
    [ValidateSet('chrome', 'msedge', 'firefox', 'webkit')]
    [string]$Browser = 'msedge'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'

function Invoke-NativeChecked {
    param([string]$Command, [string[]]$Arguments = @())
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

Push-Location -LiteralPath $repositoryRoot
try {
    $baseRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $baseRevision -notmatch '^[0-9a-f]{40}$') {
        throw 'Phase 01 closure requires a valid committed base revision'
    }
    $changes = (& git status --porcelain=v1 --untracked-files=normal | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) { $baseRevision } else { "UNCOMMITTED_WORKTREE(base=$baseRevision)" }

    if ($Live) {
        & (Join-Path $PSScriptRoot 'verify-p01-s08.ps1') -Live -ContainerRuntime $ContainerRuntime
    }
    else {
        & (Join-Path $PSScriptRoot 'verify-p01-s08.ps1')
    }
    if (-not $?) { throw 'Phase 01 S08 cumulative regression failed' }

    $unformatted = (& gofmt -l ./cmd ./internal ./tests | Out-String).Trim()
    if ($unformatted.Length -ne 0) { throw "Unformatted Go source:`n$unformatted" }
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './...', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @('build', './cmd/api', './cmd/worker', './cmd/simulator')
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'run', './cmd/contractctl', 'lint',
        'docs/atlas-prd/03-contracts/openapi.yaml',
        'docs/atlas-prd/03-contracts/asyncapi.yaml'
    )

    Push-Location -LiteralPath (Join-Path $repositoryRoot 'apps/web')
    try {
        Invoke-NativeChecked -Command 'bun' -Arguments @('run', 'check:api')
        Invoke-NativeChecked -Command 'bun' -Arguments @('test')
        Invoke-NativeChecked -Command 'bun' -Arguments @('run', 'typecheck')
        Invoke-NativeChecked -Command 'bun' -Arguments @('run', 'build')
    }
    finally { Pop-Location }

    & (Join-Path $PSScriptRoot 'test-p01-s09-operational-catalog-canary.ps1')
    if (-not $?) { throw 'Phase 01 aggregate observability verification failed' }
    & (Join-Path $PSScriptRoot 'test-p01-s09-runbook-exercises.ps1')
    if (-not $?) { throw 'Phase 01 runbook exercise verification failed' }
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test', './internal/architecture', '-run',
        '^TestPhase01ClosurePolicyIsCompleteAndHonest$', '-count=1'
    )
    & (Join-Path $PSScriptRoot 'test-p01-evidence-integrity.ps1') `
        -CatalogueRelativePath 'evidence/phase-01/acceptance/P01-S09-evidence-catalogue-precommit.json' `
        -ExpectedSlice 'P01-S09'
    if (-not $?) { throw 'Phase 01 closure evidence integrity verification failed' }

    if ($Live -or $History -or $SupplyChain) {
        $phase00Arguments = @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'verify-s08.ps1'), '-HistoricalEvidence')
        if ($Live) { $phase00Arguments += '-Live' }
        if ($History) { $phase00Arguments += '-History' }
        if ($SupplyChain) { $phase00Arguments += '-SupplyChain' }
        if ($Live -or $SupplyChain) { $phase00Arguments += @('-ContainerRuntime', $ContainerRuntime) }
        Invoke-NativeChecked -Command 'pwsh' -Arguments $phase00Arguments
    }

    if ($Live) {
        & (Join-Path $PSScriptRoot 'p01-s04.ps1') -ContainerRuntime $ContainerRuntime
        if (-not $?) { throw 'Real PostgreSQL and synthetic Keycloak population journey failed' }
        & (Join-Path $PSScriptRoot 'test-p01-s09-personas.ps1')
        if (-not $?) { throw 'Support, risk, and finance persona journeys failed' }
        & (Join-Path $PSScriptRoot 'test-p01-s09-keycloak-browser.ps1') -Browser $Browser
        if (-not $?) { throw 'Real synthetic Keycloak browser journey failed' }
        & (Join-Path $PSScriptRoot 'test-p01-s09-browser.ps1') -Browser $Browser
        if (-not $?) { throw 'Phase 01 browser state and concurrency journey failed' }
        Write-Output 'p01_live_acceptance=PASS(customer,merchant,support,risk,finance,merchant-developer)'
        Write-Output 'p01_live_adversarial=PASS(ADV-IAM-001..015,absent-surfaces-fail-closed)'
        Write-Output 'p01_live_audit=PASS(synchronous-caller-transaction-chain-inspected)'
        Write-Output 'p01_live_recovery=PASS(product-seed-v5-and-phase00-full-restore)'
    }
    else {
        Write-Output 'p01_live_acceptance=NOT_REQUESTED(use -Live)'
    }

    Write-Output 'p01_requirements=VERIFIED(30/30,current-phase-scope)'
    Write-Output 'p01_iam002=VERIFIED(login,step-up,privilege-change,tenant-switch)'
    Write-Output 'p01_iam006=VERIFIED(current-actions;future-actions=reserved-fail-closed)'
    Write-Output 'p01_admin_removal=FAIL_CLOSED(policy-not-ratified)'
    Write-Output 'p01_break_glass=DISABLED'
    Write-Output 'p01_financial_behavior=ABSENT'
    Write-Output 'p01_production_claim=ABSENT'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 'p01_verification=PASS'
}
finally {
    Pop-Location
}
