# Ralph for Claude Code - Windows PowerShell Version

Native Windows PowerShell port of Ralph, the autonomous AI development loop system.

## Requirements

- **PowerShell 7+** (not Windows PowerShell 5.1)
- **Node.js** (for Claude Code CLI)
- **Git**

### Install Dependencies

```powershell
# Using winget (Windows 10/11)
winget install Microsoft.PowerShell OpenJS.NodeJS.LTS Git.Git

# Or using Chocolatey
choco install powershell-core nodejs-lts git
```

## Installation

```powershell
# Clone the repository
git clone https://github.com/frankbria/ralph-claude-code.git
cd ralph-claude-code\windows

# Install Ralph globally
.\install.ps1

# Verify installation
ralph -Help
```

### Installation Paths

| Type | Path |
|------|------|
| Commands | `$env:LOCALAPPDATA\Ralph\bin\` |
| Scripts | `$env:LOCALAPPDATA\Ralph\` |
| Templates | `$env:LOCALAPPDATA\Ralph\templates\` |

## Quick Start

```powershell
# Create a new project
ralph-setup my-project
cd my-project

# Edit PROMPT.md with your requirements
# Then start Ralph with monitoring
ralph -Monitor
```

## Commands

| Command | Description |
|---------|-------------|
| `ralph -Monitor` | Start loop with monitoring window |
| `ralph -Status` | Show current status |
| `ralph -Help` | Show help |
| `ralph -ResetCircuit` | Reset circuit breaker |
| `ralph-setup <name>` | Create new project |
| `ralph-import <file>` | Convert PRD to project |
| `ralph-monitor` | Standalone monitor |

### Ralph Loop Options

```powershell
ralph [-Monitor] [-Calls <int>] [-Timeout <int>] [-VerboseProgress]
      [-Status] [-ResetCircuit] [-CircuitStatus] [-Help]
      [-TaskMode] [-AutoBranch] [-AutoCommit] [-StartFrom <TaskId>]
      [-TaskStatus]

-Monitor          Start with separate monitoring window
-Calls <int>      Max API calls per hour (default: 100)
-Timeout <int>    Claude timeout in minutes (default: 15)
-VerboseProgress  Show detailed progress during execution
-Status           Show current loop status
-ResetCircuit     Reset circuit breaker to CLOSED
-CircuitStatus    Show circuit breaker status

# Task Mode Options
-TaskMode         Enable task-plan integration mode
-AutoBranch       Auto-create/switch feature branches
-AutoCommit       Auto-commit on task completion
-StartFrom <id>   Start from specific task (e.g., T005)
-TaskStatus       Show task progress and exit
```

## Project Structure

When you run `ralph-setup my-project`, it creates:

```
my-project/
  PROMPT.md         # Development instructions for Ralph
  @fix_plan.md      # Prioritized task checklist
  @AGENT.md         # Build/run instructions
  specs/            # Project specifications
  src/              # Source code
  logs/             # Execution logs
  docs/generated/   # Auto-generated docs
```

## How It Works

1. **Read Instructions** - Loads `PROMPT.md` with your project requirements
2. **Execute Claude Code** - Runs Claude with current context
3. **Analyze Response** - Checks for completion signals and progress
4. **Track Progress** - Updates task lists and logs results
5. **Repeat** - Continues until project complete or limits reached

### Intelligent Exit Detection

Ralph automatically stops when it detects:
- All tasks in `@fix_plan.md` marked complete
- Multiple consecutive "done" signals from Claude
- Too many test-only loops (no implementation)
- Circuit breaker opens (no progress)

### Circuit Breaker

Prevents runaway execution by detecting stagnation:

| State | Meaning |
|-------|---------|
| CLOSED | Normal operation |
| HALF_OPEN | Monitoring (2 no-progress loops) |
| OPEN | Halted (3+ no-progress loops) |

Reset with: `ralph -ResetCircuit`

## Task Mode

Ralph supports integration with task-plan systems for structured development:

### Task Mode Workflow

```
PRD.md → task-plan → tasks/*.md → Ralph TaskMode → Automated Implementation
```

### Usage

```powershell
# Create tasks directory with feature files
# tasks/001-authentication.md, tasks/002-dashboard.md, etc.

# Start Ralph in task mode with full automation
ralph -TaskMode -AutoBranch -AutoCommit -Monitor

# Show task progress
ralph -TaskStatus

# Start from specific task
ralph -TaskMode -StartFrom T005
```

### Task File Format

```markdown
# Feature 1: User Authentication

**Feature ID:** F001
**Status:** NOT_STARTED

### T001: Login Form

**Status:** NOT_STARTED
**Priority:** P1

#### Description
Create login form component.

#### Files to Touch
- `src/Login.tsx` (new)

#### Dependencies
- None

#### Success Criteria
- [ ] Form renders
- [ ] Validation works
```

### How Task Mode Works

1. **Find Next Task** - Reads `tasks/*.md`, finds first NOT_STARTED with met dependencies
2. **Create Branch** - With `-AutoBranch`: creates `feature/F001-authentication`
3. **Inject Task** - Adds task details to PROMPT.md
4. **Execute Claude** - Runs Claude Code focused on current task
5. **Commit** - With `-AutoCommit`: creates `feat(T001): Login Form completed`
6. **Update Status** - Marks task COMPLETED
7. **Check Feature** - If all tasks done, merges to main
8. **Next Task** - Continues to next task

### Branch Strategy

```
main
  └── feature/F001-authentication
        ├── feat(T001): Login Form completed
        ├── feat(T002): Auth API completed
        └── feat(T003): Session Management completed
  └── feature/F002-dashboard
        └── ...
```

### Commit Format

```
feat(T001): Login Form completed

Completed:
- [x] Form renders
- [x] Validation works

Files:
- src/Login.tsx
```

### Task Modules

| Module | Purpose |
|--------|---------|
| `lib/TaskReader.ps1` | Parse tasks/*.md files |
| `lib/TaskStatusUpdater.ps1` | Update task statuses |
| `lib/GitBranchManager.ps1` | Branch/commit management |
| `lib/PromptInjector.ps1` | Inject task into PROMPT.md |

## Configuration

### Rate Limiting

Default: 100 API calls per hour

```powershell
# Custom limit
ralph -Calls 50
```

### Execution Timeout

Default: 15 minutes per Claude execution

```powershell
# 30 minute timeout for complex tasks
ralph -Timeout 30
```

### Thresholds

Edit `ralph_loop.ps1` to customize:

```powershell
$script:Config = @{
    MaxCallsPerHour = 100
    ClaudeTimeoutMinutes = 15
    MaxConsecutiveTestLoops = 3
    MaxConsecutiveDoneSignals = 2
}
```

## Testing

```powershell
# Install Pester
Install-Module -Name Pester -Force -SkipPublisherCheck

# Run all tests
.\tests\Run-Tests.ps1

# Run with coverage
.\tests\Run-Tests.ps1 -Coverage

# Run only unit tests
.\tests\Run-Tests.ps1 -Unit
```

## Files Reference

### Control Files (in project)

| File | Purpose |
|------|---------|
| `PROMPT.md` | Instructions for Claude |
| `@fix_plan.md` | Task checklist |
| `@AGENT.md` | Build/run commands |
| `status.json` | Current loop status |
| `.call_count` | API calls this hour |
| `.exit_signals` | Completion tracking |
| `.circuit_breaker_state` | Stagnation detection |

### Scripts

| Script | Purpose |
|--------|---------|
| `ralph_loop.ps1` | Main execution loop |
| `ralph_monitor.ps1` | Live dashboard |
| `install.ps1` | Global installation |
| `setup.ps1` | Project creation |
| `ralph_import.ps1` | PRD conversion |
| `lib\CircuitBreaker.ps1` | Stagnation detection |
| `lib\ResponseAnalyzer.ps1` | Response analysis |
| `lib\TaskReader.ps1` | Task file parsing |
| `lib\TaskStatusUpdater.ps1` | Task status updates |
| `lib\GitBranchManager.ps1` | Git branch/commit |
| `lib\PromptInjector.ps1` | PROMPT.md injection |

## Uninstall

```powershell
.\install.ps1 -Uninstall
```

Or manually:

```powershell
Remove-Item $env:LOCALAPPDATA\Ralph -Recurse -Force
# Remove from PATH manually if needed
```

## Differences from Unix Version

| Feature | Unix | Windows |
|---------|------|---------|
| Shell | Bash | PowerShell 7+ |
| JSON parsing | jq | Native (ConvertFrom-Json) |
| Terminal multiplexer | tmux | Windows Terminal / separate windows |
| Path separator | / | \ |
| Commands | ralph, ralph-monitor | ralph.cmd, ralph-monitor.cmd |
| Installation | ~/.ralph | $env:LOCALAPPDATA\Ralph |

## Troubleshooting

### PowerShell Script Errors

```
File cannot be loaded because running scripts is disabled
```

Fix:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Claude Code Not Found

```
'claude' is not recognized
```

Fix:
```powershell
npm install -g @anthropic-ai/claude-code
```

### PATH Not Updated

After installation, restart your terminal or run:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Ralph\bin;$env:PATH"
```

## License

MIT License - See [LICENSE](../LICENSE)
