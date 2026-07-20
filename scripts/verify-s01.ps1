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
    $expectedGo = (Get-Content -LiteralPath '.go-version' -Raw).Trim()
    $goVersionOutput = (& go version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $goVersionOutput -notmatch '\bgo(?<version>\d+\.\d+\.\d+)\b') {
        throw "Unable to determine the Go toolchain version: $goVersionOutput"
    }
    if ($Matches.version -ne $expectedGo) {
        throw "Go version mismatch: expected $expectedGo, observed $($Matches.version)"
    }

    $insideWorkTree = (& git rev-parse --is-inside-work-tree 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $insideWorkTree -ne 'true') {
        throw 'The workspace is not a valid Git worktree.'
    }

    $sourceRevision = (& git rev-parse HEAD 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        $sourceRevision = 'UNBORN'
    }

    Invoke-NativeChecked -Command 'go' -Arguments @('test', './...')
    Invoke-NativeChecked -Command 'go' -Arguments @('build', './cmd/api', './cmd/worker', './cmd/simulator')
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test',
        './internal/architecture',
        '-run',
        'TestArchitectureBoundaries|TestBoundaryCheckerRejectsForbiddenImport|TestBoundaryCheckerRejectsUnregisteredModule|TestImportRules|TestRepositoryLayout|TestCanonicalPRDDuplicates|TestCanonicalPRDManifest',
        '-count=1',
        '-v'
    )

    Write-Output "toolchain_go=$expectedGo"
    Write-Output 'frontend_framework=React+TypeScript'
    Write-Output 'frontend_build_toolchain=DEFERRED'
    Write-Output "source_revision=$sourceRevision"
    Write-Output 's01_verification=PASS'
}
finally {
    Pop-Location
}
