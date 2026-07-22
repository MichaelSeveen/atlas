[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$lockPath = Join-Path $repositoryRoot 'tools/supply-chain.lock.json'
$toolRoot = Join-Path $repositoryRoot '.tmp/s07-tools'
$downloadRoot = Join-Path $toolRoot 'downloads'
$extractRoot = Join-Path $toolRoot 'extract'
$binRoot = Join-Path $toolRoot 'bin'

if (-not $IsWindows -and -not $IsLinux) {
    throw 'S07 tools support Windows and Linux only.'
}
if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne [System.Runtime.InteropServices.Architecture]::X64) {
    throw 'S07 tools are locked for amd64 only.'
}

$platform = if ($IsWindows) { 'windows_amd64' } else { 'linux_amd64' }
$executableSuffix = if ($IsWindows) { '.exe' } else { '' }
$lock = Get-Content -LiteralPath $lockPath -Raw | ConvertFrom-Json
if ($lock.schema_version -ne 1) { throw 'Unsupported supply-chain tool lock schema.' }

New-Item -ItemType Directory -Force $downloadRoot, $extractRoot, $binRoot | Out-Null

foreach ($name in @('gitleaks', 'gosec', 'syft', 'grype', 'cosign')) {
    $tool = $lock.tools.$name
    $artifact = $tool.$platform
    if ($null -eq $artifact -or $artifact.url -notmatch '^https://github\.com/') {
        throw "Missing approved $platform artifact for $name."
    }
    $uri = [Uri]$artifact.url
    $archiveName = [IO.Path]::GetFileName($uri.AbsolutePath)
    $archivePath = Join-Path $downloadRoot $archiveName
    $downloadRequired = $true
    if (Test-Path -LiteralPath $archivePath) {
        $cachedHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
        $downloadRequired = $cachedHash -ne $artifact.sha256
    }
    if ($downloadRequired) {
        Invoke-WebRequest -Uri $artifact.url -OutFile $archivePath
    }
    $observed = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($observed -ne $artifact.sha256) {
        throw "Checksum mismatch for $name $($tool.version): expected $($artifact.sha256), observed $observed."
    }

    $destination = Join-Path $binRoot ($name + $executableSuffix)
    if ($name -eq 'cosign') {
        Copy-Item -LiteralPath $archivePath -Destination $destination -Force
    }
    else {
        $toolExtract = Join-Path $extractRoot $name
        if (Test-Path -LiteralPath $toolExtract) {
            $resolved = [IO.Path]::GetFullPath($toolExtract)
            $resolvedRoot = [IO.Path]::GetFullPath($extractRoot) + [IO.Path]::DirectorySeparatorChar
            if (-not $resolved.StartsWith($resolvedRoot, [StringComparison]::OrdinalIgnoreCase)) {
                throw "Refusing to clear tool extraction outside $extractRoot."
            }
            Remove-Item -LiteralPath $resolved -Recurse -Force
        }
        New-Item -ItemType Directory -Force $toolExtract | Out-Null
        if ($archiveName.EndsWith('.zip', [StringComparison]::OrdinalIgnoreCase)) {
            Expand-Archive -LiteralPath $archivePath -DestinationPath $toolExtract -Force
        }
        else {
            & tar -xzf $archivePath -C $toolExtract
            if ($LASTEXITCODE -ne 0) { throw "Could not extract $archiveName." }
        }
        $candidate = Get-ChildItem -LiteralPath $toolExtract -Recurse -File |
            Where-Object { $_.Name -eq ($name + $executableSuffix) -or $_.Name -eq $name } |
            Select-Object -First 1
        if ($null -eq $candidate) { throw "Could not find $name executable in $archiveName." }
        Copy-Item -LiteralPath $candidate.FullName -Destination $destination -Force
    }
    if (-not $IsWindows) { & chmod 0755 $destination }
    Write-Output "s07_tool=$name@$($tool.version) sha256=$observed"
}

Write-Output "s07_tool_bin=$binRoot"
