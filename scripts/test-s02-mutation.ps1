[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$temporaryParent = [IO.Path]::GetFullPath((Join-Path $repositoryRoot '.tmp'))
$mutationRoot = [IO.Path]::GetFullPath((Join-Path $temporaryParent 's02-money-mutation'))
$requiredPrefix = $temporaryParent.TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
if (-not $mutationRoot.StartsWith($requiredPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing mutation work outside repository temporary directory: $mutationRoot"
}

$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $temporaryParent 'go-build'
$env:GOMODCACHE = Join-Path $temporaryParent 'go-mod'

if (Test-Path -LiteralPath $mutationRoot) {
    Remove-Item -LiteralPath $mutationRoot -Recurse -Force
}

try {
    New-Item -ItemType Directory -Path (Join-Path $mutationRoot 'internal/platform') -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'go.mod') -Destination $mutationRoot
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'internal/platform/domainerror') -Destination (Join-Path $mutationRoot 'internal/platform') -Recurse
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'internal/platform/money') -Destination (Join-Path $mutationRoot 'internal/platform') -Recurse

    $amountPath = Join-Path $mutationRoot 'internal/platform/money/amount.go'
    $source = Get-Content -LiteralPath $amountPath -Raw
    $target = @'
func (a Amount) Add(other Amount) (Amount, error) {
	if a.currency.IsZero() || other.currency.IsZero() {
		return Amount{}, ErrInvalidAmount
	}
	if a.currency != other.currency {
		return Amount{}, ErrCurrencyMismatch
	}
'@
    $replacement = @'
func (a Amount) Add(other Amount) (Amount, error) {
	if a.currency.IsZero() || other.currency.IsZero() {
		return Amount{}, ErrInvalidAmount
	}
	if false {
		return Amount{}, ErrCurrencyMismatch
	}
'@
    if (-not $source.Contains($target)) {
        throw 'The expected currency-mismatch guard was not found; the mutation target drifted.'
    }
    $mutated = $source.Replace($target, $replacement)
    Set-Content -LiteralPath $amountPath -Value $mutated -NoNewline

    Push-Location -LiteralPath $mutationRoot
    try {
        $mutationOutput = (& go test ./internal/platform/money -run TestCheckedArithmetic -count=1 2>&1 | Out-String).Trim()
        $mutationExit = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }

    if ($mutationExit -eq 0) {
        throw 'MONEY_CURRENCY_MISMATCH mutation survived; invariant test did not fail.'
    }
    if ($mutationOutput -notmatch 'currency mismatch') {
        throw "Mutation failed for an unexpected reason:`n$mutationOutput"
    }

    Write-Output 'mutation_target=MONEY_CURRENCY_MISMATCH_GUARD'
    Write-Output 'mutation_expected=go_test_failure'
    Write-Output 'mutation_observed=go_test_failure'
    Write-Output 'mutation_currency_guard=KILLED'
}
finally {
    if (Test-Path -LiteralPath $mutationRoot) {
        Remove-Item -LiteralPath $mutationRoot -Recurse -Force
    }
}
