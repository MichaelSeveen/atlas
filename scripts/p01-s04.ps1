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
    # The realm import is part of the live proof. A normal stop preserves the
    # Keycloak data volume and can silently skip newly source-controlled users.
    # Reset only the explicitly named synthetic local environment so every run
    # proves the checked-in realm from an empty provider store.
    & (Join-Path $PSScriptRoot 's04.ps1') `
        -Action Reset `
        -Confirmation 'RESET ATLAS LOCAL' `
        -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 's05.ps1') -Action Verify -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 'p01-s04-session-repository.ps1') -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 's04.ps1') -Action Up -ContainerRuntime $ContainerRuntime
    & (Join-Path $PSScriptRoot 'configure-p01-s04-keycloak.ps1')
    & (Join-Path $PSScriptRoot 'test-p01-s04-account-enumeration.ps1')
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
