[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$canaryRoot = Join-Path $repositoryRoot '.tmp/s04-config-canary'
$repositoryPrefix = [IO.Path]::GetFullPath($repositoryRoot).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
$resolvedCanaryRoot = [IO.Path]::GetFullPath($canaryRoot)
if (-not $resolvedCanaryRoot.StartsWith($repositoryPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'S04 config canary target escapes the repository'
}

if (Test-Path -LiteralPath $canaryRoot) {
    throw 'S04 config canary target already exists; refusing an ambiguous cleanup'
}
New-Item -ItemType Directory -Path $canaryRoot -Force | Out-Null

try {
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'deploy/environments/local.json') -Destination $canaryRoot
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'deploy/environments/test.json') -Destination $canaryRoot
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'deploy/environments/staging.json') -Destination $canaryRoot
    Copy-Item -LiteralPath (Join-Path $repositoryRoot 'deploy/environments/production-reference.json') -Destination $canaryRoot

    $productionPath = Join-Path $canaryRoot 'production-reference.json'
    $content = Get-Content -LiteralPath $productionPath -Raw
    $mutated = $content.Replace('"allowed_origins": ["https://web.production-reference.atlas.invalid"]', '"allowed_origins": ["*"]')
    if ($mutated -eq $content) {
        throw 'Config canary mutation target was not found'
    }
    Set-Content -LiteralPath $productionPath -Value $mutated -NoNewline

    & go run ./cmd/envctl validate --config-dir $canaryRoot *> $null
    if ($LASTEXITCODE -eq 0) {
        throw 'Wildcard production-reference config canary survived validation'
    }
    Write-Output 'config_canary=WILDCARD_PRODUCTION_REFERENCE_ORIGIN'
    Write-Output 'config_canary_expected=validation_failure'
    Write-Output 'config_canary_observed=validation_failure'
    Write-Output 'config_canary=KILLED'
}
finally {
    if (Test-Path -LiteralPath $canaryRoot) {
        Remove-Item -LiteralPath $canaryRoot -Recurse -Force
    }
}
