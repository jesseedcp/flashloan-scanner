param(
    [string]$Config = ".\flashloan-scanner.local.yaml",
    [string]$MigrationsDir = ".\migrations",
    [switch]$SkipMigrate,
    [switch]$LoadFixture
)

$ErrorActionPreference = "Stop"

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Action
    )

    Write-Host ""
    Write-Host "==> $Name" -ForegroundColor Cyan
    & $Action
}

function Assert-PathExists {
    param(
        [string]$Path,
        [string]$Label
    )

    if (-not (Test-Path -Path $Path)) {
        throw "$Label not found: $Path"
    }
}

function Resolve-ExistingPath {
    param(
        [string]$Path,
        [string]$Label
    )

    Assert-PathExists -Path $Path -Label $Label
    return (Resolve-Path -Path $Path).Path
}

function Invoke-GoRun {
    param(
        [string[]]$CommandArgs
    )

    & go @CommandArgs
    if ($LASTEXITCODE -ne 0) {
        throw "go command failed with exit code ${LASTEXITCODE}: go $($CommandArgs -join ' ')"
    }
}

$Config = Resolve-ExistingPath -Path $Config -Label "config file"
$MigrationsDir = Resolve-ExistingPath -Path $MigrationsDir -Label "migrations directory"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    if (-not $SkipMigrate) {
        Invoke-Step -Name "Running database migrations" -Action {
            Invoke-GoRun -CommandArgs @("run", ".\cmd\flashloan-scanner", "migrate", "--config", $Config, "--migrations-dir", $MigrationsDir)
        }
    }

    if ($LoadFixture) {
        Invoke-Step -Name "Loading local flashloan fixture" -Action {
            Invoke-GoRun -CommandArgs @("run", ".\cmd\flashloan-scanner", "flashloan-fixture", "--config", $Config)
        }
    }

    Invoke-Step -Name "Running flashloan scanner once" -Action {
        Invoke-GoRun -CommandArgs @("run", ".\cmd\flashloan-scanner", "flashloan-scan", "--config", $Config)
    }
}
finally {
    Pop-Location
}

