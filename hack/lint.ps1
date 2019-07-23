<#
.Synopsis
 Runs several linting tools.
#>
Param(
    [Parameter(Mandatory=$false)] [Switch] $UpdateTools = $false # if set, will update the tools even if they already are present
)

# exit on error
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSDefaultParameterValues['*:ErrorAction'] = 'Stop'

. "$PSScriptRoot/.common_env.ps1"
. "$PSScriptRoot/.go.ps1"

function ensureToolInstalled([String]$tool, [Bool]$update) {
    $bin = [io.fileinfo]$tool | % Basename
    if ($update -or -not (Get-Command $bin -errorAction SilentlyContinue)) {
        Write-Host -ForegroundColor green "Installing $tool..."
        & $global:goBin get -u $tool

        if (-not $?) {
            throw "Failed to install $tool"
        }
    } else {
        Write-Host -ForegroundColor green "$tool already installed"
    }

    $binPath = (Get-Command $bin).Source
    Write-Host -ForegroundColor green "Using $tool from $binPath"
}

$previousPath = pwd
try {
    cd $repoRootDir

    # go fmt
    Write-Host -ForegroundColor green 'Running go fmt...'
    $formatOutput = & $global:goBin fmt $repoRootDir
    if (-not $?) {
        throw 'go fmt failed'
    }
    if ($formatOutput) {
        git --no-pager diff
        throw "go fmt modified some files: $formatOutput"
    }
    Write-Host -ForegroundColor green 'go fmt passed!'

    # go vet
    Write-Host -ForegroundColor green 'Running go vet...'
    & $global:goBin vet github.com/wk8/go-win-iscsidsc/...
    if (-not $?) {
        throw 'go vet failed'
    }
    Write-Host -ForegroundColor green 'go vet passed!'

    $goFiles = Get-ChildItem -Path $repoRootDir -Filter '*.go' -Recurse `
    | Where-Object -Property FullName -NotLike "$repoRootDir\vendor\*" `
    | % FullName

    # goimports
    ensureToolInstalled 'golang.org/x/tools/cmd/goimports' $UpdateTools
    Write-Host -ForegroundColor green 'Running goimports...'
    $importsOutput = goimports -l -w $goFiles
    if (-not $?) {
        throw 'goimports failed'
    }
    if ($importsOutput) {
        git --no-pager diff
        throw "goimports modified some files: $importsOutput"
    }
    Write-Host -ForegroundColor green 'goimports passed!'

    # golint
    ensureToolInstalled 'golang.org/x/lint/golint' $UpdateTools
    Write-Host -ForegroundColor green 'Running golint...'
    $goLintFailed = $false
    $goFiles | ForEach-Object {
        golint -set_exit_status $_
        if (-not $?) {
            $goLintFailed = $true
        }
    }
    if ($goLintFailed) {
        throw 'golint failed'
    }
    Write-Host -ForegroundColor green 'golint passed!'
} finally {
    cd $previousPath
}
