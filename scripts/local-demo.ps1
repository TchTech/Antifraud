param([string]$PgBin='C:/Program Files/PostgreSQL/18/bin')
$ErrorActionPreference='Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
function Run([scriptblock]$Command) { & $Command; if ($LASTEXITCODE) { throw "Failed: $Command" } }
New-Item -ItemType Directory -Force .local | Out-Null
if (!(Test-Path .local/pgdata/PG_VERSION)) { Run { & "$PgBin/initdb.exe" -D "$PWD/.local/pgdata" -U demo --auth=trust --encoding=UTF8 --locale=C } }
& "$PgBin/pg_ctl.exe" -D "$PWD/.local/pgdata" status *> $null
if ($LASTEXITCODE -ne 0) { Run { & "$PgBin/pg_ctl.exe" -D "$PWD/.local/pgdata" -l "$PWD/.local/postgres.log" -o '-p 55432 -h 127.0.0.1' start } }
$exists = & "$PgBin/psql.exe" -h 127.0.0.1 -p 55432 -U demo -d postgres -Atc "SELECT 1 FROM pg_database WHERE datname='antifraud_demo'"
if ($exists -ne '1') { Run { & "$PgBin/createdb.exe" -h 127.0.0.1 -p 55432 -U demo antifraud_demo } }
if (Test-Path .local/demo-processes.json) {
    $previous=Get-Content .local/demo-processes.json -Raw | ConvertFrom-Json
    foreach ($processId in @($previous.risk,$previous.gateway)) {
        $running=Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($running -and $running.Path -in @("$PWD/.local/bin/risk.exe".Replace('/','\'),"$PWD/.local/bin/gateway.exe".Replace('/','\'))) { Stop-Process -Id $processId }
    }
}
Run { & ./scripts/dev.ps1 build }
$env:DATABASE_URL='postgres://demo@127.0.0.1:55432/antifraud_demo?sslmode=disable'
Run { & .local/bin/risk.exe -migrate up }
Run { & .local/bin/network-collector.exe -once }
$riskProcess=Start-Process -FilePath "$PWD/.local/bin/risk.exe" -WorkingDirectory $PWD -WindowStyle Hidden -RedirectStandardOutput "$PWD/.local/risk.stdout.log" -RedirectStandardError "$PWD/.local/risk.stderr.log" -PassThru
$gatewayProcess=Start-Process -FilePath "$PWD/.local/bin/gateway.exe" -WorkingDirectory $PWD -WindowStyle Hidden -RedirectStandardOutput "$PWD/.local/gateway.stdout.log" -RedirectStandardError "$PWD/.local/gateway.stderr.log" -PassThru
@{risk=$riskProcess.Id;gateway=$gatewayProcess.Id} | ConvertTo-Json | Set-Content .local/demo-processes.json
Run { python scripts/demo.py }
