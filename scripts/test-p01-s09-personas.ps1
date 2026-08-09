[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$journey = Join-Path $PSScriptRoot 'test-p01-s04-oidc-http.ps1'
$personas = @(
    @{name = 'support'; username = 'synthetic-support-analyst'},
    @{name = 'risk'; username = 'synthetic-risk-analyst'},
    @{name = 'finance'; username = 'synthetic-finance-operator'}
)

Push-Location -LiteralPath $repositoryRoot
try {
    foreach ($persona in $personas) {
        & $journey -Population workforce -Username $persona.username
        if (-not $?) { throw "Synthetic $($persona.name) browser-session journey failed" }
        Write-Output "p01_s09_persona_$($persona.name)=PASS(separate_workforce_identity=true)"
    }
    Write-Output 'p01_s09_personas=PASS(support,risk,finance)'
}
finally {
    Pop-Location
}
