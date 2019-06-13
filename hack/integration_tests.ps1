<#
.Synopsis
 Runs this repo's integration tests.
#>
Param(
    [Parameter(Position = 0, Mandatory=$false)] [String] $TestCase # if left empty, will run em' all
)

# exit on error
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$PSDefaultParameterValues['*:ErrorAction'] = 'Stop'

. "$PSScriptRoot/.common_env.ps1"
. "$PSScriptRoot/.go.ps1"

runTestsForSubpackages 'integration_tests' $TestCase
