[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$GitleaksPath,

    [Parameter(Mandatory)]
    [string]$GitleaksConfig
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$temporaryParent = Join-Path $repositoryRoot '.tmp'
$canaryRoot = Join-Path $temporaryParent ("s07-deleted-secret-canary-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force $canaryRoot | Out-Null

function Invoke-GitChecked([string[]]$Arguments) {
    & git -C $canaryRoot @Arguments | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Git canary command failed: $($Arguments -join ' ')" }
}

try {
    Invoke-GitChecked @('init', '--initial-branch=main')
    Invoke-GitChecked @('config', 'user.name', 'Atlas S07 synthetic canary')
    Invoke-GitChecked @('config', 'user.email', 's07-canary@example.invalid')
    $secretPath = Join-Path $canaryRoot 'deleted-secret.txt'
    # Synthetic invalid credential exists only in a disposable repository.
    $accessKey = ('AK' + 'IAQWERTYUIOPASDFGH')
    $secretKey = ('q1W2e3R4t5Y6u7I8o9P0' + 'a1S2d3F4g5H6j7K8l9Z0')
    [IO.File]::WriteAllText($secretPath, "aws_access_key_id=$accessKey`naws_secret_access_key=$secretKey`n")
    Invoke-GitChecked @('add', 'deleted-secret.txt')
    Invoke-GitChecked @('commit', '-m', 'seed synthetic deleted-history credential')
    Remove-Item -LiteralPath $secretPath -Force
    Invoke-GitChecked @('add', '-u')
    Invoke-GitChecked @('commit', '-m', 'delete synthetic credential')

    & $GitleaksPath git $canaryRoot --config $GitleaksConfig --no-banner --redact=100 --exit-code 23 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 23) {
        throw "Deleted-history secret canary was not detected; expected exit 23, observed $LASTEXITCODE."
    }
    Write-Output 's07_deleted_history_secret_canary=PASS'
}
finally {
    if (Test-Path -LiteralPath $canaryRoot) {
        $resolved = [IO.Path]::GetFullPath($canaryRoot)
        $resolvedParent = [IO.Path]::GetFullPath($temporaryParent) + [IO.Path]::DirectorySeparatorChar
        if (-not $resolved.StartsWith($resolvedParent, [StringComparison]::OrdinalIgnoreCase) -or
            -not ([IO.Path]::GetFileName($resolved)).StartsWith('s07-deleted-secret-canary-', [StringComparison]::Ordinal)) {
            throw "Refusing to remove unexpected canary path $resolved."
        }
        Remove-Item -LiteralPath $resolved -Recurse -Force
    }
}
