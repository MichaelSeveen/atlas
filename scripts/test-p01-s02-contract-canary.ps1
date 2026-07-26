[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$canaryRoot = Join-Path $repositoryRoot '.tmp/p01-s02-contract-canary'
$resolvedRepository = [IO.Path]::GetFullPath($repositoryRoot)
$resolvedCanary = [IO.Path]::GetFullPath($canaryRoot)
if (-not $resolvedCanary.StartsWith(($resolvedRepository.TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar), [StringComparison]::OrdinalIgnoreCase)) {
    throw "Unsafe canary directory: $resolvedCanary"
}

$previousOpenAPI = $env:ATLAS_OPENAPI_PATH
$previousPolicy = $env:ATLAS_IDENTITY_ACCESS_POLICY_PATH
$env:GOTELEMETRY = 'off'
$env:GOCACHE = Join-Path $repositoryRoot '.tmp/go-build'
$env:GOMODCACHE = Join-Path $repositoryRoot '.tmp/go-mod'

function Assert-GoTestRejects([string]$TestName, [string]$Label) {
    $output = & go test ./tests/contract -run "^$TestName$" -count=1 2>&1
    if ($LASTEXITCODE -eq 0) {
        throw "$Label canary was accepted."
    }
    Write-Output "p01_s02_canary=$Label`:PASS(rejected)"
}

Push-Location -LiteralPath $repositoryRoot
try {
    if (Test-Path -LiteralPath $resolvedCanary) {
        Remove-Item -LiteralPath $resolvedCanary -Recurse -Force
    }
    New-Item -ItemType Directory -Path $resolvedCanary | Out-Null

    $openAPICanary = Join-Path $resolvedCanary 'openapi-missing-session.yaml'
    $openAPI = Get-Content -LiteralPath 'docs/atlas-prd/03-contracts/openapi.yaml' -Raw
    $mutatedOpenAPI = $openAPI.Replace('  /v1/sessions:', '  /v1/sessions-canary-removed:')
    if ($mutatedOpenAPI -eq $openAPI) { throw 'OpenAPI canary mutation did not apply.' }
    [IO.File]::WriteAllText($openAPICanary, $mutatedOpenAPI, [Text.UTF8Encoding]::new($false))
    $env:ATLAS_OPENAPI_PATH = $openAPICanary
    Assert-GoTestRejects 'TestPhase01OpenAPISurface' 'missing-openapi-path'
    Remove-Item Env:ATLAS_OPENAPI_PATH

    $policyCanary = Join-Path $resolvedCanary 'identity-access-allow.json'
    $policy = Get-Content -LiteralPath 'docs/atlas-prd/03-contracts/identity-access-policy.json' -Raw
    $mutatedPolicy = $policy.Replace('"default_authorization": "deny"', '"default_authorization": "allow"')
    if ($mutatedPolicy -eq $policy) { throw 'Policy canary mutation did not apply.' }
    [IO.File]::WriteAllText($policyCanary, $mutatedPolicy, [Text.UTF8Encoding]::new($false))
    $env:ATLAS_IDENTITY_ACCESS_POLICY_PATH = $policyCanary
    Assert-GoTestRejects 'TestPhase01IdentityAccessPolicyIsClosedAndConsistent' 'allow-by-default-policy'

    Write-Output 'p01_s02_contract_canaries=PASS'
}
finally {
    if ($null -eq $previousOpenAPI) {
        Remove-Item Env:ATLAS_OPENAPI_PATH -ErrorAction SilentlyContinue
    }
    else {
        $env:ATLAS_OPENAPI_PATH = $previousOpenAPI
    }
    if ($null -eq $previousPolicy) {
        Remove-Item Env:ATLAS_IDENTITY_ACCESS_POLICY_PATH -ErrorAction SilentlyContinue
    }
    else {
        $env:ATLAS_IDENTITY_ACCESS_POLICY_PATH = $previousPolicy
    }
    if (Test-Path -LiteralPath $resolvedCanary) {
        Remove-Item -LiteralPath $resolvedCanary -Recurse -Force
    }
    Pop-Location
}
