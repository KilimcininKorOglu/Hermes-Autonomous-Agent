# Hermes Autonomous Agent

![Version](https://img.shields.io/badge/version-v2.2.3-blue)
![Status](https://img.shields.io/badge/status-stable-green)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)

AI-powered autonomous application development system written in Go. Supports Claude, Droid, and Gemini CLIs with task-driven development, parallel execution, automatic branching, and circuit breaker protection.

## Documentation

| Document                                  | Description                          |
|-------------------------------------------|--------------------------------------|
| [User Guide](docs/USER-GUIDE.md)          | Complete usage documentation         |
| [Example Usage](docs/example-usage.md)    | Step-by-step walkthrough             |
| [Kullanım Kılavuzu](docs/USER-GUIDE.tr.md)| Türkçe kullanım dokümantasyonu       |
| [Örnek Kullanım](docs/example-usage.tr.md)| Türkçe adım adım rehber              |

## Features

- **Idea to PRD** - Generate detailed PRD from a simple idea description
- **Multi-AI Support** - Claude, Droid, and Gemini CLI providers with auto-detection
- **PRD Parser** - Convert PRD documents to structured task files
- **Task Execution Loop** - Autonomous task execution with progress tracking
- **Parallel Execution** - Multiple AI agents working simultaneously (v2.0)
- **Auto Git Operations** - Feature branches and conventional commits
- **Auto Git Tagging** - Automatic version tags when features complete
- **AI-Assisted Merge** - LLM-powered conflict resolution
- **Circuit Breaker** - Stagnation detection and recovery
- **Interactive TUI** - Dashboard, task list, and log viewer
- **Resume Support** - Continue from where you left off
- **Windows Version Info** - Task Manager shows app name and version (v2.2)

## Requirements

- Go 1.24+
- Git
- One of: Claude CLI, Droid CLI, or Gemini CLI

### AI CLI Installation

```bash
# Claude CLI
npm install -g @anthropic-ai/claude-code

# Droid CLI
curl -fsSL https://app.factory.ai/cli | sh

# Gemini CLI
npm install -g @google/gemini-cli
```

## Installation

```bash
# Clone and build
git clone https://github.com/KilimcininKorOglu/Hermes-Autonomous-Agent.git
cd Hermes-Autonomous-Agent
build.bat          # Windows (builds all platforms by default)
make               # Linux/macOS (builds all platforms by default)

# Binary outputs to bin/hermes-{os}-{arch}[.exe]
```

### Pre-built Releases

Download pre-built binaries from [GitHub Releases](https://github.com/KilimcininKorOglu/Hermes-Autonomous-Agent/releases).

### System-wide Installation

```bash
# Install to system PATH (run after building or downloading)
hermes install

# Uninstall
hermes install --uninstall
```

## Quick Start

```bash
# Initialize project
hermes init my-project
cd my-project

# Generate PRD from idea
hermes idea "e-commerce platform with user auth and payments"

# Or use existing PRD
hermes prd .hermes/docs/PRD.md

# Check status
hermes status

# Run task execution
hermes run --auto-branch --auto-commit
```

## Commands

| Command              | Description                      |
|----------------------|----------------------------------|
| `hermes init [name]` | Initialize project               |
| `hermes idea <desc>` | Generate PRD from idea           |
| `hermes prd <file>`  | Parse PRD to task files          |
| `hermes add <feat>`  | Add single feature               |
| `hermes run`         | Execute task loop                |
| `hermes status`      | Show task status table           |
| `hermes task <id>`   | Show task details                |
| `hermes log`         | View execution logs              |
| `hermes tui`         | Launch interactive TUI           |
| `hermes reset`       | Reset circuit breaker            |
| `hermes update`      | Check and install updates        |
| `hermes install`     | Install to system PATH           |

## Idea Command Options

```bash
hermes idea "e-commerce site"                    # Generate PRD in English
hermes idea "blog platform" --language tr        # Generate PRD in Turkish
hermes idea "CRM system" --interactive           # Ask additional questions
hermes idea "task manager" --dry-run             # Preview without saving
hermes idea "chat app" -o custom-prd.md          # Custom output path
```

## Run Options

```bash
hermes run                          # Auto-detect AI provider
hermes run --ai claude              # Force Claude
hermes run --ai droid               # Force Droid
hermes run --ai gemini              # Force Gemini
hermes run --auto-branch            # Create feature branches
hermes run --auto-commit            # Commit on completion
hermes run --autonomous=false       # Pause between tasks
```

## Parallel Execution (v2.0)

Execute multiple independent tasks simultaneously with AI agents:

```bash
# Enable parallel execution
hermes run --parallel

# Specify number of workers
hermes run --parallel --workers 3

# Preview execution plan (dry run)
hermes run --dry-run

# Combine with other options
hermes run --parallel --workers 5 --auto-commit
```

### Key Features

- **Dependency Graph** - Automatically respects task dependencies
- **Worker Pool** - Multiple AI agents working in parallel
- **Isolated Workspaces** - Git worktree-based isolation per task
- **Conflict Detection** - Detects file-level and semantic conflicts
- **AI-Assisted Merge** - LLM-powered conflict resolution
- **Rollback Support** - Automatic snapshot and recovery

### Configuration

Add to `.hermes/config.json`:

```json
{
  "parallel": {
    "enabled": false,
    "maxWorkers": 3,
    "strategy": "branch-per-task",
    "conflictResolution": "ai-assisted",
    "isolatedWorkspaces": true,
    "mergeStrategy": "sequential",
    "maxCostPerHour": 0,
    "failureStrategy": "continue",
    "maxRetries": 2
  }
}
```

| Option              | Default            | Description                        |
|---------------------|--------------------|------------------------------------|
| enabled             | false              | Enable parallel by default         |
| maxWorkers          | 3                  | Maximum parallel AI agents         |
| strategy            | "branch-per-task"  | Branching strategy                 |
| conflictResolution  | "ai-assisted"      | Conflict resolution method         |
| isolatedWorkspaces  | true               | Use git worktrees                  |
| mergeStrategy       | "sequential"       | How to merge results               |
| maxCostPerHour      | 0                  | Cost limit (0 = unlimited)         |
| failureStrategy     | "continue"         | fail-fast or continue              |
| maxRetries          | 2                  | Retry failed tasks                 |

## AI Providers

| Provider | Priority | Command  |
|----------|----------|----------|
| Claude   | 1        | `claude` |
| Droid    | 2        | `droid`  |
| Gemini   | 3        | `gemini` |

Auto-detection tries providers in priority order.

## Project Structure

```
my-project/
├── .hermes/                # Hermes data (gitignored)
│   ├── config.json         # Configuration
│   ├── PROMPT.md           # AI prompt (auto-managed)
│   ├── tasks/              # Task files
│   ├── logs/               # Execution logs
│   └── docs/               # PRD documents
└── ...                     # Your project files
```

## Task File Format

```markdown
# Feature 1: User Authentication

**Feature ID:** F001
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 1-2 weeks
**Status:** NOT_STARTED

## Overview

User authentication system with secure login, registration, and session management.

## Goals

- Enable secure user authentication
- Support multiple auth methods
- Implement session management

## Tasks

### T001: Database Schema

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Create database schema for users table with proper indexes and constraints.

#### Technical Details

Use PostgreSQL with UUID primary keys. Add indexes on email and username.

#### Files to Touch

- `db/migrations/001_users.sql` (new)
- `db/migrations/002_sessions.sql` (new)

#### Dependencies

- None

#### Success Criteria

- [ ] Migration runs successfully
- [ ] Rollback works
- [ ] Indexes created

## Performance Targets

- Login response time: < 200ms
- Session validation: < 50ms

## Risk Assessment

| Risk              | Probability | Impact | Mitigation           |
|-------------------|-------------|--------|----------------------|
| SQL injection     | Low         | High   | Use parameterized queries |
```

### Task Status Types

| Status       | Description                     |
|--------------|--------------------------------|
| NOT_STARTED  | Task not yet begun             |
| IN_PROGRESS  | Currently being worked on      |
| COMPLETED    | Successfully finished          |
| BLOCKED      | Cannot proceed                 |
| AT_RISK      | May not meet deadline          |
| PAUSED       | Temporarily suspended          |

## Auto Git Tagging

When all tasks in a feature are completed and the feature has a `Target Version`, Hermes automatically creates a git tag.

```bash
# Example: Feature F001 with Target Version v1.0.0 completes
# Hermes creates:
git tag -a v1.0.0 -m "Release v1.0.0: F001 - User Authentication"
```

Tags are only created if:
- All tasks in the feature have `COMPLETED` status
- Feature has `**Target Version:**` field set
- Tag doesn't already exist

## Configuration

`.hermes/config.json`:

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
  },
  "parallel": {
    "enabled": false,
    "maxWorkers": 3,
    "strategy": "branch-per-task",
    "conflictResolution": "ai-assisted",
    "isolatedWorkspaces": true,
    "mergeStrategy": "sequential",
    "maxCostPerHour": 0,
    "failureStrategy": "continue",
    "maxRetries": 2
  }
}
```

### Configuration Options

| Section    | Key                   | Default        | Description                          |
|------------|-----------------------|----------------|--------------------------------------|
| ai         | planning              | "claude"       | AI provider for PRD parsing          |
| ai         | coding                | "claude"       | AI provider for task execution       |
| ai         | timeout               | 300            | Task execution timeout (seconds)     |
| ai         | prdTimeout            | 1200           | PRD parsing timeout (seconds)        |
| ai         | maxRetries            | 10             | Maximum retry attempts               |
| ai         | streamOutput          | true           | Stream AI output to console          |
| taskMode   | autoBranch            | true           | Create feature branches              |
| taskMode   | autoCommit            | true           | Commit on task completion            |
| taskMode   | autonomous            | true           | Run without pausing between tasks    |
| taskMode   | maxConsecutiveErrors  | 5              | Stop after N consecutive errors      |
| loop       | maxCallsPerHour       | 100            | Rate limit for AI calls              |
| loop       | timeoutMinutes        | 15             | Loop timeout in minutes              |
| loop       | errorDelay            | 10             | Delay after error (seconds)          |
| paths      | hermesDir             | ".hermes"      | Hermes data directory                |
| paths      | tasksDir              | ".hermes/tasks"| Task files directory                 |
| paths      | logsDir               | ".hermes/logs" | Log files directory                  |
| paths      | docsDir               | ".hermes/docs" | Documentation directory              |

Priority: CLI flag > Project config > Global config (~/.hermes/config.json) > Defaults

## TUI Keyboard Shortcuts

| Key     | Action                     |
|---------|----------------------------|
| 1/2/3/? | Dashboard/Tasks/Logs/Help  |
| r       | Start execution            |
| s       | Stop execution             |
| Shift+R | Refresh                    |
| j/k     | Scroll                     |
| q       | Quit                       |

## Circuit Breaker

Prevents runaway execution when no progress is detected.

| State     | Meaning                            |
|-----------|------------------------------------|
| CLOSED    | Normal operation                   |
| HALF_OPEN | Monitoring (2 no-progress loops)   |
| OPEN      | Halted (requires `hermes reset`)   |

## Development

```bash
# Build for all platforms (default)
build.bat              # Windows
make                   # Linux/macOS

# Build for current platform only
build.bat build        # Windows
make build             # Linux/macOS

# Test
build.bat test
make test

# Windows builds include version info
# Requires go-winres: go install github.com/tc-hib/go-winres@latest
```

## Changelog

### v2.2.3
- Windows version info in Task Manager/Properties
- Dynamic version from git tags
- GoReleaser integration for releases

### v2.1.x
- Full status support (BLOCKED, AT_RISK, PAUSED)
- Enhanced prompt templates
- PRD filename sanitization

### v2.0.0
- Parallel task execution with worker pools
- Git worktree isolation
- AI-assisted merge conflict resolution
- Dependency graph scheduling

## License

MIT License
