# Hermes Autonomous Agent - User Guide

Complete guide to using Hermes, the AI-powered autonomous application development system.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Project Initialization](#project-initialization)
4. [Idea to PRD Generation](#idea-to-prd-generation)
5. [PRD Parsing](#prd-parsing)
6. [Adding Features](#adding-features)
7. [Task Execution](#task-execution)
8. [Status and Monitoring](#status-and-monitoring)
9. [Interactive TUI](#interactive-tui)
10. [Configuration](#configuration)
11. [Circuit Breaker](#circuit-breaker)
12. [Troubleshooting](#troubleshooting)

---

## Installation

### Prerequisites

- Go 1.24 or higher
- Git
- One of the following AI CLIs:
  - Claude CLI: `npm install -g @anthropic-ai/claude-code`
  - Droid CLI: `curl -fsSL https://app.factory.ai/cli | sh`
  - Gemini CLI: `npm install -g @google/gemini-cli`

### Building from Source

```bash
# Clone the repository
git clone https://github.com/YourUsername/hermes.git
cd hermes

# Build for your platform
build.bat              # Windows
make build             # Linux/macOS

# Binary location
bin/hermes-windows-amd64.exe    # Windows
bin/hermes-linux-amd64          # Linux
bin/hermes-darwin-arm64         # macOS Apple Silicon
```

### Verifying Installation

```bash
hermes --version
hermes --help
```

---

## Quick Start

```bash
# 1. Initialize a new project
hermes init my-project
cd my-project

# 2. Generate PRD from idea (new in v1.1.0)
hermes idea "e-commerce platform with user auth and payments"

# Or create your PRD manually in .hermes/docs/PRD.md

# 3. Parse PRD into tasks
hermes prd .hermes/docs/PRD.md

# 4. View generated tasks
hermes status

# 5. Start autonomous execution
hermes run --auto-branch --auto-commit
```

---

## Project Initialization

The `hermes init` command sets up the project structure.

### Usage

```bash
# Initialize in current directory
hermes init

# Initialize in a new directory
hermes init my-project
```

### What Gets Created

```
my-project/
├── .git/                    # Git repository (if not exists)
├── .gitignore               # Comprehensive gitignore
└── .hermes/                 # Hermes directory (gitignored)
    ├── config.json          # Project configuration
    ├── PROMPT.md            # AI prompt template
    ├── tasks/               # Task files directory
    ├── logs/                # Execution logs
    └── docs/                # Documentation (place PRD here)
```

### Generated .gitignore

The init command creates a comprehensive `.gitignore` including:

- `.hermes/` - All Hermes data
- `node_modules/`, `vendor/`, `venv/` - Dependencies
- `dist/`, `build/`, `bin/` - Build outputs
- `.env`, `.env.local` - Environment files
- `.idea/`, `.vscode/` - IDE files
- `*.log`, `logs/` - Log files

### Initial Commit

After initialization, Hermes creates an initial commit:

```
chore: Initialize project with Hermes
```

---

## Idea to PRD Generation

Generate a detailed PRD from a simple idea or description. This feature was added in v1.1.0.

### Usage

```bash
hermes idea <description> [flags]
```

### Flags

| Flag            | Short | Default               | Description                            |
|-----------------|-------|-----------------------|----------------------------------------|
| `--output`      | `-o`  | `.hermes/docs/PRD.md` | Output file path                       |
| `--dry-run`     |       | false                 | Preview without writing file           |
| `--interactive` | `-i`  | false                 | Interactive mode (additional questions)|
| `--language`    | `-l`  | `en`                  | PRD language (en/tr)                   |
| `--timeout`     |       | 600                   | AI timeout in seconds                  |
| `--debug`       |       | false                 | Enable debug output                    |

### Examples

```bash
# Generate PRD in English
hermes idea "e-commerce website with shopping cart"

# Generate PRD in Turkish
hermes idea "blog platformu" --language tr

# Interactive mode with additional questions
hermes idea "CRM system" --interactive

# Preview without saving
hermes idea "task manager" --dry-run

# Custom output path
hermes idea "chat application" -o docs/chat-prd.md
```

### Interactive Mode

When using `--interactive`, Hermes asks additional questions:

- Target audience
- Preferred technology stack
- Expected scale (small/medium/large/enterprise)
- Expected timeline
- Must-have features

These answers help generate a more tailored PRD.

### Output

The generated PRD includes:

1. Project Overview (name, description, target audience, objectives)
2. Features (with user stories and acceptance criteria)
3. Technical Requirements (stack, architecture, integrations)
4. Non-Functional Requirements (security, scalability, accessibility)
5. Success Metrics (KPIs, success criteria)
6. Timeline & Milestones

---

## PRD Parsing

Convert a Product Requirements Document into structured task files.

### Usage

```bash
hermes prd <prd-file> [flags]
```

### Flags

| Flag             | Default | Description                      |
|------------------|---------|----------------------------------|
| `--dry-run`      | false   | Preview output without writing   |
| `--timeout`      | 1200    | Timeout in seconds               |
| `--max-retries`  | 10      | Maximum retry attempts           |
| `--debug`        | false   | Enable debug output              |

### Examples

```bash
# Parse PRD
hermes prd .hermes/docs/PRD.md

# Preview without creating files
hermes prd requirements.md --dry-run

# With longer timeout for large PRDs
hermes prd large-prd.md --timeout 1800
```

### PRD Format Recommendations

Your PRD should include:

- Project overview
- Feature descriptions
- User stories
- Technical requirements
- Acceptance criteria

### Generated Task Files

Task files are created in `.hermes/tasks/` with the format:

```
.hermes/tasks/
├── 001-user-authentication.md
├── 002-product-catalog.md
├── 003-shopping-cart.md
└── 004-checkout.md
```

---

## Adding Features

Add individual features without re-parsing the entire PRD.

### Usage

```bash
hermes add <feature-description> [flags]
```

### Flags

| Flag        | Default | Description                      |
|-------------|---------|----------------------------------|
| `--dry-run` | false   | Preview output without writing   |
| `--timeout` | 300     | Timeout in seconds               |
| `--debug`   | false   | Enable debug output              |

### Examples

```bash
# Add a new feature
hermes add "user authentication with JWT"

# Add with preview
hermes add "dark mode toggle" --dry-run

# Add complex feature
hermes add "real-time notifications using WebSockets"
```

### ID Continuity

Hermes automatically assigns the next available IDs:

- Feature IDs: F001, F002, F003...
- Task IDs: T001, T002... (continues across all features)

---

## Task Execution

Execute tasks using AI with automatic progress tracking.

### Usage

```bash
hermes run [flags]
```

### Flags

| Flag            | Default     | Description                         |
|-----------------|-------------|-------------------------------------|
| `--ai`          | auto        | AI provider (claude/droid/gemini)   |
| `--auto-branch` | from config | Create feature branches             |
| `--auto-commit` | from config | Commit on task completion           |
| `--autonomous`  | true        | Run without pausing                 |
| `--timeout`     | from config | AI timeout in seconds               |
| `--debug`       | false       | Enable debug output                 |

### Examples

```bash
# Basic run with auto-detect AI
hermes run

# Full automation
hermes run --auto-branch --auto-commit

# Use specific AI provider
hermes run --ai gemini

# Interactive mode (pause between tasks)
hermes run --autonomous=false

# With custom timeout
hermes run --timeout 600
```

### AI Provider Priority

When using `--ai auto` (default), providers are tried in order:

1. Claude (`claude` command)
2. Droid (`droid` command)
3. Gemini (`gemini` command)

### Execution Flow

1. Load next incomplete task
2. Set task status to `IN_PROGRESS`
3. Create feature branch (if `--auto-branch`)
4. Inject task into AI prompt
5. Execute AI with task instructions
6. Analyze response for completion
7. Set task status to `COMPLETED`
8. Commit changes (if `--auto-commit`)
9. Repeat until all tasks complete

### Branch Naming

Feature branches follow the format:

```
feature/F001-authentication
feature/F002-product-catalog
```

### Commit Format

Commits use conventional format:

```
feat(T001): Database Schema completed
feat(T002): User Registration API completed
```

### Stopping Execution

Press `Ctrl+C` to gracefully stop execution.

---

## Status and Monitoring

### Task Status

View all tasks and their status:

```bash
hermes status
```

#### Filtering

```bash
# Filter by status
hermes status --filter IN_PROGRESS
hermes status --filter COMPLETED
hermes status --filter NOT_STARTED
hermes status --filter BLOCKED

# Filter by priority
hermes status --priority P1
hermes status --priority P2
```

#### Output

```
+--------+---------------------------+--------------+----------+--------+
| ID     | Name                      | Status       | Priority | Feature|
+--------+---------------------------+--------------+----------+--------+
| T001   | Database Schema           | COMPLETED    | P1       | F001   |
| T002   | User Registration API     | IN_PROGRESS  | P1       | F001   |
| T003   | Email Verification        | NOT_STARTED  | P1       | F001   |
+--------+---------------------------+--------------+----------+--------+

Task Progress
----------------------------------------
[##########--------------------] 33.3%

Total:       3
Completed:   1
In Progress: 1
Not Started: 1
Blocked:     0
----------------------------------------
```

### Task Details

View detailed information about a specific task:

```bash
# Using full ID
hermes task T001

# Using short ID
hermes task 1
hermes task 001
```

#### Output

```
Task: T001
--------------------------------------------------
Name:     Database Schema
Status:   COMPLETED
Priority: P1
Feature:  F001

Files to Touch:
  - db/migrations/001_users.sql
  - db/schema.go

Dependencies:
  - None

Success Criteria:
  - Migration runs successfully
  - Rollback works correctly
  - Schema matches design document
```

### Viewing Logs

View execution logs:

```bash
# Show last 50 lines
hermes log

# Show last N lines
hermes log -n 100

# Follow log in real-time
hermes log -f

# Filter by level
hermes log --level ERROR
hermes log --level WARN
```

#### Log Levels

| Level   | Color  | Description         |
|---------|--------|---------------------|
| ERROR   | Red    | Error messages      |
| WARN    | Yellow | Warning messages    |
| SUCCESS | Green  | Success messages    |
| INFO    | White  | Informational       |
| DEBUG   | Gray   | Debug information   |

---

## Interactive TUI

Launch the interactive terminal user interface:

```bash
hermes tui
```

### Screens

| Key | Screen    | Description                              |
|-----|-----------|------------------------------------------|
| 1   | Dashboard | Progress overview and circuit breaker    |
| 2   | Tasks     | Task list with filtering                 |
| 3   | Logs      | Real-time log viewer                     |
| ?   | Help      | Keyboard shortcuts reference             |

### Dashboard Screen

Displays:

- Overall progress bar
- Circuit breaker status
- Current/next task
- Task statistics

### Tasks Screen

Features:

- Scrollable task list
- Status filtering
- Task detail view

#### Task Filters

| Key | Filter      |
|-----|-------------|
| a   | All tasks   |
| c   | Completed   |
| p   | In Progress |
| n   | Not Started |
| b   | Blocked     |

### Logs Screen

Features:

- Scrollable log viewer
- Color-coded log levels
- Auto-scroll toggle

### Keyboard Shortcuts

| Key       | Action                     |
|-----------|----------------------------|
| 1/2/3/?   | Switch screens             |
| r         | Start task execution       |
| s         | Stop execution             |
| Shift+R   | Manual refresh             |
| Enter     | Open task detail           |
| Esc       | Back to previous screen    |
| j/k       | Scroll down/up             |
| g         | Go to top                  |
| Shift+G   | Go to bottom               |
| f         | Toggle auto-scroll (logs)  |
| q         | Quit                       |

---

## Configuration

### Configuration Files

Hermes uses layered configuration:

1. CLI flags (highest priority)
2. Project config: `.hermes/config.json`
3. Global config: `~/.hermes/config.json`
4. Default values (lowest priority)

### Configuration Options

```json
{
  "ai": {
    "planning": "claude",
    "coding": "claude",
    "timeout": 300,
    "prdTimeout": 1200,
    "maxRetries": 10,
    "streamOutput": true
  },
  "taskMode": {
    "autoBranch": true,
    "autoCommit": true,
    "autonomous": true,
    "maxConsecutiveErrors": 5
  },
  "loop": {
    "maxCallsPerHour": 100,
    "timeoutMinutes": 15,
    "errorDelay": 10
  },
  "paths": {
    "hermesDir": ".hermes",
    "tasksDir": ".hermes/tasks",
    "logsDir": ".hermes/logs",
    "docsDir": ".hermes/docs"
  }
}
```

### AI Configuration

| Option         | Type   | Default  | Description                     |
|----------------|--------|----------|---------------------------------|
| `planning`     | string | "claude" | AI for PRD parsing              |
| `coding`       | string | "claude" | AI for task execution           |
| `timeout`      | int    | 300      | Task execution timeout (sec)    |
| `prdTimeout`   | int    | 1200     | PRD parsing timeout (sec)       |
| `maxRetries`   | int    | 10       | Maximum retry attempts          |
| `streamOutput` | bool   | true     | Stream AI output                |

### Task Mode Configuration

| Option                 | Type | Default | Description                    |
|------------------------|------|---------|--------------------------------|
| `autoBranch`           | bool | true    | Create feature branches        |
| `autoCommit`           | bool | true    | Commit on completion           |
| `autonomous`           | bool | true    | Run without pausing            |
| `maxConsecutiveErrors` | int  | 5       | Stop after N consecutive errors|

### Loop Configuration

| Option           | Type | Default | Description               |
|------------------|------|---------|---------------------------|
| `maxCallsPerHour`| int  | 100     | Rate limit                |
| `timeoutMinutes` | int  | 15      | Loop timeout              |
| `errorDelay`     | int  | 10      | Delay after error (sec)   |

---

## Circuit Breaker

The circuit breaker prevents runaway execution when no progress is detected.

### States

| State     | Description                                     |
|-----------|-------------------------------------------------|
| CLOSED    | Normal operation, execution allowed             |
| HALF_OPEN | Monitoring mode, 2 loops without progress       |
| OPEN      | Execution halted, requires manual reset         |

### Thresholds

- **HALF_OPEN**: Triggered after 2 consecutive loops without progress
- **OPEN**: Triggered after 3 consecutive loops without progress

### Progress Detection

Hermes analyzes AI responses for:

- File modifications
- Code changes
- Completion signals
- Error patterns

### Viewing Status

Circuit breaker status appears in:

```bash
hermes status    # Shows if not CLOSED
hermes tui       # Dashboard screen
```

### Resetting

When the circuit breaker opens, reset it with:

```bash
hermes reset
```

Output:

```
Current state: OPEN
Reason: No progress for 3 loops, opening circuit

Circuit breaker reset successfully.
You can now run 'hermes run' to continue.
```

### Automatic Recovery

The circuit breaker automatically recovers when progress is detected:

- State returns to CLOSED
- No manual intervention required

---

## Troubleshooting

### Common Issues

#### No AI Provider Found

```
Error: no AI provider available (install claude or droid)
```

**Solution**: Install at least one AI CLI:

```bash
npm install -g @anthropic-ai/claude-code
# or
curl -fsSL https://app.factory.ai/cli | sh
# or
npm install -g @google/gemini-cli
```

#### No Tasks Found

```
Error: no tasks found, run 'hermes prd <file>' first
```

**Solution**: Parse a PRD file first:

```bash
hermes prd .hermes/docs/PRD.md
```

#### Circuit Breaker Open

```
Circuit breaker is OPEN - execution halted
```

**Solution**: Reset the circuit breaker:

```bash
hermes reset
```

#### Task Not Found

```
Error: task T001 not found
```

**Solution**: Check available tasks:

```bash
hermes status
```

### Log Analysis

Check logs for detailed error information:

```bash
# View recent errors
hermes log --level ERROR

# Follow logs in real-time
hermes log -f
```

### Debug Mode

Enable debug output for more information:

```bash
hermes run --debug
hermes prd file.md --debug
```

### Getting Help

```bash
hermes --help
hermes run --help
hermes prd --help
```

---

## Task File Format Reference

### Feature Header

```markdown
# Feature N: Feature Name
**Feature ID:** FXXX
**Status:** NOT_STARTED
```

### Task Definition

```markdown
### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** file1.go, file2.go
**Dependencies:** T001, T002
**Success Criteria:**
- Criterion 1
- Criterion 2
- Criterion 3
```

### Status Values

| Status       | Description                    |
|--------------|--------------------------------|
| NOT_STARTED  | Task has not been started      |
| IN_PROGRESS  | Task is currently being worked |
| COMPLETED    | Task is finished               |
| BLOCKED      | Task is blocked by dependency  |

### Priority Values

| Priority | Description     |
|----------|-----------------|
| P1       | Critical        |
| P2       | High            |
| P3       | Medium          |
| P4       | Low             |

---

## Best Practices

### PRD Writing

1. Be specific about requirements
2. Include acceptance criteria
3. Define dependencies clearly
4. Break down large features

### Task Management

1. Start with P1 tasks
2. Keep tasks small and focused
3. Define clear success criteria
4. Specify files to modify

### Execution

1. Use `--auto-branch` for clean history
2. Use `--auto-commit` for incremental saves
3. Monitor with `hermes tui`
4. Check logs regularly

### Recovery

1. Check `hermes status` for progress
2. Use `hermes log -f` to monitor
3. Reset circuit breaker if stuck
4. Review task dependencies
