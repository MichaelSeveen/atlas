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
    if ($LASTEXITCODE -ne 0) {
        $baseRevision = 'UNBORN'
    }
    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) {
        $baseRevision
    }
    else {
        "UNCOMMITTED_WORKTREE(base=$baseRevision)"
    }

    $unformatted = (& gofmt -l ./cmd ./internal ./tests | Out-String).Trim()
    if ($unformatted.Length -ne 0) {
        throw "Unformatted Go source:`n$unformatted"
    }
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'build',
        './cmd/api',
        './cmd/worker',
        './cmd/simulator',
        './cmd/envctl',
        './cmd/dbctl',
        './cmd/contractctl'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @('vet', './...')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './...', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'run',
        './cmd/contractctl',
        'lint',
        'docs/atlas-prd/03-contracts/openapi.yaml',
        'docs/atlas-prd/03-contracts/asyncapi.yaml'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'run',
        './cmd/dbctl',
        'verify',
        '--migration-dir',
        'db/migrations'
    )

    Push-Location -LiteralPath (Join-Path $repositoryRoot 'apps/web')
    try {
        Invoke-NativeChecked -Command 'bun' -Arguments @('test')
        Invoke-NativeChecked -Command 'bun' -Arguments @('run', 'build')
    }
    finally {
        Pop-Location
    }

    & (Join-Path $PSScriptRoot 'verify-p01-s03.ps1')
    if (-not $?) {
        throw 'Phase 01 S03 regression verification failed'
    }

    if ($Live) {
        & (Join-Path $PSScriptRoot 'p01-s04.ps1') -ContainerRuntime $ContainerRuntime
        Write-Output 'p01_s04_live_verification=PASS'
    }
    else {
        Write-Output 'p01_s04_live_verification=NOT_REQUESTED'
    }

    Write-Output 'p01_s04_scope=oidc-bff,durable-session,current-principal,revocation,step-up-boundary'
    Write-Output 'p01_s04_financial_state=ABSENT'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 'p01_s04_verification=PASS'
}
finally {
    Pop-Location
}
