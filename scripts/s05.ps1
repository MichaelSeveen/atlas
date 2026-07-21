[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidateSet('Up', 'Migrate', 'Verify', 'BackupRestore', 'Down')]
    [string]$Action,

    [Parameter()]
    [ValidateSet('podman', 'docker')]
    [string]$ContainerRuntime = 'podman'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repositoryRoot 'deploy/local/compose.yaml'
$configDirectory = Join-Path $repositoryRoot 'deploy/environments'
$stateRoot = Join-Path $repositoryRoot '.tmp/environments'
$runtimeFile = Join-Path $stateRoot 'local/runtime.env'

function Invoke-NativeChecked {
    param([string]$Command, [string[]]$Arguments = @())
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: $Command $($Arguments -join ' ')"
    }
}

function Initialize-DatabaseEnvironment {
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'validate', '--config-dir', $configDirectory)
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/envctl', 'prepare', '--environment', 'local', '--config-dir', $configDirectory, '--state-root', $stateRoot)
    Invoke-NativeChecked -Command 'go' -Arguments @('run', './cmd/dbctl', 'verify', '--migration-dir', (Join-Path $repositoryRoot 'db/migrations'))
}

function Invoke-Compose {
    param([string[]]$Arguments)
    Invoke-NativeChecked -Command $ContainerRuntime -Arguments (@('compose', '--env-file', $runtimeFile, '--file', $composeFile) + $Arguments)
}

function Wait-Postgres {
    param([string]$Service = 'postgres')
    $deadline = [DateTimeOffset]::UtcNow.AddMinutes(2)
    $composeArguments = @('compose', '--env-file', $runtimeFile, '--file', $composeFile)
    if ($Service -eq 'postgres-restore') {
        $composeArguments += @('--profile', 'recovery')
    }
    $composeArguments += @('exec', '-T', $Service, 'pg_isready', '-h', '127.0.0.1', '-p', '5432')
    do {
        & $ContainerRuntime @composeArguments *> $null
        if ($LASTEXITCODE -eq 0) { return }
        Start-Sleep -Seconds 2
    } while ([DateTimeOffset]::UtcNow -lt $deadline)
    throw "$Service did not become ready before the bounded deadline"
}

function Start-DatabaseFoundation {
    Invoke-Compose -Arguments @('up', '--detach', 'postgres', 'nats')
    Wait-Postgres
}

function Invoke-DatabaseScript {
    param([string]$Path)
    Invoke-Compose -Arguments @('exec', '-T', 'postgres', 'sh', $Path)
}

function Initialize-RolesAndMigrations {
    Invoke-DatabaseScript -Path '/database/roles/bootstrap.sh'
    Invoke-DatabaseScript -Path '/database/tools/apply-migrations.sh'
}

function Test-RealBroker {
    $server = Invoke-RestMethod -TimeoutSec 5 -Uri 'http://127.0.0.1:18222/varz'
    $jetstream = Invoke-RestMethod -TimeoutSec 5 -Uri 'http://127.0.0.1:18222/jsz'
    if ($null -eq $server.jetstream -or $null -eq $jetstream.memory -or $null -eq $jetstream.storage) {
        throw 'real NATS JetStream integration dependency is unavailable'
    }
    Write-Output 'database_integration_broker=REAL_NATS_JETSTREAM'
}

function Invoke-BackupRestore {
    Invoke-DatabaseScript -Path '/database/recovery/backup.sh'
    $started = [DateTimeOffset]::UtcNow
    Invoke-Compose -Arguments @('--profile', 'recovery', 'up', '--detach', '--force-recreate', 'postgres-restore')
    Wait-Postgres -Service 'postgres-restore'
    Invoke-Compose -Arguments @('--profile', 'recovery', 'exec', '-T', 'postgres-restore', 'sh', '/recovery/verify-restore.sh')
    $elapsed = [Math]::Ceiling(([DateTimeOffset]::UtcNow - $started).TotalSeconds)
    Write-Output "database_restore_rto_seconds=$elapsed"
    Invoke-Compose -Arguments @('--profile', 'recovery', 'stop', 'postgres-restore')
}

Push-Location -LiteralPath $repositoryRoot
try {
    switch ($Action) {
        'Up' {
            Initialize-DatabaseEnvironment
            Start-DatabaseFoundation
            Write-Output 's05_database_up=PASS'
        }
        'Migrate' {
            Initialize-DatabaseEnvironment
            Start-DatabaseFoundation
            Initialize-RolesAndMigrations
            Write-Output 's05_database_migrate=PASS'
        }
        'Verify' {
            Initialize-DatabaseEnvironment
            Start-DatabaseFoundation
            Initialize-RolesAndMigrations
            Invoke-DatabaseScript -Path '/database/tests/migration_lanes.sh'
            Invoke-DatabaseScript -Path '/database/tests/role_matrix.sh'
            Invoke-DatabaseScript -Path '/database/tests/lock_timeout.sh'
            Test-RealBroker
            Invoke-BackupRestore
            Write-Output 's05_database_verify=PASS'
        }
        'BackupRestore' {
            Initialize-DatabaseEnvironment
            Start-DatabaseFoundation
            Initialize-RolesAndMigrations
            Invoke-BackupRestore
            Write-Output 's05_backup_restore=PASS'
        }
        'Down' {
            if (Test-Path -LiteralPath $runtimeFile) {
                Invoke-Compose -Arguments @('--profile', 'recovery', 'down', '--remove-orphans')
            }
            Write-Output 's05_database_down=PASS'
        }
    }
}
finally {
    Pop-Location
}
