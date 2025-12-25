# Resume Mode Unit Tests

$scriptRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

# Load modules
. "$scriptRoot\lib\TaskReader.ps1"
. "$scriptRoot\lib\TaskStatusUpdater.ps1"

Describe "Resume Mode Functions" {
    Context "Test-ShouldResume" {
        BeforeEach {
            $testDir = Join-Path $env:TEMP "Hermes-resume-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$testDir\tasks" -Force | Out-Null
            Push-Location $testDir
        }
        
        AfterEach {
            Pop-Location
            Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return false when no run-state.md exists" {
            Test-ShouldResume -BasePath "." | Should Be $false
        }
        
        It "Should return false when status is COMPLETED" {
            $lines = @(
                "# Task Plan Run State",
                "",
                "**Status:** COMPLETED",
                "",
                "## Current Position",
                "",
                "- **Current Task:** T001"
            )
            Set-Content -Path "tasks\run-state.md" -Value ($lines -join "`r`n")
            
            Test-ShouldResume -BasePath "." | Should Be $false
        }
        
        It "Should return true when status is IN_PROGRESS with task" {
            $lines = @(
                "# Task Plan Run State",
                "",
                "**Status:** IN_PROGRESS",
                "",
                "## Current Position",
                "",
                "- **Current Task:** T003",
                "- **Next Task:** T004"
            )
            Set-Content -Path "tasks\run-state.md" -Value ($lines -join "`r`n")
            
            Test-ShouldResume -BasePath "." | Should Be $true
        }
    }
    
    Context "Get-ResumeInfo" {
        BeforeEach {
            $testDir = Join-Path $env:TEMP "Hermes-resume-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$testDir\tasks" -Force | Out-Null
            Push-Location $testDir
        }
        
        AfterEach {
            Pop-Location
            Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return null when no run-state exists" {
            $info = Get-ResumeInfo -BasePath "."
            $info | Should Be $null
        }
        
        It "Should return resume info with NextTaskId" {
            $lines = @(
                "# Task Plan Run State",
                "",
                "**Status:** IN_PROGRESS",
                "",
                "## Current Position",
                "",
                "- **Current Feature:** F001",
                "- **Current Branch:** feature/F001-test",
                "- **Current Task:** T003",
                "- **Next Task:** T004"
            )
            Set-Content -Path "tasks\run-state.md" -Value ($lines -join "`r`n")
            
            $info = Get-ResumeInfo -BasePath "."
            $info | Should Not Be $null
            $info.ResumeTaskId | Should Be "T004"
            $info.CurrentBranch | Should Be "feature/F001-test"
        }
    }
    
    Context "Get-ExecutionQueue" {
        BeforeEach {
            $testDir = Join-Path $env:TEMP "Hermes-queue-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$testDir\tasks" -Force | Out-Null
            Push-Location $testDir
            
            $lines = @(
                "# Feature 1: Test",
                "",
                "**Feature ID:** F001",
                "",
                "### T001: Task 1",
                "",
                "**Status:** COMPLETED",
                "**Priority:** P1",
                "",
                "### T002: Task 2",
                "",
                "**Status:** NOT_STARTED",
                "**Priority:** P2",
                "",
                "### T003: Task 3",
                "",
                "**Status:** NOT_STARTED",
                "**Priority:** P1"
            )
            Set-Content -Path "tasks\001-test.md" -Value ($lines -join "`r`n")
        }
        
        AfterEach {
            Pop-Location
            Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return only non-completed tasks" {
            $queue = Get-ExecutionQueue -BasePath "."
            $queue.Count | Should Be 2
        }
        
        It "Should sort by priority" {
            $queue = Get-ExecutionQueue -BasePath "."
            $queue[0].Priority | Should Be "P1"
        }
    }
}
