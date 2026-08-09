[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$catalogPath = Join-Path $repositoryRoot 'deploy/observability/catalog.json'
$requiredMetrics = @(
    'atlas.identity.api_credential.authentication.count',
    'atlas.identity.api_credential.anomaly.count',
    'atlas.identity.api_credential.rate_rejection.count',
    'atlas.identity.api_credential.rate_fallback.count'
)
$requiredAlerts = @('P01-S08-ALR-001', 'P01-S08-ALR-002', 'P01-S08-ALR-003')

function Assert-CredentialCatalogPolicy {
    param([Parameter(Mandatory)][object]$Catalog)

    $metrics = @{}
    foreach ($metric in $Catalog.metrics) {
        if ($metrics.ContainsKey($metric.name) -or [string]::IsNullOrWhiteSpace($metric.owner)) {
            throw 'metric identity or ownership is invalid'
        }
        $metrics[$metric.name] = $metric
        foreach ($label in $metric.labels.PSObject.Properties.Name) {
            if ($label -match '(?i)(request_id|correlation_id|trace_id|tenant|actor|user|principal|session|credential_id|network_address|email|account)') {
                throw 'high-cardinality credential identity label is forbidden'
            }
        }
    }
    foreach ($name in $requiredMetrics) {
        if (-not $metrics.ContainsKey($name) -or $metrics[$name].status -ne 'emitted') {
            throw "required emitted credential metric is absent: $name"
        }
    }

    $alerts = @{}
    foreach ($alert in $Catalog.alerts) { $alerts[$alert.id] = $alert }
    foreach ($id in $requiredAlerts) {
        if (-not $alerts.ContainsKey($id)) { throw "required credential alert is absent: $id" }
        $alert = $alerts[$id]
        if ([string]::IsNullOrWhiteSpace($alert.owner) -or
            $alert.runbook -ne 'docs/runbooks/API_CREDENTIAL_COMPROMISE_AND_ROTATION.md' -or
            $alert.test -ne 'scripts/test-p01-s08-credential-alert-catalog-canary.ps1' -or
            -not $metrics.ContainsKey($alert.metric)) {
            throw "credential alert ownership or linkage is invalid: $id"
        }
    }
}

$canonical = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
Assert-CredentialCatalogPolicy -Catalog $canonical

$identityMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$metric = @($identityMutation.metrics | Where-Object { $_.name -eq 'atlas.identity.api_credential.anomaly.count' })[0]
$metric.labels | Add-Member -NotePropertyName 'credential_id' -NotePropertyValue @('unbounded')
$killed = $false
try { Assert-CredentialCatalogPolicy -Catalog $identityMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded credential identity-label mutation was accepted' }
Write-Output 'credential_identity_label_mutation=KILLED'

$alertMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$alertMutation.alerts = @($alertMutation.alerts | Where-Object { $_.id -ne 'P01-S08-ALR-001' })
$killed = $false
try { Assert-CredentialCatalogPolicy -Catalog $alertMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded missing credential-anomaly alert was accepted' }
Write-Output 'credential_anomaly_alert_mutation=KILLED'

$runbookMutation = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
@($runbookMutation.alerts | Where-Object { $_.id -eq 'P01-S08-ALR-002' })[0].runbook = ''
$killed = $false
try { Assert-CredentialCatalogPolicy -Catalog $runbookMutation } catch { $killed = $true }
if (-not $killed) { throw 'Seeded credential alert runbook mutation was accepted' }
Write-Output 'credential_alert_runbook_mutation=KILLED'

Write-Output 'p01_s08_credential_observability=PASS'
