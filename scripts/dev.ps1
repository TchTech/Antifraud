param([ValidateSet('generate','check','e2e','demo','migrate','build','android')][string]$Task='check')
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
$env:GOCACHE = "$PWD/.local/go-build"
$env:GOMODCACHE = "$PWD/.local/go-mod"
function Run([scriptblock]$Command) { & $Command; if ($LASTEXITCODE) { throw "Command failed: $Command ($LASTEXITCODE)" } }
switch ($Task) {
    generate { Run { & ./scripts/generate.ps1 } }
    check { Run { go vet ./... }; Run { go test ./... } }
    e2e { Run { go test -tags integration -count=1 -v ./tests/... } }
    migrate { Run { go run ./cmd/risk -migrate up } }
    build {
        New-Item -ItemType Directory -Force .local/bin | Out-Null
        Run { go build -o .local/bin/gateway.exe ./cmd/gateway }
        Run { go build -o .local/bin/risk.exe ./cmd/risk }
        Run { go build -o .local/bin/network-collector.exe ./cmd/network-collector }
    }
    demo { Run { docker compose up --build -d --wait }; Run { python scripts/demo.py } }
    android { Run { & ./apps/mobile/gradlew.bat -p apps/mobile assembleDebug --no-daemon } }
}
