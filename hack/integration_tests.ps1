<#
.Synopsis
 Runs this repo's integration tests.
#>
Param(
    [Parameter(Position = 0, Mandatory=$false)] [String] $TestCase # if left empty, will run em' all
)

. "$PSScriptRoot/.common_env.ps1"

runTestsForSubpackages 'integration_tests' $TestCase
