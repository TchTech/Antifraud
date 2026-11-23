param(
    [string]$PgBin = 'C:/Program Files/PostgreSQL/18/bin',
    [switch]$NoApp
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)

function Run([scriptblock]$Command) { & $Command; if ($LASTEXITCODE) { throw "Command failed: $Command" } }

Write-Host "`n=== Запуск учебного стенда Сессионного антифрода ===" -ForegroundColor Cyan

# 1. Проверка наличия PostgreSQL
if (!(Test-Path "$PgBin/pg_ctl.exe")) {
    throw "PostgreSQL 18 не найден в '$PgBin'. Укажите верный путь через параметр -PgBin."
}

New-Item -ItemType Directory -Force .local | Out-Null

# 2. Инициализация кластера при необходимости
if (!(Test-Path .local/pgdata/PG_VERSION)) {
    Write-Host "-> Инициализация локального pgdata..." -ForegroundColor Yellow
    Run { & "$PgBin/initdb.exe" -D "$PWD/.local/pgdata" -U demo --auth=trust --encoding=UTF8 --locale=C }
}

# 3. Запуск PostgreSQL на порту 55432
& "$PgBin/pg_ctl.exe" -D "$PWD/.local/pgdata" status *> $null
if ($LASTEXITCODE -ne 0) {
    Write-Host "-> Запуск PostgreSQL на 127.0.0.1:55432..." -ForegroundColor Yellow
    Run { & "$PgBin/pg_ctl.exe" -D "$PWD/.local/pgdata" -l "$PWD/.local/postgres.log" -o '-p 55432 -h 127.0.0.1' start }
    Start-Sleep -Seconds 2
}

# 4. Очистка и пересоздание antifraud_demo для свежих синтетических сигналов
Write-Host "-> Подготовка чистой БД antifraud_demo (свежие синтетические метки)..." -ForegroundColor Yellow
& "$PgBin/dropdb.exe" -h 127.0.0.1 -p 55432 -U demo --if-exists antifraud_demo *> $null
Run { & "$PgBin/createdb.exe" -h 127.0.0.1 -p 55432 -U demo antifraud_demo }

# 5. Остановка предыдущих процессов risk/gateway
if (Test-Path .local/demo-processes.json) {
    $previous = Get-Content .local/demo-processes.json -Raw | ConvertFrom-Json
    foreach ($processId in @($previous.risk, $previous.gateway)) {
        $running = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($running) { Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue }
    }
}
Get-Process -Name risk, gateway -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

# 6. Сборка бинарников
Write-Host "-> Сборка сервисов (gateway, risk, network-collector)..." -ForegroundColor Yellow
Run { & ./scripts/dev.ps1 build }

# 7. Миграции и посев сигналов
$env:DATABASE_URL = 'postgres://demo@127.0.0.1:55432/antifraud_demo?sslmode=disable'
Write-Host "-> Применение Goose миграций..." -ForegroundColor Yellow
Run { & .local/bin/risk.exe -migrate up }

Write-Host "-> Посев синтетических сигналов..." -ForegroundColor Yellow
Run { & .local/bin/network-collector.exe -once }

# 8. Запуск Risk и Gateway
Write-Host "-> Запуск сервисов Risk (:8081) и Gateway (:8080)..." -ForegroundColor Yellow
$riskProcess = Start-Process -FilePath "$PWD/.local/bin/risk.exe" -WorkingDirectory $PWD -WindowStyle Hidden -RedirectStandardOutput "$PWD/.local/risk.stdout.log" -RedirectStandardError "$PWD/.local/risk.stderr.log" -PassThru
$gatewayProcess = Start-Process -FilePath "$PWD/.local/bin/gateway.exe" -WorkingDirectory $PWD -WindowStyle Hidden -RedirectStandardOutput "$PWD/.local/gateway.stdout.log" -RedirectStandardError "$PWD/.local/gateway.stderr.log" -PassThru

@{risk = $riskProcess.Id; gateway = $gatewayProcess.Id} | ConvertTo-Json | Set-Content .local/demo-processes.json

Start-Sleep -Seconds 1

# Проверка ответа gateway
try {
    $resp = Invoke-RestMethod -Uri "http://127.0.0.1:8080/v1/state/demo-legal" -Method Get -TimeoutSec 3
    Write-Host "-> Gateway готов и слушает порт 8080!" -ForegroundColor Green
} catch {
    Write-Warning "Gateway инициализируется: $_"
}

# 9. Настройка ADB и запуск Android приложения
if (!$NoApp) {
    $adb = Get-Command adb -ErrorAction SilentlyContinue
    if ($adb) {
        $devices = & adb devices | Select-String -Pattern "\bdevice\b"
        if ($devices) {
            Write-Host "-> Проброс порта 8080 на устройство через adb reverse..." -ForegroundColor Yellow
            & adb reverse tcp:8080 tcp:8080 2>$null
            Write-Host "-> Открытие экрана приложения в эмуляторе..." -ForegroundColor Yellow
            $prevEA = $ErrorActionPreference
            $ErrorActionPreference = 'Continue'
            & adb shell am start -n edu.antifraud.demo/.MainActivity 2>$null | Out-Null
            $ErrorActionPreference = $prevEA
            Write-Host "-> Приложение готово на экране!" -ForegroundColor Green
        } else {
            Write-Host "-> Android устройство/эмулятор не обнаружен." -ForegroundColor DarkYellow
        }
    }
}

Write-Host "`n[OK] Платформа успешно запущена!" -ForegroundColor Green
Write-Host "Шлюз API: http://127.0.0.1:8080 (в приложении: http://10.0.2.2:8080 или http://127.0.0.1:8080)"
Write-Host "Для остановки стенда выполните: .\scripts\stop-stand.ps1`n"
