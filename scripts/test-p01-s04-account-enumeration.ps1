[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('all', 'customer', 'merchant', 'workforce')]
    [string]$Population = 'all',

    [Parameter()]
    [ValidateRange(5, 50)]
    [int]$SampleCount = 9,

    [Parameter()]
    [ValidateRange(1, 2000)]
    [double]$MaxMedianDeltaMilliseconds = 150,

    [Parameter()]
    [ValidateRange(1.0, 10.0)]
    [double]$MaxMedianRatio = 2.5,

    [Parameter()]
    [ValidateRange(1, 5000)]
    [double]$MaxP95DeltaMilliseconds = 500
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
$apiOrigin = 'http://127.0.0.1:18080'
$knownUsernames = @{
    customer = 'synthetic-customer'
    merchant = 'synthetic-merchant-operator'
    workforce = 'synthetic-workforce-operator'
}

function Read-RuntimeEnvironment {
    if (-not (Test-Path -LiteralPath $runtimeFile -PathType Leaf)) {
        throw 'Prepared local runtime environment is absent'
    }
    $values = @{}
    foreach ($line in Get-Content -LiteralPath $runtimeFile) {
        if ([String]::IsNullOrWhiteSpace($line) -or $line.StartsWith('#')) {
            continue
        }
        $parts = $line.Split('=', 2)
        if ($parts.Count -ne 2 -or [String]::IsNullOrWhiteSpace($parts[0])) {
            throw 'Prepared local runtime environment contains a malformed entry'
        }
        $values[$parts[0]] = $parts[1]
    }
    return $values
}

function Get-LoginFailureObservation {
    param(
        [Parameter(Mandatory)][ValidateSet('customer', 'merchant', 'workforce')]
        [string]$IdentityPopulation,
        [Parameter(Mandatory)][string]$Username,
        [Parameter(Mandatory)][string]$WrongPassword
    )

    $handler = [Net.Http.HttpClientHandler]::new()
    $handler.AllowAutoRedirect = $false
    $handler.CookieContainer = [Net.CookieContainer]::new()
    $client = [Net.Http.HttpClient]::new($handler)
    $client.Timeout = [TimeSpan]::FromSeconds(15)
    [void]$client.DefaultRequestHeaders.TryAddWithoutValidation('Accept-Language', 'en')
    try {
        $begin = $client.GetAsync(
            "$apiOrigin/v1/auth/login?population=$IdentityPopulation&return_to=%2F$IdentityPopulation"
        ).GetAwaiter().GetResult()
        try {
            if ([int]$begin.StatusCode -ne 303 -or $null -eq $begin.Headers.Location) {
                throw "OIDC login initiation returned status $([int]$begin.StatusCode)"
            }
            $authorizationURI = $begin.Headers.Location
        }
        finally {
            $begin.Dispose()
        }

        $authorization = $client.GetAsync($authorizationURI).GetAwaiter().GetResult()
        try {
            if ([int]$authorization.StatusCode -ne 200) {
                throw "Synthetic identity authorization form returned status $([int]$authorization.StatusCode)"
            }
            $html = $authorization.Content.ReadAsStringAsync().GetAwaiter().GetResult()
            $authorizationCookies = @(
                $authorization.Headers.GetValues('Set-Cookie') |
                    ForEach-Object { ([string]$_).Split(';', 2)[0] }
            ) -join '; '
            if ([String]::IsNullOrWhiteSpace($authorizationCookies)) {
                throw 'Synthetic identity authorization form did not issue browser transaction cookies'
            }
        }
        finally {
            $authorization.Dispose()
        }

        $formMatch = [regex]::Match(
            $html,
            '<form\b[^>]*\bid="kc-form-login"[^>]*\baction="(?<action>[^"]+)"',
            [Text.RegularExpressions.RegexOptions]::IgnoreCase
        )
        if (-not $formMatch.Success) {
            throw 'Synthetic identity authorization form action was absent'
        }
        $formAction = [Uri][Net.WebUtility]::HtmlDecode($formMatch.Groups['action'].Value)
        if ($formAction.Scheme -ne 'http' -or
            $formAction.Host -ne '127.0.0.1' -or
            $formAction.Port -ne 18081 -or
            -not $formAction.AbsolutePath.StartsWith(
                "/realms/atlas-$IdentityPopulation-local/login-actions/"
            )) {
            throw 'Synthetic identity authorization form action escaped the allow-listed realm'
        }

        $fields = [Collections.Generic.List[Collections.Generic.KeyValuePair[string, string]]]::new()
        $fields.Add([Collections.Generic.KeyValuePair[string, string]]::new('username', $Username))
        $fields.Add([Collections.Generic.KeyValuePair[string, string]]::new('password', $WrongPassword))
        $fields.Add([Collections.Generic.KeyValuePair[string, string]]::new('credentialId', ''))
        $fields.Add([Collections.Generic.KeyValuePair[string, string]]::new('login', 'Sign In'))
        $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::Post, $formAction)
        try {
            [void]$request.Headers.TryAddWithoutValidation('Cookie', $authorizationCookies)
            $request.Content = [Net.Http.FormUrlEncodedContent]::new($fields)
            $stopwatch = [Diagnostics.Stopwatch]::StartNew()
            $failure = $client.SendAsync($request).GetAwaiter().GetResult()
            $stopwatch.Stop()
            try {
                $body = $failure.Content.ReadAsStringAsync().GetAwaiter().GetResult()
                if ([int]$failure.StatusCode -ne 200 -or $null -ne $failure.Headers.Location) {
                    throw "Synthetic identity rejection returned status $([int]$failure.StatusCode)"
                }
                $errorMatches = [regex]::Matches(
                    $body,
                    '>\s*Invalid username or password\.\s*<',
                    [Text.RegularExpressions.RegexOptions]::IgnoreCase -bor
                        [Text.RegularExpressions.RegexOptions]::Singleline
                )
                if ($errorMatches.Count -ne 1) {
                    throw 'Synthetic identity rejection copy is not the approved generic value'
                }
                $errorCopy = 'Invalid username or password.'
                return [pscustomobject]@{
                    Status = [int]$failure.StatusCode
                    ErrorCopy = $errorCopy
                    DurationMilliseconds = [double]$stopwatch.Elapsed.TotalMilliseconds
                }
            }
            finally {
                $failure.Dispose()
            }
        }
        finally {
            $request.Dispose()
        }
    }
    finally {
        $client.Dispose()
        $handler.Dispose()
    }
}

function Get-Median {
    param([Parameter(Mandatory)][double[]]$Values)

    $sorted = @($Values | Sort-Object)
    $middle = [int][Math]::Floor($sorted.Count / 2)
    if (($sorted.Count % 2) -eq 1) {
        return [double]$sorted[$middle]
    }
    return ([double]$sorted[$middle - 1] + [double]$sorted[$middle]) / 2
}

function Get-Percentile95 {
    param([Parameter(Mandatory)][double[]]$Values)

    $sorted = @($Values | Sort-Object)
    $index = [Math]::Max(0, [Math]::Ceiling(0.95 * $sorted.Count) - 1)
    return [double]$sorted[$index]
}

function Test-PopulationEnumerationResistance {
    param(
        [Parameter(Mandatory)][ValidateSet('customer', 'merchant', 'workforce')]
        [string]$IdentityPopulation,
        [Parameter(Mandatory)][string]$WrongPassword
    )

    $knownUsername = $knownUsernames[$IdentityPopulation]
    $absentUsername = "synthetic-absent-$IdentityPopulation"

    [void](Get-LoginFailureObservation `
        -IdentityPopulation $IdentityPopulation `
        -Username $knownUsername `
        -WrongPassword $WrongPassword)
    [void](Get-LoginFailureObservation `
        -IdentityPopulation $IdentityPopulation `
        -Username $absentUsername `
        -WrongPassword $WrongPassword)

    $knownDurations = [Collections.Generic.List[double]]::new()
    $absentDurations = [Collections.Generic.List[double]]::new()
    for ($sample = 0; $sample -lt $SampleCount; $sample++) {
        $order = if (($sample % 2) -eq 0) {
            @(
                @{Kind = 'known'; Username = $knownUsername},
                @{Kind = 'absent'; Username = $absentUsername}
            )
        }
        else {
            @(
                @{Kind = 'absent'; Username = $absentUsername},
                @{Kind = 'known'; Username = $knownUsername}
            )
        }
        foreach ($candidate in $order) {
            $observation = Get-LoginFailureObservation `
                -IdentityPopulation $IdentityPopulation `
                -Username $candidate.Username `
                -WrongPassword $WrongPassword
            if ($candidate.Kind -eq 'known') {
                $knownDurations.Add($observation.DurationMilliseconds)
            }
            else {
                $absentDurations.Add($observation.DurationMilliseconds)
            }
        }
    }

    $knownMedian = Get-Median -Values $knownDurations.ToArray()
    $absentMedian = Get-Median -Values $absentDurations.ToArray()
    $knownP95 = Get-Percentile95 -Values $knownDurations.ToArray()
    $absentP95 = Get-Percentile95 -Values $absentDurations.ToArray()
    $medianDelta = [Math]::Abs($knownMedian - $absentMedian)
    $p95Delta = [Math]::Abs($knownP95 - $absentP95)
    $smallerMedian = [Math]::Min($knownMedian, $absentMedian)
    $medianRatio = if ($smallerMedian -lt 5) {
        1.0
    }
    else {
        [Math]::Max($knownMedian, $absentMedian) / $smallerMedian
    }

    if ($medianDelta -gt $MaxMedianDeltaMilliseconds -or
        $medianRatio -gt $MaxMedianRatio -or
        $p95Delta -gt $MaxP95DeltaMilliseconds) {
        throw (
            "Synthetic $IdentityPopulation authentication timing exceeded its enumeration bound: " +
            "median_delta_ms=$([Math]::Round($medianDelta, 1)), " +
            "median_ratio=$([Math]::Round($medianRatio, 2)), " +
            "p95_delta_ms=$([Math]::Round($p95Delta, 1))"
        )
    }

    $invariant = [Globalization.CultureInfo]::InvariantCulture
    Write-Output (
        'p01_s04_account_enumeration=' +
        "PASS(population=$IdentityPopulation," +
        "samples=$SampleCount,status=200,error=generic," +
        "median_delta_ms=$($medianDelta.ToString('0.0', $invariant))," +
        "median_ratio=$($medianRatio.ToString('0.00', $invariant))," +
        "p95_delta_ms=$($p95Delta.ToString('0.0', $invariant)))"
    )
}

$runtime = Read-RuntimeEnvironment
if (-not $runtime.ContainsKey('ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD') -or
    [String]::IsNullOrWhiteSpace($runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'])) {
    throw 'Prepared local runtime environment is missing the synthetic OIDC test credential'
}
$wrongPassword = $runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'] + '-incorrect'
$populations = if ($Population -eq 'all') {
    @('customer', 'merchant', 'workforce')
}
else {
    @($Population)
}
foreach ($identityPopulation in $populations) {
    Test-PopulationEnumerationResistance `
        -IdentityPopulation $identityPopulation `
        -WrongPassword $wrongPassword
}

Write-Output 'p01_s04_account_enumeration_matrix=PASS(copy=status=timing-bounded,repeated-attempt-outcome=uniform)'
