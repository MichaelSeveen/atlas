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

Push-Location -LiteralPath $repositoryRoot
try {
    & (Join-Path $PSScriptRoot 's05.ps1') -Action Verify -ContainerRuntime $ContainerRuntime

    $runtime = Read-RuntimeEnvironment
    foreach ($required in @('ATLAS_POSTGRES_API_USER', 'ATLAS_POSTGRES_API_PASSWORD', 'ATLAS_POSTGRES_DB')) {
        if (-not $runtime.ContainsKey($required) -or [String]::IsNullOrWhiteSpace($runtime[$required])) {
            throw "Prepared local runtime environment is missing $required"
        }
    }
    $databaseUser = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_USER'])
    $databasePassword = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_API_PASSWORD'])
    $databaseName = [Uri]::EscapeDataString($runtime['ATLAS_POSTGRES_DB'])
    $env:ATLAS_P01_DATABASE_URL = "postgres://${databaseUser}:${databasePassword}@127.0.0.1:15432/${databaseName}?sslmode=disable"
    try {
        Invoke-NativeChecked -Command 'go' -Arguments @(
            'test',
            './internal/identity/persistence',
            '-run',
            '^TestMembershipRepositoryRealPostgresTenantIsolation$',
            '-count=1'
        )
    }
    finally {
        Remove-Item Env:ATLAS_P01_DATABASE_URL -ErrorAction SilentlyContinue
    }

    Write-Output 'p01_s03_real_postgres_repository=PASS'
    Write-Output 'p01_s03_live=PASS'
}
finally {
    Pop-Location
}
