[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman',

    [Parameter(Mandatory)]
    [string]$ToolBin
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$reportRoot = Join-Path $repositoryRoot '.tmp/s07-reports'
$head = (& git -C $repositoryRoot rev-parse HEAD | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $head -notmatch '^[0-9a-f]{40}$') { throw 'A valid Git source revision is required.' }
$dirty = ((& git -C $repositoryRoot status --porcelain=v1 | Out-String).Trim()).Length -ne 0
$sourceRevision = if ($dirty) { "UNCOMMITTED_WORKTREE(base=$head)" } else { $head }
$imageRevision = if ($dirty) { "uncommitted-$head" } else { $head }
$backendImage = "localhost/atlas-backend:$head"
$webImage = "localhost/atlas-web:$head"
$toolSuffix = if ($IsWindows) { '.exe' } else { '' }
$syft = Join-Path $ToolBin ("syft$toolSuffix")
$grype = Join-Path $ToolBin ("grype$toolSuffix")
. (Join-Path $PSScriptRoot 'compose.ps1')

if ($IsWindows -and $ContainerRuntime -eq 'podman') {
    & podman info --format '{{.Host.Arch}}' *> $null
    if ($LASTEXITCODE -ne 0) {
        $env:ATLAS_FORCE_PODMAN_WSL = 'true'
        Write-Output 's07_container_transport=podman-wsl-fallback'
    }
}

if (Test-Path -LiteralPath $reportRoot) {
    $resolved = [IO.Path]::GetFullPath($reportRoot)
    $expectedParent = [IO.Path]::GetFullPath((Join-Path $repositoryRoot '.tmp')) + [IO.Path]::DirectorySeparatorChar
    if (-not $resolved.StartsWith($expectedParent, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to clear report path $resolved."
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
New-Item -ItemType Directory -Force $reportRoot | Out-Null

function Invoke-NativeChecked([string]$Command, [string[]]$Arguments) {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Command failed: $Command $($Arguments -join ' ')" }
}

function Invoke-ContainerChecked([string[]]$Arguments) {
    Invoke-AtlasContainer -ContainerRuntime $ContainerRuntime -RepositoryRoot $repositoryRoot -Arguments $Arguments
}

function New-FrontendSPDX([string]$OutputPath) {
    $manifest = Get-Content -LiteralPath (Join-Path $repositoryRoot 'apps/web/package.json') -Raw | ConvertFrom-Json
    $packages = [Collections.Generic.List[object]]::new()
    $relationships = [Collections.Generic.List[object]]::new()
    $packages.Add([ordered]@{
        name = [string]$manifest.name; SPDXID = 'SPDXRef-Atlas-Web'; versionInfo = [string]$manifest.version
        downloadLocation = 'NOASSERTION'; filesAnalyzed = $false; licenseConcluded = 'NOASSERTION'
        licenseDeclared = 'NOASSERTION'; copyrightText = 'NOASSERTION'
    })
    $allPins = [ordered]@{}
    $packageManager = ([string]$manifest.packageManager).Split('@')
    $allPins['bun'] = $packageManager[$packageManager.Count - 1]
    foreach ($group in @('dependencies', 'devDependencies')) {
        foreach ($property in $manifest.$group.PSObject.Properties) {
            $allPins[$property.Name] = [string]$property.Value
        }
    }
    foreach ($entry in $allPins.GetEnumerator()) {
        $id = 'SPDXRef-' + ($entry.Key -replace '[^A-Za-z0-9.-]', '-')
        $packages.Add([ordered]@{
            name = $entry.Key; SPDXID = $id; versionInfo = $entry.Value
            downloadLocation = 'NOASSERTION'; filesAnalyzed = $false; licenseConcluded = 'NOASSERTION'
            licenseDeclared = 'NOASSERTION'; copyrightText = 'NOASSERTION'
        })
        $relationships.Add([ordered]@{ spdxElementId = 'SPDXRef-Atlas-Web'; relationshipType = 'DEPENDS_ON'; relatedSpdxElement = $id })
    }
    $document = [ordered]@{
        spdxVersion = 'SPDX-2.3'; dataLicense = 'CC0-1.0'; SPDXID = 'SPDXRef-DOCUMENT'
        name = "atlas-frontend-$head"; documentNamespace = "https://github.com/MichaelSeveen/atlas/sbom/frontend/$head"
        creationInfo = [ordered]@{ created = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'); creators = @('Tool: Atlas-S07-package-json-spdx/1') }
        packages = $packages; relationships = $relationships
    }
    $document | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $OutputPath -Encoding utf8NoBOM
}

Push-Location -LiteralPath $repositoryRoot
try {
    Invoke-ContainerChecked @('build', '--build-arg', "SOURCE_REVISION=$imageRevision", '--tag', $backendImage, '--file', 'deploy/local/Containerfile.backend', '.')
    Invoke-ContainerChecked @('build', '--build-arg', "SOURCE_REVISION=$imageRevision", '--tag', $webImage, '--file', 'apps/web/Containerfile', 'apps/web')

    $sboms = [ordered]@{
        backend_source = Join-Path $reportRoot 'backend-source.spdx.json'
        frontend_source = Join-Path $reportRoot 'frontend-source.spdx.json'
        backend_image = Join-Path $reportRoot 'backend-image.spdx.json'
        web_image = Join-Path $reportRoot 'web-image.spdx.json'
    }
    Invoke-NativeChecked $syft @('dir:.', '--exclude', './.git', '--exclude', './.tmp', '--exclude', './apps/web/node_modules', '-o', "spdx-json=$($sboms.backend_source)")
    New-FrontendSPDX $sboms.frontend_source
    $backendArchive = Join-Path $reportRoot 'backend-image.oci.tar'
    $webArchive = Join-Path $reportRoot 'web-image.oci.tar'
    if ($ContainerRuntime -eq 'podman') {
        Invoke-ContainerChecked @('save', '--format', 'oci-archive', '--output', '.tmp/s07-reports/backend-image.oci.tar', $backendImage)
        Invoke-ContainerChecked @('save', '--format', 'oci-archive', '--output', '.tmp/s07-reports/web-image.oci.tar', $webImage)
        Invoke-NativeChecked $syft @("oci-archive:$backendArchive", '-o', "spdx-json=$($sboms.backend_image)")
        Invoke-NativeChecked $syft @("oci-archive:$webArchive", '-o', "spdx-json=$($sboms.web_image)")
    }
    else {
        Invoke-ContainerChecked @('save', '--output', $backendArchive, $backendImage)
        Invoke-ContainerChecked @('save', '--output', $webArchive, $webImage)
        Invoke-NativeChecked $syft @("docker-archive:$backendArchive", '-o', "spdx-json=$($sboms.backend_image)")
        Invoke-NativeChecked $syft @("docker-archive:$webArchive", '-o', "spdx-json=$($sboms.web_image)")
    }

    $backendSBOM = Get-Content -LiteralPath $sboms.backend_source -Raw
    $frontendSBOM = Get-Content -LiteralPath $sboms.frontend_source -Raw
    $backendImageSBOM = Get-Content -LiteralPath $sboms.backend_image -Raw
    $webImageSBOM = Get-Content -LiteralPath $sboms.web_image -Raw
    if ($backendSBOM -notmatch 'github\.com/jackc/pgx') { throw 'Backend source SBOM does not identify the pinned PostgreSQL dependency.' }
    if ($frontendSBOM -notmatch 'react') { throw 'Frontend SBOM does not identify React.' }
    if ($backendImageSBOM -notmatch 'github\.com/MichaelSeveen/atlas') { throw 'Backend image SBOM does not identify the Atlas Go binaries.' }
    if ($webImageSBOM -notmatch '@atlas/web') { throw 'Web image SBOM does not identify the Atlas React package.' }
    foreach ($entry in $sboms.GetEnumerator()) {
        $source = Get-Content -LiteralPath $entry.Value -Raw
        if ($source -match '(?i)AGPL-|SSPL-') { throw "Denied license detected in $($entry.Key) SBOM." }
        $scanPath = Join-Path $reportRoot ($entry.Key.Replace('_', '-') + '.grype.json')
        Invoke-NativeChecked $grype @("sbom:$($entry.Value)", '--fail-on', 'critical', '--output', 'json', '--file', $scanPath)
    }

    $artifacts = [ordered]@{}
    foreach ($item in @(@{ name = 'backend'; image = $backendImage }, @{ name = 'web'; image = $webImage })) {
        $raw = (Invoke-AtlasContainer -ContainerRuntime $ContainerRuntime -RepositoryRoot $repositoryRoot -Arguments @('image', 'inspect', $item.image) | Out-String)
        $inspect = ($raw | ConvertFrom-Json)[0]
        if ([string]::IsNullOrWhiteSpace([string]$inspect.Config.User) -or [string]$inspect.Config.User -match '^(0|root)(:|$)') {
            throw "$($item.name) image does not declare a non-root user."
        }
        if ($inspect.Config.Labels.'org.opencontainers.image.revision' -ne $imageRevision) {
            throw "$($item.name) image revision label does not match the tested source."
        }
        Invoke-ContainerChecked @('run', '--rm', '--read-only', '--cap-drop=ALL', '--security-opt', 'no-new-privileges', '--entrypoint', '/bin/sh', $item.image, '-c', 'test ! -w /')
        $runtimeUID = (Invoke-AtlasContainer -ContainerRuntime $ContainerRuntime -RepositoryRoot $repositoryRoot -Arguments @('run', '--rm', '--read-only', '--cap-drop=ALL', '--security-opt', 'no-new-privileges', '--entrypoint', '/bin/sh', $item.image, '-c', 'id -u') | Out-String).Trim()
        if ($runtimeUID -eq '0' -or $runtimeUID -notmatch '^\d+$') {
            throw "$($item.name) image did not execute as a verifiable non-root user."
        }
        if ($item.name -eq 'web') {
            $bunVersion = (Invoke-AtlasContainer -ContainerRuntime $ContainerRuntime -RepositoryRoot $repositoryRoot -Arguments @('run', '--rm', '--read-only', '--cap-drop=ALL', '--security-opt', 'no-new-privileges', '--entrypoint', '/usr/local/bin/bun', $item.image, '--version') | Out-String).Trim()
            if ($bunVersion -ne '1.3.0') {
                throw "Web image Bun runtime mismatch: expected 1.3.0, observed $bunVersion."
            }
        }
        $digest = [string]$inspect.Digest
        if ($digest -notmatch '^sha256:[0-9a-f]{64}$') {
            $digest = [string]$inspect.Id
        }
        if ($digest -notmatch '^sha256:[0-9a-f]{64}$') { throw "$($item.name) image has no immutable SHA-256 identity." }
        $artifacts[$item.name] = [ordered]@{ image = $item.image; digest = $digest; user = [string]$inspect.Config.User }
    }

    $manifest = [ordered]@{
        schema_version = 1
        source_revision = $sourceRevision
        image_revision = $imageRevision
        base_revision = $head
        generated_at = [DateTime]::UtcNow.ToString('o')
        artifacts = $artifacts
        sbom_sha256 = [ordered]@{}
    }
    foreach ($entry in $sboms.GetEnumerator()) {
        $manifest.sbom_sha256[$entry.Key] = (Get-FileHash -LiteralPath $entry.Value -Algorithm SHA256).Hash.ToLowerInvariant()
    }
    $manifest | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $reportRoot 'manifest.json') -Encoding utf8NoBOM
    Write-Output "s07_supply_source_revision=$sourceRevision"
    Write-Output 's07_sbom_surfaces=backend-source,frontend-source,backend-image,web-image'
    Write-Output 's07_vulnerability_threshold=critical'
    Write-Output 's07_image_runtime=non-root,read-only,cap-drop,no-new-privileges'
    Write-Output 's07_supply_chain=PASS'
}
finally {
    Pop-Location
}
