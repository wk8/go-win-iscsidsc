## Meant to be sourced by the other scripts in this dir

# exit on error
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSDefaultParameterValues['*:ErrorAction'] = 'Stop'

if (-not $env:TEMP) {
    throw 'No temp directory set in the TEMP env variable?'
}
$disksDir = "$env:TEMP\go-win-iscsids-disks\"
$disksExtension = '.vhdx'

function disksDirForTarget([String]$targetIQN) {
    # Windows paths can't contain colons, hostnames can't contain tildes
    return $disksDir + $targetIQN.Replace(':', '~')
}

function targetIQNFromDisksDir([String]$targetDir) {
    return (Get-Item $targetDir).BaseName.Replace('~', ':')
}

# meant for best-effort cleanups
function tryIgnore([ScriptBlock]$callback) {
    try {
        $callback.Invoke()
    } catch {
        # intentionally left blank
    }
}

$repoRootDir = Split-Path -Parent $PSScriptRoot
