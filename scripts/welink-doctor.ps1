param(
  [ValidateSet('analysis-only', 'manual-sync', 'decrypt-first')]
  [string]$Mode = 'analysis-only',
  [string]$Platform = 'auto',
  [string]$DataDir = '',
  [string]$SourceDataDir = '',
  [string]$WorkDir = '',
  [string]$MsgDir = '',
  [switch]$WriteEnv
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ArgsList = @('--mode', $Mode, '--platform', $Platform)
if ($DataDir) { $ArgsList += @('--data-dir', $DataDir) }
if ($SourceDataDir) { $ArgsList += @('--source-data-dir', $SourceDataDir) }
if ($WorkDir) { $ArgsList += @('--work-dir', $WorkDir) }
if ($MsgDir) { $ArgsList += @('--msg-dir', $MsgDir) }
if ($WriteEnv) { $ArgsList += '--write-env' }
if (Get-Command py -ErrorAction SilentlyContinue) {
  py -3 "$ScriptDir/welink_doctor.py" @ArgsList
} else {
  python "$ScriptDir/welink_doctor.py" @ArgsList
}
