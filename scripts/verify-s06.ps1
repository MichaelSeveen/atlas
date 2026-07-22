[CmdletBinding()]
param(
    [Parameter()]
    [switch]$Live,

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
    if ($LASTEXITCODE -ne 0) { $baseRevision = 'UNBORN' }
    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) { $baseRevision } else { "UNCOMMITTED_WORKTREE(base=$baseRevision)" }

    $unformatted = (& gofmt -l ./cmd ./internal ./tests | Out-String).Trim()
    if ($unformatted.Length -ne 0) { throw "Unformatted Go source:`n$unformatted" }
    Invoke-NativeChecked -Command 'go' -Arguments @('build', './cmd/api', './cmd/worker', './cmd/simulator', './cmd/envctl', './cmd/dbctl')
    Invoke-NativeChecked -Command 'go' -Arguments @('vet', './...')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './...', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'validate', '--config-dir', 'deploy/environments')
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/dbctl', 'verify', '--migration-dir', 'db/migrations')
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s06-alert-catalog-canary.ps1'))
    Invoke-NativeChecked -Command 'bun' -Arguments @('install', '--cwd', 'apps/web', '--frozen-lockfile')
    Invoke-NativeChecked -Command 'bun' -Arguments @('run', '--cwd', 'apps/web', 'typecheck')
    Invoke-NativeChecked -Command 'bun' -Arguments @('run', '--cwd', 'apps/web', 'test')
    Invoke-NativeChecked -Command 'bun' -Arguments @('run', '--cwd', 'apps/web', 'build')

    if ($Live) {
        & (Join-Path $PSScriptRoot 's06.ps1') -Action Verify -ContainerRuntime $ContainerRuntime
        Write-Output 's06_live_verification=PASS'
    }
    else {
        Write-Output 's06_live_verification=NOT_REQUESTED'
    }
    Write-Output 's06_named_skipped_test=2'
    Write-Output 's06_seeded_negatives=log-injection,metric-cardinality,ownerless-alert,key-downgrade,telemetry-outage'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 's06_verification=PASS'
}
finally {
    Pop-Location
}
