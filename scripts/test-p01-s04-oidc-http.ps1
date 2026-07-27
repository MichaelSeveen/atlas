[CmdletBinding()]
param(
    [Parameter()]
    [ValidateSet('customer', 'merchant', 'workforce')]
    [string]$Population = 'customer'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
$apiOrigin = 'http://127.0.0.1:18080'
$usernames = @{
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

function Invoke-NoRedirect {
    param(
        [Parameter(Mandatory)][ValidateSet('Get', 'Post')][string]$Method,
        [Parameter(Mandatory)][Uri]$Uri,
        [Parameter()][AllowNull()][hashtable]$Form,
        [Parameter()][AllowNull()][string]$JSON,
        [Parameter()][AllowEmptyString()][string]$Cookie = '',
        [Parameter()][AllowNull()][hashtable]$Headers
    )

    $request = [Net.Http.HttpRequestMessage]::new(
        [Net.Http.HttpMethod]::$Method,
        $Uri
    )
    try {
        if (-not [String]::IsNullOrWhiteSpace($Cookie)) {
            [void]$request.Headers.TryAddWithoutValidation('Cookie', $Cookie)
        }
        if ($null -ne $Headers) {
            foreach ($key in $Headers.Keys) {
                [void]$request.Headers.TryAddWithoutValidation([string]$key, [string]$Headers[$key])
            }
        }
        if ($null -ne $Form) {
            if ($PSBoundParameters.ContainsKey('JSON')) {
                throw 'A synthetic request cannot contain both form and JSON content'
            }
            $fields = [Collections.Generic.List[Collections.Generic.KeyValuePair[string, string]]]::new()
            foreach ($key in $Form.Keys) {
                $fields.Add(
                    [Collections.Generic.KeyValuePair[string, string]]::new(
                        [string]$key,
                        [string]$Form[$key]
                    )
                )
            }
            $request.Content = [Net.Http.FormUrlEncodedContent]::new($fields)
        }
        elseif ($PSBoundParameters.ContainsKey('JSON')) {
            $request.Content = [Net.Http.StringContent]::new(
                $JSON,
                [Text.Encoding]::UTF8,
                'application/json'
            )
        }
        return $script:httpClient.SendAsync($request).GetAwaiter().GetResult()
    }
    finally {
        $request.Dispose()
    }
}

function Update-CookieHeader {
    param(
        [Parameter()][AllowEmptyString()][string]$Existing = '',
        [Parameter(Mandatory)][Net.Http.HttpResponseMessage]$Response
    )

    $cookies = @{}
    foreach ($pair in $Existing.Split(';', [StringSplitOptions]::RemoveEmptyEntries)) {
        $parts = $pair.Trim().Split('=', 2)
        if ($parts.Count -eq 2 -and -not [String]::IsNullOrWhiteSpace($parts[0])) {
            $cookies[$parts[0]] = $parts[1]
        }
    }
    if ($Response.Headers.Contains('Set-Cookie')) {
        foreach ($header in $Response.Headers.GetValues('Set-Cookie')) {
            $pair = ([string]$header).Split(';', 2)[0]
            $parts = $pair.Split('=', 2)
            if ($parts.Count -eq 2 -and -not [String]::IsNullOrWhiteSpace($parts[0])) {
                $cookies[$parts[0]] = $parts[1]
            }
        }
    }
    return @(
        $cookies.GetEnumerator() |
            Sort-Object -Property Key |
            ForEach-Object { "$($_.Key)=$($_.Value)" }
    ) -join '; '
}

$runtime = Read-RuntimeEnvironment
if (-not $runtime.ContainsKey('ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD') -or
    [String]::IsNullOrWhiteSpace($runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'])) {
    throw 'Prepared local runtime environment is missing the synthetic OIDC test credential'
}

$handler = [Net.Http.HttpClientHandler]::new()
$handler.AllowAutoRedirect = $false
$handler.CookieContainer = [Net.CookieContainer]::new()
$script:httpClient = [Net.Http.HttpClient]::new($handler)
$script:httpClient.Timeout = [TimeSpan]::FromSeconds(15)
try {
    $begin = Invoke-NoRedirect `
        -Method Get `
        -Uri "$apiOrigin/v1/auth/login?population=$Population&return_to=%2F$Population"
    if ([int]$begin.StatusCode -ne 303 -or $null -eq $begin.Headers.Location) {
        throw "OIDC login initiation returned status $([int]$begin.StatusCode)"
    }

    $authorization = Invoke-NoRedirect -Method Get -Uri $begin.Headers.Location
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
    $formMatch = [regex]::Match(
        $html,
        '<form\b[^>]*\bid="kc-form-login"[^>]*\baction="(?<action>[^"]+)"',
        [Text.RegularExpressions.RegexOptions]::IgnoreCase
    )
    if (-not $formMatch.Success) {
        throw 'Synthetic identity authorization form action was absent'
    }
    $formAction = [Net.WebUtility]::HtmlDecode($formMatch.Groups['action'].Value)
    $parsedAction = [Uri]$formAction
    if ($parsedAction.Scheme -ne 'http' -or $parsedAction.Host -ne '127.0.0.1' -or
        $parsedAction.Port -ne 18081 -or
        -not $parsedAction.AbsolutePath.StartsWith("/realms/atlas-$Population-local/login-actions/")) {
        throw 'Synthetic identity authorization form action escaped the allow-listed realm'
    }

    $credentialResponse = Invoke-NoRedirect `
        -Method Post `
        -Uri $formAction `
        -Cookie $authorizationCookies `
        -Form @{
            username = $usernames[$Population]
            password = $runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD']
            credentialId = ''
            login = 'Sign In'
        }
    if ([int]$credentialResponse.StatusCode -notin @(302, 303) -or
        $null -eq $credentialResponse.Headers.Location) {
        throw "Synthetic identity credential submission returned status $([int]$credentialResponse.StatusCode)"
    }
    $providerCookies = Update-CookieHeader `
        -Existing $authorizationCookies `
        -Response $credentialResponse

    $callback = $credentialResponse.Headers.Location
    if ($callback.Scheme -ne 'http' -or $callback.Host -ne '127.0.0.1' -or
        $callback.Port -ne 18080 -or $callback.AbsolutePath -ne '/v1/auth/callback') {
        throw 'Synthetic identity provider callback escaped the Atlas API'
    }
    $callbackQuery = [Web.HttpUtility]::ParseQueryString($callback.Query)
    $queryKeys = @($callbackQuery.AllKeys | Where-Object { $null -ne $_ } | Sort-Object)
    Write-Output "p01_s04_callback_query_keys=$($queryKeys -join ',')"
    $issuer = [Uri]$callbackQuery['iss']
    $sessionState = [string]$callbackQuery['session_state']
    Write-Output (
        'p01_s04_callback_shape=' +
        "code:$($callbackQuery['code'].Length)," +
        "state:$($callbackQuery['state'].Length)," +
        "issuer:$($callbackQuery['iss'].Length)," +
        "issuer_scheme:$($issuer.Scheme)," +
        "issuer_host:$($issuer.Host)," +
        "session_state:$($sessionState.Length)," +
        "session_state_safe:$([bool]($sessionState -match '^[A-Za-z0-9._~-]+$'))"
    )

    $complete = Invoke-NoRedirect -Method Get -Uri $callback
    if ($Population -eq 'workforce') {
        $denialBody = $complete.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        if ([int]$complete.StatusCode -ne 401 -or
            $denialBody -notmatch '"code"\s*:\s*"OIDC_TRANSACTION_INVALID"') {
            throw "Baseline workforce authentication did not fail closed; status was $([int]$complete.StatusCode)"
        }
        Write-Output 'p01_s04_workforce_assurance=PASS(baseline-denied=true,session-issued=false)'
        return
    }
    if ([int]$complete.StatusCode -ne 303 -or
        [string]$complete.Headers.Location -ne "http://127.0.0.1:13000/$Population") {
        throw "Atlas OIDC callback returned status $([int]$complete.StatusCode) or an unsafe application redirect"
    }
    $sessionCookies = @($complete.Headers.GetValues('Set-Cookie'))
    if ($sessionCookies.Count -ne 1) {
        throw 'Atlas OIDC callback did not issue exactly one session cookie'
    }
    $sessionCookie = [string]$sessionCookies[0]
    if (-not $sessionCookie.StartsWith('__Host-atlas_session=') -or
        $sessionCookie -notmatch '(?i);\s*Path=/' -or
        $sessionCookie -notmatch '(?i);\s*Secure' -or
        $sessionCookie -notmatch '(?i);\s*HttpOnly' -or
        $sessionCookie -notmatch '(?i);\s*SameSite=Lax' -or
        $sessionCookie -match '(?i);\s*Domain=') {
        throw 'Atlas OIDC callback session cookie flags are unsafe'
    }
    Write-Output "p01_s04_callback_status=$([int]$complete.StatusCode)"
    Write-Output 'p01_s04_session_cookie=PASS(host-only=true,secure=true,http-only=true,same-site=lax)'

    $sessionCookiePair = $sessionCookie.Split(';', 2)[0]
    $current = Invoke-NoRedirect `
        -Method Get `
        -Uri "$apiOrigin/v1/me" `
        -Cookie $sessionCookiePair
    if ([int]$current.StatusCode -ne 200 -or
        -not $current.Headers.Contains('X-Atlas-CSRF-Token')) {
        throw "Atlas current-principal check returned status $([int]$current.StatusCode) without a CSRF token"
    }
    $csrfToken = @($current.Headers.GetValues('X-Atlas-CSRF-Token'))[0]
    if ($csrfToken.Length -ne 43) {
        throw 'Atlas current-principal CSRF token is malformed'
    }
    $currentBody = $current.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $currentDocument = $currentBody | ConvertFrom-Json
    $expectedPrincipalType = if ($Population -eq 'merchant') { 'merchant_user' } else { 'customer' }
    if ($currentBody -match '(?i)(access_token|refresh_token|id_token|authorization_code)' -or
        $currentDocument.type -ne $expectedPrincipalType) {
        throw 'Atlas current-principal response exposed a token or crossed the state-bound population'
    }
    Write-Output "p01_s04_current_principal=PASS(population=$Population,csrf=present,tokens=absent)"

    $sessions = Invoke-NoRedirect `
        -Method Get `
        -Uri "$apiOrigin/v1/sessions" `
        -Cookie $sessionCookiePair
    $sessionDocument = $sessions.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    $currentSessions = @($sessionDocument.data | Where-Object { $_.current -eq $true })
    if ([int]$sessions.StatusCode -ne 200 -or
        $currentSessions.Count -ne 1 -or
        -not ([string]$currentSessions[0].id).StartsWith('ses_') -or
        $currentSessions[0].population -ne $Population) {
        throw "Atlas session inventory did not identify exactly one current $Population session"
    }
    Write-Output "p01_s04_session_inventory=PASS(population=$Population,current=1)"

    $stepUpBody = '{"action":"identity.approval.decide"}'
    $stepUpKey = "p01-s04-$Population-" + [Guid]::NewGuid().ToString('N')
    $missingCSRF = Invoke-NoRedirect `
        -Method Post `
        -Uri "$apiOrigin/v1/step-up/challenges" `
        -Cookie $sessionCookiePair `
        -JSON $stepUpBody `
        -Headers @{'Idempotency-Key' = "p01-s04-$Population-step-up-missing-csrf"}
    if ([int]$missingCSRF.StatusCode -ne 403) {
        throw "Atlas step-up accepted a cookie-authenticated request without CSRF; status was $([int]$missingCSRF.StatusCode)"
    }

    $stepUp = Invoke-NoRedirect `
        -Method Post `
        -Uri "$apiOrigin/v1/step-up/challenges" `
        -Cookie $sessionCookiePair `
        -JSON $stepUpBody `
        -Headers @{
            'Idempotency-Key' = $stepUpKey
            'X-Atlas-CSRF-Token' = $csrfToken
        }
    $stepUpBodyText = $stepUp.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $stepUpDocument = $stepUpBodyText | ConvertFrom-Json
    $stepUpAuthorization = [Uri]$stepUpDocument.authorization_url
    $stepUpQuery = [Web.HttpUtility]::ParseQueryString($stepUpAuthorization.Query)
    if ([int]$stepUp.StatusCode -ne 201 -or
        $stepUpDocument.action -ne 'identity.approval.decide' -or
        $stepUpDocument.required_assurance -ne 'stepped_up' -or
        $stepUpAuthorization.Host -ne '127.0.0.1' -or
        $stepUpAuthorization.Port -ne 18081 -or
        $stepUpQuery['prompt'] -ne 'login' -or
        $stepUpQuery['max_age'] -ne '0' -or
        $stepUpQuery['acr_values'] -ne '2 3') {
        throw 'Atlas step-up initiation did not force a fresh higher-assurance synthetic authentication'
    }
    $stepUpReplay = Invoke-NoRedirect `
        -Method Post `
        -Uri "$apiOrigin/v1/step-up/challenges" `
        -Cookie $sessionCookiePair `
        -JSON $stepUpBody `
        -Headers @{
            'Idempotency-Key' = $stepUpKey
            'X-Atlas-CSRF-Token' = $csrfToken
        }
    $stepUpReplayBody = $stepUpReplay.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    if ([int]$stepUpReplay.StatusCode -ne 201 -or
        $stepUpReplayBody -cne $stepUpBodyText -or
        [string]$stepUpReplay.Headers.Location -cne [string]$stepUp.Headers.Location) {
        throw 'Atlas step-up idempotency replay did not return the exact stored response'
    }
    $stepUpConflict = Invoke-NoRedirect `
        -Method Post `
        -Uri "$apiOrigin/v1/step-up/challenges" `
        -Cookie $sessionCookiePair `
        -JSON '{"action":"identity.approval.execute"}' `
        -Headers @{
            'Idempotency-Key' = $stepUpKey
            'X-Atlas-CSRF-Token' = $csrfToken
        }
    $stepUpConflictBody = $stepUpConflict.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    if ([int]$stepUpConflict.StatusCode -ne 409 -or
        $stepUpConflictBody -notmatch '"code"\s*:\s*"IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST"') {
        throw 'Atlas step-up idempotency key accepted a different normalized request'
    }
    Write-Output "p01_s04_step_up_idempotency=PASS(population=$Population,replay=exact,changed_request=409)"

    $stepUpProviderResponse = Invoke-NoRedirect `
        -Method Get `
        -Uri $stepUpAuthorization `
        -Cookie $providerCookies
    $providerCookies = Update-CookieHeader `
        -Existing $providerCookies `
        -Response $stepUpProviderResponse
    $stepUpCallback = $null
    for ($attempt = 0; $attempt -lt 2; $attempt++) {
        if ([int]$stepUpProviderResponse.StatusCode -in @(302, 303) -and
            $null -ne $stepUpProviderResponse.Headers.Location) {
            $candidate = $stepUpProviderResponse.Headers.Location
            if ($candidate.Scheme -ne 'http' -or $candidate.Host -ne '127.0.0.1' -or
                $candidate.Port -ne 18080 -or $candidate.AbsolutePath -ne '/v1/auth/callback') {
                throw 'Synthetic step-up callback escaped the Atlas API'
            }
            $stepUpCallback = $candidate
            break
        }
        if ([int]$stepUpProviderResponse.StatusCode -ne 200) {
            throw "Synthetic step-up form $attempt returned status $([int]$stepUpProviderResponse.StatusCode)"
        }
        $stepUpHTML = $stepUpProviderResponse.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        $stepUpFormMatch = [regex]::Match(
            $stepUpHTML,
            '<form\b[^>]*\bid="kc-form-login"[^>]*\baction="(?<action>[^"]+)"',
            [Text.RegularExpressions.RegexOptions]::IgnoreCase
        )
        if (-not $stepUpFormMatch.Success) {
            throw 'Synthetic step-up credential form action was absent'
        }
        $stepUpFormAction = [Uri][Net.WebUtility]::HtmlDecode(
            $stepUpFormMatch.Groups['action'].Value
        )
        if ($stepUpFormAction.Scheme -ne 'http' -or
            $stepUpFormAction.Host -ne '127.0.0.1' -or
            $stepUpFormAction.Port -ne 18081 -or
            -not $stepUpFormAction.AbsolutePath.StartsWith(
                "/realms/atlas-$Population-local/login-actions/"
            )) {
            throw 'Synthetic step-up credential form escaped the allow-listed realm'
        }
        $stepUpProviderResponse = Invoke-NoRedirect `
            -Method Post `
            -Uri $stepUpFormAction `
            -Cookie $providerCookies `
            -Form @{
                username = $usernames[$Population]
                password = $runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD']
                credentialId = ''
                login = 'Sign In'
            }
        $providerCookies = Update-CookieHeader `
            -Existing $providerCookies `
            -Response $stepUpProviderResponse
    }
    if ($null -eq $stepUpCallback -and
        [int]$stepUpProviderResponse.StatusCode -in @(302, 303) -and
        $null -ne $stepUpProviderResponse.Headers.Location) {
        $candidate = $stepUpProviderResponse.Headers.Location
        if ($candidate.Scheme -ne 'http' -or $candidate.Host -ne '127.0.0.1' -or
            $candidate.Port -ne 18080 -or $candidate.AbsolutePath -ne '/v1/auth/callback') {
            throw 'Synthetic step-up callback escaped the Atlas API'
        }
        $stepUpCallback = $candidate
    }
    if ($null -eq $stepUpCallback) {
        throw 'Synthetic step-up did not complete within the bounded credential flow'
    }

    $stepUpComplete = Invoke-NoRedirect -Method Get -Uri $stepUpCallback
    if ([int]$stepUpComplete.StatusCode -ne 303 -or
        [string]$stepUpComplete.Headers.Location -ne "http://127.0.0.1:13000/$Population") {
        $stepUpDenial = $stepUpComplete.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        throw "Atlas step-up callback failed with status $([int]$stepUpComplete.StatusCode): $stepUpDenial"
    }
    $rotatedCookies = @($stepUpComplete.Headers.GetValues('Set-Cookie'))
    if ($rotatedCookies.Count -ne 1) {
        throw 'Atlas step-up callback did not issue exactly one rotated session cookie'
    }
    $rotatedCookiePair = ([string]$rotatedCookies[0]).Split(';', 2)[0]
    if ($rotatedCookiePair -eq $sessionCookiePair) {
        throw 'Atlas step-up callback did not rotate the opaque session cookie'
    }
    $rotatedCurrent = Invoke-NoRedirect `
        -Method Get `
        -Uri "$apiOrigin/v1/me" `
        -Cookie $rotatedCookiePair
    $rotatedBody = $rotatedCurrent.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $rotatedDocument = $rotatedBody | ConvertFrom-Json
    if ([int]$rotatedCurrent.StatusCode -ne 200 -or
        $rotatedDocument.assurance -ne 'stepped_up' -or
        -not $rotatedCurrent.Headers.Contains('X-Atlas-CSRF-Token')) {
        throw 'Atlas rotated session did not carry the verified stepped-up assurance'
    }
    $rotatedCSRF = @($rotatedCurrent.Headers.GetValues('X-Atlas-CSRF-Token'))[0]
    if ($rotatedCSRF.Length -ne 43 -or $rotatedCSRF -eq $csrfToken) {
        throw 'Atlas rotated session did not issue a rotation-bound CSRF token'
    }
    $oldSession = Invoke-NoRedirect `
        -Method Get `
        -Uri "$apiOrigin/v1/me" `
        -Cookie $sessionCookiePair
    if ([int]$oldSession.StatusCode -ne 401) {
        throw "Atlas pre-step-up session remained authoritative with status $([int]$oldSession.StatusCode)"
    }
    Write-Output "p01_s04_step_up_rotation=PASS(population=$Population,loa=2,assurance=stepped_up,old_cookie=401,mfa_claim=false)"

    $logout = Invoke-NoRedirect `
        -Method Post `
        -Uri "$apiOrigin/v1/logout" `
        -Cookie $rotatedCookiePair `
        -Headers @{'X-Atlas-CSRF-Token' = $rotatedCSRF}
    if ([int]$logout.StatusCode -ne 204) {
        throw "Atlas logout returned status $([int]$logout.StatusCode)"
    }
    $clearedCookies = @($logout.Headers.GetValues('Set-Cookie'))
    if ($clearedCookies.Count -ne 1 -or
        -not ([string]$clearedCookies[0]).StartsWith('__Host-atlas_session=') -or
        [string]$clearedCookies[0] -notmatch '(?i);\s*Max-Age=0') {
        throw 'Atlas logout did not expire the host-bound session cookie'
    }
    $revoked = Invoke-NoRedirect `
        -Method Get `
        -Uri "$apiOrigin/v1/me" `
        -Cookie $rotatedCookiePair
    if ([int]$revoked.StatusCode -ne 401) {
        throw "Revoked Atlas session remained authoritative with status $([int]$revoked.StatusCode)"
    }
    Write-Output 'p01_s04_logout=PASS(revoked=true,old_cookie=401)'
}
finally {
    $script:httpClient.Dispose()
    $handler.Dispose()
    if ($runtime.ContainsKey('ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD')) {
        $runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'] = $null
    }
}
