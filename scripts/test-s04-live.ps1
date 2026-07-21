[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Assert-Http {
    param(
        [Parameter(Mandatory)]
        [string]$Uri,

        [Parameter()]
        [int]$Status = 200,

        [Parameter()]
        [string]$Contains = ''
    )

    $response = Invoke-WebRequest -UseBasicParsing -SkipHttpErrorCheck -TimeoutSec 5 -Uri $Uri
    if ($response.StatusCode -ne $Status) {
        throw "Unexpected HTTP status for $Uri"
    }
    if ($Contains.Length -gt 0 -and $response.Content -notlike "*$Contains*") {
        throw "Expected safe marker is absent for $Uri"
    }
    return $response
}

$live = Assert-Http -Uri 'http://127.0.0.1:18080/health/live' -Contains '"status":"alive"'
$ready = Assert-Http -Uri 'http://127.0.0.1:18080/health/ready' -Contains '"status":"ready"'
$version = Assert-Http -Uri 'http://127.0.0.1:18080/version' -Contains '"contract_version":"2026-07-20"'
$runtime = Assert-Http -Uri 'http://127.0.0.1:13000/runtime-config.json' -Contains 'SYNTHETIC DATA ONLY'
$customer = Assert-Http -Uri 'http://127.0.0.1:13000/customer'
$merchant = Assert-Http -Uri 'http://127.0.0.1:13000/merchant'
$workforce = Assert-Http -Uri 'http://127.0.0.1:13000/workforce'
$identity = Assert-Http -Uri 'http://127.0.0.1:18081/realms/atlas-customer-local/.well-known/openid-configuration' -Contains 'atlas-customer-local'
$broker = Assert-Http -Uri 'http://127.0.0.1:18222/varz' -Contains 'jetstream'
$objects = Assert-Http -Uri 'http://127.0.0.1:19000/minio/health/live'

foreach ($response in @($live, $ready, $version, $runtime, $customer, $merchant, $workforce)) {
    if ($response.Headers['Cache-Control'] -notcontains 'no-store') {
        throw 'Foundation response omitted no-store cache policy'
    }
}

$unknown = Assert-Http -Uri 'http://127.0.0.1:13000/not-a-shell' -Status 404
if ($unknown.Content -like '*postgres*' -or $unknown.Content -like '*keycloak*') {
    throw 'Unknown shell response leaked topology'
}

Write-Output 'api_live=200'
Write-Output 'api_ready=200'
Write-Output 'api_version=200'
Write-Output 'web_shells=customer,merchant,workforce'
Write-Output 'identity_realm=atlas-customer-local'
Write-Output 'broker_jetstream=READY'
Write-Output 'object_storage=READY'
Write-Output 's04_live_smoke=PASS'
