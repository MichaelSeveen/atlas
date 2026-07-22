[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$catalogPath = Join-Path $repositoryRoot 'deploy/observability/catalog.json'

function Assert-CatalogPolicy {
    param([Parameter(Mandatory)][object]$Catalog)

    if ($Catalog.version -ne 1 -or $Catalog.cardinality_budget_per_metric -lt 1 -or $Catalog.cardinality_budget_per_metric -gt 128) {
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
