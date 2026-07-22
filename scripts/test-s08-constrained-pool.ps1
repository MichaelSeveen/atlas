[CmdletBinding()]
param(
    [Parameter()]
    [switch]$RequireRace
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'

function Invoke-NativeChecked {
    param([string]$Command, [string[]]$Arguments = @())
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

if (-not (Test-Path -LiteralPath $runtimeFile)) {
    throw 'Prepared local runtime state is required for the constrained-pool test.'
}

foreach ($line in Get-Content -LiteralPath $runtimeFile) {
    if ([string]::IsNullOrWhiteSpace($line) -or $line.StartsWith('#')) { continue }
    $separator = $line.IndexOf('=')
    if ($separator -lt 1) { throw 'Prepared runtime state is malformed.' }
    $name = $line.Substring(0, $separator)
    $value = $line.Substring($separator + 1)
    [Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

$env:ATLAS_POSTGRES_HOST = '127.0.0.1'
$env:ATLAS_POSTGRES_PORT = '15432'
$env:ATLAS_S08_DATABASE_INTEGRATION = '1'
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'

$cgoEnabled = (& go env CGO_ENABLED | Out-String).Trim()
if ($RequireRace -and $cgoEnabled -ne '1') {
    throw 'The required hosted constrained-pool lane cannot run the Go race detector because CGO is disabled.'
}
$arguments = @('test')
if ($cgoEnabled -eq '1') { $arguments += '-race' }
$arguments += @('./internal/platform/database', '-run', '^TestMostAgentsSkip10ConstrainedDatabasePool$', '-count=1', '-v')

Push-Location -LiteralPath $repositoryRoot
try {
    Invoke-NativeChecked -Command 'go' -Arguments $arguments
    Write-Output 's08_constrained_pool_connections=1'
    if ($cgoEnabled -eq '1') {
        Write-Output 's08_constrained_pool_race=PASS'
        Write-Output 's08_named_skipped_test_10=PASS'
    }
    else {
        Write-Output 's08_constrained_pool_race=NOT_AVAILABLE(cgo-disabled-host;required-in-hosted-Linux-lane)'
        Write-Output 's08_named_skipped_test_10=PARTIAL(concurrent-real-database-pass;race-proof-pending-hosted-S08-run)'
    }
}
finally {
    Remove-Item Env:ATLAS_S08_DATABASE_INTEGRATION -ErrorAction SilentlyContinue
    Pop-Location
}
