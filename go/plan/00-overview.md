# Hermes Go Rewrite - Overview

## Project Goal

Rewrite Hermes Autonomous Agent from PowerShell to Go using the `claude-code-sdk-go` library.

## Current PowerShell Structure

```
hermes-claude-code/
├── hermes_loop.ps1          # Main execution loop
├── hermes-prd.ps1           # PRD parser
├── hermes-add.ps1           # Feature addition
├── hermes_monitor.ps1       # Live dashboard
├── setup.ps1                # Project setup
├── install.ps1              # Global installation
└── lib/
    ├── AIProvider.ps1       # AI CLI abstraction
    ├── ConfigManager.ps1    # Configuration
    ├── TaskReader.ps1       # Task file parsing
    ├── TaskStatusUpdater.ps1
    ├── GitBranchManager.ps1
    ├── CircuitBreaker.ps1
    ├── ResponseAnalyzer.ps1
    ├── PromptInjector.ps1
    ├── TableFormatter.ps1
    ├── FeatureAnalyzer.ps1
    └── Logger.ps1
```

## Target Go Structure

```
hermes-go/
├── cmd/
│   └── hermes/main.go           # Single CLI with subcommands
├── internal/
│   ├── ai/
│   │   ├── provider.go          # AI provider interface
│   │   ├── claude.go            # Claude implementation (uses SDK)
│   │   └── stream.go            # Stream output handling
│   ├── config/
│   │   ├── config.go            # Config struct and loading
│   │   └── defaults.go          # Default configuration
│   ├── task/
│   │   ├── reader.go            # Task file parsing
│   │   ├── writer.go            # Task file writing
│   │   ├── status.go            # Status updates
│   │   └── types.go             # Task/Feature types
│   ├── git/
│   │   ├── branch.go            # Branch management
│   │   └── commit.go            # Commit operations
│   ├── circuit/
│   │   ├── breaker.go           # Circuit breaker logic
│   │   └── state.go             # State management
│   ├── prompt/
│   │   ├── injector.go          # PROMPT.md injection
│   │   └── templates.go         # Prompt templates
│   ├── analyzer/
│   │   ├── response.go          # Response analysis
│   │   └── feature.go           # Feature analysis
│   ├── ui/
│   │   ├── table.go             # Table formatting
│   │   ├── logger.go            # Logging
│   │   └── progress.go          # Progress display
│   └── tui/
│       ├── app.go               # Main TUI app
│       ├── styles.go            # Shared styles
│       ├── layout.go            # Responsive layout
│       ├── dashboard.go         # Dashboard screen
│       ├── tasks.go             # Task list screen
│       ├── task_detail.go       # Task detail screen
│       ├── logs.go              # Log viewer screen
│       ├── settings.go          # Settings screen
│       └── help.go              # Help screen
├── pkg/
│   └── hermes/
│       └── hermes.go            # Public API (if needed)
├── templates/
│   └── PROMPT.md                # Default prompt template
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Key Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/severity1/claude-code-sdk-go` | Claude Code CLI SDK |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration |
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/lipgloss` | TUI styling and layout |
| `github.com/charmbracelet/bubbles` | TUI components (table, viewport, spinner) |
| `github.com/fatih/color` | Colored output (non-TUI mode) |

## Benefits of Go Rewrite

1. **Single Binary**: No runtime dependencies, easy distribution
2. **Cross-Platform**: Native Windows/Linux/Mac builds
3. **Performance**: Faster startup, lower memory usage
4. **Type Safety**: Compile-time error checking
5. **Concurrency**: Native goroutines for parallel operations
6. **Testing**: Built-in testing framework

## Implementation Phases

| Phase | Description | Plan Document |
|-------|-------------|---------------|
| 01 | Project setup and dependencies | [01-project-setup.md](01-project-setup.md) |
| 02 | Configuration management | [02-config-management.md](02-config-management.md) |
| 03 | Task file parsing | [03-task-parsing.md](03-task-parsing.md) |
| 04 | AI provider integration | [04-ai-provider.md](04-ai-provider.md) |
| 05 | Git operations | [05-git-operations.md](05-git-operations.md) |
| 06 | Circuit breaker | [06-circuit-breaker.md](06-circuit-breaker.md) |
| 07 | Prompt injection | [07-prompt-injection.md](07-prompt-injection.md) |
| 08 | Response analysis | [08-response-analysis.md](08-response-analysis.md) |
| 09 | UI and logging | [09-ui-logging.md](09-ui-logging.md) |
| 10 | CLI commands | [10-cli-commands.md](10-cli-commands.md) |
| 11 | Testing | [11-testing.md](11-testing.md) |
| 12 | TUI screens | [12-tui-screens.md](12-tui-screens.md) |

## Estimated Timeline

- **Phase 01-03**: Foundation (config, tasks) - 1-2 days
- **Phase 04-06**: Core logic (AI, git, circuit) - 2-3 days
- **Phase 07-09**: Support (prompt, analysis, UI) - 1-2 days
- **Phase 10-11**: CLI and testing - 2-3 days
- **Phase 12**: TUI screens - 2-3 days

Total: ~2 weeks for full implementation

## Quick Start (After Implementation)

```bash
# Build
cd go && make build

# Install globally
make install

# Initialize project
hermes setup my-project

# Parse PRD
hermes prd docs/PRD.md

# Add feature
hermes add "user authentication"

# Run task mode
hermes run --auto-branch --auto-commit

# Show status
hermes status

# Launch TUI
hermes tui
```
