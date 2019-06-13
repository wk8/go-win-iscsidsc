<#
.Synopsis
 Runs this repo's unit tests.
#>
Param(
    [Parameter(Position = 0, Mandatory=$false)] [String] $Subpackage, # if left empty, will run em' all
    [Parameter(Position = 1, Mandatory=$false)] [String] $TestCase # if left empty, will run em' all
)

. "$PSScriptRoot/.common_env.ps1"

if ($Subpackage -and $Subpackage -ne '') {
    runTestsForSubpackages $Subpackage $TestCase
} else {
    $subpackages = Get-ChildItem -Path $repoRootDir -Recurse -Depth 1 -Filter '*test.go' `
        | ForEach-Object { New-Object PSObject -Property @{subpackage=(Split-Path -Parent $_.FullName.SubString($repoRootDir.length)).TrimStart("\/.")} } `
        | Select subpackage `
        | Get-Unique -AsString `
        | Where-Object -Property subpackage -ne 'integration_tests' `
        | ForEach-Object { $_.subpackage }
    runTestsForSubpackages $subpackages $TestCase
}
