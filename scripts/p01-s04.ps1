[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot

Push-Location -LiteralPath $repositoryRoot
try {
    & (Join-Path $PSScriptRoot 's04.ps1') -Action Down -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 's05.ps1') -Action Verify -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 'p01-s04-session-repository.ps1') -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 's04.ps1') -Action Up -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 'configure-p01-s04-keycloak.ps1')
    foreach ($population in @('customer', 'merchant', 'workforce')) {
        & (Join-Path $PSScriptRoot 'test-p01-s04-oidc-http.ps1') -Population $population
    }

    Write-Output 'p01_s04_live_database=REAL_POSTGRESQL'
    Write-Output 'p01_s04_live_identity_provider=SYNTHETIC_KEYCLOAK'
    Write-Output 'p01_s04_live=PASS'
}
finally {
    Pop-Location
}
