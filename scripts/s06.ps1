[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidateSet('Up', 'Verify', 'Down')]
    [string]$Action,

    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

switch ($Action) {
    'Up' {
        & (Join-Path $PSScriptRoot 's05.ps1') -Action Migrate -ContainerRuntime $ContainerRuntime
        & (Join-Path $PSScriptRoot 's04.ps1') -Action Up -ContainerRuntime $ContainerRuntime
        Write-Output 's06_environment_up=PASS'
    }
    'Verify' {
        & (Join-Path $PSScriptRoot 's05.ps1') -Action Migrate -ContainerRuntime $ContainerRuntime
        & (Join-Path $PSScriptRoot 's04.ps1') -Action Up -ContainerRuntime $ContainerRuntime
        & (Join-Path $PSScriptRoot 'test-s06-live.ps1') -ContainerRuntime $ContainerRuntime
        Write-Output 's06_environment_verify=PASS'
    }
    'Down' {
        & (Join-Path $PSScriptRoot 's04.ps1') -Action Down -ContainerRuntime $ContainerRuntime
        Write-Output 's06_environment_down=PASS'
    }
}
