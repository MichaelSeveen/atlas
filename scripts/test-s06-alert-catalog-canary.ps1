[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$catalogPath = Join-Path $repositoryRoot 'deploy/observability/catalog.json'

function Assert-CatalogPolicy {
    param([Parameter(Mandatory)][object]$Catalog)

    if ($Catalog.version -ne 1 -or $Catalog.cardinality_budget_per_metric -lt 1 -or $Catalog.cardinality_budget_per_metric -gt 384) {
        throw 'catalog metadata is invalid'
    }
    $metricNames = @{}
    foreach ($metric in $Catalog.metrics) {
        if ($metricNames.ContainsKey($metric.name) -or [string]::IsNullOrWhiteSpace($metric.owner)) {
            throw 'metric identity or ownership is invalid'
        }
        $metricNames[$metric.name] = $true
        foreach ($label in $metric.labels.PSObject.Properties.Name) {
            if ($label -match '(?i)(request_id|correlation_id|trace_id|tenant|actor|user|email|account)') {
                throw 'high-cardinality identity label is forbidden'
            }
        }
    }
    foreach ($alert in $Catalog.alerts) {
        if ([string]::IsNullOrWhiteSpace($alert.owner) -or [string]::IsNullOrWhiteSpace($alert.runbook) -or
            [string]::IsNullOrWhiteSpace($alert.test) -or -not $metricNames.ContainsKey($alert.metric)) {
            throw 'alert ownership or linkage is invalid'
        }
    }

    $alertsByID = @{}
    foreach ($alert in $Catalog.alerts) {
        if ($alertsByID.ContainsKey($alert.id)) {
            throw 'duplicate alert identifier is forbidden'
        }
        $alertsByID[$alert.id] = $alert
    }
    foreach ($requiredID in @('P01-S05-ALR-001', 'P01-S06-ALR-001')) {
        if (-not $alertsByID.ContainsKey($requiredID)) {
            throw "required identity authorization alert is absent: $requiredID"
        }
    }
    $authorizationAlert = $alertsByID['P01-S06-ALR-001']
    if ($authorizationAlert.metric -ne 'atlas.identity.operation.count' -or
        $authorizationAlert.runbook -ne 'docs/runbooks/AUTHORIZATION_INCIDENT_AND_INVALIDATION.md' -or
        $authorizationAlert.condition -notmatch 'organization_member_list' -or
        $authorizationAlert.condition -notmatch 'rejected' -or
        $authorizationAlert.condition -notmatch '404') {
        throw 'authorization concealment alert lost its bounded emitted signal or runbook'
    }
}

$canonical = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
Assert-CatalogPolicy -Catalog $canonical

$identityMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$identityMutation.metrics[0].labels | Add-Member -NotePropertyName 'request_id' -NotePropertyValue @('unbounded')
$killed = $false
try { Assert-CatalogPolicy -Catalog $identityMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded high-cardinality metric label was accepted' }
Write-Output 'metric_identity_label_mutation=KILLED'

$ownerMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$ownerMutation.alerts[0].owner = ''
$killed = $false
try { Assert-CatalogPolicy -Catalog $ownerMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded ownerless alert was accepted' }
Write-Output 'ownerless_alert_mutation=KILLED'

$authorizationMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$authorizationMutation.alerts = @($authorizationMutation.alerts | Where-Object { $_.id -ne 'P01-S06-ALR-001' })
$killed = $false
try { Assert-CatalogPolicy -Catalog $authorizationMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded missing authorization alert was accepted' }
Write-Output 'authorization_alert_mutation=KILLED'
