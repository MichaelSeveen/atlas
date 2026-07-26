[CmdletBinding()]
param(
    [Parameter()]
    [switch]$RotateTestCredential
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$runtimeFile = Join-Path $repositoryRoot '.tmp/environments/local/runtime.env'
$identityOrigin = 'http://127.0.0.1:18081'
$clientID = 'atlas-bff-local'
$callback = 'http://127.0.0.1:18080/v1/auth/callback'
$webOrigin = 'http://127.0.0.1:13000'

function Rotate-SyntheticTestCredential {
    if (-not (Test-Path -LiteralPath $runtimeFile -PathType Leaf)) {
        throw 'Prepared local runtime environment is absent'
    }
    $bytes = [byte[]]::new(32)
    [Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
    $value = $null
    try {
        $value = [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', '-').Replace('/', '_')
        $lines = @(Get-Content -LiteralPath $runtimeFile)
        $replaced = 0
        for ($index = 0; $index -lt $lines.Count; $index++) {
            if ($lines[$index].StartsWith('ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD=')) {
                $lines[$index] = "ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD=$value"
                $replaced++
            }
        }
        if ($replaced -ne 1) {
            throw 'Prepared local runtime environment has an invalid synthetic test credential inventory'
        }
        $temporary = Join-Path (Split-Path -Parent $runtimeFile) ([IO.Path]::GetRandomFileName())
        try {
            [IO.File]::WriteAllText(
                $temporary,
                (($lines -join "`n") + "`n"),
                [Text.UTF8Encoding]::new($false)
            )
            Move-Item -LiteralPath $temporary -Destination $runtimeFile -Force
        }
        finally {
            if (Test-Path -LiteralPath $temporary) {
                Remove-Item -LiteralPath $temporary -Force
            }
        }
    }
    finally {
        [Array]::Clear($bytes, 0, $bytes.Length)
        $value = $null
    }
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

function Set-ObjectProperty {
    param(
        [Parameter(Mandatory)][object]$InputObject,
        [Parameter(Mandatory)][string]$Name,
        [Parameter()][AllowNull()][object]$Value
    )

    $InputObject | Add-Member -NotePropertyName $Name -NotePropertyValue $Value -Force
}

function Invoke-IdentityAdmin {
    param(
        [Parameter(Mandatory)][ValidateSet('Get', 'Post', 'Put')][string]$Method,
        [Parameter(Mandatory)][string]$Path,
        [Parameter()][AllowNull()][object]$Body
    )

    $arguments = @{
        Method = $Method
        Uri = "$identityOrigin$Path"
        Headers = @{ Authorization = "Bearer $script:adminToken" }
        TimeoutSec = 15
    }
    if ($PSBoundParameters.ContainsKey('Body')) {
        $arguments.ContentType = 'application/json'
        $arguments.Body = $Body | ConvertTo-Json -Depth 20 -Compress
    }
    return Invoke-RestMethod @arguments
}

if ($RotateTestCredential) {
    Rotate-SyntheticTestCredential
    Write-Output 'p01_s04_keycloak_test_credential=ROTATED(runtime-only)'
}

$runtime = Read-RuntimeEnvironment
foreach ($required in @(
    'ATLAS_KEYCLOAK_ADMIN',
    'ATLAS_KEYCLOAK_ADMIN_PASSWORD',
    'ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'
)) {
    if (-not $runtime.ContainsKey($required) -or [String]::IsNullOrWhiteSpace($runtime[$required])) {
        throw "Prepared local runtime environment is missing $required"
    }
}

$script:adminToken = $null
try {
    $tokenResponse = Invoke-RestMethod `
        -Method Post `
        -Uri "$identityOrigin/realms/master/protocol/openid-connect/token" `
        -ContentType 'application/x-www-form-urlencoded' `
        -TimeoutSec 15 `
        -Body @{
            client_id = 'admin-cli'
            grant_type = 'password'
            username = $runtime['ATLAS_KEYCLOAK_ADMIN']
            password = $runtime['ATLAS_KEYCLOAK_ADMIN_PASSWORD']
        }
    $script:adminToken = [string]$tokenResponse.access_token
    if ([String]::IsNullOrWhiteSpace($script:adminToken)) {
        throw 'Synthetic identity provider did not issue an administrative control-plane token'
    }

    $populations = @(
        @{
            Realm = 'atlas-customer-local'
            Username = 'synthetic-customer'
            Subject = '00000000-0000-4000-8000-000000000101'
            Email = 'customer@synthetic.invalid'
            FirstName = 'Synthetic'
            LastName = 'Customer'
            IdleSeconds = 1800
            MaximumSeconds = 43200
        },
        @{
            Realm = 'atlas-merchant-local'
            Username = 'synthetic-merchant-operator'
            Subject = '00000000-0000-4000-8000-000000000201'
            Email = 'merchant-operator@synthetic.invalid'
            FirstName = 'Synthetic'
            LastName = 'Merchant Operator'
            IdleSeconds = 1200
            MaximumSeconds = 28800
        },
        @{
            Realm = 'atlas-workforce-local'
            Username = 'synthetic-workforce-operator'
            Subject = '00000000-0000-4000-8000-000000000301'
            Email = 'workforce-operator@synthetic.invalid'
            FirstName = 'Synthetic'
            LastName = 'Workforce Operator'
            IdleSeconds = 600
            MaximumSeconds = 3600
        }
    )

    foreach ($population in $populations) {
        $realmName = $population.Realm
        $realm = Invoke-IdentityAdmin -Method Get -Path "/admin/realms/$realmName"
        Set-ObjectProperty -InputObject $realm -Name 'registrationAllowed' -Value $false
        Set-ObjectProperty -InputObject $realm -Name 'resetPasswordAllowed' -Value $false
        Set-ObjectProperty -InputObject $realm -Name 'rememberMe' -Value $false
        Set-ObjectProperty -InputObject $realm -Name 'ssoSessionIdleTimeout' -Value $population.IdleSeconds
        Set-ObjectProperty -InputObject $realm -Name 'ssoSessionMaxLifespan' -Value $population.MaximumSeconds
        Set-ObjectProperty -InputObject $realm -Name 'accessTokenLifespan' -Value 300
        Invoke-IdentityAdmin -Method Put -Path "/admin/realms/$realmName" -Body $realm | Out-Null

        $clientResponse = Invoke-IdentityAdmin -Method Get -Path "/admin/realms/$realmName/clients?clientId=$clientID"
        $clients = @($clientResponse | ForEach-Object { $_ })
        if ($clients.Count -eq 0) {
            $newClient = @{
                clientId = $clientID
                name = 'Atlas BFF - Synthetic Local'
                enabled = $true
                publicClient = $true
                standardFlowEnabled = $true
                directAccessGrantsEnabled = $false
                implicitFlowEnabled = $false
                serviceAccountsEnabled = $false
                redirectUris = @($callback)
                webOrigins = @($webOrigin)
                attributes = @{
                    'pkce.code.challenge.method' = 'S256'
                    'post.logout.redirect.uris' = "$webOrigin/signed-out"
                }
            }
            Invoke-IdentityAdmin -Method Post -Path "/admin/realms/$realmName/clients" -Body $newClient | Out-Null
            $clientResponse = Invoke-IdentityAdmin -Method Get -Path "/admin/realms/$realmName/clients?clientId=$clientID"
            $clients = @($clientResponse | ForEach-Object { $_ })
        }
        if ($clients.Count -ne 1) {
            throw "Synthetic realm $realmName did not contain exactly one $clientID client"
        }

        $client = Invoke-IdentityAdmin -Method Get -Path "/admin/realms/$realmName/clients/$($clients[0].id)"
        Set-ObjectProperty -InputObject $client -Name 'enabled' -Value $true
        Set-ObjectProperty -InputObject $client -Name 'publicClient' -Value $true
        Set-ObjectProperty -InputObject $client -Name 'standardFlowEnabled' -Value $true
        Set-ObjectProperty -InputObject $client -Name 'directAccessGrantsEnabled' -Value $false
        Set-ObjectProperty -InputObject $client -Name 'implicitFlowEnabled' -Value $false
        Set-ObjectProperty -InputObject $client -Name 'serviceAccountsEnabled' -Value $false
        Set-ObjectProperty -InputObject $client -Name 'redirectUris' -Value @($callback)
        Set-ObjectProperty -InputObject $client -Name 'webOrigins' -Value @($webOrigin)
        Set-ObjectProperty -InputObject $client -Name 'attributes' -Value @{
            'pkce.code.challenge.method' = 'S256'
            'post.logout.redirect.uris' = "$webOrigin/signed-out"
        }
        Invoke-IdentityAdmin -Method Put -Path "/admin/realms/$realmName/clients/$($clients[0].id)" -Body $client | Out-Null

        $encodedUsername = [Uri]::EscapeDataString($population.Username)
        $userResponse = Invoke-IdentityAdmin -Method Get -Path "/admin/realms/$realmName/users?username=$encodedUsername&exact=true"
        $users = @($userResponse | ForEach-Object { $_ })
        if ($users.Count -ne 1 -or [string]$users[0].id -ne [string]$population.Subject) {
            throw "Synthetic realm $realmName user identity does not match the source-controlled external subject"
        }
        if (-not [bool]$users[0].enabled) {
            throw "Synthetic realm $realmName user is disabled"
        }

        $user = Invoke-IdentityAdmin -Method Get -Path "/admin/realms/$realmName/users/$($users[0].id)"
        Set-ObjectProperty -InputObject $user -Name 'enabled' -Value $true
        Set-ObjectProperty -InputObject $user -Name 'emailVerified' -Value $true
        Set-ObjectProperty -InputObject $user -Name 'email' -Value $population.Email
        Set-ObjectProperty -InputObject $user -Name 'firstName' -Value $population.FirstName
        Set-ObjectProperty -InputObject $user -Name 'lastName' -Value $population.LastName
        Set-ObjectProperty -InputObject $user -Name 'requiredActions' -Value @()
        Invoke-IdentityAdmin -Method Put -Path "/admin/realms/$realmName/users/$($users[0].id)" -Body $user | Out-Null

        $credential = @{
            type = 'password'
            value = $runtime['ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD']
            temporary = $false
        }
        Invoke-IdentityAdmin `
            -Method Put `
            -Path "/admin/realms/$realmName/users/$($users[0].id)/reset-password" `
            -Body $credential | Out-Null
    }

    Write-Output 'p01_s04_keycloak_clients=PASS(count=3,public=true,pkce=S256,direct_grants=false)'
    Write-Output 'p01_s04_keycloak_subjects=PASS(count=3,passwords=runtime-only)'
}
finally {
    $script:adminToken = $null
    if ($null -ne $runtime) {
        foreach ($key in @(
            'ATLAS_KEYCLOAK_ADMIN_PASSWORD',
            'ATLAS_SYNTHETIC_OIDC_TEST_PASSWORD'
        )) {
            if ($runtime.ContainsKey($key)) {
                $runtime[$key] = $null
            }
        }
    }
}
