[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$exercisePath = Join-Path $repositoryRoot 'docs/engineering/phase-01-operational-exercises.json'
$requiredScenarios = @(
    'identity-provider-unavailable',
    'authorization-stale-or-incident',
    'audit-persistence-unavailable',
    'merchant-api-credential-compromise'
)

function Assert-Exercises {
    param([Parameter(Mandatory)][object]$Policy)

    if ($Policy.schema_version -ne 1 -or $Policy.phase -ne 'P01-S09' -or
        $Policy.environment -ne 'synthetic-local-reference') {
        throw 'Phase 01 operational exercise identity is invalid'
    }
    $seen = @{}
    foreach ($exercise in $Policy.exercises) {
        if ($seen.ContainsKey($exercise.scenario) -or $exercise.status -ne 'exercised-synthetic' -or
            [string]::IsNullOrWhiteSpace($exercise.expected_policy) -or
            [string]::IsNullOrWhiteSpace($exercise.runtime_evidence)) {
            throw "operational exercise is incomplete or duplicated: $($exercise.scenario)"
        }
        $runbookPath = Join-Path $repositoryRoot $exercise.runbook
        if (-not (Test-Path -LiteralPath $runbookPath -PathType Leaf)) {
            throw "operational exercise runbook is absent: $($exercise.runbook)"
        }
        $runbook = Get-Content -LiteralPath $runbookPath -Raw
        if ($runbook -notmatch '(?i)(detect|triage|scope|failure)' -or $runbook -notmatch '(?i)(contain|response)' -or
            $runbook -notmatch '(?i)(recover|verification)' -or $runbook -notmatch '(?i)(evidence|audit)') {
            throw "operational runbook lacks detection, containment, recovery, or evidence steps: $($exercise.runbook)"
        }
        $seen[$exercise.scenario] = $true
    }
    foreach ($scenario in $requiredScenarios) {
        if (-not $seen.ContainsKey($scenario)) { throw "required operational scenario is absent: $scenario" }
    }
    if (($Policy.limitations -join ' ') -notmatch '(?i)break-glass remains disabled' -or
        ($Policy.limitations -join ' ') -notmatch '(?i)no production') {
        throw 'operational exercise limitations overstate Phase 01 readiness'
    }
}

$canonical = Get-Content -LiteralPath $exercisePath -Raw | ConvertFrom-Json
Assert-Exercises -Policy $canonical

$unsafeMutation = Get-Content -LiteralPath $exercisePath -Raw | ConvertFrom-Json
$unsafeMutation.exercises[2].expected_policy = 'write the mutation and repair Audit later'
$killed = $false
try {
    Assert-Exercises -Policy $unsafeMutation
    if ($unsafeMutation.exercises[2].expected_policy -notmatch '(?i)(roll back|fail closed)') {
        throw 'unsafe deferred Audit mutation accepted'
    }
}
catch { $killed = $true }
if (-not $killed) { throw 'Seeded unsafe Audit-outage response was accepted' }
Write-Output 'audit_outage_runbook_mutation=KILLED'

$missingScenario = Get-Content -LiteralPath $exercisePath -Raw | ConvertFrom-Json
$missingScenario.exercises = @($missingScenario.exercises | Where-Object { $_.scenario -ne 'authorization-stale-or-incident' })
$killed = $false
try { Assert-Exercises -Policy $missingScenario } catch { $killed = $true }
if (-not $killed) { throw 'Seeded missing authorization exercise was accepted' }
Write-Output 'authorization_exercise_mutation=KILLED'

Write-Output 'p01_s09_runbook_exercises=PASS'
