[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
$composeFile = Join-Path $repositoryRoot 'deploy/local/compose.yaml'
$testDatabase = 'atlas_p01_s04_session_test'
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'
. (Join-Path $PSScriptRoot 'compose.ps1')

function Read-RuntimeEnvironment {
    if (-not (Test-Path -LiteralPath $runtimeFile -PathType Leaf)) {
        throw 'Prepared local runtime environment is absent'
    }
    $values = @{}
    foreach ($line in Get-Content -LiteralPath $runtimeFile) {
        if ([String]::IsNullOrWhiteSpace($line) -or $line.StartsWith('#')) {
            continue
        }
        $parts = $line.Split('=', 2)
        if ($parts.Count -ne 2 -or [String]::IsNullOrWhiteSpace($parts[0])) {
            throw 'Prepared local runtime environment contains a malformed entry'
        }
        $values[$parts[0]] = $parts[1]
    }
    return $values
}

function Invoke-SessionDatabaseAction {
    param([Parameter(Mandatory)][ValidateSet('create', 'drop')][string]$Action)

    Invoke-AtlasCompose `
        -ContainerRuntime $ContainerRuntime `
        -RuntimeFile $runtimeFile `
        -ComposeFile $composeFile `
        -Arguments @(
            'exec',
            '-T',
            '-e',
            "ATLAS_P01_S04_SESSION_TEST_ACTION=$Action",
            'postgres',
            'sh',
            '/database/tests/phase01_session_repository.sh'
        )
}

function Invoke-NativeChecked {
    param([string]$Command, [string[]]$Arguments = @())

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

Push-Location -LiteralPath $repositoryRoot
try {
    $runtime = Read-RuntimeEnvironment
    foreach ($required in @(
        'ATLAS_POSTGRES_API_USER',
        'ATLAS_POSTGRES_API_PASSWORD',
        'ATLAS_POSTGRES_MIGRATION_USER',
        'ATLAS_POSTGRES_MIGRATION_PASSWORD'
    )) {
        if (-not $runtime.ContainsKey($required) -or [String]::IsNullOrWhiteSpace($runtime[$required])) {
            throw "Prepared local runtime environment is missing $required"
        }
    }

    $databasePrepared = $false
    try {
        Invoke-SessionDatabaseAction -Action create
        $databasePrepared = $true

        $apiUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_USER'])
        $apiPassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_PASSWORD'])
        $migrationUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_MIGRATION_USER'])
        $migrationPassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_MIGRATION_PASSWORD'])
        $database = [Uri]::EscapeDataString($testDatabase)
        $env:ATLAS_P01_DATABASE_URL = "postgres://${apiUser}:${apiPassword}@127.0.0.1:15432/${database}?sslmode=disable"
        $env:ATLAS_P01_MIGRATION_DATABASE_URL = "postgres://${migrationUser}:${migrationPassword}@127.0.0.1:15432/${database}?sslmode=disable"
        try {
            Invoke-NativeChecked -Command 'go' -Arguments @(
                'test',
                './internal/identity/persistence',
                '-run',
                '^TestSessionStoreRealPostgresOIDCRevocationAndAuthorityInvalidation$',
                '-count=1'
            )
        }
        finally {
            Remove-Item Env:ATLAS_P01_DATABASE_URL -ErrorAction SilentlyContinue
            Remove-Item Env:ATLAS_P01_MIGRATION_DATABASE_URL -ErrorAction SilentlyContinue
        }
    }
    finally {
        if ($databasePrepared) {
            Invoke-SessionDatabaseAction -Action drop
        }
        if ($runtime.ContainsKey('ATLAS_POSTGRES_API_PASSWORD')) {
            $runtime['ATLAS_POSTGRES_API_PASSWORD'] = $null
        }
        if ($runtime.ContainsKey('ATLAS_POSTGRES_MIGRATION_PASSWORD')) {
            $runtime['ATLAS_POSTGRES_MIGRATION_PASSWORD'] = $null
        }
    }

    Write-Output 'p01_s04_real_postgres_session_repository=PASS'
}
finally {
    Pop-Location
}
