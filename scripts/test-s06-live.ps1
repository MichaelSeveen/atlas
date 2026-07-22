[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repositoryRoot 'deploy/local/compose.yaml'
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
$traceID = '4bf92f3577b34da6a3ce929d0e0e4736'
$traceparent = "00-$traceID-00f067aa0ba902b7-01"
. (Join-Path $PSScriptRoot 'compose.ps1')

if (-not (Test-Path -LiteralPath $runtimeFile)) {
    throw 'Prepared local runtime state is required for the S06 live test'
}

function Invoke-Compose {
    param([string[]]$Arguments)
    Invoke-AtlasCompose -ContainerRuntime $ContainerRuntime -RuntimeFile $runtimeFile -ComposeFile $composeFile -Arguments $Arguments
}

function Get-CollectorLogs {
    return (Invoke-AtlasCompose -ContainerRuntime $ContainerRuntime -RuntimeFile $runtimeFile -ComposeFile $composeFile -Arguments @('logs', 'otel-collector') 2>&1 | Out-String)
}

$headers = @{
    'traceparent' = $traceparent
    'X-Request-Id' = 'req_01JAT1AS00000000000001'
    'X-Correlation-Id' = 'cor_01JAT1AS00000000000001'
}
$response = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -TimeoutSec 5 -Headers $headers -Uri 'http://127.0.0.1:18080/health/ready'
if ($response.StatusCode -ne 200 -or $response.Headers['traceparent'] -notmatch "^00-$traceID-") {
    throw 'Golden request did not preserve validated trace continuity'
}

$deadline = [DateTimeOffset]::UtcNow.AddSeconds(30)
do {
    Start-Sleep -Seconds 2
    $collectorLogs = Get-CollectorLogs
    $completeTrace = $collectorLogs -like "*$traceID*" -and
        $collectorLogs -like '*GET /health/ready*' -and
        $collectorLogs -like '*readiness.check*' -and
        $collectorLogs -like '*database.schema_readiness*'
    $completeMetrics = $collectorLogs -like '*http.server.request.count*' -and
        $collectorLogs -like '*atlas.database.pool.connections*' -and
        $collectorLogs -like '*atlas.build.info*'
} while ((-not $completeTrace -or -not $completeMetrics) -and [DateTimeOffset]::UtcNow -lt $deadline)

if (-not $completeTrace) { throw 'Collector did not export the complete API/readiness/database golden trace' }
if (-not $completeMetrics) { throw 'Collector did not export RED/database/build metrics' }

try {
    Invoke-Compose -Arguments @('stop', 'otel-collector')
    $outage = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -TimeoutSec 5 -Uri 'http://127.0.0.1:18080/health/ready'
    if ($outage.StatusCode -ne 200) {
        throw 'Collector outage incorrectly changed authoritative readiness'
    }
}
finally {
    Invoke-Compose -Arguments @('start', 'otel-collector')
}

Write-Output "golden_trace_id=$traceID"
Write-Output 'golden_trace_spans=api,readiness,database'
Write-Output 'telemetry_outage_readiness=200'
Write-Output 's06_live_observability=PASS'
