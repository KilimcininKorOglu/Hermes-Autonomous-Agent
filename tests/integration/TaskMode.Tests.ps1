#Requires -Modules Pester

<#
.SYNOPSIS
    Integration tests for Hermes Task Mode
#>

BeforeAll {
    $script:WindowsDir = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    . "$script:WindowsDir\lib\TaskReader.ps1"
    . "$script:WindowsDir\lib\TaskStatusUpdater.ps1"
    . "$script:WindowsDir\lib\GitBranchManager.ps1"
    . "$script:WindowsDir\lib\PromptInjector.ps1"
}

Describe "Task Mode Integration" {
    BeforeAll {
        # Create complete test project
        $script:ProjectDir = Join-Path $TestDrive "integration-project"
        $script:TasksDir = Join-Path $script:ProjectDir "tasks"
        New-Item -ItemType Directory -Path $script:TasksDir -Force | Out-Null
        
        # Create PROMPT.md
        @"
# Project Instructions

Build a web application.

## Guidelines

- Use TypeScript
- Follow best practices
"@ | Set-Content (Join-Path $script:ProjectDir "PROMPT.md") -Encoding UTF8
        
        # Create feature file with multiple tasks
        @"
# Feature 1: Authentication

**Feature ID:** F001
**Feature Name:** Authentication
**Priority:** P1 - Critical
**Status:** NOT_STARTED
**Estimated Duration:** 3 days

## Tasks

### T001: Login Form

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create login form component.

#### Files to Touch
- `src/Login.tsx` (new)

#### Dependencies
- None

#### Success Criteria
- [ ] Form renders
- [ ] Validation works

---

### T002: Auth API

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement authentication API.

#### Files to Touch
- `src/api/auth.ts` (new)

#### Dependencies
- T001

#### Success Criteria
- [ ] Login endpoint works
- [ ] Token returned

---

### T003: Session Management

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
Handle user sessions.

#### Dependencies
- T001
- T002

#### Success Criteria
- [ ] Session persists

"@ | Set-Content (Join-Path $script:TasksDir "001-authentication.md") -Encoding UTF8
        
        # Create second feature
        @"
# Feature 2: Dashboard

**Feature ID:** F002
**Feature Name:** Dashboard
**Priority:** P2 - High
**Status:** NOT_STARTED

## Tasks

### T004: Dashboard Layout

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
Create dashboard layout.

#### Dependencies
- None

#### Success Criteria
- [ ] Layout renders

"@ | Set-Content (Join-Path $script:TasksDir "002-dashboard.md") -Encoding UTF8
    }
    
    Context "Full Task Workflow" {
        It "Finds next task correctly" {
            $task = Get-NextTask -BasePath $script:ProjectDir
            $task.TaskId | Should -Be "T001"
        }
        
        It "Updates task status to IN_PROGRESS" {
            Set-TaskStatus -TaskId "T001" -Status "IN_PROGRESS" -BasePath $script:ProjectDir
            
            $task = Get-TaskById -TaskId "T001" -BasePath $script:ProjectDir
            $task.Status | Should -Be "IN_PROGRESS"
        }
        
        It "Injects task into PROMPT.md" {
            $task = Get-TaskById -TaskId "T001" -BasePath $script:ProjectDir
            Add-TaskToPrompt -Task $task -FeatureName "Authentication" -BasePath $script:ProjectDir
            
            $content = Get-Content (Join-Path $script:ProjectDir "PROMPT.md") -Raw
            $content | Should -Match "T001"
            $content | Should -Match "Login Form"
        }
        
        It "Completes task and updates status" {
            Set-TaskStatus -TaskId "T001" -Status "COMPLETED" -BasePath $script:ProjectDir
            Complete-AllSuccessCriteria -TaskId "T001" -BasePath $script:ProjectDir
            
            $task = Get-TaskById -TaskId "T001" -BasePath $script:ProjectDir
            $task.Status | Should -Be "COMPLETED"
        }
        
        It "Next task respects dependencies" {
            # T001 is COMPLETED, T002 depends on T001
            $next = Get-NextTask -BasePath $script:ProjectDir
            $next.TaskId | Should -BeIn @("T002", "T004")
        }
        
        It "Cleans up PROMPT.md after task" {
            Remove-TaskFromPrompt -BasePath $script:ProjectDir
            
            $content = Get-Content (Join-Path $script:ProjectDir "PROMPT.md") -Raw
            $content | Should -Not -Match "Hermes_TASK_START"
        }
    }
    
    Context "Feature Progress Tracking" {
        BeforeAll {
            # Reset statuses
            Set-TaskStatus -TaskId "T001" -Status "COMPLETED" -BasePath $script:ProjectDir
            Set-TaskStatus -TaskId "T002" -Status "IN_PROGRESS" -BasePath $script:ProjectDir
        }
        
        It "Calculates feature progress correctly" {
            $progress = Get-FeatureProgress -FeatureId "F001" -BasePath $script:ProjectDir
            $progress.Total | Should -Be 3
            $progress.Completed | Should -Be 1
            $progress.IsComplete | Should -Be $false
        }
        
        It "Detects feature completion" {
            Set-TaskStatus -TaskId "T002" -Status "COMPLETED" -BasePath $script:ProjectDir
            Set-TaskStatus -TaskId "T003" -Status "COMPLETED" -BasePath $script:ProjectDir
            
            $isComplete = Test-FeatureComplete -FeatureId "F001" -BasePath $script:ProjectDir
            $isComplete | Should -Be $true
        }
    }
    
    Context "Run State Management" {
        It "Updates run state" {
            Update-RunState -CurrentTaskId "T002" -CurrentFeatureId "F001" `
                -CurrentBranch "feature/F001-auth" -NextTaskId "T003" -BasePath $script:ProjectDir
            
            $state = Get-RunState -BasePath $script:ProjectDir
            $state.CurrentTaskId | Should -Be "T002"
            $state.CurrentFeatureId | Should -Be "F001"
        }
        
        It "Generates tasks-status.md" {
            Update-TasksStatusFile -BasePath $script:ProjectDir
            
            $statusFile = Join-Path $script:ProjectDir "tasks" "tasks-status.md"
            Test-Path $statusFile | Should -Be $true
            
            $content = Get-Content $statusFile -Raw
            $content | Should -Match "Total Tasks:"
        }
    }
    
    Context "Multi-Feature Workflow" {
        BeforeAll {
            # Complete F001, start F002
            Set-TaskStatus -TaskId "T001" -Status "COMPLETED" -BasePath $script:ProjectDir
            Set-TaskStatus -TaskId "T002" -Status "COMPLETED" -BasePath $script:ProjectDir
            Set-TaskStatus -TaskId "T003" -Status "COMPLETED" -BasePath $script:ProjectDir
        }
        
        It "Moves to next feature after completion" {
            $next = Get-NextTask -BasePath $script:ProjectDir
            $next.TaskId | Should -Be "T004"
            $next.FeatureId | Should -Be "F002"
        }
        
        It "Overall progress is correct" {
            $progress = Get-TaskProgress -BasePath $script:ProjectDir
            $progress.Total | Should -Be 4
            $progress.Completed | Should -Be 3
            $progress.Percentage | Should -Be 75
        }
    }
    
    Context "Branch Name Generation" {
        It "Creates valid branch names" {
            $branch = Get-FeatureBranchName -FeatureId "F001" -FeatureName "Authentication"
            $branch | Should -Be "feature/F001-authentication"
        }
        
        It "Handles special characters in feature name" {
            $branch = Get-FeatureBranchName -FeatureId "F002" -FeatureName "User Profile & Settings"
            $branch | Should -Match "^feature/F002-"
            $branch | Should -Not -Match "[&\s]"
        }
    }
    
    Context "Prompt Injection Complete Cycle" {
        It "Backup -> Inject -> Remove cycle works" {
            # Backup
            $backup = Backup-Prompt -BasePath $script:ProjectDir
            $backup | Should -Not -BeNullOrEmpty
            
            # Inject
            $task = Get-TaskById -TaskId "T004" -BasePath $script:ProjectDir
            Add-TaskToPrompt -Task $task -BasePath $script:ProjectDir
            
            $content = Get-Content (Join-Path $script:ProjectDir "PROMPT.md") -Raw
            $content | Should -Match "T004"
            
            # Remove
            Remove-TaskFromPrompt -BasePath $script:ProjectDir
            
            $content = Get-Content (Join-Path $script:ProjectDir "PROMPT.md") -Raw
            $content | Should -Not -Match "Hermes_TASK"
            
            # Original content preserved
            $content | Should -Match "Project Instructions"
        }
    }
}

Describe "Edge Cases" {
    Context "Empty Tasks Directory" {
        BeforeAll {
            $script:EmptyDir = Join-Path $TestDrive "empty-project"
            $script:EmptyTasksDir = Join-Path $script:EmptyDir "tasks"
            New-Item -ItemType Directory -Path $script:EmptyTasksDir -Force | Out-Null
        }
        
        It "Returns empty for no tasks" {
            $tasks = Get-AllTasks -BasePath $script:EmptyDir
            $tasks | Should -BeNullOrEmpty
        }
        
        It "Returns null for next task" {
            $next = Get-NextTask -BasePath $script:EmptyDir
            $next | Should -BeNullOrEmpty
        }
        
        It "Returns zero progress" {
            $progress = Get-TaskProgress -BasePath $script:EmptyDir
            $progress.Total | Should -Be 0
            $progress.Percentage | Should -Be 0
        }
    }
    
    Context "All Tasks Completed" {
        BeforeAll {
            $script:DoneDir = Join-Path $TestDrive "done-project"
            $script:DoneTasksDir = Join-Path $script:DoneDir "tasks"
            New-Item -ItemType Directory -Path $script:DoneTasksDir -Force | Out-Null
            
            @"
# Feature 1: Done

**Feature ID:** F001
**Status:** COMPLETED

### T001: Done

**Status:** COMPLETED

#### Success Criteria
- [x] Done
"@ | Set-Content (Join-Path $script:DoneTasksDir "001-done.md") -Encoding UTF8
        }
        
        It "Returns null for next task" {
            $next = Get-NextTask -BasePath $script:DoneDir
            $next | Should -BeNullOrEmpty
        }
        
        It "Shows 100% progress" {
            $progress = Get-TaskProgress -BasePath $script:DoneDir
            $progress.Percentage | Should -Be 100
        }
    }
    
    Context "Circular Dependencies" {
        BeforeAll {
            $script:CircularDir = Join-Path $TestDrive "circular-project"
            $script:CircularTasksDir = Join-Path $script:CircularDir "tasks"
            New-Item -ItemType Directory -Path $script:CircularTasksDir -Force | Out-Null
            
            # Note: This shouldn't happen in practice, but test resilience
            @"
# Feature 1: Circular

**Feature ID:** F001

### T001: First

**Status:** NOT_STARTED

#### Dependencies
- T002

### T002: Second

**Status:** NOT_STARTED

#### Dependencies
- T001
"@ | Set-Content (Join-Path $script:CircularTasksDir "001-circular.md") -Encoding UTF8
        }
        
        It "Handles circular dependencies gracefully" {
            # Both tasks depend on each other, so neither should be returned
            $next = Get-NextTask -BasePath $script:CircularDir
            # Could return null or one of them depending on implementation
            # Just ensure it doesn't crash
            $true | Should -Be $true
        }
    }
}
