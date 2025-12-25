# Incremental PRD Update Unit Tests

$scriptRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

# Load the hermes-prd script to get functions (without executing main)
# We'll define the functions locally for testing

function Get-ExistingTaskState {
    param([string]$TasksDir = "tasks")
    
    $state = @{
        Features = @{}
        HighestFeatureId = 0
        HighestTaskId = 0
        HasTasks = $false
    }
    
    if (-not (Test-Path $TasksDir)) {
        return $state
    }
    
    $files = Get-ChildItem -Path $TasksDir -Filter "*.md" -ErrorAction SilentlyContinue
    if (-not $files -or $files.Count -eq 0) {
        return $state
    }
    
    foreach ($file in $files) {
        if ($file.Name -match "status") { continue }
        
        $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
        if (-not $content) { continue }
        
        $featureIdMatch = [regex]::Match($content, "\*\*Feature ID:\*\*\s*(F(\d+))")
        if (-not $featureIdMatch.Success) { continue }
        
        $featureId = $featureIdMatch.Groups[1].Value
        $featureNum = [int]$featureIdMatch.Groups[2].Value
        
        if ($featureNum -gt $state.HighestFeatureId) {
            $state.HighestFeatureId = $featureNum
        }
        
        $nameMatch = [regex]::Match($content, "^# Feature \d+:\s*(.+)$", [System.Text.RegularExpressions.RegexOptions]::Multiline)
        $featureName = if ($nameMatch.Success) { $nameMatch.Groups[1].Value.Trim() } else { "" }
        
        $statusMatch = [regex]::Match($content, "\*\*Status:\*\*\s*(NOT_STARTED|IN_PROGRESS|COMPLETED|BLOCKED)")
        $featureStatus = if ($statusMatch.Success) { $statusMatch.Groups[1].Value } else { "NOT_STARTED" }
        
        $taskMatches = [regex]::Matches($content, "### (T(\d+)):\s*(.+)")
        $tasks = @{}
        
        foreach ($tm in $taskMatches) {
            $taskId = $tm.Groups[1].Value
            $taskNum = [int]$tm.Groups[2].Value
            $taskName = $tm.Groups[3].Value.Trim()
            
            if ($taskNum -gt $state.HighestTaskId) {
                $state.HighestTaskId = $taskNum
            }
            
            $taskStatusPattern = "### $taskId.*?\*\*Status:\*\*\s*(NOT_STARTED|IN_PROGRESS|COMPLETED|BLOCKED)"
            $taskStatusMatch = [regex]::Match($content, $taskStatusPattern, [System.Text.RegularExpressions.RegexOptions]::Singleline)
            $taskStatus = if ($taskStatusMatch.Success) { $taskStatusMatch.Groups[1].Value } else { "NOT_STARTED" }
            
            $tasks[$taskId] = @{
                Name = $taskName
                Status = $taskStatus
            }
        }
        
        $state.Features[$featureId] = @{
            Name = $featureName
            Status = $featureStatus
            Tasks = $tasks
            FileName = $file.Name
            FilePath = $file.FullName
        }
        
        $state.HasTasks = $true
    }
    
    return $state
}

function Test-FeatureHasProgress {
    param([hashtable]$Feature)
    
    foreach ($task in $Feature.Tasks.Values) {
        if ($task.Status -ne "NOT_STARTED") {
            return $true
        }
    }
    return $false
}

Describe "Incremental PRD Update" {
    Context "Get-ExistingTaskState" {
        BeforeEach {
            $testDir = Join-Path $env:TEMP "Hermes-incr-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$testDir\tasks" -Force | Out-Null
            Push-Location $testDir
        }
        
        AfterEach {
            Pop-Location
            Remove-Item -Recurse -Force $testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return empty state for non-existent directory" {
            $state = Get-ExistingTaskState -TasksDir "nonexistent"
            $state.HasTasks | Should Be $false
            $state.HighestFeatureId | Should Be 0
            $state.HighestTaskId | Should Be 0
        }
        
        It "Should return empty state for empty directory" {
            $state = Get-ExistingTaskState -TasksDir "tasks"
            $state.HasTasks | Should Be $false
        }
        
        It "Should parse existing features correctly" {
            # Create content with proper spacing
            $lines = @(
                "# Feature 1: User Auth",
                "",
                "**Feature ID:** F001",
                "**Status:** IN_PROGRESS",
                "",
                "### T001: Login",
                "",
                "**Status:** COMPLETED",
                "",
                "### T002: Register",
                "",
                "**Status:** NOT_STARTED"
            )
            $content = $lines -join "`r`n"
            Set-Content -Path "tasks\001-user-auth.md" -Value $content -NoNewline
            
            $state = Get-ExistingTaskState -TasksDir "tasks"
            $state.HasTasks | Should Be $true
            $state.HighestFeatureId | Should Be 1
            $state.HighestTaskId | Should Be 2
            $state.Features["F001"].Name | Should Be "User Auth"
            $state.Features["F001"].Status | Should Be "IN_PROGRESS"
            $state.Features["F001"].Tasks["T001"].Status | Should Be "COMPLETED"
            $state.Features["F001"].Tasks["T002"].Status | Should Be "NOT_STARTED"
        }
        
        It "Should find highest IDs across multiple files" {
            $lines1 = @(
                "# Feature 1: First",
                "",
                "**Feature ID:** F001",
                "",
                "### T001: Task 1",
                "",
                "**Status:** NOT_STARTED",
                "",
                "### T002: Task 2",
                "",
                "**Status:** NOT_STARTED"
            )
            $lines2 = @(
                "# Feature 2: Second",
                "",
                "**Feature ID:** F002",
                "",
                "### T003: Task 3",
                "",
                "**Status:** NOT_STARTED",
                "",
                "### T004: Task 4",
                "",
                "**Status:** NOT_STARTED"
            )
            Set-Content -Path "tasks\001-first.md" -Value ($lines1 -join "`r`n") -NoNewline
            Set-Content -Path "tasks\002-second.md" -Value ($lines2 -join "`r`n") -NoNewline
            
            $state = Get-ExistingTaskState -TasksDir "tasks"
            $state.HighestFeatureId | Should Be 2
            $state.HighestTaskId | Should Be 4
            $state.Features.Count | Should Be 2
        }
        
        It "Should skip status files" {
            $content = @"
# Task Status

Some status content
"@
            Set-Content -Path "tasks\tasks-status.md" -Value $content
            
            $state = Get-ExistingTaskState -TasksDir "tasks"
            $state.HasTasks | Should Be $false
        }
    }
    
    Context "Test-FeatureHasProgress" {
        It "Should return false when all tasks are NOT_STARTED" {
            $feature = @{
                Tasks = @{
                    "T001" = @{ Status = "NOT_STARTED" }
                    "T002" = @{ Status = "NOT_STARTED" }
                }
            }
            Test-FeatureHasProgress -Feature $feature | Should Be $false
        }
        
        It "Should return true when any task is IN_PROGRESS" {
            $feature = @{
                Tasks = @{
                    "T001" = @{ Status = "NOT_STARTED" }
                    "T002" = @{ Status = "IN_PROGRESS" }
                }
            }
            Test-FeatureHasProgress -Feature $feature | Should Be $true
        }
        
        It "Should return true when any task is COMPLETED" {
            $feature = @{
                Tasks = @{
                    "T001" = @{ Status = "COMPLETED" }
                    "T002" = @{ Status = "NOT_STARTED" }
                }
            }
            Test-FeatureHasProgress -Feature $feature | Should Be $true
        }
    }
}
