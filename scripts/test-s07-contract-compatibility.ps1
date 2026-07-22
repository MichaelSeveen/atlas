[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$BaseRef
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$baselineRoot = Join-Path $repositoryRoot '.tmp/s07-contract-baseline'
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'
New-Item -ItemType Directory -Force $baselineRoot | Out-Null

Push-Location -LiteralPath $repositoryRoot
try {
    foreach ($name in @('openapi.yaml', 'asyncapi.yaml')) {
        $relative = "docs/atlas-prd/03-contracts/$name"
        $bytes = & git show "${BaseRef}:$relative" 2>$null
        if ($LASTEXITCODE -ne 0) { throw "Cannot read $relative from $BaseRef." }
        $baseline = Join-Path $baselineRoot $name
        [IO.File]::WriteAllLines($baseline, [string[]]$bytes, [Text.UTF8Encoding]::new($false))
        & go run ./cmd/contractctl compare --baseline $baseline --candidate $relative
        if ($LASTEXITCODE -ne 0) { throw "$name compatibility comparison failed." }
    }
    Write-Output "s07_contract_base=$BaseRef"
    Write-Output 's07_contract_compatibility=PASS'
}
finally {
    Pop-Location
}
