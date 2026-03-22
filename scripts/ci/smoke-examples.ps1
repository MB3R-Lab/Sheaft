param(
    [string]$OutRoot = ".tmp/smoke-examples",
    [string]$BinPath = "./bin/sheaft.exe",
    [int]$ServePort = 18080
)

$ErrorActionPreference = "Stop"

if (Test-Path $OutRoot) {
    Remove-Item -Recurse -Force $OutRoot
}

New-Item -ItemType Directory -Force (Join-Path $OutRoot "policy") | Out-Null
New-Item -ItemType Directory -Force (Join-Path $OutRoot "analysis") | Out-Null

$repoRoot = (Get-Location).Path.Replace('\', '/')
$serveConfig = Join-Path $OutRoot "sheaft.serve.yaml"
@"
schema_version: "1.0"
listen: ":$ServePort"

artifact:
  path: "$repoRoot/examples/outputs/snapshot.sample.json"
  mode: file

analysis_file: "$repoRoot/configs/analysis.example.yaml"

poll_interval: 30s
watch_fs: true
watch_polling: true

history:
  max_items: 20
  disk_dir: "$repoRoot/.sheaft/history"
"@ | Set-Content -Path $serveConfig -Encoding utf8

& $BinPath run `
    --model examples/outputs/model.sample.json `
    --policy configs/gate.policy.example.yaml `
    --out-dir (Join-Path $OutRoot "policy") `
    --seed 42

& $BinPath run `
    --model examples/outputs/snapshot.sample.json `
    --analysis configs/analysis.example.yaml `
    --out-dir (Join-Path $OutRoot "analysis")

$serveOut = Join-Path $OutRoot "serve.stdout.log"
$serveErr = Join-Path $OutRoot "serve.stderr.log"
$readyOut = Join-Path $OutRoot "readyz.json"
$reportOut = Join-Path $OutRoot "current-report.json"

$proc = Start-Process -FilePath $BinPath `
    -ArgumentList @("serve", "--config", $serveConfig) `
    -PassThru `
    -RedirectStandardOutput $serveOut `
    -RedirectStandardError $serveErr

try {
    $healthy = $false
    for ($attempt = 0; $attempt -lt 20; $attempt++) {
        try {
            $null = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:$ServePort/healthz" -TimeoutSec 3
            $healthy = $true
            break
        } catch {
            Start-Sleep -Seconds 1
        }
    }

    if (-not $healthy) {
        if (Test-Path $serveOut) { Get-Content $serveOut | Write-Error }
        if (Test-Path $serveErr) { Get-Content $serveErr | Write-Error }
        throw "sheaft serve did not become reachable on :$ServePort"
    }

    $ready = Invoke-RestMethod -Uri "http://127.0.0.1:$ServePort/readyz" -TimeoutSec 3
    $ready | ConvertTo-Json -Depth 10 | Set-Content -Path $readyOut
    if (-not $ready.ready) {
        throw "sheaft serve started but did not reach ready=true"
    }

    Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:$ServePort/current-report" -TimeoutSec 3 |
        Select-Object -ExpandProperty Content |
        Set-Content -Path $reportOut
} finally {
    if ($proc -and -not $proc.HasExited) {
        Stop-Process -Id $proc.Id -Force
    }
}
