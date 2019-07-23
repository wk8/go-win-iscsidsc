<#
.Synopsis
 Sets up an iSCSI target for testing & dev purposes.
#>
Param(
    [Parameter(Position = 0, Mandatory = $false)] [String] $TargetIQN,
    [Parameter(Mandatory=$false)] [Switch] $RandomizeIQN = $false, # if true, all 'X' characters in the IQN will be replaced by random alpha-numeric characters, à la mktemp
                                                                   # it will also ensure that whatever random name it comes up with is not taken yet
    [Parameter(Mandatory=$false)] [Switch] $TestIQN = $false, # if true, equivalent to $TargetIQN = $TestIQNPattern and $RandomizeIQN = $true
    [Parameter(Mandatory=$false)] [String] $WriteIQNTo = $false, # if set, the IQN will be written to this file after the target's creation
                                                        # comes in handy to retrieve the IQN when randomized
    [Parameter(Mandatory=$false)] [Switch] $OverwriteIQNFile = $false, # if $WriteIQNTo points to a file that already exists, this script
                                                                       # will refuse to overwrite it unless $OverwriteIQNFile is true

    [Parameter(Mandatory=$false)] [String[]] $ClientIQNs = @('*'), # wildcard, matches all
    [Parameter(Mandatory=$false)] [String] $ChapUser, # if left empty, no CHAP
    [Parameter(Mandatory=$false)] [String] $ChapPassword, # if left empty, no CHAP
    [Parameter(Mandatory=$false)] [Int32] $DiskCount = 5,
    [Parameter(Mandatory=$false)] [UInt64] $DiskSizeBytes = 100 * 1024 * 1024 # 100MB
)

# exit on error
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSDefaultParameterValues['*:ErrorAction'] = 'Stop'

. "$PSScriptRoot/.common_env.ps1"

function ensureDisksExist([String]$targetIQN, [Int32]$count, [UInt64]$sizeBytes) {
    $callback = {
        param($diskPath)

        if ([System.IO.File]::Exists($diskPath)) {
            Write-Host -ForegroundColor green "Disk $i for target $targetIQN already exists"
        } else {
            Write-Host -ForegroundColor green "Creating disk $i for target $targetIQN"
            tryIgnore { Remove-IscsiVirtualDisk -Path $diskPath } $true
            New-IscsiVirtualDisk -Path $diskPath -SizeBytes $sizeBytes
        }
    }
    iterateOverDiskFiles $targetIQN $count $callback
}

function ensureTargetExists([String]$targetIQN) {
    $currentTarget = Get-IscsiServerTarget -TargetName $targetIQN -ErrorAction SilentlyContinue
    if ($currentTarget) {
        Write-Host -ForegroundColor green "Target $targetIQN already exists"
    } else {
        Write-Host -ForegroundColor green "Creating target $targetIQN"
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
    if (-not (Test-Path -Path $targetDir)){
        New-Item -ItemType directory -Path $targetDir
    }

    for ($i = 0; $i -lt $count; $i++) {
        $diskPath = $targetDir + '\' + $i + $disksExtension
        $callback.Invoke($diskPath, $i)
    }
}

function replaceXs([String]$s) {
    $start = 0 # inclusive
    while ($start -lt $s.Length) {
        $end = $start # exclusive
        while ($end -lt $s.Length -and $s[$end] -ceq 'X') {
            $end++
        }
        $diff = $end - $start
        if ($diff -gt 0) {
            $s = $s.Remove($start, $diff).Insert($start, $(randomString $diff))
        }
        $start = $end + 1
    }
    echo $s
}

# it is important here to _not_ have any upper-case letter, as Windows' ISCSI APIs are not
# case-sensitive, but golang is...
# inspired from https://gist.github.com/marcgeld/4891bbb6e72d7fdb577920a6420c1dfb
function randomString([Int]$length) {
    echo ( -join ((0x30..0x39) + (0x61..0x7A) | Get-Random -Count $length | % {[char]$_}) )
}

#########################
# Start of main section #
#########################

if ($TestIQN) {
    $TargetIQN = $TestIQNPattern
    $RandomizeIQN = $true
}

if (-not $TargetIQN -or $TargetIQN -eq '') {
    throw 'Need to specify one of $TargetIQN or $TestIQN'
}

if ($RandomizeIQN) {
    $maxTries = 20
    $tries = 0
    do {
        if ($tries -ge $maxTries) {
            throw "Unable to find unused random IQN from pattern $TargetIQN after $tries attempts"
        }
        $candidate = replaceXs $TargetIQN
        $tries++
    } while (Test-Path -Path $(disksDirForTarget $candidate))

    $TargetIQN = $candidate
    Write-Host -ForegroundColor green "Creating target with IQN: $TargetIQN"
}

ensureDisksExist $TargetIQN $DiskCount $DiskSizeBytes
ensureTargetExists $TargetIQN
setTargetSettings $TargetIQN $ClientIQNs $ChapUser $ChapPassword
addDisksToTarget $TargetIQN $DiskCount

if ($WriteIQNTo -and -not ($WriteIQNTo -eq '')) {
    if ((Test-Path -Path $WriteIQNTo) -and -not $OverwriteIQNFile) {
        throw "Refusing to overwrite $WriteIQNTo"
    }

    $parentDir = Split-Path -Parent $WriteIQNTo
    if ($parentDir -and -not (Test-Path -Path $parentDir)){
        New-Item -ItemType directory -Path $parentDir
    }

    echo $TargetIQN | Set-Content $WriteIQNTo
    Write-Host -ForegroundColor green "TargetIQN written to $WriteIQNTo"
}
