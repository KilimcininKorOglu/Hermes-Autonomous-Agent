# Ralph for Claude Code - User Guide

Autonomous AI development loop system for Windows PowerShell. Supports Claude, Droid, and Aider CLIs.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Installation](#2-installation)
3. [Quick Start](#3-quick-start)
4. [Commands](#4-commands)
5. [Task Mode](#5-task-mode)
6. [PRD Parser](#6-prd-parser)
7. [Feature Add](#7-feature-add)
8. [AI Provider System](#8-ai-provider-system)
9. [Module Details](#9-module-details)
10. [Configuration](#10-configuration)
11. [Troubleshooting](#11-troubleshooting)

---

## 1. Overview

### What is Ralph?

Ralph is a system that automates software development by running AI CLI tools (Claude Code, Droid, Aider) in an autonomous loop.

### Key Features

| Feature            | Description                                      |
|--------------------|--------------------------------------------------|
| Multi-AI Support   | Supports Claude, Droid, and Aider CLIs           |
| Task Mode          | Structured task-based development                |
| Auto Branch        | Automatic Git branch creation for each feature   |
| Auto Commit        | Automatic commit on task completion              |
| Autonomous Mode    | Continuous operation without user intervention   |
| Resume             | Resume from where it left off after interruption |
| Circuit Breaker    | Stagnation detection and protection              |
| ASCII Status       | Progress display in colored table format         |

### Architecture

```
ralph-claude-code/
├── ralph_loop.ps1          # Main execution loop
├── ralph-prd.ps1           # PRD to task conversion
├── ralph-add.ps1           # Single feature addition
├── ralph_monitor.ps1       # Live monitoring panel
├── install.ps1             # Global installation
├── setup.ps1               # Project creation
├── lib/                    # PowerShell modules
│   ├── AIProvider.ps1      # AI CLI abstraction
│   ├── TaskReader.ps1      # Task file reading
│   ├── TaskStatusUpdater.ps1 # Status updates
│   ├── GitBranchManager.ps1  # Git operations
│   ├── TableFormatter.ps1  # ASCII table formatting
│   ├── CircuitBreaker.ps1  # Stagnation detection
│   ├── ResponseAnalyzer.ps1 # AI response analysis
│   ├── PromptInjector.ps1  # PROMPT.md management
│   └── FeatureAnalyzer.ps1 # Feature analysis
├── templates/              # Project templates
└── tests/unit/             # Pester tests
```

---

## 2. Installation

### Requirements

| Requirement    | Description                           |
|----------------|---------------------------------------|
| PowerShell 7+  | NOT Windows PowerShell 5.1            |
| Git            | Version control                       |
| AI CLI         | At least one: claude, droid, or aider |

### Installing Dependencies

```powershell
# Using winget
winget install Microsoft.PowerShell Git.Git

# Using Chocolatey
choco install powershell-core git

# AI CLI installation (install at least one)
npm install -g @anthropic-ai/claude-code  # Claude
pip install aider-chat                     # Aider
```

### Installing Ralph

```powershell
# Clone the repository
git clone https://github.com/frankbria/ralph-claude-code.git
cd ralph-claude-code

# Install globally
.\install.ps1

# Verify installation
ralph -Help
```

### Installation Paths

| Type      | Path                              |
|-----------|-----------------------------------|
| Commands  | `$env:LOCALAPPDATA\Ralph\bin\`    |
| Scripts   | `$env:LOCALAPPDATA\Ralph\`        |
| Templates | `$env:LOCALAPPDATA\Ralph\templates\` |

### Uninstalling

```powershell
.\install.ps1 -Uninstall
```

---

## 3. Quick Start

### Creating a New Project

```powershell
# Create project
ralph-setup my-project
cd my-project

# Edit PROMPT.md
notepad PROMPT.md

# Start Ralph
ralph -Monitor
```

### Working with Task Mode

```powershell
# Create tasks from PRD
ralph-prd docs/PRD.md

# Start task mode
ralph -TaskMode -AutoBranch -AutoCommit

# Run in autonomous mode
ralph -TaskMode -Autonomous
```

### Project Structure

```
my-project/
├── PROMPT.md           # Main instructions
├── @fix_plan.md        # Task list
├── @AGENT.md           # Build/run instructions
├── specs/              # Project specifications
├── src/                # Source code
├── logs/               # Log files
├── tasks/              # Task files (Task Mode)
│   ├── 001-feature.md  # Feature files
│   ├── tasks-status.md # Status tracking
│   └── run-state.md    # Resume checkpoint
└── status.json         # Live status
```

---

## 4. Commands

### Main Commands

| Command                | Description                    |
|------------------------|--------------------------------|
| `ralph -Monitor`       | Start with monitoring window   |
| `ralph -Status`        | Show current status            |
| `ralph -Help`          | Show help message              |
| `ralph -ResetCircuit`  | Reset circuit breaker          |
| `ralph-setup <name>`   | Create new project             |
| `ralph-prd <file>`     | Convert PRD to tasks           |
| `ralph-add "feature"`  | Add single feature             |
| `ralph-monitor`        | Standalone monitoring panel    |

### Ralph Loop Parameters

```powershell
ralph [-Monitor] [-Calls <int>] [-Timeout <int>] [-VerboseProgress]
      [-Status] [-ResetCircuit] [-CircuitStatus] [-Help]
      [-AI <provider>] [-TaskMode] [-AutoBranch] [-AutoCommit]
      [-StartFrom <TaskId>] [-TaskStatus]
```

| Parameter          | Default | Description                            |
|--------------------|---------|----------------------------------------|
| `-AI`              | auto    | AI provider: claude, droid, aider, auto |
| `-Calls`           | 100     | Maximum API calls per hour             |
| `-Timeout`         | 15      | AI timeout (minutes)                   |
| `-Monitor`         | -       | Open monitoring window                 |
| `-VerboseProgress` | -       | Show detailed progress                 |

### Task Mode Parameters

| Parameter               | Description                           |
|-------------------------|---------------------------------------|
| `-TaskMode`             | Enable task-plan integration          |
| `-AutoBranch`           | Auto-create feature branches          |
| `-AutoCommit`           | Auto-commit on task completion        |
| `-StartFrom T005`       | Start from specific task              |
| `-TaskStatus`           | Show task progress table              |
| `-Autonomous`           | Run continuously without pausing      |
| `-MaxConsecutiveErrors` | Error threshold (default: 5)          |

### Filtering Parameters

| Parameter         | Example   | Description          |
|-------------------|-----------|----------------------|
| `-StatusFilter`   | COMPLETED | Filter by status     |
| `-FeatureFilter`  | F001      | Filter by feature    |
| `-PriorityFilter` | P1        | Filter by priority   |

---

## 5. Task Mode

### What is Task Mode?

Task Mode is a development mode that works with structured task files. Each feature and task is defined in separate files.

### Workflow

```
PRD.md -> ralph-prd -> tasks/*.md -> ralph -TaskMode -> Automatic Implementation
```

### Task File Format

```markdown
# Feature 1: User Authentication

**Feature ID:** F001
**Feature Name:** User Authentication
**Priority:** P1 - Critical
**Status:** NOT_STARTED
**Estimated Duration:** 1-2 weeks

## Overview

User authentication system description.

## Tasks

### T001: Database Schema

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Create users table.

#### Files to Touch
- `migrations/001_users.sql` (new)
- `src/models/user.ts` (new)

#### Dependencies
- None

#### Success Criteria
- [ ] Migration successful
- [ ] Rollback works
- [ ] Index defined
```

### ID System

| Type       | Format        | Example           |
|------------|---------------|-------------------|
| Feature ID | F + 3 digits  | F001, F002, F003  |
| Task ID    | T + 3 digits  | T001, T002, T003  |

**Important:** Task IDs continue across ALL features:
- F001: T001, T002, T003
- F002: T004, T005, T006

### Branch Strategy

```
main
  └── feature/F001-user-authentication
        ├── feat(T001): Database Schema completed
        ├── feat(T002): Registration API completed
        └── feat(T003): Login API completed
  └── feature/F002-dashboard
        └── ...
```

### Commit Format

```
feat(T001): Task Name completed

Completed:
- [x] Migration successful
- [x] Rollback works

Files:
- migrations/001_users.sql
- src/models/user.ts
```

### Automatic Resume

Ralph automatically resumes from where it left off using `run-state.md`:

```powershell
# First run - interrupted at T003
ralph -TaskMode -AutoBranch -AutoCommit
# ... interruption or context limit

# Next run - resumes from T004
ralph -TaskMode -AutoBranch -AutoCommit
# Output: "Previous run detected - Resuming from T004..."
```

### Status Display

```powershell
# Full status table
ralph -TaskStatus

# Filtered status
ralph -TaskStatus -StatusFilter BLOCKED
ralph -TaskStatus -FeatureFilter F001
ralph -TaskStatus -PriorityFilter P1

# Combined filter
ralph -TaskStatus -StatusFilter NOT_STARTED -PriorityFilter P1
```

---

## 6. PRD Parser

### Usage

```powershell
ralph-prd <prd-file> [-AI <provider>] [-DryRun] [-Force] [-Clean]
```

### Parameters

| Parameter     | Description                              |
|---------------|------------------------------------------|
| `<prd-file>`  | PRD markdown file                        |
| `-AI`         | AI provider (default: auto)              |
| `-DryRun`     | Preview without creating files           |
| `-OutputDir`  | Output directory (default: tasks)        |
| `-Timeout`    | AI timeout in seconds (default: 1200)    |
| `-MaxRetries` | Retry count (default: 10)                |
| `-Force`      | Overwrite NOT_STARTED features           |
| `-Clean`      | Delete all existing tasks, start fresh   |

### Incremental Mode

By default, `ralph-prd` runs in incremental mode:

```powershell
# First run - creates all features
ralph-prd docs/PRD.md

# PRD updated, new features added
ralph-prd docs/PRD.md
# Only adds NEW features, preserves existing progress

# Clean start
ralph-prd docs/PRD.md -Clean
```

### Incremental Mode Behavior

| Existing Status | Behavior                         |
|-----------------|----------------------------------|
| COMPLETED       | Never overwritten                |
| IN_PROGRESS     | Preserved                        |
| NOT_STARTED     | Can be overwritten with `-Force` |
| New             | Added                            |

### Example Output

```
[INFO] Reading PRD: docs/PRD.md
[INFO] PRD size: 45000 characters, 800 lines
[INFO] Using AI: claude
[INFO] Attempt 1/10...
[OK] AI completed successfully

[OK] Created: tasks/001-user-authentication.md (F001, T001-T004)
[OK] Created: tasks/002-dashboard.md (F002, T005-T008)
[OK] Created: tasks/tasks-status.md

Summary:
  Features: 2
  Tasks: 8
  Estimated: 12 days

Next: Run 'ralph -TaskMode -AutoBranch -AutoCommit' to start
```

---

## 7. Feature Add

### Usage

```powershell
ralph-add <feature> [-AI <provider>] [-DryRun] [-Priority <P1-P4>]
```

### Input Types

```powershell
# Inline description
ralph-add "user registration system"

# Read from file
ralph-add @docs/webhook-spec.md

# Specify priority
ralph-add "password reset" -Priority P1

# Preview
ralph-add "email verification" -DryRun
```

### Parameters

| Parameter    | Description                        |
|--------------|------------------------------------|
| `<feature>`  | Feature description or @file-path  |
| `-AI`        | AI provider (default: auto)        |
| `-DryRun`    | Preview without creating files     |
| `-Priority`  | Priority: P1, P2, P3, P4           |
| `-OutputDir` | Output directory (default: tasks)  |
| `-Timeout`   | AI timeout in seconds (default: 300) |

### Example Output

```
[INFO] Reading feature input...
[INFO] Source: inline description
[INFO] Next Feature ID: F003
[INFO] Next Task ID: T012
[INFO] Using AI: claude
[INFO] Analyzing feature with claude...

==================================================
  Feature added!
==================================================

  Feature ID: F003
  File:       tasks/003-email-verification.md
  Name:       Email Verification
  Priority:   P2 - High
  Tasks:      4 (T012-T015)
  Effort:     3 days (total)

==================================================

Next: Run 'ralph -TaskMode -AutoBranch -AutoCommit' to implement
```

---

## 8. AI Provider System

### Supported Providers

| Provider | Command  | Description        |
|----------|----------|--------------------|
| Claude   | `claude` | Claude Code CLI    |
| Droid    | `droid`  | Factory Droid CLI  |
| Aider    | `aider`  | Aider AI CLI       |

### Auto Detection

Ralph automatically detects available AI CLIs. Priority order:

1. `claude` (highest)
2. `droid`
3. `aider`

### Provider Selection

```powershell
# Auto detection (default)
ralph -TaskMode
ralph-prd docs/PRD.md
ralph-add "feature"

# Specific provider
ralph -TaskMode -AI droid
ralph-prd docs/PRD.md -AI claude
ralph-add "feature" -AI aider
```

### Provider Check

```powershell
# List available providers
ralph-prd -List
```

### AIProvider.ps1 Functions

| Function               | Description                        |
|------------------------|------------------------------------|
| `Test-AIProvider`      | Check if provider is available     |
| `Get-AutoProvider`     | Return first available provider    |
| `Get-AvailableProviders` | List all available providers     |
| `Invoke-TaskExecution` | Run AI for Task Mode               |
| `Invoke-AIWithRetry`   | Run AI with retry logic            |
| `Write-AIProviderList` | Display available providers        |

---

## 9. Module Details

### TaskReader.ps1

Reads and parses task files.

| Function                   | Description                     |
|----------------------------|---------------------------------|
| `Get-AllTasks`             | Return all tasks                |
| `Get-AllFeatures`          | Return all features             |
| `Get-TaskById`             | Find task by ID                 |
| `Get-FeatureById`          | Find feature by ID              |
| `Get-NextTask`             | Find next task                  |
| `Get-TaskProgress`         | Overall progress statistics     |
| `Test-TaskDependenciesMet` | Check dependencies              |

### TaskStatusUpdater.ps1

Updates task statuses and manages resume mechanism.

| Function            | Description                    |
|---------------------|--------------------------------|
| `Set-TaskStatus`    | Update task status             |
| `Set-FeatureStatus` | Update feature status          |
| `Update-RunState`   | Update resume checkpoint       |
| `Test-ShouldResume` | Check if resume is needed      |
| `Get-ResumeInfo`    | Get resume details             |
| `Get-ExecutionQueue`| Priority-sorted queue          |

### GitBranchManager.ps1

Manages Git branch and commit operations.

| Function               | Description                  |
|------------------------|------------------------------|
| `New-FeatureBranch`    | Create feature branch        |
| `Switch-ToFeatureBranch` | Switch to branch           |
| `New-TaskCommit`       | Create task commit           |
| `Merge-FeatureToMain`  | Merge to main                |
| `Get-CurrentBranch`    | Get current branch name      |
| `Test-BranchExists`    | Check if branch exists       |

### CircuitBreaker.ps1

Stagnation detection and protection.

| State     | Meaning                                    |
|-----------|--------------------------------------------|
| CLOSED    | Normal operation                           |
| HALF_OPEN | Monitoring mode (2 loops without progress) |
| OPEN      | Halted (3+ loops without progress)         |

| Function                  | Description          |
|---------------------------|----------------------|
| `Initialize-CircuitBreaker` | Initialize         |
| `Add-LoopResult`          | Record loop result   |
| `Test-ShouldHalt`         | Should halt?         |
| `Reset-CircuitBreaker`    | Reset                |
| `Show-CircuitStatus`      | Show status          |

### TableFormatter.ps1

ASCII table formatting.

| Function                 | Description                |
|--------------------------|----------------------------|
| `Format-TaskTable`       | Create task table          |
| `Write-TaskTable`        | Write colored table        |
| `Get-FilteredTasks`      | Filtered tasks             |
| `Show-EnhancedTaskStatus`| Show enhanced status       |

### PromptInjector.ps1

Injects task information into PROMPT.md.

| Function                  | Description              |
|---------------------------|--------------------------|
| `Add-TaskToPrompt`        | Add task section         |
| `Remove-TaskFromPrompt`   | Remove task section      |
| `Backup-Prompt`           | Backup PROMPT.md         |
| `Get-CurrentTaskFromPrompt` | Get current task ID    |

### FeatureAnalyzer.ps1

Feature analysis and file creation.

| Function             | Description                  |
|----------------------|------------------------------|
| `Get-NextIds`        | Next Feature/Task ID         |
| `Read-FeatureInput`  | Read input (inline/@file)    |
| `Build-FeaturePrompt`| Create AI prompt             |
| `Parse-FeatureOutput`| Parse AI output              |
| `Write-FeatureFile`  | Write feature file           |

---

## 10. Configuration

### ralph_loop.ps1 Configuration

```powershell
$script:Config = @{
    AIProvider = "claude"        # Resolved AI provider
    AITimeoutMinutes = 15        # Timeout duration
    MaxCallsPerHour = 100        # Hourly API limit
    MaxConsecutiveErrors = 5     # Error threshold
    TasksDir = "tasks"           # Task directory
    LogDir = "logs"              # Log directory
    StatusFile = "status.json"   # Status file
}
```

### Project Control Files

| File                     | Purpose                          |
|--------------------------|----------------------------------|
| `PROMPT.md`              | Instructions fed to AI each loop |
| `tasks/*.md`             | Feature and task definitions     |
| `tasks/run-state.md`     | Resume checkpoint                |
| `tasks/tasks-status.md`  | Status tracking                  |
| `status.json`            | Live loop status                 |
| `.circuit_breaker_state` | Circuit breaker state            |

### PROMPT.md Status Block

AI should output this block at the end of each response:

```
---RALPH_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
RECOMMENDATION: <next action>
---END_RALPH_STATUS---
```

---

## 11. Troubleshooting

### Script Execution Error

```
File cannot be loaded because running scripts is disabled
```

**Solution:**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### AI CLI Not Found

```
'claude' is not recognized
```

**Solution:**
```powershell
# Claude
npm install -g @anthropic-ai/claude-code

# Aider
pip install aider-chat
```

### PATH Not Updated

Restart terminal after installation or:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Ralph\bin;$env:PATH"
```

### Circuit Breaker Opened

```
CIRCUIT BREAKER OPENED - Execution halted
```

**Solution:**
1. Review recent logs:
   ```powershell
   Get-Content logs\ralph.log -Tail 20
   ```

2. Check AI output:
   ```powershell
   Get-ChildItem logs\*_output_*.log | Sort-Object LastWriteTime -Descending | Select-Object -First 1
   ```

3. Fix the issue and reset:
   ```powershell
   ralph -ResetCircuit
   ```

### Task Not Found

```
Task not found: T005
```

**Solution:**
- Ensure task ID is typed correctly
- Check that `tasks/` directory exists
- List existing tasks with `ralph -TaskStatus`

### Resume Not Working

Resume mechanism depends on `run-state.md`:

```powershell
# Check run-state.md
Get-Content tasks/run-state.md

# Manually start from specific task
ralph -TaskMode -StartFrom T005
```

### Syntax Check

Check for syntax errors in PowerShell files:

```powershell
$errors = $null
$null = [System.Management.Automation.Language.Parser]::ParseFile(
    'path/to/file.ps1', 
    [ref]$null, 
    [ref]$errors
)
if ($errors.Count -gt 0) { 
    $errors | ForEach-Object { Write-Host $_.Message } 
}
```

### Running Tests

```powershell
# Run all tests
Import-Module Pester -Force
Invoke-Pester -Path tests/unit/

# Single test file
Invoke-Pester -Path tests/unit/AIProvider.Tests.ps1

# Detailed output
Invoke-Pester -Path tests/unit/ -PassThru
```

---

## Additional Resources

- [README.md](../README.md) - Project overview
- [CLAUDE.md](../CLAUDE.md) - Guide for Claude Code
- [AGENTS.md](../AGENTS.md) - Guide for AI agents
- [plan/](../plan/) - Development plans

---

**Version:** 1.0  
**Last Updated:** 2025-12-25
