[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'

function Invoke-NativeChecked {
    param(
        [Parameter(Mandatory)]
        [string]$Command,

        [Parameter()]
        [string[]]$Arguments = @()
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

Push-Location -LiteralPath $repositoryRoot
try {
    $baseRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        $baseRevision = 'UNBORN'
    }
    $worktreeChanges = (& git status --porcelain=v1 | Out-String).Trim()
    if ($worktreeChanges.Length -eq 0) {
        $sourceRevision = $baseRevision
    }
    else {
        $sourceRevision = "UNCOMMITTED_WORKTREE(base=$baseRevision)"
    }

    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'verify-s01.ps1'))
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './internal/platform/...', '-count=1')
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test',
        './internal/architecture',
        '-run',
        'TestArchitectureBoundaries|TestBoundaryCheckerRejectsFloatingPointMoney|TestBoundaryCheckerRejectsDirectTimeNow|TestBoundaryCheckerRejectsDotImportedTime|TestDomainPolicyAllowsNonMoneyFloatAndClockAdapter',
        '-count=1',
        '-v'
    )
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './internal/platform/money', '-run=^$', '-fuzz=^FuzzParseMinorUnits$', '-fuzztime=100x')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './internal/platform/money', '-run=^$', '-fuzz=^FuzzCheckedAddition$', '-fuzztime=100x')
    Invoke-NativeChecked -Command 'go' -Arguments @('test', './internal/platform/identifier', '-run=^$', '-fuzz=^FuzzParse$', '-fuzztime=100x')
    Invoke-NativeChecked -Command 'pwsh' -Arguments @('-NoProfile', '-File', (Join-Path $PSScriptRoot 'test-s02-mutation.ps1'))

    Write-Output 's02_implementation_scope=GO_ONLY'
    Write-Output 's02_platform_packages=actor,clock,correlation,domainerror,identifier,money'
    Write-Output 'static_canaries=FLOAT_MONEY,TIME_NOW,DOT_TIME_IMPORT'
    Write-Output 'fuzz_campaigns=FuzzParseMinorUnits,FuzzCheckedAddition,FuzzParse'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 's02_verification=PASS'
}
finally {
    Pop-Location
}
