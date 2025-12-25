# TableFormatter Unit Tests

$scriptRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

# Load modules
. "$scriptRoot\lib\TaskReader.ps1"
. "$scriptRoot\lib\TaskStatusUpdater.ps1"
. "$scriptRoot\lib\TableFormatter.ps1"

Describe "TableFormatter Module" {
    Context "Get-StatusColor" {
        It "Should return Green for COMPLETED" {
            Get-StatusColor -Status "COMPLETED" | Should Be "Green"
        }
        
        It "Should return Yellow for IN_PROGRESS" {
            Get-StatusColor -Status "IN_PROGRESS" | Should Be "Yellow"
        }
        
        It "Should return Gray for NOT_STARTED" {
            Get-StatusColor -Status "NOT_STARTED" | Should Be "Gray"
        }
        
        It "Should return Red for BLOCKED" {
            Get-StatusColor -Status "BLOCKED" | Should Be "Red"
        }
        
        It "Should return White for unknown status" {
            Get-StatusColor -Status "UNKNOWN" | Should Be "White"
        }
    }
    
    Context "Format-TableSeparator" {
        It "Should create top separator with correct length" {
            $sep = Format-TableSeparator -Widths @(5, 10) -Type "Top"
            $sep.Length | Should BeGreaterThan 15
        }
        
        It "Should create middle separator with correct length" {
            $sep = Format-TableSeparator -Widths @(5, 10) -Type "Middle"
            $sep.Length | Should BeGreaterThan 15
        }
        
        It "Should create bottom separator with correct length" {
            $sep = Format-TableSeparator -Widths @(5, 10) -Type "Bottom"
            $sep.Length | Should BeGreaterThan 15
        }
    }
    
    Context "Format-TableRow" {
        It "Should format row with values" {
            $row = Format-TableRow -Values @("A", "B") -Widths @(5, 10)
            $row | Should Match "A"
            $row | Should Match "B"
        }
        
        It "Should truncate long values" {
            $row = Format-TableRow -Values @("VeryLongValue") -Widths @(8)
            $row | Should Match "\.\.\."
        }
        
        It "Should pad short values" {
            $row = Format-TableRow -Values @("X") -Widths @(10)
            $row.Length | Should BeGreaterThan 10
        }
    }
    
    Context "Format-TaskTable" {
        It "Should return message for empty tasks" {
            $result = Format-TaskTable -Tasks @()
            $result | Should Be "No tasks found."
        }
        
        It "Should format tasks as table" {
            $tasks = @(
                @{ TaskId = "T001"; Name = "Test Task"; Status = "COMPLETED"; Priority = "P1"; FeatureId = "F001" }
            )
            $result = Format-TaskTable -Tasks $tasks
            $result.Count | Should BeGreaterThan 3
            ($result -join "`n") | Should Match "T001"
            ($result -join "`n") | Should Match "Test Task"
        }
    }
    
    Context "Get-FilteredTasks" {
        BeforeEach {
            $testDir = Join-Path $env:TEMP "Hermes-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$testDir\tasks" -Force | Out-Null
            Push-Location $testDir
            
            $content = @"
# Feature 1: Test

**Feature ID:** F001
**Status:** IN_PROGRESS

### T001: Task 1

**Status:** COMPLETED
**Priority:** P1

### T002: Task 2

**Status:** IN_PROGRESS
**Priority:** P2

### T003: Task 3

**Status:** NOT_STARTED
**Priority:** P1
"@
            Set-Content -Path "tasks\001-test.md" -Value $content
        }
        
        AfterEach {
            Pop-Location
            Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return all tasks without filter" {
            $tasks = Get-FilteredTasks -BasePath "."
            $tasks.Count | Should Be 3
        }
        
        It "Should filter by status" {
            $tasks = Get-FilteredTasks -StatusFilter "COMPLETED" -BasePath "."
            $tasks.Count | Should Be 1
            $tasks[0].TaskId | Should Be "T001"
        }
        
        It "Should filter by priority" {
            $tasks = Get-FilteredTasks -PriorityFilter "P1" -BasePath "."
            $tasks.Count | Should Be 2
        }
        
        It "Should filter by feature" {
            $tasks = Get-FilteredTasks -FeatureFilter "F001" -BasePath "."
            $tasks.Count | Should Be 3
        }
        
        It "Should combine filters" {
            $tasks = Get-FilteredTasks -StatusFilter "NOT_STARTED" -PriorityFilter "P1" -BasePath "."
            $tasks.Count | Should Be 1
            $tasks[0].TaskId | Should Be "T003"
        }
    }
}
