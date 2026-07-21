[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$temporaryRoot = Join-Path $repositoryRoot '.tmp/s04-seed-canary'
$source = Join-Path $repositoryRoot 'deploy/seeds/foundation.json'
$mutant = Join-Path $temporaryRoot 'unknown-tenant.json'
$repositoryPrefix = [IO.Path]::GetFullPath($repositoryRoot).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$resolvedTemporaryRoot = [IO.Path]::GetFullPath($temporaryRoot)
if (-not $resolvedTemporaryRoot.StartsWith($repositoryPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'S04 seed canary target escapes the repository'
}

Push-Location -LiteralPath $repositoryRoot
try {
    if (Test-Path -LiteralPath $temporaryRoot) {
        throw 'S04 seed canary target already exists; refusing an ambiguous cleanup'
    }
    New-Item -ItemType Directory -Path $temporaryRoot | Out-Null
    $document = Get-Content -LiteralPath $source -Raw | ConvertFrom-Json
    $document.users[0].tenant_id = '01J00000000000000000000000'
    $document | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $mutant -Encoding utf8NoBOM
    & go run ./cmd/envctl seed-checksum --manifest $mutant 2>$null
    if ($LASTEXITCODE -eq 0) {
        throw 'Unknown-tenant seed mutation survived validation'
    }
    Write-Output 'seed_canary=UNKNOWN_TENANT_REFERENCE'
    Write-Output 'seed_canary_expected=validation_failure'
    Write-Output 'seed_canary_observed=validation_failure'
    Write-Output 'seed_canary=KILLED'
}
finally {
    if (Test-Path -LiteralPath $temporaryRoot) {
        Remove-Item -LiteralPath $temporaryRoot -Recurse -Force
    }
    Pop-Location
}
