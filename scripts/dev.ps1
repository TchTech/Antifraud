param(
    [ValidateSet('generate','lint','compose-lint','check','e2e','protection','evaluate','evaluate-burst','load','proof','demo','migrate','build','android')][string]$Task='check',
    [string]$Baseline,
    [string]$Output,
    [string]$ServerRef,
    [string]$Profile,
    [switch]$CheckBudget
)
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
$env:GOCACHE = "$PWD/.local/go-build"
$env:GOMODCACHE = "$PWD/.local/go-mod"
function Run([scriptblock]$Command) { & $Command; if ($LASTEXITCODE) { throw "Command failed: $Command ($LASTEXITCODE)" } }
function Lint {
    Run { python scripts/lint.py }
    Run { go vet ./... }
}
switch ($Task) {
    generate { Run { & ./scripts/generate.ps1 } }
    lint { Lint }
    compose-lint { Run { python scripts/lint.py --compose } }
    check { Lint; Run { go test ./... } }
    e2e { Run { go test -tags integration -count=1 -v ./tests/... } }
    protection { Run { go test -tags integration -count=1 -json -run '^TestSyntheticProtectionControls$' ./tests } }
    evaluate {
        $evaluationArgs = @()
        if ($Baseline) { $evaluationArgs += @('--baseline', $Baseline) }
        if ($Output) { $evaluationArgs += @('--output', $Output) }
        Run { python scripts/evaluate.py @evaluationArgs }
    }
    'evaluate-burst' {
        Run { python scripts/evaluate.py }
        Run { python scripts/evaluate.py --burst --baseline .local/evaluation/metrics.json --output .local/evaluation-burst }
    }
    load {
        $loadArgs = @()
        if ($Baseline) { $loadArgs += @('--baseline', $Baseline) }
        if ($Output) { $loadArgs += @('--output', $Output) }
        if ($ServerRef) { $loadArgs += @('--server-ref', $ServerRef) }
        if ($Profile) { $loadArgs += @('--profile', $Profile) }
        if ($CheckBudget) { $loadArgs += '--check' }
        Run { python scripts/load.py @loadArgs }
    }
    proof {
        $version = (Get-Content formal/lean-toolchain -Raw).Trim() -replace '^leanprover/lean4:v', ''
        $cachedLean = Join-Path $PWD ".local/lean-toolchain/lean-$version-windows/bin/lean.exe"
        $leanCommand = if (Test-Path -LiteralPath $cachedLean) { $cachedLean } else { 'lean' }
        Push-Location formal
        try { Run { & $leanCommand -DwarningAsError=true Authorization.lean } }
        finally { Pop-Location }
    }
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
