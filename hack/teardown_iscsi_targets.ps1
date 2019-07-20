<#
.Synopsis
 Tears down one or all iSCSI target(s) created by setup_iscsi_target.ps1.
#>
Param(
    [Parameter(Position = 0, Mandatory=$false)] [String] $TargetIQN # if left empty, will remove em' all
)

# exit on error
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSDefaultParameterValues['*:ErrorAction'] = 'Stop'

. "$PSScriptRoot/.common_env.ps1"

$global:serviceStopped = $false

function removeTargetDir([String]$targetDir)  {
    $targetIQN = targetIQNFromDisksDir $targetDir
    Write-Host -ForegroundColor green "Removing target $targetIQN"

    try {
        removeTarget $targetDir $targetIQN
    } catch {
        if (-not $global:serviceStopped -and ((Get-Service WinTarget).Status -eq 'Running')) {
            Write-Host -ForegroundColor green 'Stopping the service...'
            Stop-Service WinTarget
            $global:serviceStopped = $true

            removeTarget $targetDir $targetIQN
        } else {
            throw
        }
    }
}

function removeTarget([String]$targetDir, [String]$targetIQN) {
    Get-ChildItem $targetDir -Filter "*$disksExtension" | Foreach-Object {
        tryIgnore { Remove-IscsiVirtualDiskTargetMapping -TargetName $targetIQN -Path $_.FullName }
        tryIgnore { Remove-IscsiVirtualDisk -Path $_.FullName }
    }

    tryIgnore { Remove-IscsiServerTarget -TargetName $targetIQN -errorAction SilentlyContinue }
    if (Test-Path -Path $targetDir) {
        rm -Force -Recurse -Verbose $targetDir
    }
}

#########################
# Start of main section #
#########################

if ($TargetIQN) {
    $targetDir = disksDirForTarget $targetIQN
    if(Test-Path -Path $targetDir){
        removeTargetDir $targetDir
    } else {
        throw "Directory $targetDir does not exist"
    }
} elseif(Test-Path -Path $disksDir) {
    # remove them all
    Get-ChildItem $disksDir | ForEach-Object {
        if ((Get-Item $_.FullName) -is [System.IO.DirectoryInfo])
        {
            removeTargetDir $_.FullName
        }
    }

    # and attempt to remove any cruft left over from failed teardowns
    Get-IscsiVirtualDisk | Foreach-Object {
        if ($_.Path.StartsWith($disksDir)) {
            tryIgnore { Remove-IscsiVirtualDisk -Path $_.Path }
        }
    }
    Get-IscsiServerTarget | Foreach-Object {
        if ($_.TargetIQN.ToString().StartsWith($TestIQNPrefix)) {
            removeTargetDir $(disksDirForTarget $_.TargetIQN)
        }
    }
}

if ($global:serviceStopped) {
    Write-Host -ForegroundColor green 'Re-starting the service'
    Start-Service WinTarget
}
