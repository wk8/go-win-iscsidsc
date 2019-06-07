<#
.Synopsis
 Manages this repo's go dependencies with glide (https://glide.sh)
#>
Param(
    [ValidateSet('install', 'update', IgnoreCase = $true)] [Parameter(Position = 0, Mandatory = $false)] [String] $Action = 'install'
)

. "$PSScriptRoot/.common_env.ps1"

# see https://github.com/Masterminds/glide/releases
$glideVersion = '0.13.2' # latest as of 4/20/19

$platform = '386'
if ([Environment]::Is64BitProcess) {
    $platform = 'amd64'
}
$glideURL = "https://github.com/Masterminds/glide/releases/download/v$glideVersion/glide-v$glideVersion-windows-$platform.zip"
$glideBin = "$repoRootDir/dev/glide-$glideVersion.exe"

if ([System.IO.File]::Exists($glideBin)) {
    echo 'Glide already up-to-date'
} else {
    echo "Downloading glide from $glideURL"

    $downloadDir = "$env:TEMP/glide-$glideVersion"
    $archivePath = "$downloadDir/glide.zip"
    $extractDir = "$downloadDir/unzipped"
    if(-not (Test-Path -Path $extractDir)){
        New-Item -ItemType directory -Path $extractDir
    }

    wget $glideURL -OutFile $archivePath
    Expand-Archive -Force $archivePath $extractDir

    $destDir = Split-Path -Parent $glideBin
    if(-not (Test-Path -Path $destDir)){
        New-Item -ItemType directory -Path $destDir
    }
    cp -Verbose "$extractDir/windows-$platform/glide.exe" $glideBin
    rm -Recurse -Force -Verbose $downloadDir
}

& $glideBin $Action -v

if (-not $?) { throw "'$glideBin $Action' failed" }
