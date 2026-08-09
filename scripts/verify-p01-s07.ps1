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
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
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

function Read-RuntimeEnvironment {
    if (-not (Test-Path -LiteralPath $runtimeFile -PathType Leaf)) {
        throw 'Prepared local runtime environment is absent'
    }
    $values = @{}
    foreach ($line in Get-Content -LiteralPath $runtimeFile) {
        if ([String]::IsNullOrWhiteSpace($line) -or $line.StartsWith('#')) { continue }
        $parts = $line.Split('=', 2)
        if ($parts.Count -ne 2 -or [String]::IsNullOrWhiteSpace($parts[0])) {
            throw 'Prepared local runtime environment contains a malformed entry'
        }
        $values[$parts[0]] = $parts[1]
    }
    return $values
}

Push-Location -LiteralPath $repositoryRoot
try {
    $baseRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { $baseRevision = 'UNBORN' }
    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) { $baseRevision } else { "UNCOMMITTED_WORKTREE(base=$baseRevision)" }

    if ($Live) {
        & (Join-Path $PSScriptRoot 'verify-p01-s06.ps1') -Live -ContainerRuntime $ContainerRuntime
    }
    else {
        & (Join-Path $PSScriptRoot 'verify-p01-s06.ps1')
    }
    if (-not $?) { throw 'Phase 01 S06 regression verification failed' }

    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test', './internal/operations/...', './internal/identity/...',
        './cmd/api/internal/server', './tests/contract', './internal/architecture', '-count=1'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'run', './cmd/contractctl', 'lint',
        'docs/atlas-prd/03-contracts/openapi.yaml',
        'docs/atlas-prd/03-contracts/asyncapi.yaml'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'run', './cmd/dbctl', 'verify', '--migration-dir', 'db/migrations'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'build', './cmd/api', './cmd/worker', './cmd/simulator'
    )

    & (Join-Path $PSScriptRoot 'test-s07-approval-alert-catalog-canary.ps1')
    if (-not $?) { throw 'Approval observability catalogue verification failed' }

    $s07Catalogue = 'evidence/phase-01/approvals/P01-S07-evidence-catalogue-postcommit.json'
    & (Join-Path $PSScriptRoot 'test-p01-evidence-integrity.ps1') `
        -CatalogueRelativePath $s07Catalogue `
        -ExpectedSlice 'P01-S07'
    if (-not $?) { throw 'Phase 01 S07 evidence integrity verification failed' }

    if ($Live) {
        $runtime = Read-RuntimeEnvironment
        foreach ($required in @(
            'ATLAS_POSTGRES_API_USER', 'ATLAS_POSTGRES_API_PASSWORD',
            'ATLAS_POSTGRES_MIGRATION_USER', 'ATLAS_POSTGRES_MIGRATION_PASSWORD',
            'ATLAS_POSTGRES_DB'
        )) {
            if (-not $runtime.ContainsKey($required) -or [String]::IsNullOrWhiteSpace($runtime[$required])) {
                throw "Prepared local runtime environment is missing $required"
            }
        }
        $apiUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_USER'])
        $apiPassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_PASSWORD'])
        $migrationUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_MIGRATION_USER'])
        $migrationPassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_MIGRATION_PASSWORD'])
        $database = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_DB'])
        $env:ATLAS_P01_DATABASE_URL = "postgres://${apiUser}:${apiPassword}@127.0.0.1:15432/${database}?sslmode=disable"
        $env:ATLAS_P01_MIGRATION_DATABASE_URL = "postgres://${migrationUser}:${migrationPassword}@127.0.0.1:15432/${database}?sslmode=disable"
        try {
            Invoke-NativeChecked -Command 'go' -Arguments @(
                'test', './internal/operations/persistence', '-run',
                '^TestApprovalRealPostgresSeparationReauthorizationIntegrityConcurrencyAndAuditRollback$',
                '-count=1'
            )
        }
        finally {
            Remove-Item Env:ATLAS_P01_DATABASE_URL -ErrorAction SilentlyContinue
            Remove-Item Env:ATLAS_P01_MIGRATION_DATABASE_URL -ErrorAction SilentlyContinue
            $runtime['ATLAS_POSTGRES_API_PASSWORD'] = $null
            $runtime['ATLAS_POSTGRES_MIGRATION_PASSWORD'] = $null
        }
        & (Join-Path $PSScriptRoot 's05.ps1') -Action Verify -ContainerRuntime $ContainerRuntime
        if (-not $?) { throw 'Phase 01 S07 database role and recovery verification failed' }
        Write-Output 'p01_s07_live_verification=PASS'
    }
    else {
        Write-Output 'p01_s07_live_verification=NOT_REQUESTED'
    }

    Write-Output 'p01_s07_scope=typed-approval,maker-checker,payload-integrity,execution-reauthorization,terminal-state,replay'
    Write-Output 'p01_s07_executable_action=identity.organization.membership.change_admin'
    Write-Output 'p01_s07_approval_event=ABSENT'
    Write-Output 'p01_s07_worker_job=ABSENT'
    Write-Output 'p01_s07_credential_behavior=ABSENT'
    Write-Output 'p01_s07_financial_state=ABSENT'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 'p01_s07_verification=PASS'
}
finally {
    Pop-Location
}
