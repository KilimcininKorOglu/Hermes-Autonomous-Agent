# Hermes for Claude Code - User Guide

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

### What is Hermes?

Hermes is a system that automates software development by running AI CLI tools (Claude Code, Droid, Aider) in an autonomous loop.

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
hermes-claude-code/
├── hermes_loop.ps1          # Main execution loop
├── hermes-prd.ps1           # PRD to task conversion
├── hermes-add.ps1           # Single feature addition
├── hermes_monitor.ps1       # Live monitoring panel
├── install.ps1             # Global installation
├── setup.ps1               # Project creation
├── lib/                    # PowerShell modules
│   ├── AIProvider.ps1      # AI CLI abstraction
│   ├── CircuitBreaker.ps1  # Stagnation detection
│   ├── ConfigManager.ps1   # Configuration management
│   ├── FeatureAnalyzer.ps1 # Feature analysis
│   ├── GitBranchManager.ps1  # Git operations
│   ├── Logger.ps1          # Logging system
│   ├── PromptInjector.ps1  # PROMPT.md management
│   ├── ResponseAnalyzer.ps1 # AI response analysis
│   ├── TableFormatter.ps1  # ASCII table formatting
│   ├── TaskReader.ps1      # Task file reading
│   └── TaskStatusUpdater.ps1 # Status updates
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

### Installing Hermes

```powershell
# Clone the repository
git clone https://github.com/frankbria/hermes-claude-code.git
cd hermes-claude-code

# Install globally
.\install.ps1

# Verify installation
hermes -Help
```

### Installation Paths

| Type      | Path                              |
|-----------|-----------------------------------|
| Commands  | `$env:LOCALAPPDATA\Hermes\bin\`    |
| Scripts   | `$env:LOCALAPPDATA\Hermes\`        |
| Templates | `$env:LOCALAPPDATA\Hermes\templates\` |

### Uninstalling

```powershell
.\install.ps1 -Uninstall
```

---

## 3. Quick Start

### Creating a New Project

```powershell
# Create project
hermes-setup my-project
cd my-project

# Edit PROMPT.md
notepad PROMPT.md

# Start Hermes
hermes -Monitor
```

### Working with Task Mode

```powershell
# Copy PRD to .hermes/docs/
copy docs/PRD.md .hermes/docs/PRD.md

# Create tasks from PRD
hermes-prd .hermes/docs/PRD.md

# Start task mode
hermes -TaskMode -AutoBranch -AutoCommit

# Run in autonomous mode (no pauses between tasks)
hermes -TaskMode -AutoBranch -AutoCommit -Autonomous
```

### Project Structure

```
my-project/
├── .gitignore          # Contains ".hermes/" to exclude from git
├── .hermes/            # All Hermes files (gitignored)
│   ├── config.json     # Project configuration
│   ├── PROMPT.md       # AI prompt file
│   ├── tasks/          # Task files (001-xxx.md format)
│   ├── logs/           # Execution logs
│   ├── docs/           # PRD and documentation
│   ├── status.json     # Current status
│   ├── progress.json   # Progress tracking
│   └── .circuit_breaker_state
├── src/                # Project code (created by AI, git tracked)
├── tests/              # Test files
└── README.md           # Project readme
```

---

## 4. Commands

### Main Commands

| Command                | Description                    |
|------------------------|--------------------------------|
| `hermes -Monitor`       | Start with monitoring window   |
| `hermes -Status`        | Show current status            |
| `hermes -Help`          | Show help message              |
| `hermes -ResetCircuit`  | Reset circuit breaker          |
| `hermes-setup <name>`   | Create new project             |
| `hermes-prd <file>`     | Convert PRD to tasks           |
| `hermes-add "feature"`  | Add single feature             |
| `hermes-monitor`        | Standalone monitoring panel    |

### Hermes Loop Parameters

```powershell
hermes [-Monitor] [-Calls <int>] [-Timeout <int>] [-VerboseProgress]
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
.hermes/docs/PRD.md -> hermes-prd -> .hermes/tasks/*.md -> hermes -TaskMode -> Implementation
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

Hermes automatically resumes from where it left off using `run-state.md`:

```powershell
# First run - interrupted at T003
hermes -TaskMode -AutoBranch -AutoCommit
# ... interruption or context limit

# Next run - resumes from T004
hermes -TaskMode -AutoBranch -AutoCommit
# Output: "Previous run detected - Resuming from T004..."
```

### Status Display

```powershell
# Full status table
hermes -TaskStatus

# Filtered status
hermes -TaskStatus -StatusFilter BLOCKED
hermes -TaskStatus -FeatureFilter F001
hermes -TaskStatus -PriorityFilter P1

# Combined filter
hermes -TaskStatus -StatusFilter NOT_STARTED -PriorityFilter P1
```

---

## 6. PRD Parser

### Usage

```powershell
hermes-prd <prd-file> [-AI <provider>] [-DryRun] [-Force] [-Clean]
```

### Parameters

| Parameter     | Description                                   |
|---------------|-----------------------------------------------|
| `<prd-file>`  | PRD markdown file                             |
| `-AI`         | AI provider (default: from config or claude)  |
| `-DryRun`     | Preview without creating files                |
| `-OutputDir`  | Output directory (default: .hermes/tasks)     |
| `-Timeout`    | AI timeout in seconds (default: 1200)         |
| `-MaxRetries` | Retry count (default: 10)                     |
| `-Force`      | Overwrite NOT_STARTED features                |
| `-Clean`      | Delete all existing tasks, start fresh        |

### Incremental Mode

By default, `hermes-prd` runs in incremental mode:

```powershell
# First run - creates all features
hermes-prd .hermes/docs/PRD.md

# PRD updated, new features added
hermes-prd .hermes/docs/PRD.md
# Only adds NEW features, preserves existing progress

# Clean start
hermes-prd .hermes/docs/PRD.md -Clean
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
[INFO] Reading PRD: .hermes/docs/PRD.md
[INFO] PRD size: 5946 characters, 264 lines
[INFO] Using AI: claude (timeout: 1200s, retries: 10)
[INFO] Parsing PRD with claude...

[INFO] Attempt 1/10...
[DEBUG] Starting claude execution at 14:36:39...
[DEBUG] claude completed in 166.5 seconds
[OK] AI completed successfully

[OK] Created: 001-user-authentication.md (F001, T001-T007)
[OK] Created: 002-product-catalog.md (F002, T008-T014)
[OK] Created: tasks-status.md

[SUCCESS] Created 2 features with 14 tasks
Summary:
  New Features: 2
  New Tasks: 14
  Estimated: 0 days
  Attempts: 1

Next: Run 'hermes -TaskMode -AutoBranch -AutoCommit' to start
```

---

## 7. Feature Add

### Usage

```powershell
hermes-add <feature> [-AI <provider>] [-DryRun] [-Priority <P1-P4>]
```

### Input Types

```powershell
# Inline description
hermes-add "user registration system"

# Read from file
hermes-add @docs/webhook-spec.md

# Specify priority
hermes-add "password reset" -Priority P1

# Preview
hermes-add "email verification" -DryRun
```

### Parameters

| Parameter    | Description                                  |
|--------------|----------------------------------------------|
| `<feature>`  | Feature description or @file-path            |
| `-AI`        | AI provider (default: from config or claude) |
| `-DryRun`    | Preview without creating files               |
| `-Priority`  | Priority: P1, P2, P3, P4                     |
| `-OutputDir` | Output directory (default: .hermes/tasks)    |
| `-Timeout`   | AI timeout in seconds (default: 300)         |

### Example Output

```
[INFO] Reading feature input...
[INFO] Source: inline description
[INFO] Next Feature ID: F003
[INFO] Next Task ID: T012
[INFO] Using AI: claude (planning task)
[INFO] Analyzing feature with claude...

==================================================
  Feature added!
==================================================

  Feature ID: F003
  File:       .hermes/tasks/003-email-verification.md
  Name:       Email Verification
  Priority:   P2 - High
  Tasks:      4 (T012-T015)
  Effort:     3 days (total)

==================================================

Next: Run 'hermes -TaskMode -AutoBranch -AutoCommit' to implement
```

---

## 8. AI Provider System

### Supported Providers

| Provider | Command  | Description        |
|----------|----------|--------------------|
| Claude   | `claude` | Claude Code CLI    |
| Droid    | `droid`  | Factory Droid CLI  |
| Aider    | `aider`  | Aider AI CLI       |

### Task-Based AI Selection

Hermes uses different AI providers for different task types:

| Task Type | Default AI | Description                          |
|-----------|------------|--------------------------------------|
| Planning  | `claude`   | PRD parsing, feature addition        |
| Coding    | `droid`    | Task execution, code implementation  |

This is configured in `.hermes/config.json`:

```json
{
  "ai": {
    "planning": "claude",
    "coding": "droid"
  }
}
```

### Auto Detection

If no config exists, Hermes automatically detects available AI CLIs. Priority order:

1. `claude` (highest)
2. `droid`
3. `aider`

### Provider Selection

```powershell
# Uses task-based selection (planning=claude, coding=droid)
hermes -TaskMode
hermes-prd .hermes/docs/PRD.md
hermes-add "feature"

# Override with specific provider
hermes -TaskMode -AI claude
hermes-prd .hermes/docs/PRD.md -AI droid
hermes-add "feature" -AI aider
```

### Provider Check

```powershell
# List available providers
hermes-prd -List
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

### ConfigManager.ps1

Configuration management for global and project settings.

| Function               | Description                      |
|------------------------|----------------------------------|
| `Get-HermesConfig`     | Get merged configuration         |
| `Get-ConfigValue`      | Get specific config value        |
| `Set-ConfigValue`      | Set config value                 |
| `Get-AIForTask`        | Get AI provider for task type    |
| `Initialize-ProjectConfig` | Create project config        |

### Logger.ps1

Logging system for session tracking.

| Function            | Description                    |
|---------------------|--------------------------------|
| `Initialize-Logger` | Start logging session          |
| `Write-Log`         | Write log entry                |
| `Write-LogSection`  | Write section header           |
| `Close-Logger`      | End logging session            |

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

### Project Configuration (.hermes/config.json)

```json
{
  "ai": {
    "planning": "claude",
    "coding": "droid",
    "timeout": 300,
    "prdTimeout": 1200,
    "maxRetries": 10
  },
  "taskMode": {
    "autoBranch": false,
    "autoCommit": false,
    "autonomous": false,
    "maxConsecutiveErrors": 5
  },
  "loop": {
    "maxCallsPerHour": 100,
    "timeoutMinutes": 15
  },
  "paths": {
    "hermesDir": ".hermes",
    "tasksDir": ".hermes/tasks",
    "logsDir": ".hermes/logs",
    "docsDir": ".hermes/docs"
  }
}
```

### Project Control Files

| File                              | Purpose                          |
|-----------------------------------|----------------------------------|
| `.hermes/config.json`             | Project configuration            |
| `.hermes/PROMPT.md`               | Instructions fed to AI each loop |
| `.hermes/tasks/*.md`              | Feature and task definitions     |
| `.hermes/tasks/run-state.md`      | Resume checkpoint                |
| `.hermes/tasks/tasks-status.md`   | Status tracking                  |
| `.hermes/status.json`             | Live loop status                 |
| `.hermes/progress.json`           | Progress tracking                |
| `.hermes/.circuit_breaker_state`  | Circuit breaker state            |
| `.hermes/logs/`                   | AI output and session logs       |

### PROMPT.md Status Block

AI should output this block at the end of each response:

```
---HERMES_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
RECOMMENDATION: <next action>
---END_HERMES_STATUS---
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
$env:PATH = "$env:LOCALAPPDATA\Hermes\bin;$env:PATH"
```

### Circuit Breaker Opened

```
CIRCUIT BREAKER OPENED - Execution halted
```

**Solution:**

1. Review recent logs:

   ```powershell
   Get-Content .hermes\logs\hermes-loop-*.log -Tail 20
   ```

2. Check AI output:

   ```powershell
   Get-ChildItem .hermes\logs\*_output_*.log | Sort-Object LastWriteTime -Descending | Select-Object -First 1
   ```

3. Fix the issue and reset:

   ```powershell
   hermes -ResetCircuit
   ```

### Task Not Found

```
Task not found: T005
```

**Solution:**

- Ensure task ID is typed correctly
- Check that `.hermes/tasks/` directory exists
- List existing tasks with `hermes -TaskStatus`

### Resume Not Working

Resume mechanism depends on `run-state.md`:

```powershell
# Check run-state.md
Get-Content .hermes/tasks/run-state.md

# Manually start from specific task
hermes -TaskMode -StartFrom T005
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

---

**Version:** 1.1  
**Last Updated:** 2025-12-26
