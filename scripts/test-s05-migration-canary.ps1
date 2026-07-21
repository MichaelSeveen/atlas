[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$temporaryRoot = [IO.Path]::GetFullPath((Join-Path $repositoryRoot '.tmp'))
$canaryRoot = [IO.Path]::GetFullPath((Join-Path $temporaryRoot 's05-migration-canary'))
if (-not $canaryRoot.StartsWith($temporaryRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Migration canary target escaped the repository temporary root'
}

function Reset-Canary {
    if (Test-Path -LiteralPath $canaryRoot) {
        Remove-Item -LiteralPath $canaryRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Path $canaryRoot -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'db/migrations') -Destination $canaryRoot -Recurse
}

function Assert-VerificationFails {
    & go run ./cmd/dbctl verify --migration-dir (Join-Path $canaryRoot 'migrations') *> $null
    if ($LASTEXITCODE -eq 0) {
        throw 'Seeded released-migration violation was accepted'
    }
}

Push-Location -LiteralPath $repositoryRoot
try {
    Reset-Canary
    Add-Content -LiteralPath (Join-Path $canaryRoot 'migrations/000001_foundation_control_schema.sql') -Value '-- seeded immutable-history mutation'
    Assert-VerificationFails
    Write-Output 'migration_checksum_mutation=KILLED'

    Reset-Canary
    Remove-Item -LiteralPath (Join-Path $canaryRoot 'migrations/000002_recovery_probe.metadata.json') -Force
    Assert-VerificationFails
    Write-Output 'migration_metadata_deletion=KILLED'
}
finally {
    Pop-Location
    if (Test-Path -LiteralPath $canaryRoot) {
        Remove-Item -LiteralPath $canaryRoot -Recurse -Force
    }
}
