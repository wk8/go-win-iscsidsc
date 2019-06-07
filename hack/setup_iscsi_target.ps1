<#
.Synopsis
 Sets up an iSCSI target for testing & dev purposes.
#>
Param(
    [Parameter(Position = 0, Mandatory = $true)] [String] $TargetIQN,
    [Parameter(Mandatory=$false)] [String[]] $ClientIQNs = @('*'), # wildcard, matches all
    [Parameter(Mandatory=$false)] [String] $ChapUser, # if left empty, no CHAP
    [Parameter(Mandatory=$false)] [String] $ChapPassword, # if left empty, no CHAP
    [Parameter(Mandatory=$false)] [Int32] $DiskCount = 5,
    [Parameter(Mandatory=$false)] [UInt64] $DiskSizeBytes = 100 * 1024 * 1024 # 100MB
)

. "$PSScriptRoot/.common_env.ps1"

function ensureDisksExist([String]$targetIQN, [Int32]$count, [UInt64]$sizeBytes) {
    $callback = {
        param($diskPath)

        if ([System.IO.File]::Exists($diskPath)) {
            echo "Disk $i for target $targetIQN already exists"
        } else {
            echo "Creating disk $i for target $targetIQN"
            tryIgnore { Remove-IscsiVirtualDisk -Path $diskPath }
            New-IscsiVirtualDisk -Path $diskPath -SizeBytes $sizeBytes
        }
    }
    iterateOverDiskFiles $targetIQN $count $callback
}

function ensureTargetExists([String]$targetIQN) {
    $currentTarget = Get-IscsiServerTarget -TargetName $targetIQN -ErrorAction SilentlyContinue
    if ($currentTarget) {
        echo "Target $targetIQN already exists"
    } else {
        echo "Creating target $targetIQN"
        New-IscsiServerTarget -TargetName $targetIQN
    }
}

function setTargetSettings([String]$targetIQN, [String[]]$clientIQNs, [String]$chapUser, [String]$chapPassword) {
    for ($i = 0; $i -lt $clientIQNs.Count; $i++) {
        $clientIQNs[$i] = 'IQN:' + $clientIQNs[$i]
    }

    $params = @{
        TargetName = $targetIQN
        TargetIqn = $targetIQN
        Description = 'Testing target for go-win-iscsidsc'
        Enable = $true
        InitiatorIds = $clientIQNs
    }

    if ($chapUser -and $chapPassword) {
        $params['EnableChap'] = $true
        $password = "$chapPassword" | ConvertTo-SecureString -asPlainText -Force
        $chap = New-Object System.Management.Automation.PSCredential($chapUser, $password)
        $params['Chap'] = $chap
    } else {
        $params['EnableChap'] = $false
    }

    Set-IscsiServerTarget @params
}

function addDisksToTarget([String]$targetIQN, [Int32]$count) {
    $callback = {
        param($diskPath, $i)

        Add-IscsiVirtualDiskTargetMapping -TargetName $targetIQN -Path $diskPath -Lun $i
    }
    iterateOverDiskFiles $targetIQN $count $callback
}

function iterateOverDiskFiles([String]$targetIQN, [Int32]$count, [ScriptBlock]$callback) {
    $targetDir = disksDirForTarget $targetIQN
    if(-not (Test-Path -Path $targetDir)){
        New-Item -ItemType directory -Path $targetDir
    }

    for ($i = 0; $i -lt $count; $i++) {
        $diskPath = $targetDir + '\' + $i + $disksExtension
        $callback.Invoke($diskPath, $i)
    }
}

ensureDisksExist $TargetIQN $DiskCount $DiskSizeBytes
ensureTargetExists $TargetIQN
setTargetSettings $TargetIQN $ClientIQNs $ChapUser $ChapPassword
addDisksToTarget $TargetIQN $DiskCount
