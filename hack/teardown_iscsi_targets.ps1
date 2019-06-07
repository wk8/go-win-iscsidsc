<#
.Synopsis
 Tears down one or all iSCSI target(s) created by setup_iscsi_target.ps1.
#>
Param(
    [Parameter(Position = 0, Mandatory=$false)] [String] $TargetIQN # if left empty, will remove em' all
)

. "$PSScriptRoot/.common_env.ps1"

$global:serviceStopped = $false

function removeTargetDir([String]$targetDir)  {
    $targetIQN = targetIQNFromDisksDir $targetDir
    echo "Removing target $targetIQN"

    try {
        removeTarget $targetDir $targetIQN
    } catch {
        if (-not $global:serviceStopped -and ((Get-Service WinTarget).Status -eq 'Running')) {
            echo "Stopping the service..."
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
        tryIgnore { Remove-IscsiVirtualDisk -Path $diskPath }
    }

    tryIgnore { Remove-IscsiServerTarget -TargetName $targetIQN -errorAction SilentlyContinue }
    rm -Force -Recurse -Verbose $targetDir
}

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
}

if ($global:serviceStopped) {
    echo "Re-starting the service"
    Start-Service WinTarget
}
