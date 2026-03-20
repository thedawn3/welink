param(
  [ValidateSet('analysis-only', 'manual-sync', 'decrypt-first')]
  [string]$Mode = $(if ($env:WELINK_MODE) { $env:WELINK_MODE } else { 'analysis-only' }),
  [string]$Platform = $(if ($env:WELINK_PLATFORM) { $env:WELINK_PLATFORM } else { 'auto' }),
  [string]$DataDir = $(if ($env:WELINK_ANALYSIS_DATA_DIR) { $env:WELINK_ANALYSIS_DATA_DIR } else { $env:WELINK_DATA_DIR }),
  [string]$SourceDataDir = $env:WELINK_SOURCE_DATA_DIR,
  [string]$WorkDir = $env:WELINK_WORK_DIR,
  [string]$MsgDir = $env:WELINK_MSG_DIR,
  [string]$WechatDecryptDir = $env:WELINK_WECHAT_DECRYPT_DIR
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

& $DoctorScript -Mode $Mode -Platform $Platform -DataDir $DataDir -SourceDataDir $SourceDataDir -WorkDir $WorkDir -MsgDir $MsgDir -WechatDecryptDir $WechatDecryptDir -WriteEnv

Push-Location $RepoRoot
try {
  docker compose up -d --build
  $ResolvedMode = Get-EnvValueFromFile -Key "WELINK_MODE" -Fallback $Mode
  $ResolvedPlatform = Get-EnvValueFromFile -Key "WELINK_PLATFORM" -Fallback $Platform
  $ResolvedSourceDataDir = Get-EnvValueFromFile -Key "WELINK_SOURCE_DATA_DIR" -Fallback $SourceDataDir
  $ResolvedWorkDir = Get-EnvValueFromFile -Key "WELINK_WORK_DIR" -Fallback $WorkDir
  $ResolvedWechatDecryptDir = Get-EnvValueFromFile -Key "WELINK_WECHAT_DECRYPT_DIR" -Fallback $WechatDecryptDir
  $FrontendPort = Get-EnvValueFromFile -Key "WELINK_FRONTEND_PORT" -Fallback "3000"
  $BackendPort = Get-EnvValueFromFile -Key "WELINK_BACKEND_PORT" -Fallback "8080"
  Write-Host "WeLink started."
  Write-Host "Local frontend: http://localhost:$FrontendPort"
  Write-Host "Local backend : http://localhost:$BackendPort"
  if ($ResolvedWechatDecryptDir) {
    Write-Host "wechat-decrypt: $ResolvedWechatDecryptDir"
  }

  if ($ResolvedMode -eq "decrypt-first") {
    Write-Host ""
    Write-Host "decrypt-first mode detected."
    Write-Host "Backend is configured to auto-start decrypt on boot."
    Write-Host "Manual override example:"
    Write-Host "curl -Method Post http://localhost:$BackendPort/api/system/decrypt/start -ContentType 'application/json' -Body '{`"platform`":`"$ResolvedPlatform`",`"source_data_dir`":`"$ResolvedSourceDataDir`",`"work_dir`":`"$ResolvedWorkDir`",`"auto_refresh`":true,`"wal_enabled`":true}'"
    Write-Host ""
    Write-Host "Check runtime:"
    Write-Host "curl http://localhost:$BackendPort/api/system/runtime"
  }
} finally {
  Pop-Location
}
