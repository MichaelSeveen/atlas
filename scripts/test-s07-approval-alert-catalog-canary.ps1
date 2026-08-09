[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$catalogPath = Join-Path $repositoryRoot 'deploy/observability/catalog.json'
$requiredMetrics = @(
    'atlas.operations.approval.operation.count',
    'atlas.operations.approval.operation.duration',
    'atlas.operations.approval.status.count',
    'atlas.operations.approval.age',
    'atlas.operations.approval.conflict.count',
    'atlas.operations.approval.integrity_failure.count'
)
$requiredAlerts = @(
    'P01-S07-ALR-001',
    'P01-S07-ALR-002',
    'P01-S07-ALR-003',
    'P01-S07-ALR-004'
)

function Assert-ApprovalCatalogPolicy {
    param([Parameter(Mandatory)][object]$Catalog)

    if ($Catalog.version -ne 1 -or $Catalog.cardinality_budget_per_metric -lt 1 -or
        $Catalog.cardinality_budget_per_metric -gt 384) {
        throw 'catalog metadata is invalid'
    }
    $metrics = @{}
    foreach ($metric in $Catalog.metrics) {
        if ($metrics.ContainsKey($metric.name) -or [string]::IsNullOrWhiteSpace($metric.owner)) {
            throw 'metric identity or ownership is invalid'
        }
        $metrics[$metric.name] = $metric
        foreach ($label in $metric.labels.PSObject.Properties.Name) {
            if ($label -match '(?i)(request_id|correlation_id|trace_id|tenant|actor|user|principal|session|approval_id|target|email|account)') {
                throw 'high-cardinality approval identity label is forbidden'
            }
        }
    }
    foreach ($name in $requiredMetrics) {
        if (-not $metrics.ContainsKey($name) -or $metrics[$name].status -ne 'emitted') {
            throw "required emitted approval metric is absent: $name"
        }
    }

    $alerts = @{}
    foreach ($alert in $Catalog.alerts) {
        if ($alerts.ContainsKey($alert.id)) { throw 'duplicate alert identifier is forbidden' }
        $alerts[$alert.id] = $alert
    }
    foreach ($id in $requiredAlerts) {
        if (-not $alerts.ContainsKey($id)) { throw "required approval alert is absent: $id" }
        $alert = $alerts[$id]
        if ([string]::IsNullOrWhiteSpace($alert.owner) -or
            $alert.runbook -ne 'docs/runbooks/APPROVAL_INTEGRITY_AND_RECOVERY.md' -or
            $alert.test -ne 'scripts/test-s07-approval-alert-catalog-canary.ps1' -or
            -not $metrics.ContainsKey($alert.metric)) {
            throw "approval alert ownership or linkage is invalid: $id"
        }
    }
    if ($alerts['P01-S07-ALR-003'].metric -ne 'atlas.operations.approval.integrity_failure.count' -or
        $alerts['P01-S07-ALR-003'].condition -notmatch 'any approval payload or maker-checker integrity failure') {
        throw 'approval integrity page lost its fail-closed emitted signal'
    }
}

$canonical = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
Assert-ApprovalCatalogPolicy -Catalog $canonical

$identityMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$metric = @($identityMutation.metrics | Where-Object { $_.name -eq 'atlas.operations.approval.operation.count' })[0]
$metric.labels | Add-Member -NotePropertyName 'approval_id' -NotePropertyValue @('unbounded')
$killed = $false
try { Assert-ApprovalCatalogPolicy -Catalog $identityMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded approval identity-label mutation was accepted' }
Write-Output 'approval_identity_label_mutation=KILLED'

$alertMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$alertMutation.alerts = @($alertMutation.alerts | Where-Object { $_.id -ne 'P01-S07-ALR-003' })
$killed = $false
try { Assert-ApprovalCatalogPolicy -Catalog $alertMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded missing approval-integrity alert was accepted' }
Write-Output 'approval_integrity_alert_mutation=KILLED'

$runbookMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
@($runbookMutation.alerts | Where-Object { $_.id -eq 'P01-S07-ALR-001' })[0].runbook = ''
$killed = $false
try { Assert-ApprovalCatalogPolicy -Catalog $runbookMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded approval alert runbook mutation was accepted' }
Write-Output 'approval_alert_runbook_mutation=KILLED'

Write-Output 'p01_s07_approval_observability=PASS'
