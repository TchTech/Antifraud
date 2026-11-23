param([string]$PgBin = 'C:/Program Files/PostgreSQL/18/bin')

$ErrorActionPreference = 'SilentlyContinue'
Set-Location (Split-Path $PSScriptRoot -Parent)

Write-Host "`n=== Остановка учебного стенда ===" -ForegroundColor Cyan

if (Test-Path .local/demo-processes.json) {
    $p = Get-Content .local/demo-processes.json -Raw | ConvertFrom-Json
    foreach ($id in @($p.risk, $p.gateway)) {
        Stop-Process -Id $id -Force -ErrorAction SilentlyContinue
    }
    Remove-Item .local/demo-processes.json -Force -ErrorAction SilentlyContinue
}

Get-Process -Name risk, gateway -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Write-Host "-> Сервисы Risk и Gateway остановлены." -ForegroundColor Yellow

if (Test-Path "$PgBin/pg_ctl.exe") {
    & "$PgBin/pg_ctl.exe" -D "$PWD/.local/pgdata" stop *> $null
    Write-Host "-> PostgreSQL остановлен." -ForegroundColor Yellow
}

Write-Host "[OK] Стенд полностью остановлен.`n" -ForegroundColor Green
