# Autonomous Mode Unit Tests

$scriptRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

# Load modules
. "$scriptRoot\lib\TaskReader.ps1"
. "$scriptRoot\lib\TaskStatusUpdater.ps1"

Describe "Autonomous Mode Functions" {
    # Define functions locally for testing
    function Get-ProgressBar {
        param(
            [int]$Percentage,
            [int]$Width = 20
        )
        
        $filled = [Math]::Floor(($Percentage / 100) * $Width)
        $empty = $Width - $filled
        
        $filledChar = [char]0x2588
        $emptyChar = [char]0x2591
        
        return "[" + ($filledChar.ToString() * $filled) + ($emptyChar.ToString() * $empty) + "]"
    }
    
    function Test-FeatureCompleted {
        param([string]$FeatureId)
        $fp = Get-FeatureProgress -FeatureId $FeatureId -BasePath "."
        return ($fp.Completed -eq $fp.Total -and $fp.Total -gt 0)
    }
    Context "Get-ProgressBar" {
        It "Should create empty bar for 0%" {
            $bar = Get-ProgressBar -Percentage 0 -Width 10
            $bar | Should Match "^\[.{10}\]$"
            $bar | Should Not Match [char]0x2588
        }
        
        It "Should create full bar for 100%" {
            $bar = Get-ProgressBar -Percentage 100 -Width 10
            $bar | Should Match "^\[.{10}\]$"
        }
        
        It "Should create half bar for 50%" {
            $bar = Get-ProgressBar -Percentage 50 -Width 10
            $bar | Should Match "^\[.{10}\]$"
        }
        
        It "Should handle default width" {
            $bar = Get-ProgressBar -Percentage 50
            $bar | Should Match "^\[.{20}\]$"
        }
    }
    
    Context "Test-FeatureCompleted" {
        BeforeEach {
            $testDir = Join-Path $env:TEMP "ralph-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$testDir\tasks" -Force | Out-Null
            Push-Location $testDir
        }
        
        AfterEach {
            Pop-Location
            Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return false for empty feature" {
            $content = @"
# Feature 1: Test

**Feature ID:** F001
**Status:** NOT_STARTED

### T001: Task 1

**Status:** NOT_STARTED
"@
            Set-Content -Path "tasks\001-test.md" -Value $content
            Test-FeatureCompleted -FeatureId "F001" | Should Be $false
        }
        
        It "Should return true when all tasks completed" {
            $content = @"
# Feature 1: Test

**Feature ID:** F001
**Status:** IN_PROGRESS

### T001: Task 1

**Status:** COMPLETED

### T002: Task 2

**Status:** COMPLETED
"@
            Set-Content -Path "tasks\001-test.md" -Value $content
            Test-FeatureCompleted -FeatureId "F001" | Should Be $true
        }
        
        It "Should return false with mixed statuses" {
            $content = @"
# Feature 1: Test

**Feature ID:** F001
**Status:** IN_PROGRESS

### T001: Task 1

**Status:** COMPLETED

### T002: Task 2

**Status:** NOT_STARTED
"@
            Set-Content -Path "tasks\001-test.md" -Value $content
            Test-FeatureCompleted -FeatureId "F001" | Should Be $false
        }
    }
    
}
