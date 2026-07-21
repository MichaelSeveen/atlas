Set-StrictMode -Version Latest

function ConvertTo-AtlasWslPath {
    param([Parameter(Mandatory)][string]$Path)

    $absolute = [IO.Path]::GetFullPath($Path)
    if ($absolute.Length -lt 4 -or $absolute[1] -ne ':' -or $absolute[2] -ne [IO.Path]::DirectorySeparatorChar) {
        throw 'Podman WSL fallback requires an absolute drive path'
    }
    $drive = [char]::ToLowerInvariant($absolute[0])
    $tail = $absolute.Substring(3).Replace([IO.Path]::DirectorySeparatorChar, '/')
    return "/mnt/$drive/$tail"
}

function Invoke-AtlasCompose {
    param(
        [Parameter(Mandatory)][ValidateSet('podman', 'docker')][string]$ContainerRuntime,
        [Parameter(Mandatory)][string]$RuntimeFile,
        [Parameter(Mandatory)][string]$ComposeFile,
        [Parameter()][string[]]$Arguments = @()
    )

    if ($ContainerRuntime -eq 'docker' -or
        $null -ne (Get-Command 'podman-compose' -ErrorAction SilentlyContinue) -or
        $null -ne (Get-Command 'docker-compose' -ErrorAction SilentlyContinue)) {
        $commandArguments = @('compose', '--env-file', $RuntimeFile, '--file', $ComposeFile) + $Arguments
        & $ContainerRuntime @commandArguments
        if ($LASTEXITCODE -ne 0) {
            throw "Compose command failed with exit code ${LASTEXITCODE}: $($Arguments -join ' ')"
        }
        return
    }

    $wslRuntimeFile = ConvertTo-AtlasWslPath -Path $RuntimeFile
    $wslComposeFile = ConvertTo-AtlasWslPath -Path $ComposeFile
    $wslArguments = @(
        '-d', 'podman-machine-default', '-u', 'root', '--',
        'podman-compose', '--env-file', $wslRuntimeFile, '--file', $wslComposeFile
    ) + $Arguments
    & wsl.exe @wslArguments
    if ($LASTEXITCODE -ne 0) {
        throw "Podman WSL compose command failed with exit code ${LASTEXITCODE}: $($Arguments -join ' ')"
    }
}
