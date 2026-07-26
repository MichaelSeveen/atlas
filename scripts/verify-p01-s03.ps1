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
    if ($unformatted.Length -ne 0) {
        throw "Unformatted Go source:`n$unformatted"
    }
    Invoke-NativeChecked -Command 'go' -Arguments @('build', './cmd/api', './cmd/worker', './cmd/simulator', './cmd/envctl', './cmd/dbctl')
    Invoke-NativeChecked -Command 'go' -Arguments @('vet', './...')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './...', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/dbctl', 'verify', '--migration-dir', 'db/migrations')
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s05-migration-canary.ps1'))
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-p01-s03-persistence-canary.ps1'))
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'verify-p01-s02.ps1'))

    if ($Live) {
        & (Join-Path $PSScriptRoot 'p01-s03.ps1') -ContainerRuntime $ContainerRuntime
        Write-Output 'p01_s03_live_verification=PASS'
    }
    else {
        Write-Output 'p01_s03_live_verification=NOT_REQUESTED'
    }

    Write-Output 'p01_s03_scope=no-http-handler,no-oidc-exchange,no-event,no-worker-input,no-financial-state'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 'p01_s03_verification=PASS'
}
finally {
    Pop-Location
}
