[CmdletBinding()]
param(
    [string]$BaseRef,
    [string]$HeadRef = 'HEAD',
    [AllowEmptyString()]
    [string]$PullRequestBody = '',
    [string[]]$ChangedPath = @(),
    [string]$PolicyPath = '.github/solo-maintainer-policy.json'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Test-SensitivePath {
    param(
        [Parameter(Mandatory)] [string]$Path,
        [Parameter(Mandatory)] [string[]]$Patterns
    )

    $normalized = ($Path -replace '\\', '/') -replace '^\./', ''
    foreach ($pattern in $Patterns) {
        if ($pattern.EndsWith('/')) {
            if ($normalized.StartsWith($pattern, [System.StringComparison]::Ordinal)) { return $true }
        }
        elseif ($normalized.Equals($pattern, [System.StringComparison]::Ordinal)) { return $true }
    }
    return $false
}

function Assert-Attestations {
    param(
        [Parameter(Mandatory)] [string]$Body,
        [Parameter(Mandatory)] [string[]]$Required
    )

    foreach ($attestation in $Required) {
        $pattern = '(?im)^\s*-\s*\[[xX]\]\s*' + [regex]::Escape($attestation) + '(?:\s|$)'
        if ($Body -notmatch $pattern) {
            throw "Sensitive PR is missing checked solo-maintainer attestation: $attestation"
        }
    }
}

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$resolvedPolicy = Join-Path $repositoryRoot $PolicyPath
$policy = Get-Content -LiteralPath $resolvedPolicy -Raw | ConvertFrom-Json

if ($policy.schema_version -ne 1 -or $policy.mode -ne 'solo-maintainer-synthetic-portfolio') {
    throw 'Solo-maintainer policy identity is invalid.'
}
if ($policy.independent_review_status -ne 'unavailable-not-claimed') {
    throw 'Solo-maintainer policy must not claim independent review.'
}

$patterns = @($policy.sensitive_paths)
$required = @($policy.required_attestations)
if ($patterns.Count -lt 10 -or $required.Count -lt 6) {
    throw 'Solo-maintainer sensitive-path or attestation coverage is incomplete.'
}

$seededSensitive = '.github/workflows/pr.yml'
if (-not (Test-SensitivePath -Path $seededSensitive -Patterns $patterns)) {
    throw 'Sensitive-path seeded canary was not detected.'
}
$seededBody = ($required | ForEach-Object { "- [x] $_ seeded-canary" }) -join "`n"
Assert-Attestations -Body $seededBody -Required $required
$seededFailureObserved = $false
try {
    Assert-Attestations -Body '- [x] requirements-and-threats-reviewed only' -Required $required
}
catch {
    $seededFailureObserved = $true
}
if (-not $seededFailureObserved) {
    throw 'Incomplete-attestation seeded canary was accepted.'
}

if ($ChangedPath.Count -eq 0 -and $BaseRef) {
    $output = & git -C $repositoryRoot diff --name-only "$BaseRef...$HeadRef"
    if ($LASTEXITCODE -ne 0) { throw "Unable to inspect changed paths from $BaseRef to $HeadRef." }
    $ChangedPath = @($output | Where-Object { $_ })
}

$sensitive = @($ChangedPath | Where-Object { Test-SensitivePath -Path $_ -Patterns $patterns })
if ($sensitive.Count -gt 0) {
    Assert-Attestations -Body $PullRequestBody -Required $required
    Write-Output ('solo_governance_sensitive_paths=' + (($sensitive | Sort-Object -Unique) -join ','))
    Write-Output 'solo_governance_attestations=PASS'
}
else {
    Write-Output 'solo_governance_attestations=NOT_APPLICABLE(no-sensitive-path-change)'
}

Write-Output 'solo_governance_independent_review=UNAVAILABLE_NOT_CLAIMED'
Write-Output 'solo_governance_seeded_canaries=PASS'
Write-Output 'solo_governance_verification=PASS'
