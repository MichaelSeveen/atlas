[CmdletBinding()]
param(
    [Parameter()]
    [switch]$History,

    [Parameter()]
    [switch]$SupplyChain,

    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

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
    $baseRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw 'S07 requires a valid Git revision.' }
    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) { $baseRevision } else { "UNCOMMITTED_WORKTREE(base=$baseRevision)" }

    & (Join-Path $PSScriptRoot 'test-solo-maintainer-governance.ps1')
    if (-not $?) { throw 'Solo-maintainer governance canaries failed.' }

    $unformatted = (& gofmt -l ./cmd ./internal ./tests | Out-String).Trim()
    if ($unformatted.Length -ne 0) { throw "Unformatted Go source:`n$unformatted" }
    Invoke-NativeChecked 'go' @('build', './cmd/api', './cmd/worker', './cmd/simulator', './cmd/envctl', './cmd/dbctl', './cmd/contractctl')
    Invoke-NativeChecked 'go' @('vet', './...')
    Invoke-NativeChecked 'go' @('test', './...', '-count=1')
    $cgoEnabled = (& go env CGO_ENABLED | Out-String).Trim()
    if ($cgoEnabled -eq '1') {
        Invoke-NativeChecked 'go' @('test', '-race', './internal/architecture', './internal/contractcompat', './internal/platform/...', '-count=1')
        Write-Output 's07_race=PASS'
    }
    else {
        Write-Output 's07_race=NOT_AVAILABLE(cgo-disabled-host;required-on-GitHub-Linux)'
    }
    Invoke-NativeChecked 'go' @('run', './cmd/contractctl', 'lint', 'docs/atlas-prd/03-contracts/openapi.yaml', 'docs/atlas-prd/03-contracts/asyncapi.yaml')
    Invoke-NativeChecked 'go' @('run', './cmd/dbctl', 'verify', '--migration-dir', 'db/migrations')
    & (Join-Path $PSScriptRoot 'test-s07-live-contract-examples.ps1')
    if (-not $?) { throw 'Live contract example test failed.' }
    Invoke-NativeChecked 'bun' @('install', '--cwd', 'apps/web', '--frozen-lockfile')
    Invoke-NativeChecked 'bun' @('run', '--cwd', 'apps/web', 'lint')
    Invoke-NativeChecked 'bun' @('run', '--cwd', 'apps/web', 'test')
    Invoke-NativeChecked 'bun' @('run', '--cwd', 'apps/web', 'build')

    if ($History -or $SupplyChain) {
        & (Join-Path $PSScriptRoot 'bootstrap-s07-tools.ps1')
        if (-not $?) { throw 'S07 tool bootstrap failed.' }
        $toolBin = Join-Path $repositoryRoot '.tmp/s07-tools/bin'
    }

    if ($History) {
        $suffix = if ($IsWindows) { '.exe' } else { '' }
        $gitleaks = Join-Path $toolBin ("gitleaks$suffix")
        $gitleaksConfig = Join-Path $repositoryRoot '.gitleaks.toml'
        Invoke-NativeChecked $gitleaks @('dir', '.', '--config', $gitleaksConfig, '--no-banner', '--redact=100')
        Invoke-NativeChecked $gitleaks @('git', '.', '--config', $gitleaksConfig, '--no-banner', '--redact=100')
        & (Join-Path $PSScriptRoot 'test-s07-history-secret-canary.ps1') -GitleaksPath $gitleaks -GitleaksConfig $gitleaksConfig
        if (-not $?) { throw 'Deleted-history secret canary failed.' }

        $goTools = Get-Content -LiteralPath (Join-Path $repositoryRoot 'tools/go-tools.lock.json') -Raw | ConvertFrom-Json
        $gosec = Join-Path $toolBin ("gosec$suffix")
        if ($IsWindows) {
            # Gosec 2.25 does not complete within a bounded time on this Windows host.
            # GitHub Linux runs the full set and CodeQL supplies an independent taint lane.
            Write-Output 's07_gosec=NOT_AVAILABLE(unbounded-Windows-analysis;full-Gosec-and-CodeQL-required-on-GitHub-Linux)'
        }
        else {
            # Tool downloads and module/build caches live below .tmp. They are
            # verified independently and are not repository-owned source.
            Invoke-NativeChecked $gosec @('-quiet', '-exclude-generated', '-exclude-dir', '.tmp', './...')
            Write-Output 's07_gosec=PASS(full)'
        }
        Invoke-NativeChecked 'go' @('run', [string]$goTools.tools.govulncheck, './...')
        Write-Output 's07_security_scans=secret-worktree,secret-history,govulncheck,gosec-hosted'
    }
    else {
        Write-Output 's07_security_scans=NOT_REQUESTED(use -History)'
    }

    if ($SupplyChain) {
        & (Join-Path $PSScriptRoot 'test-s07-supply-chain.ps1') -ContainerRuntime $ContainerRuntime -ToolBin $toolBin
        if (-not $?) { throw 'S07 supply-chain test failed.' }
    }
    else {
        Write-Output 's07_supply_chain=NOT_REQUESTED(use -SupplyChain)'
    }

    Write-Output 's07_seeded_negatives=mutable-action,breaking-openapi,breaking-asyncapi,unresolved-reference,deleted-history-secret,sensitive-path,incomplete-solo-attestation'
    Write-Output 's07_named_skipped_tests=1,7'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 's07_hosted_enforcement=UNVERIFIED'
    Write-Output 's07_verification=PASS'
}
finally {
    Pop-Location
}
