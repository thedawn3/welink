param(
  [ValidateSet('analysis-only', 'manual-sync', 'decrypt-first')]
  [string]$Mode = 'analysis-only',
  [string]$Platform = 'auto',
  [string]$DataDir = '',
  [string]$SourceDataDir = '',
  [string]$WorkDir = '',
  [string]$MsgDir = '',
  [string]$WechatDecryptDir = '',
  [switch]$WriteEnv
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ArgsList = @('--mode', $Mode, '--platform', $Platform)
if ($DataDir) { $ArgsList += @('--data-dir', $DataDir) }
if ($SourceDataDir) { $ArgsList += @('--source-data-dir', $SourceDataDir) }
if ($WorkDir) { $ArgsList += @('--work-dir', $WorkDir) }
if ($MsgDir) { $ArgsList += @('--msg-dir', $MsgDir) }
if ($WechatDecryptDir) { $ArgsList += @('--wechat-decrypt-dir', $WechatDecryptDir) }
if ($WriteEnv) { $ArgsList += '--write-env' }

if (Get-Command py -ErrorAction SilentlyContinue) {
  py -3 "$ScriptDir/welink_doctor.py" @ArgsList
  exit $LASTEXITCODE
}

if (Get-Command python -ErrorAction SilentlyContinue) {
  python "$ScriptDir/welink_doctor.py" @ArgsList
  exit $LASTEXITCODE
}

throw "Python 3 not found. Install Python and make sure 'py -3' or 'python' is available in PATH before running welink-doctor.ps1."
