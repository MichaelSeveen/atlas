[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$catalogPath = Join-Path $repositoryRoot 'deploy/observability/catalog.json'
$requiredAlerts = @(
    'P01-S05-ALR-001',
    'P01-S06-ALR-001',
    'P01-S08-ALR-001',
    'P01-S09-ALR-001',
    'P01-S09-ALR-002'
)
$requiredPanels = @(
    'atlas.identity.operation.count',
    'atlas.identity.provider.request.count',
    'atlas.operations.approval.operation.count',
    'atlas.operations.approval.status.count',
    'atlas.operations.approval.conflict.count',
    'atlas.operations.approval.integrity_failure.count',
    'atlas.identity.api_credential.authentication.count',
    'atlas.identity.api_credential.anomaly.count',
    'atlas.identity.api_credential.rate_rejection.count',
    'atlas.identity.break_glass.attempt.count'
)

function Assert-Phase01OperationalPolicy {
    param([Parameter(Mandatory)][object]$Catalog)

    $metrics = @{}
    foreach ($metric in $Catalog.metrics) {
        if ($metrics.ContainsKey($metric.name)) { throw "duplicate metric: $($metric.name)" }
        $metrics[$metric.name] = $metric
        foreach ($label in $metric.labels.PSObject.Properties.Name) {
            if ($label -match '(?i)(request_id|correlation_id|trace_id|tenant|actor|user|principal|session|credential_id|network_address|email|account)') {
                throw "unbounded or identifying metric label: $label"
            }
        }
    }
    $breakGlass = $metrics['atlas.identity.break_glass.attempt.count']
    if (-not $breakGlass -or $breakGlass.status -ne 'definition-only') {
        throw 'break-glass telemetry must remain explicitly definition-only while the capability is disabled'
    }

    $alerts = @{}
    foreach ($alert in $Catalog.alerts) { $alerts[$alert.id] = $alert }
    foreach ($id in $requiredAlerts) {
        if (-not $alerts.ContainsKey($id)) { throw "required Phase 01 alert is absent: $id" }
        $alert = $alerts[$id]
        if (-not $metrics.ContainsKey($alert.metric) -or [string]::IsNullOrWhiteSpace($alert.owner) -or
            [string]::IsNullOrWhiteSpace($alert.condition) -or [string]::IsNullOrWhiteSpace($alert.rationale) -or
            -not (Test-Path -LiteralPath (Join-Path $repositoryRoot $alert.runbook) -PathType Leaf)) {
            throw "Phase 01 alert linkage is invalid: $id"
        }
    }
    foreach ($id in @('P01-S09-ALR-001', 'P01-S09-ALR-002')) {
        if ($alerts[$id].test -ne 'scripts/test-p01-s09-operational-catalog-canary.ps1') {
            throw "S09 alert lacks its mutation test: $id"
        }
    }

    $dashboard = @($Catalog.dashboards | Where-Object { $_.name -eq 'phase-01-identity-access-tenancy' })
    if ($dashboard.Count -ne 1 -or $dashboard[0].owner -ne 'security-on-call') {
        throw 'aggregate Phase 01 dashboard identity or ownership is invalid'
    }
    foreach ($panel in $requiredPanels) {
        if ($dashboard[0].panels -notcontains $panel -or -not $metrics.ContainsKey($panel)) {
            throw "aggregate Phase 01 dashboard panel is absent or unknown: $panel"
        }
    }
}

$canonical = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
Assert-Phase01OperationalPolicy -Catalog $canonical

$missingAlert = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$missingAlert.alerts = @($missingAlert.alerts | Where-Object { $_.id -ne 'P01-S09-ALR-002' })
$killed = $false
try { Assert-Phase01OperationalPolicy -Catalog $missingAlert } catch { $killed = $true }
if (-not $killed) { throw 'Seeded missing repeated-step-up alert was accepted' }
Write-Output 'repeated_step_up_alert_mutation=KILLED'

$falseBreakGlassClaim = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
@($falseBreakGlassClaim.metrics | Where-Object { $_.name -eq 'atlas.identity.break_glass.attempt.count' })[0].status = 'emitted'
$killed = $false
try { Assert-Phase01OperationalPolicy -Catalog $falseBreakGlassClaim } catch { $killed = $true }
if (-not $killed) { throw 'Seeded false break-glass emission claim was accepted' }
Write-Output 'break_glass_claim_mutation=KILLED'

$missingDashboard = Get-Content -LiteralPath $catalogPath -Raw | ConvertFrom-Json
$missingDashboard.dashboards = @($missingDashboard.dashboards | Where-Object { $_.name -ne 'phase-01-identity-access-tenancy' })
$killed = $false
try { Assert-Phase01OperationalPolicy -Catalog $missingDashboard } catch { $killed = $true }
if (-not $killed) { throw 'Seeded missing aggregate Phase 01 dashboard was accepted' }
Write-Output 'phase01_dashboard_mutation=KILLED'

Write-Output 'p01_s09_operational_catalog=PASS'
