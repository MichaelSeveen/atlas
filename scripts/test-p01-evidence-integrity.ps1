[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = [IO.Path]::GetFullPath((Split-Path -Parent $PSScriptRoot))
$policyPath = Join-Path $repositoryRoot 'docs/engineering/phase-01-evidence-policy.json'
$policy = Get-Content -LiteralPath $policyPath -Raw | ConvertFrom-Json
$postcommitPath = Join-Path $repositoryRoot $policy.phase_01_catalogue.postcommit_path
$precommitPath = Join-Path $repositoryRoot $policy.phase_01_catalogue.precommit_path
$cataloguePath = if (Test-Path -LiteralPath $postcommitPath) { $postcommitPath } else { $precommitPath }

function Get-CurrentSourceIdentity {
    $revision = (& git -C $repositoryRoot rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $revision -notmatch '^[a-f0-9]{40}$') {
        throw 'Phase 01 evidence requires a 40-hex Git base revision.'
    }
    $changes = (& git -C $repositoryRoot status --porcelain=v1 | Out-String).Trim()
    if ($changes.Length -eq 0) { return $revision }
    return "UNCOMMITTED_WORKTREE(base=$revision)"
}

function Test-AllowedPostcommitEvidencePath([string]$Relative) {
    $normalized = $Relative.Replace('\', '/')
    if ($normalized.StartsWith('evidence/phase-01/identity-session/', [StringComparison]::Ordinal)) {
        return $true
    }
    return $normalized -in @(
        'AGENTS.md',
        'docs/atlas-prd/06-governance/EVIDENCE_INDEX.md',
        'docs/atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv',
        'docs/atlas-prd/MANIFEST.sha256',
        'docs/engineering/IMPLEMENTATION_STATUS.md',
        'docs/engineering/PHASE-01-PLAN.md'
    )
}

function Get-AcceptedSourceIdentities([string]$DeclaredSource) {
    $currentSource = Get-CurrentSourceIdentity
    if ($DeclaredSource -eq $currentSource) {
        return @($currentSource)
    }
    if ($DeclaredSource -notmatch '^[a-f0-9]{40}$') {
        throw "Stale evidence source identity: $DeclaredSource"
    }

    $currentHead = (& git -C $repositoryRoot rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $currentHead -notmatch '^[a-f0-9]{40}$') {
        throw 'Phase 01 post-commit evidence requires a committed current revision.'
    }
    & git -C $repositoryRoot merge-base --is-ancestor $DeclaredSource $currentHead
    if ($LASTEXITCODE -ne 0) {
        throw "Stale evidence source identity: $DeclaredSource is not an ancestor of $currentHead"
    }

    $changedPaths = @()
    if ($DeclaredSource -ne $currentHead) {
        $changedPaths += @(& git -C $repositoryRoot diff --name-only $DeclaredSource $currentHead)
        if ($LASTEXITCODE -ne 0) {
            throw 'Inspecting committed Phase 01 evidence changes failed.'
        }
    }
    $changedPaths += @(& git -C $repositoryRoot diff --name-only $currentHead)
    if ($LASTEXITCODE -ne 0) {
        throw 'Inspecting dirty Phase 01 evidence changes failed.'
    }
    $changedPaths += @(& git -C $repositoryRoot ls-files --others --exclude-standard)
    if ($LASTEXITCODE -ne 0) {
        throw 'Inspecting untracked Phase 01 evidence changes failed.'
    }
    foreach ($changedPath in @($changedPaths | Select-Object -Unique)) {
        if (-not (Test-AllowedPostcommitEvidencePath $changedPath)) {
            throw "Stale evidence source identity because code/config changed after ${DeclaredSource}: $changedPath"
        }
    }
    return @($DeclaredSource)
}

function Test-SafeRelativePath([string]$Relative) {
    if ([IO.Path]::IsPathRooted($Relative)) { return $false }
    $normalized = $Relative.Replace('\', '/')
    if ($normalized -match '(^|/)\.\.(/|$)') { return $false }
    $resolved = [IO.Path]::GetFullPath((Join-Path $repositoryRoot $Relative))
    return $resolved.StartsWith(($repositoryRoot.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar), [StringComparison]::OrdinalIgnoreCase)
}

function Assert-Catalogue([object]$Catalogue, [string[]]$AcceptedSources, [bool]$VerifyFiles) {
    if ($Catalogue.schema_version -ne 1 -or
        $Catalogue.phase -ne 'PHASE-01_IDENTITY_ACCESS_TENANCY' -or
        $Catalogue.slice -ne 'P01-S04') {
        throw 'Phase 01 evidence catalogue identity is invalid.'
    }
    if ($Catalogue.source_revision -notin $AcceptedSources) {
        throw "Stale evidence source identity: $($Catalogue.source_revision)"
    }
    if ([string]::IsNullOrWhiteSpace([string]$Catalogue.sanitization)) {
        throw 'Evidence sanitization statement is required.'
    }
    $seen = @{}
    foreach ($artifact in @($Catalogue.artifacts)) {
        $evidenceID = [string]$artifact.evidence_id
        if ($evidenceID -notmatch '^EVD-P01-S0[1-4]-[A-Z0-9-]+$') {
            throw "Invalid evidence ID: $evidenceID"
        }
        if ($seen.ContainsKey($evidenceID)) {
            throw "Duplicate evidence ID: $evidenceID"
        }
        $seen[$evidenceID] = $true
        if (-not (Test-SafeRelativePath ([string]$artifact.path))) {
            throw "Unsafe artifact path: $($artifact.path)"
        }
        if ([string]$artifact.sha256 -notmatch '^[a-f0-9]{64}$') {
            throw "Invalid artifact digest for $evidenceID"
        }
        if ($artifact.result -ne 'PASS') {
            throw "Artifact does not pass: $evidenceID"
        }
        if ($VerifyFiles) {
            $artifactPath = Join-Path $repositoryRoot ([string]$artifact.path)
            if (-not (Test-Path -LiteralPath $artifactPath -PathType Leaf)) {
                throw "Missing evidence artifact: $($artifact.path)"
            }
            $actual = (Get-FileHash -LiteralPath $artifactPath -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actual -ne [string]$artifact.sha256) {
                throw "Evidence artifact digest mismatch: $evidenceID"
            }
        }
    }
    if ($seen.Count -lt 4) {
        throw 'Current Phase 01 evidence catalogue is incomplete.'
    }
}

function Copy-JsonObject([object]$Value) {
    return ($Value | ConvertTo-Json -Depth 20 | ConvertFrom-Json)
}

function Assert-CanaryRejected([scriptblock]$Mutation, [string]$Label, [string[]]$AcceptedSources) {
    $candidate = Copy-JsonObject $script:catalogue
    & $Mutation $candidate
    $rejected = $false
    try {
        Assert-Catalogue $candidate $AcceptedSources $true
    }
    catch {
        $rejected = $true
    }
    if (-not $rejected) { throw "Evidence canary was accepted: $Label" }
    Write-Output "p01_evidence_canary=$Label`:PASS(rejected)"
}

Push-Location -LiteralPath $repositoryRoot
try {
    if (-not (Test-Path -LiteralPath $cataloguePath -PathType Leaf)) {
        throw "Phase 01 evidence catalogue is missing: $cataloguePath"
    }
    $script:catalogue = Get-Content -LiteralPath $cataloguePath -Raw | ConvertFrom-Json
    $acceptedSources = Get-AcceptedSourceIdentities ([string]$script:catalogue.source_revision)
    Assert-Catalogue $script:catalogue $acceptedSources $true

    $sidecarPath = "$cataloguePath.sha256"
    if (-not (Test-Path -LiteralPath $sidecarPath -PathType Leaf)) {
        throw "Evidence catalogue sidecar is missing: $sidecarPath"
    }
    $expectedCatalogueHash = ((Get-Content -LiteralPath $sidecarPath -Raw).Trim() -split '\s+')[0].ToLowerInvariant()
    $actualCatalogueHash = (Get-FileHash -LiteralPath $cataloguePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expectedCatalogueHash -ne $actualCatalogueHash) {
        throw 'Evidence catalogue sidecar digest mismatch.'
    }

    Assert-CanaryRejected { param($c) $c.artifacts[0].sha256 = ('0' * 64) } 'artifact-tamper' $acceptedSources
    Assert-CanaryRejected { param($c) $c.source_revision = ('0' * 40) } 'stale-source' $acceptedSources
    Assert-CanaryRejected {
        param($c)
        $c.artifacts = @($c.artifacts) + @(Copy-JsonObject $c.artifacts[0])
    } 'duplicate-evidence-id' $acceptedSources
    Assert-CanaryRejected { param($c) $c.artifacts[0].path = '../escape.txt' } 'unsafe-artifact-path' $acceptedSources

    Write-Output "p01_evidence_catalogue=$([IO.Path]::GetRelativePath($repositoryRoot, $cataloguePath).Replace('\', '/'))"
    Write-Output "p01_evidence_source=$([string]$script:catalogue.source_revision)"
    Write-Output 'p01_evidence_integrity=PASS'
}
finally {
    Pop-Location
}
