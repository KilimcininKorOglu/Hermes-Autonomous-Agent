# Hermes Autonomous Agent

AI-powered autonomous application development system written in Go. Supports Claude, Droid, and Gemini CLIs with task-driven development, automatic branching, and circuit breaker protection.

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
- **Auto Git Operations** - Feature branches and conventional commits
- **Circuit Breaker** - Stagnation detection and recovery
- **Interactive TUI** - Dashboard, task list, and log viewer
- **Resume Support** - Continue from where you left off

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
git clone https://github.com/YourUsername/hermes.git
cd hermes
build.bat          # Windows
make build         # Linux/macOS

# Binary outputs to bin/hermes-{os}-{arch}[.exe]
```

## Quick Start

```bash
# Initialize project
hermes init my-project
cd my-project

# Generate PRD from idea (new in v1.1.0)
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
| `hermes idea <desc>` | Generate PRD from idea (v1.1.0)  |
| `hermes prd <file>`  | Parse PRD to task files          |
| `hermes add <feat>`  | Add single feature               |
| `hermes run`         | Execute task loop                |
| `hermes status`      | Show task status table           |
| `hermes task <id>`   | Show task details                |
| `hermes log`         | View execution logs              |
| `hermes tui`         | Launch interactive TUI           |
| `hermes reset`       | Reset circuit breaker            |

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
**Status:** NOT_STARTED

### T001: Database Schema
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** db/migrations/001_users.sql
**Dependencies:** None
**Success Criteria:**
- Migration runs successfully
- Rollback works
```

## Configuration

`.hermes/config.json`:

```json
{
  "ai": {
    "planning": "claude",
    "coding": "claude",
    "timeout": 300
  },
  "taskMode": {
    "autoBranch": true,
    "autoCommit": true,
    "autonomous": true
  }
}
```

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
# Build
build.bat              # Windows
make build             # Linux/macOS

# Test
build.bat test
make test

# Build all platforms
build.bat build-all
make build-all-platforms
```

## License

MIT License
