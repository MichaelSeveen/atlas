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
$composeFile = Join-Path $repositoryRoot 'deploy/local/compose.yaml'
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

function Wait-RedisPort {
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        $client = [Net.Sockets.TcpClient]::new()
        try {
            $connection = $client.ConnectAsync('127.0.0.1', 16379)
            if ($connection.Wait(1000) -and $client.Connected) {
                return
            }
        }
        catch {
            # The bounded loop owns the retry; no endpoint or credential is logged.
        }
        finally {
            $client.Dispose()
        }
        Start-Sleep -Milliseconds 250
    }
    throw 'Prepared local Redis service did not become reachable within the bounded deadline'
}

Push-Location -LiteralPath $repositoryRoot
try {
    $baseRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { $baseRevision = 'UNBORN' }
    $changes = (& git status --porcelain=v1 | Out-String).Trim()
    $sourceRevision = if ($changes.Length -eq 0) { $baseRevision } else { "UNCOMMITTED_WORKTREE(base=$baseRevision)" }

    if ($Live) {
        & (Join-Path $PSScriptRoot 'verify-p01-s07.ps1') -Live -ContainerRuntime $ContainerRuntime
    }
    else {
        & (Join-Path $PSScriptRoot 'verify-p01-s07.ps1')
    }
    if (-not $?) { throw 'Phase 01 S07 regression verification failed' }

    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test', './internal/identity/...', './cmd/api/internal/server',
        './tests/contract', './internal/architecture', '-count=1'
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

    & (Join-Path $PSScriptRoot 'test-p01-s08-credential-alert-catalog-canary.ps1')
    if (-not $?) { throw 'API credential observability catalogue verification failed' }

    $s08Catalogue = 'evidence/phase-01/api-credentials/P01-S08-evidence-catalogue-precommit.json'
    & (Join-Path $PSScriptRoot 'test-p01-evidence-integrity.ps1') `
        -CatalogueRelativePath $s08Catalogue `
        -ExpectedSlice 'P01-S08'
    if (-not $?) { throw 'Phase 01 S08 evidence integrity verification failed' }

    if ($Live) {
        $runtime = Read-RuntimeEnvironment
        foreach ($required in @(
            'ATLAS_POSTGRES_API_USER', 'ATLAS_POSTGRES_API_PASSWORD',
            'ATLAS_POSTGRES_MIGRATION_USER', 'ATLAS_POSTGRES_MIGRATION_PASSWORD',
            'ATLAS_POSTGRES_DB', 'ATLAS_REDIS_PASSWORD'
        )) {
            if (-not $runtime.ContainsKey($required) -or [String]::IsNullOrWhiteSpace($runtime[$required])) {
                throw "Prepared local runtime environment is missing $required"
            }
        }
        . (Join-Path $PSScriptRoot 'compose.ps1')
        Invoke-AtlasCompose -ContainerRuntime $ContainerRuntime -RuntimeFile $runtimeFile `
            -ComposeFile $composeFile -Arguments @('up', '--detach', 'redis')
        Wait-RedisPort
        $apiUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_USER'])
        $apiPassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_PASSWORD'])
        $migrationUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_MIGRATION_USER'])
        $migrationPassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_MIGRATION_PASSWORD'])
        $database = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_DB'])
        $env:ATLAS_P01_DATABASE_URL = "postgres://${apiUser}:${apiPassword}@127.0.0.1:15432/${database}?sslmode=disable"
        $env:ATLAS_P01_MIGRATION_DATABASE_URL = "postgres://${migrationUser}:${migrationPassword}@127.0.0.1:15432/${database}?sslmode=disable"
        $env:ATLAS_P01_REDIS_ADDR = '127.0.0.1:16379'
        $env:ATLAS_P01_REDIS_PASSWORD = $runtime['ATLAS_REDIS_PASSWORD']
        try {
            Invoke-NativeChecked -Command 'go' -Arguments @(
                'test', './internal/identity/persistence', '-run',
                '^TestAPICredentialRealPostgresOneTimeStateRotationRevocationAndAuditRollback$',
                '-count=1'
            )
            Invoke-NativeChecked -Command 'go' -Arguments @(
                'test', './internal/identity/ratelimit', '-run',
                '^TestRealRedisEnforcesCredentialTenantAndNetworkDimensions$',
                '-count=1'
            )
        }
        finally {
            Remove-Item Env:ATLAS_P01_DATABASE_URL -ErrorAction SilentlyContinue
            Remove-Item Env:ATLAS_P01_MIGRATION_DATABASE_URL -ErrorAction SilentlyContinue
            Remove-Item Env:ATLAS_P01_REDIS_ADDR -ErrorAction SilentlyContinue
            Remove-Item Env:ATLAS_P01_REDIS_PASSWORD -ErrorAction SilentlyContinue
            $runtime['ATLAS_POSTGRES_API_PASSWORD'] = $null
            $runtime['ATLAS_POSTGRES_MIGRATION_PASSWORD'] = $null
            $runtime['ATLAS_REDIS_PASSWORD'] = $null
        }
        Write-Output 'p01_s08_live_verification=PASS'
    }
    else {
        Write-Output 'p01_s08_live_verification=NOT_REQUESTED'
    }

    Write-Output 'p01_s08_scope=machine-credential,lifecycle,one-time-secret,tenant-environment-audience-scope-binding,three-dimensional-rate-limit'
    Write-Output 'p01_s08_secret_persistence=VERIFIER_ONLY'
    Write-Output 'p01_s08_redis_authority=ABSENT'
    Write-Output 'p01_s08_financial_scope=ABSENT'
    Write-Output 'p01_s08_production_secret_manager=ABSENT'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 'p01_s08_verification=PASS'
}
finally {
    Pop-Location
}
