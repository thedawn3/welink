param(
  [string]$DataDir = $env:WELINK_DATA_DIR,
  [string]$MsgDir = $env:WELINK_MSG_DIR
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..")
$DoctorScript = Join-Path $ScriptDir "welink-doctor.ps1"
$EnvFile = Join-Path $RepoRoot ".env"

function Get-EnvValueFromFile {
  param(
    [string]$Key,
    [string]$Fallback
  )

  if (Test-Path $EnvFile) {
    $line = Get-Content $EnvFile | Where-Object { $_ -match "^${Key}=" } | Select-Object -Last 1
    if ($line) {
      return ($line -replace "^${Key}=", "")
    }
  }
  return $Fallback
}

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
  $FrontendPort = Get-EnvValueFromFile -Key "WELINK_FRONTEND_PORT" -Fallback "3000"
  $BackendPort = Get-EnvValueFromFile -Key "WELINK_BACKEND_PORT" -Fallback "8080"
  Write-Host "WeLink started."
  Write-Host "Local frontend: http://localhost:$FrontendPort"
  Write-Host "Local backend : http://localhost:$BackendPort"
} finally {
  Pop-Location
}
