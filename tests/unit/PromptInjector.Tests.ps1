#Requires -Modules Pester

BeforeAll {
    . "$PSScriptRoot\..\..\lib\PromptInjector.ps1"
}

Describe "PromptInjector Module" {
    BeforeAll {
        $script:TestDir = Join-Path $TestDrive "test-project"
        New-Item -ItemType Directory -Path $script:TestDir -Force | Out-Null
        
        # Create test PROMPT.md
        $promptContent = @"
# Project Instructions

This is the main prompt file.

## Guidelines

Follow best practices.
"@
        Set-Content -Path (Join-Path $script:TestDir "PROMPT.md") -Value $promptContent -Encoding UTF8
        
        # Test task
        $script:TestTask = @{
            TaskId = "T001"
            Name = "Create Login Form"
            Description = "Build a login form with email and password"
            Priority = "P1"
            FeatureId = "F001"
            FilesToTouch = @("src/Login.tsx", "src/login.css")
            Dependencies = @()
            SuccessCriteria = @("Form renders", "Validation works")
            TechnicalDetails = "Use React Hook Form"
        }
    }
    
    Context "Get-PromptPath" {
        It "Returns correct path" {
            $path = Get-PromptPath -BasePath $script:TestDir
            $path | Should -Be (Join-Path $script:TestDir "PROMPT.md")
        }
    }
    
    Context "Test-PromptExists" {
        It "Returns true when PROMPT.md exists" {
            Test-PromptExists -BasePath $script:TestDir | Should -Be $true
        }
        
        It "Returns false when PROMPT.md does not exist" {
            Test-PromptExists -BasePath "C:\nonexistent" | Should -Be $false
        }
    }
    
    Context "Get-TaskPromptSection" {
        It "Generates section with task ID" {
            $section = Get-TaskPromptSection -Task $script:TestTask
            $section | Should -Match "T001"
        }
        
        It "Includes task name" {
            $section = Get-TaskPromptSection -Task $script:TestTask
            $section | Should -Match "Create Login Form"
        }
        
        It "Includes files to touch" {
            $section = Get-TaskPromptSection -Task $script:TestTask
            $section | Should -Match "src/Login.tsx"
        }
        
        It "Includes success criteria" {
            $section = Get-TaskPromptSection -Task $script:TestTask
            $section | Should -Match "Form renders"
        }
        
        It "Includes Hermes_STATUS example" {
            $section = Get-TaskPromptSection -Task $script:TestTask
            $section | Should -Match "---Hermes_STATUS---"
        }
        
        It "Has start and end markers" {
            $section = Get-TaskPromptSection -Task $script:TestTask
            $section | Should -Match "Hermes_TASK_START"
            $section | Should -Match "Hermes_TASK_END"
        }
    }
    
    Context "Add-TaskToPrompt" {
        It "Adds task section to PROMPT.md" {
            Add-TaskToPrompt -Task $script:TestTask -BasePath $script:TestDir | Should -Be $true
            
            $content = Get-Content (Join-Path $script:TestDir "PROMPT.md") -Raw
            $content | Should -Match "Hermes_TASK_START"
            $content | Should -Match "T001"
        }
        
        It "Updates existing task section" {
            $newTask = $script:TestTask.Clone()
            $newTask.TaskId = "T002"
            $newTask.Name = "New Task"
            
            Add-TaskToPrompt -Task $newTask -BasePath $script:TestDir | Should -Be $true
            
            $content = Get-Content (Join-Path $script:TestDir "PROMPT.md") -Raw
            $content | Should -Match "T002"
            # Should only have one task section
            ([regex]::Matches($content, "Hermes_TASK_START")).Count | Should -Be 1
        }
    }
    
    Context "Get-CurrentTaskFromPrompt" {
        It "Returns current task ID" {
            Add-TaskToPrompt -Task $script:TestTask -BasePath $script:TestDir
            
            $taskId = Get-CurrentTaskFromPrompt -BasePath $script:TestDir
            $taskId | Should -Not -BeNullOrEmpty
        }
    }
    
    Context "Test-TaskSectionExists" {
        It "Returns true when section exists" {
            Add-TaskToPrompt -Task $script:TestTask -BasePath $script:TestDir
            Test-TaskSectionExists -BasePath $script:TestDir | Should -Be $true
        }
    }
    
    Context "Remove-TaskFromPrompt" {
        It "Removes task section" {
            Add-TaskToPrompt -Task $script:TestTask -BasePath $script:TestDir
            Remove-TaskFromPrompt -BasePath $script:TestDir | Should -Be $true
            
            $content = Get-Content (Join-Path $script:TestDir "PROMPT.md") -Raw
            $content | Should -Not -Match "Hermes_TASK_START"
        }
        
        It "Preserves original content" {
            Remove-TaskFromPrompt -BasePath $script:TestDir
            
            $content = Get-Content (Join-Path $script:TestDir "PROMPT.md") -Raw
            $content | Should -Match "Project Instructions"
        }
    }
    
    Context "Get-MinimalTaskPrompt" {
        It "Generates compact prompt" {
            $minimal = Get-MinimalTaskPrompt -Task $script:TestTask
            $minimal.Length | Should -BeLessThan 1000
        }
        
        It "Includes task ID and name" {
            $minimal = Get-MinimalTaskPrompt -Task $script:TestTask
            $minimal | Should -Match "T001"
            $minimal | Should -Match "Create Login Form"
        }
    }
    
    Context "Backup-Prompt and Restore-Prompt" {
        It "Creates backup" {
            $backupPath = Backup-Prompt -BasePath $script:TestDir
            $backupPath | Should -Not -BeNullOrEmpty
            Test-Path $backupPath | Should -Be $true
        }
        
        It "Restores from backup" {
            $backupPath = Backup-Prompt -BasePath $script:TestDir
            
            # Modify PROMPT.md
            "Modified content" | Set-Content (Join-Path $script:TestDir "PROMPT.md")
            
            # Restore
            Restore-Prompt -BackupPath $backupPath -BasePath $script:TestDir | Should -Be $true
            
            $content = Get-Content (Join-Path $script:TestDir "PROMPT.md") -Raw
            $content | Should -Match "Project Instructions"
        }
    }
    
    Context "Get-LatestBackup" {
        It "Returns most recent backup" {
            Backup-Prompt -BasePath $script:TestDir
            Start-Sleep -Milliseconds 100
            $latest = Backup-Prompt -BasePath $script:TestDir
            
            $found = Get-LatestBackup -BasePath $script:TestDir
            $found | Should -Be $latest
        }
    }
}
