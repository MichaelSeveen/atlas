[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot

function Invoke-NativeChecked {
    param([string]$Command, [string[]]$Arguments)
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

Push-Location -LiteralPath $repositoryRoot
try {
    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test',
        './internal/identity',
        '-run',
        '^TestIdentitySeedPolicyAndSubjectMutationsAreRejected$',
        '-count=1'
    )
    Write-Output 'p01_s03_seed_policy_mutations=KILLED'

    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test',
        './internal/identity/persistence',
        '-run',
        '^TestMissingTenantPredicateMutationIsRejected$',
        '-count=1'
    )
    Write-Output 'p01_s03_missing_tenant_predicate=KILLED'

    Invoke-NativeChecked -Command 'go' -Arguments @(
        'test',
        './internal/audit/...',
        '-count=1'
    )
    Write-Output 'p01_s03_audit_outage_and_scope_mutations=KILLED'
}
finally {
    Pop-Location
}
