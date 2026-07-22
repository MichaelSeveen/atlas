[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$precommitCataloguePath = Join-Path $repositoryRoot 'evidence/phase-00/acceptance/S08-evidence-catalogue-precommit.json'
$postcommitCataloguePath = Join-Path $repositoryRoot 'evidence/phase-00/acceptance/S08-evidence-catalogue-postcommit.json'
$cataloguePath = if (Test-Path -LiteralPath $postcommitCataloguePath) { $postcommitCataloguePath } else { $precommitCataloguePath }
$sidecarPath = "$cataloguePath.sha256"

function Get-SourceIdentity {
    param([string]$Root)
    $head = (& git -C $Root rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $head -notmatch '^[0-9a-f]{40}$') {
        throw 'Evidence verification requires a valid committed base revision.'
    }
    $changes = (& git -C $Root status --porcelain=v1 --untracked-files=normal | Out-String).Trim()
    if ($changes.Length -eq 0) { return $head }
    return "UNCOMMITTED_WORKTREE(base=$head)"
}

function Get-AcceptedSourceIdentities {
    param(
        [string]$Root,
        [string]$DeclaredSource
    )
    $currentSource = Get-SourceIdentity -Root $Root
    if ($DeclaredSource -eq $currentSource) { return @($currentSource) }
    if ($currentSource -notmatch '^[0-9a-f]{40}$' -or $DeclaredSource -notmatch '^[0-9a-f]{40}$') {
        throw "Evidence catalogue source revision is stale: got $DeclaredSource, want $currentSource"
    }
    & git -C $Root merge-base --is-ancestor $DeclaredSource $currentSource
    if ($LASTEXITCODE -ne 0) {
        throw "Evidence catalogue source revision is stale: $DeclaredSource is not an ancestor of $currentSource"
    }
    $allowedEvidenceChanges = @(
        'AGENTS.md',
        'docs/atlas-prd/06-governance/EVIDENCE_INDEX.md',
        'docs/atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv',
        'docs/atlas-prd/MANIFEST.sha256',
        'docs/engineering/IMPLEMENTATION_STATUS.md',
        'docs/engineering/PHASE-00-PLAN.md'
    )
    $changedPaths = @(& git -C $Root diff --name-only $DeclaredSource $currentSource)
    if ($LASTEXITCODE -ne 0) { throw 'Inspecting post-implementation evidence changes failed.' }
    foreach ($changedPath in $changedPaths) {
        $allowed = $changedPath -like 'evidence/phase-00/acceptance/*' -or $changedPath -in $allowedEvidenceChanges
        if (-not $allowed) {
            throw "Evidence catalogue source revision is stale because descendant code/config changed: $changedPath"
        }
    }
    return @($currentSource, $DeclaredSource)
}

function Test-Catalogue {
    param(
        [string]$Root,
        [string]$Path,
        [string[]]$AcceptedSources
    )
    $catalogue = Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
    if ($catalogue.schema_version -ne 1 -or $catalogue.evidence_id -ne 'EVD-P00-S08-001') {
        throw 'Evidence catalogue identity or schema is invalid.'
    }
    if ($catalogue.source_revision -notin $AcceptedSources) {
        throw "Evidence catalogue source revision is stale: got $($catalogue.source_revision)"
    }
    if ($catalogue.artifacts.Count -lt 10) {
        throw 'Evidence catalogue does not cover every Phase 00 slice and the S08 acceptance record.'
    }
    $seenIDs = @{}
    $seenPaths = @{}
    $rootPath = [IO.Path]::GetFullPath($Root).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    foreach ($artifact in $catalogue.artifacts) {
        if ($artifact.evidence_id -notmatch '^[A-Z0-9-]{5,64}$' -or $seenIDs.ContainsKey($artifact.evidence_id)) {
            throw 'Evidence catalogue contains an invalid or duplicate evidence ID.'
        }
        if ([IO.Path]::IsPathRooted($artifact.path) -or $artifact.path -match '(^|[\\/])\.\.([\\/]|$)' -or $seenPaths.ContainsKey($artifact.path)) {
            throw 'Evidence catalogue contains an unsafe or duplicate path.'
        }
        if ($artifact.sha256 -notmatch '^[0-9a-f]{64}$') {
            throw 'Evidence catalogue contains an invalid SHA-256 digest.'
        }
        $artifactPath = [IO.Path]::GetFullPath((Join-Path $Root $artifact.path))
        if (-not $artifactPath.StartsWith($rootPath, [StringComparison]::OrdinalIgnoreCase) -or -not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) {
            throw "Evidence artifact is absent or outside the repository root: $($artifact.path)"
        }
        $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $artifactPath).Hash.ToLowerInvariant()
        if ($actual -ne $artifact.sha256) {
            throw "Evidence artifact digest mismatch: $($artifact.path)"
        }
        $seenIDs[$artifact.evidence_id] = $true
        $seenPaths[$artifact.path] = $true
    }
}

function Assert-Failure {
    param([scriptblock]$Action, [string]$ExpectedMessage)
    try {
        & $Action
    }
    catch {
        if ($_.Exception.Message -notlike "*$ExpectedMessage*") { throw }
        return
    }
    throw "Seeded evidence failure was accepted: $ExpectedMessage"
}

$declaredSource = (Get-Content -LiteralPath $cataloguePath -Raw | ConvertFrom-Json).source_revision
$acceptedSources = Get-AcceptedSourceIdentities -Root $repositoryRoot -DeclaredSource $declaredSource
Test-Catalogue -Root $repositoryRoot -Path $cataloguePath -AcceptedSources $acceptedSources
$expectedCatalogueDigest = ((Get-Content -LiteralPath $sidecarPath -Raw).Trim() -split '\s+')[0]
$actualCatalogueDigest = (Get-FileHash -Algorithm SHA256 -LiteralPath $cataloguePath).Hash.ToLowerInvariant()
if ($expectedCatalogueDigest -ne $actualCatalogueDigest) {
    throw 'Evidence catalogue digest does not match its sidecar.'
}

$canaryParent = Join-Path $repositoryRoot '.tmp/s08-evidence-canary'
$canaryRoot = Join-Path $canaryParent ([Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $canaryRoot -Force | Out-Null
try {
    $catalogue = Get-Content -LiteralPath $cataloguePath -Raw | ConvertFrom-Json
    foreach ($artifact in $catalogue.artifacts) {
        $target = Join-Path $canaryRoot $artifact.path
        New-Item -ItemType Directory -Path (Split-Path -Parent $target) -Force | Out-Null
        Copy-Item -LiteralPath (Join-Path $repositoryRoot $artifact.path) -Destination $target
    }
    $canaryCatalogue = Join-Path $canaryRoot 'catalogue.json'
    Copy-Item -LiteralPath $cataloguePath -Destination $canaryCatalogue
    Test-Catalogue -Root $canaryRoot -Path $canaryCatalogue -AcceptedSources $acceptedSources

    $tamperedArtifact = Join-Path $canaryRoot $catalogue.artifacts[0].path
    Add-Content -LiteralPath $tamperedArtifact -Value 'synthetic-tamper-canary'
    Assert-Failure -ExpectedMessage 'digest mismatch' -Action {
        Test-Catalogue -Root $canaryRoot -Path $canaryCatalogue -AcceptedSources $acceptedSources
    }
    Copy-Item -LiteralPath (Join-Path $repositoryRoot $catalogue.artifacts[0].path) -Destination $tamperedArtifact -Force

    $catalogue.source_revision = '0000000000000000000000000000000000000000'
    $catalogue | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $canaryCatalogue -Encoding utf8
    Assert-Failure -ExpectedMessage 'source revision is stale' -Action {
        Test-Catalogue -Root $canaryRoot -Path $canaryCatalogue -AcceptedSources $acceptedSources
    }
}
finally {
    $resolvedCanary = [IO.Path]::GetFullPath($canaryRoot)
    $resolvedParent = [IO.Path]::GetFullPath($canaryParent).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    if ($resolvedCanary.StartsWith($resolvedParent, [StringComparison]::OrdinalIgnoreCase)) {
        Remove-Item -LiteralPath $resolvedCanary -Recurse -Force
    }
}

Write-Output "s08_evidence_catalogue_sha256=$actualCatalogueDigest"
Write-Output 's08_evidence_tamper_canary=PASS'
Write-Output 's08_evidence_stale-source_canary=PASS'
Write-Output 's08_evidence_integrity=PASS'
