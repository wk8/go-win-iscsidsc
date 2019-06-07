<#
.Synopsis
 Runs this repo's integration tests.
#>
Param(
    [Parameter(Position = 0, Mandatory=$false)] [String] $TestCase # if left empty, will run em' all
)

. "$PSScriptRoot/.common_env.ps1"

$previousPath = pwd

try {
    cd "$repoRootDir/integration_tests"

    $testArgs = @(
    'test'
    '-v'
    '-count=1'
    )

    if ($TestCase -and $TestCase -ne "")
    {
        $testArgs += "-run=$TestCase"
    }

    & go $testArgs

    if (-not $?) { throw 'tests failed' }
} finally {
    cd $previousPath
}
