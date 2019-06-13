## Meant to be sourced by the other scripts in this dir that need to use go

$global:goBin = 'go'
if ((Test-Path env:GO_WIN_ISCSI_GOBIN) -and ($env:GO_WIN_ISCSI_GOBIN -ne '')) {
    $global:goBin = $env:GO_WIN_ISCSI_GOBIN
    echo "#### Using go bin from $global:goBin ###"
    & $global:goBin version
    if (-not$?) {
        throw "$global:goBin doesn't seem to be pointing to a go binary?"
    }
}

function runTestsForSubpackages([String[]]$subpackages, [String]$testCase) {
    $subpackages = $subpackages | ForEach-Object { $subpackage = $_.TrimStart("\/."); "./$subpackage" }

    $displaySubpackages = 'subpackage'
    if ($subpackages.Length -gt 1) {
        $displaySubpackages += 's'
    }
    $displaySubpackages += " $subpackages"

    echo "Running tests for $displaySubpackages"

    $testArgs = @(
    'test'
    '-v'
    '-count=1'
    )
    $testArgs += $subpackages

    if ($testCase -and $testCase -ne '') {
        $testArgs += "-run=$testCase"
    }

    $previousPath = pwd
    try {
        cd $repoRootDir

        & $global:goBin $testArgs

        if (-not$?) {
            throw "tests failed for $displaySubpackages"
        }
    } finally {
        cd $previousPath
    }
}
