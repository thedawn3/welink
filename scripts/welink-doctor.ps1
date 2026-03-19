param(
  [string]$Platform = 'auto',
  [string]$DataDir = '',
  [string]$MsgDir = '',
  [switch]$WriteEnv
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ArgsList = @('--platform', $Platform)
if ($DataDir) { $ArgsList += @('--data-dir', $DataDir) }
if ($MsgDir) { $ArgsList += @('--msg-dir', $MsgDir) }
if ($WriteEnv) { $ArgsList += '--write-env' }
if (Get-Command py -ErrorAction SilentlyContinue) {
  py -3 "$ScriptDir/welink_doctor.py" @ArgsList
} else {
  python "$ScriptDir/welink_doctor.py" @ArgsList
}
