#Requires -Version 7.0

<#
.SYNOPSIS
    Ralph Test Runner
.DESCRIPTION
    Runs all Pester tests for the Ralph Windows PowerShell version
.PARAMETER Unit
    Run only unit tests
.PARAMETER Integration
    Run only integration tests
.PARAMETER Coverage
    Enable code coverage reporting
.PARAMETER OutputFile
    Path to save test results (NUnit XML format)
.EXAMPLE
    .\Run-Tests.ps1
.EXAMPLE
    .\Run-Tests.ps1 -Unit -Coverage
#>

[CmdletBinding()]
param(
    [switch]$Unit,
    [switch]$Integration,
    [switch]$Coverage,
    [string]$OutputFile
)

$ErrorActionPreference = "Stop"

# Check if Pester is installed
$pesterModule = Get-Module -ListAvailable -Name Pester | Where-Object { $_.Version.Major -ge 5 }
if (-not $pesterModule) {
    Write-Host "Pester 5.x is required but not installed." -ForegroundColor Red
    Write-Host ""
    Write-Host "Install Pester with:" -ForegroundColor Yellow
    Write-Host "  Install-Module -Name Pester -Force -SkipPublisherCheck"
    Write-Host ""
    exit 1
}

Import-Module Pester -MinimumVersion 5.0

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptRoot

Write-Host ""
Write-Host "Ralph for Windows - Test Runner" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Determine test paths
$testPaths = @()
if ($Unit) {
    $testPaths += Join-Path $scriptRoot "unit"
    Write-Host "Running: Unit tests" -ForegroundColor Gray
}
elseif ($Integration) {
    $testPaths += Join-Path $scriptRoot "integration"
    Write-Host "Running: Integration tests" -ForegroundColor Gray
}
else {
    $testPaths += $scriptRoot
    Write-Host "Running: All tests" -ForegroundColor Gray
}

# Create Pester configuration
$config = New-PesterConfiguration

$config.Run.Path = $testPaths
$config.Run.Exit = $true
$config.Output.Verbosity = "Detailed"

# Code coverage
if ($Coverage) {
    Write-Host "Code coverage: Enabled" -ForegroundColor Gray
    $config.CodeCoverage.Enabled = $true
    $config.CodeCoverage.Path = @(
        (Join-Path $projectRoot "lib\*.ps1"),
        (Join-Path $projectRoot "ralph_loop.ps1"),
        (Join-Path $projectRoot "ralph_monitor.ps1"),
        (Join-Path $projectRoot "setup.ps1"),
        (Join-Path $projectRoot "ralph_import.ps1")
    )
    $config.CodeCoverage.OutputPath = Join-Path $scriptRoot "coverage.xml"
    $config.CodeCoverage.OutputFormat = "JaCoCo"
}

# Test result output
if ($OutputFile) {
    Write-Host "Output file: $OutputFile" -ForegroundColor Gray
    $config.TestResult.Enabled = $true
    $config.TestResult.OutputPath = $OutputFile
    $config.TestResult.OutputFormat = "NUnitXml"
}

Write-Host ""

# Run tests
try {
    $result = Invoke-Pester -Configuration $config
    
    Write-Host ""
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host "Test Summary" -ForegroundColor Cyan
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host "  Total:   $($result.TotalCount)" -ForegroundColor White
    Write-Host "  Passed:  $($result.PassedCount)" -ForegroundColor Green
    Write-Host "  Failed:  $($result.FailedCount)" -ForegroundColor $(if ($result.FailedCount -gt 0) { "Red" } else { "White" })
    Write-Host "  Skipped: $($result.SkippedCount)" -ForegroundColor Yellow
    Write-Host ""
    
    if ($Coverage -and $result.CodeCoverage) {
        $coverage = $result.CodeCoverage
        $coveragePercent = [Math]::Round(($coverage.CommandsExecutedCount / $coverage.CommandsAnalyzedCount) * 100, 2)
        Write-Host "Code Coverage: $coveragePercent%" -ForegroundColor $(if ($coveragePercent -ge 80) { "Green" } elseif ($coveragePercent -ge 60) { "Yellow" } else { "Red" })
        Write-Host ""
    }
    
    if ($result.FailedCount -gt 0) {
        exit 1
    }
}
catch {
    Write-Host "Error running tests: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
