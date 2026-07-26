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
$identityOrigin = 'http://127.0.0.1:18081'
$clientID = 'atlas-bff-local'
$redirectURL = 'http://127.0.0.1:18080/v1/auth/callback'
$usernames = @{
    customer = 'synthetic-customer'
    merchant = 'synthetic-merchant-operator'
    workforce = 'synthetic-workforce-operator'
}

function Read-RuntimeEnvironment {
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

function ConvertTo-Base64URL {
    param([Parameter(Mandatory)][byte[]]$Bytes)
    return [Convert]::ToBase64String($Bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

function ConvertFrom-Base64URL {
    param([Parameter(Mandatory)][string]$Value)
    $encoded = $Value.Replace('-', '+').Replace('_', '/')
    switch ($encoded.Length % 4) {
        2 { $encoded += '==' }
        3 { $encoded += '=' }
        1 { throw 'ID token payload encoding is malformed' }
    }
    return [Convert]::FromBase64String($encoded)
}

function Invoke-NoRedirect {
    param(
        [Parameter(Mandatory)][ValidateSet('Get', 'Post')][string]$Method,
        [Parameter(Mandatory)][Uri]$Uri,
        [Parameter()][AllowNull()][hashtable]$Form,
        [Parameter()][AllowEmptyString()][string]$Cookie = '',
        [Parameter()][AllowEmptyString()][string]$HostHeader = ''
    )

    $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::$Method, $Uri)
    try {
        if (-not [String]::IsNullOrWhiteSpace($Cookie)) {
            [void]$request.Headers.TryAddWithoutValidation('Cookie', $Cookie)
        }
        if (-not [String]::IsNullOrWhiteSpace($HostHeader)) {
            $request.Headers.Host = $HostHeader
        }
        if ($null -ne $Form) {
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
        return $script:httpClient.SendAsync($request).GetAwaiter().GetResult()
    }
    finally {
        $request.Dispose()
    }
}

if (-not (Test-Path -LiteralPath $runtimeFile -PathType Leaf)) {
    throw 'Prepared local runtime environment is absent'
}
$runtime = Read-RuntimeEnvironment
if (-not $runtime.ContainsKey('ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD') -or
    [String]::IsNullOrWhiteSpace($runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'])) {
    throw 'Prepared local runtime environment is missing the synthetic OIDC test credential'
}

$state = ConvertTo-Base64URL -Bytes ([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
$nonce = ConvertTo-Base64URL -Bytes ([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
$verifier = ConvertTo-Base64URL -Bytes ([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
$challenge = ConvertTo-Base64URL -Bytes (
    [Security.Cryptography.SHA256]::HashData([Text.Encoding]::ASCII.GetBytes($verifier))
)
$realm = "atlas-$Population-local"
$authorizationURL = (
    "$identityOrigin/realms/$realm/protocol/openid-connect/auth" +
    "?client_id=$([Uri]::EscapeDataString($clientID))" +
    "&redirect_uri=$([Uri]::EscapeDataString($redirectURL))" +
    '&response_type=code&scope=openid' +
    "&state=$([Uri]::EscapeDataString($state))" +
    "&nonce=$([Uri]::EscapeDataString($nonce))" +
    "&code_challenge=$([Uri]::EscapeDataString($challenge))" +
    '&code_challenge_method=S256'
)

$handler = [Net.Http.HttpClientHandler]::new()
$handler.AllowAutoRedirect = $false
$handler.UseCookies = $false
$script:httpClient = [Net.Http.HttpClient]::new($handler)
$script:httpClient.Timeout = [TimeSpan]::FromSeconds(15)
$idToken = $null
$accessToken = $null
try {
    $authorization = Invoke-NoRedirect -Method Get -Uri $authorizationURL
    if ([int]$authorization.StatusCode -ne 200) {
        throw "Synthetic identity authorization form returned status $([int]$authorization.StatusCode)"
    }
    $cookies = @(
        $authorization.Headers.GetValues('Set-Cookie') |
            ForEach-Object { ([string]$_).Split(';', 2)[0] }
    ) -join '; '
    $html = $authorization.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $formMatch = [regex]::Match(
        $html,
        '<form\b[^>]*\bid="kc-form-login"[^>]*\baction="(?<action>[^"]+)"',
        [Text.RegularExpressions.RegexOptions]::IgnoreCase
    )
    if (-not $formMatch.Success -or [String]::IsNullOrWhiteSpace($cookies)) {
        throw 'Synthetic identity authorization transaction is incomplete'
    }
    $formAction = [Net.WebUtility]::HtmlDecode($formMatch.Groups['action'].Value)

    $credentialResponse = Invoke-NoRedirect `
        -Method Post `
        -Uri $formAction `
        -Cookie $cookies `
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
    $callback = $credentialResponse.Headers.Location
    $callbackQuery = [Web.HttpUtility]::ParseQueryString($callback.Query)
    if ($callbackQuery['state'] -ne $state -or [String]::IsNullOrWhiteSpace($callbackQuery['code'])) {
        throw 'Synthetic identity authorization response did not preserve the one-time state'
    }

    $tokenURL = "$identityOrigin/realms/$realm/protocol/openid-connect/token"
    $tokenResponse = Invoke-NoRedirect `
        -Method Post `
        -Uri $tokenURL `
        -HostHeader 'keycloak:8080' `
        -Form @{
            client_id = $clientID
            grant_type = 'authorization_code'
            redirect_uri = $redirectURL
            code = $callbackQuery['code']
            code_verifier = $verifier
        }
    if ([int]$tokenResponse.StatusCode -ne 200) {
        throw "Synthetic identity token exchange returned status $([int]$tokenResponse.StatusCode)"
    }
    $tokenDocument = $tokenResponse.Content.ReadAsStringAsync().GetAwaiter().GetResult() | ConvertFrom-Json
    $idToken = [string]$tokenDocument.id_token
    $accessToken = [string]$tokenDocument.access_token
    $segments = $idToken.Split('.')
    if ($segments.Count -ne 3) {
        throw 'Synthetic identity provider returned a malformed ID token envelope'
    }
    $claimsJSON = [Text.Encoding]::UTF8.GetString((ConvertFrom-Base64URL -Value $segments[1]))
    $claims = $claimsJSON | ConvertFrom-Json
    $audiences = @($claims.aud)
    $nonceValid = [string]$claims.nonce -eq $nonce
    $audienceValid = $audiences -contains $clientID
    $authTimePresent = $null -ne $claims.PSObject.Properties['auth_time'] -and [int64]$claims.auth_time -gt 0
    $atHashPresent = $null -ne $claims.PSObject.Properties['at_hash'] -and
        -not [String]::IsNullOrWhiteSpace([string]$claims.at_hash)
    Write-Output "p01_s04_id_token_issuer=$([string]$claims.iss)"
    Write-Output "p01_s04_id_token_shape=audience:$audienceValid,nonce:$nonceValid,acr:$([string]$claims.acr),auth_time:$authTimePresent,at_hash:$atHashPresent"
}
finally {
    $idToken = $null
    $accessToken = $null
    $state = $null
    $nonce = $null
    $verifier = $null
    $runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'] = $null
    $script:httpClient.Dispose()
    $handler.Dispose()
}
