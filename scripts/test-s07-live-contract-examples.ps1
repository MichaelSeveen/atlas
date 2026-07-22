[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'
$workingRoot = Join-Path $repositoryRoot '.tmp/s07-live-contract'
$suffix = if ($IsWindows) { '.exe' } else { '' }
$binary = Join-Path $workingRoot ("api$suffix")
$stdout = Join-Path $workingRoot 'api.stdout.txt'
$stderr = Join-Path $workingRoot 'api.stderr.txt'
New-Item -ItemType Directory -Force $workingRoot | Out-Null

$listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
$listener.Start()
$port = ([Net.IPEndPoint]$listener.LocalEndpoint).Port
$listener.Stop()
$baseURL = "http://127.0.0.1:$port"
$previousAddress = $env:ATLAS_HTTP_ADDR
$process = $null
$client = [Net.Http.HttpClient]::new()

Push-Location -LiteralPath $repositoryRoot
try {
    & go build -o $binary ./cmd/api
    if ($LASTEXITCODE -ne 0) { throw 'Could not build live contract API.' }
    $env:ATLAS_HTTP_ADDR = "127.0.0.1:$port"
    if ($IsWindows) {
        $process = Start-Process -FilePath $binary -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    }
    else {
        $process = Start-Process -FilePath $binary -PassThru -RedirectStandardOutput $stdout -RedirectStandardError $stderr
    }

    $live = $null
    for ($attempt = 0; $attempt -lt 50; $attempt++) {
        if ($process.HasExited) { throw "Live contract API exited early with code $($process.ExitCode)." }
        try {
            $live = $client.GetAsync("$baseURL/health/live").GetAwaiter().GetResult()
            if ([int]$live.StatusCode -eq 200) { break }
        }
        catch { }
        Start-Sleep -Milliseconds 100
    }
    if ($null -eq $live -or [int]$live.StatusCode -ne 200) { throw 'Live API did not become available.' }

    $ready = $client.GetAsync("$baseURL/health/ready").GetAwaiter().GetResult()
    $version = $client.GetAsync("$baseURL/version").GetAwaiter().GetResult()
    $liveBody = $live.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    $readyBody = $ready.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    $versionBody = $version.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    if ($liveBody.status -ne 'alive') { throw 'OpenAPI liveness example does not match the live server.' }
    if ([int]$ready.StatusCode -ne 503 -or $readyBody.code -ne 'DEPENDENCY_DEGRADED') { throw 'OpenAPI standalone-readiness example does not match the live server.' }
    if ([int]$version.StatusCode -ne 200 -or $versionBody.source_revision -ne 'development' -or $versionBody.contract_version -ne '2026-07-20') {
        throw 'OpenAPI version example does not match the live server.'
    }
    foreach ($response in @($live, $ready, $version)) {
        if ($response.Headers.CacheControl.ToString() -ne 'no-store') { throw 'Live contract response is missing Cache-Control: no-store.' }
        $requestIDs = $response.Headers.GetValues('X-Request-Id')
        if ($null -eq $requestIDs -or [string]::IsNullOrWhiteSpace([string]($requestIDs | Select-Object -First 1))) { throw 'Live contract response is missing X-Request-Id.' }
    }
    Write-Output 's07_live_contract_examples=PASS'
}
finally {
    if ($null -ne $process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
        $process.WaitForExit()
    }
    $env:ATLAS_HTTP_ADDR = $previousAddress
    $client.Dispose()
    Pop-Location
}
