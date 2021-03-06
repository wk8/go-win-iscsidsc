## Meant to be sourced by the other scripts in this dir that need to use go

$global:goBin = 'go'
if ((Test-Path env:GO_WIN_ISCSI_GOBIN) -and ($env:GO_WIN_ISCSI_GOBIN -ne '')) {
    $global:goBin = $env:GO_WIN_ISCSI_GOBIN
    Write-Host -ForegroundColor yellow "### Using go bin from $global:goBin ###"
    & $global:goBin version
    if (-not$?) {
        throw "$global:goBin doesn't seem to be pointing to a go binary?"
    }
}

if ((Test-Path env:GOPATH) -and ($env:GOPATH -ne '')) {
    $env:Path += ";$env:GOPATH/bin"
}

if ((Test-Path env:GOROOT) -and ($env:GOROOT -ne '')) {
    $env:Path += ";$env:GOROOT/bin"
}

function runTestsForSubpackages([String[]]$subpackages, [String]$testCase) {
    $subpackages = $subpackages | ForEach-Object { $subpackage = $_.TrimStart("\/."); "./$subpackage" }

    $displaySubpackages = 'subpackage'
    if ($subpackages.Length -gt 1) {
        $displaySubpackages += 's'
    }
    $displaySubpackages += " $subpackages"

    Write-Host -ForegroundColor green "[$(Get-Date)] Running tests for $displaySubpackages"

    $testArgs = @(
        'test'
        '-v'
        '-count=1'
        # some tests (eg integration tests) can take a while on smaller boxes
        '-timeout=60m'
    )
    $testArgs += $subpackages

    if ($testCase -and $testCase -ne '') {
        $testArgs += "-run=$testCase"
    }

    $previousPath = pwd
    try {
        cd $repoRootDir

        & $global:goBin $testArgs

        if (-not $?) {
            throw "tests failed for $displaySubpackages"
        }
    } finally {
        cd $previousPath
    }
}
