param(
  [string]$DataDir = $env:WELINK_DATA_DIR,
  [string]$MsgDir = $env:WELINK_MSG_DIR
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..")
$DoctorScript = Join-Path $ScriptDir "welink-doctor.ps1"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "docker command not found. Install Docker Desktop first."
}

try {
  docker compose version | Out-Null
} catch {
  throw "'docker compose' is unavailable. Enable Docker Compose v2 in Docker Desktop."
}

& $DoctorScript -DataDir $DataDir -MsgDir $MsgDir -WriteEnv

Push-Location $RepoRoot
try {
  docker compose up -d --build
  Write-Host "WeLink started."
  Write-Host "Frontend: http://localhost:3000"
  Write-Host "Backend : http://localhost:8080"
} finally {
  Pop-Location
}
