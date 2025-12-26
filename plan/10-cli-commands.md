# Phase 10: CLI Commands

## Goal

Implement single `hermes` binary with subcommands using Cobra framework.

## Commands Overview

| Command | Description |
|---------|-------------|
| `hermes run` | Run task execution loop |
| `hermes prd <file>` | Parse PRD to task files |
| `hermes add <feature>` | Add single feature |
| `hermes init [name]` | Initialize project |
| `hermes status` | Show task status |
| `hermes tui` | Launch interactive TUI |
| `hermes reset` | Reset circuit breaker |

## Go Implementation

### 10.1 Main Entry Point

```go
// cmd/hermes/main.go
package main

import (
    "os"
    
    "github.com/spf13/cobra"
    "hermes/internal/cmd"
)

var version = "dev"

func main() {
    rootCmd := &cobra.Command{
        Use:     "hermes",
        Short:   "Hermes Autonomous Agent",
        Long:    "Autonomous AI development loop for task-driven development",
        Version: version,
    }
    
    // Add subcommands
    rootCmd.AddCommand(cmd.NewRunCmd())
    rootCmd.AddCommand(cmd.NewPrdCmd())
    rootCmd.AddCommand(cmd.NewAddCmd())
    rootCmd.AddCommand(cmd.NewInitCmd())
    rootCmd.AddCommand(cmd.NewStatusCmd())
    rootCmd.AddCommand(cmd.NewTuiCmd())
    rootCmd.AddCommand(cmd.NewResetCmd())
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### 10.2 Run Command

```go
// internal/cmd/run.go
package cmd

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/spf13/cobra"
    "hermes/internal/ai"
    "hermes/internal/analyzer"
    "hermes/internal/circuit"
    "hermes/internal/config"
    "hermes/internal/git"
    "hermes/internal/prompt"
    "hermes/internal/task"
    "hermes/internal/ui"
)

type runOptions struct {
    autoBranch bool
    autoCommit bool
    autonomous bool
    aiProvider string
    timeout    int
    debug      bool
}

func NewRunCmd() *cobra.Command {
    opts := &runOptions{}
    
    cmd := &cobra.Command{
        Use:   "run",
        Short: "Run task execution loop",
        Long:  "Execute tasks from task files using AI providers",
        Example: `  hermes run
  hermes run --auto-branch --auto-commit
  hermes run --ai claude --autonomous`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runExecute(opts)
        },
    }
    
    cmd.Flags().BoolVar(&opts.autoBranch, "auto-branch", false, "Create feature branches automatically")
    cmd.Flags().BoolVar(&opts.autoCommit, "auto-commit", false, "Commit on task completion")
    cmd.Flags().BoolVar(&opts.autonomous, "autonomous", false, "Run without pausing between tasks")
    cmd.Flags().StringVar(&opts.aiProvider, "ai", "auto", "AI provider (claude, auto)")
    cmd.Flags().IntVar(&opts.timeout, "timeout", 300, "AI timeout in seconds")
    cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")
    
    return cmd
}

func runExecute(opts *runOptions) error {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle Ctrl+C
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        fmt.Println("\nReceived interrupt, shutting down...")
        cancel()
    }()
    
    // Load config
    cfg, err := config.Load(".")
    if err != nil {
        return err
    }
    
    // Initialize logger
    logger, err := ui.NewLogger(".", opts.debug)
    if err != nil {
        return err
    }
    defer logger.Close()
    
    ui.PrintHeader("Hermes Autonomous Agent")
    
    // Initialize components
    reader := task.NewReader(".")
    breaker := circuit.New(".")
    gitOps := git.New(".")
    injector := prompt.NewInjector(".")
    respAnalyzer := analyzer.NewResponseAnalyzer()
    
    // Check for tasks
    if !reader.HasTasks() {
        return fmt.Errorf("no tasks found, run 'hermes prd <file>' first")
    }
    
    // Get AI provider
    provider := ai.GetProvider(config.GetAIForTask("coding", opts.aiProvider, cfg))
    if provider == nil {
        return fmt.Errorf("no AI provider available")
    }
    
    logger.Info("Using AI provider: %s", provider.Name())
    
    loopNumber := 0
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        loopNumber++
        logger.Info("Loop %d starting...", loopNumber)
        
        // Check circuit breaker
        canExecute, err := breaker.CanExecute()
        if err != nil {
            return err
        }
        if !canExecute {
            breaker.PrintHaltMessage()
            return nil
        }
        
        // Get next task
        nextTask, err := reader.GetNextTask()
        if err != nil {
            return err
        }
        if nextTask == nil {
            logger.Success("All tasks completed!")
            return nil
        }
        
        logger.Info("Working on task: %s - %s", nextTask.ID, nextTask.Name)
        
        // Handle branching
        if opts.autoBranch && gitOps.IsRepository() {
            feature, _ := reader.GetFeatureByID(nextTask.FeatureID)
            if feature != nil {
                gitOps.EnsureOnFeatureBranch(feature.ID, feature.Name)
            }
        }
        
        // Inject task into prompt
        injector.AddTask(nextTask)
        promptContent, _ := injector.Read()
        
        // Execute AI using claude-code-sdk-go
        executor := ai.NewTaskExecutor(provider, ".")
        result, err := executor.ExecuteTask(ctx, nextTask, promptContent)
        
        if err != nil {
            logger.Error("AI execution failed: %v", err)
            breaker.AddLoopResult(false, true, loopNumber)
            continue
        }
        
        // Analyze response
        analysis := respAnalyzer.Analyze(result.Output)
        
        // Update circuit breaker
        breaker.AddLoopResult(analysis.HasProgress, false, loopNumber)
        
        // Update task status if complete
        if analysis.IsComplete {
            statusUpdater := task.NewStatusUpdater(".")
            statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusCompleted)
            
            // Auto-commit
            if opts.autoCommit && gitOps.HasUncommittedChanges() {
                gitOps.StageAll()
                gitOps.CommitTask(nextTask.ID, nextTask.Name)
            }
            
            logger.Success("Task %s completed", nextTask.ID)
        }
        
        // Pause between tasks if not autonomous
        if !opts.autonomous && analysis.IsComplete {
            logger.Info("Press Enter to continue or Ctrl+C to stop...")
            fmt.Scanln()
        }
    }
}
```

### 10.3 PRD Command

```go
// internal/cmd/prd.go
package cmd

import (
    "context"
    "fmt"
    "os"
    "time"
    
    "github.com/spf13/cobra"
    "hermes/internal/ai"
    "hermes/internal/config"
    "hermes/internal/ui"
)

type prdOptions struct {
    aiProvider string
    dryRun     bool
    timeout    int
    maxRetries int
    debug      bool
}

func NewPrdCmd() *cobra.Command {
    opts := &prdOptions{}
    
    cmd := &cobra.Command{
        Use:   "prd <file>",
        Short: "Parse PRD to task files",
        Long:  "Parse a Product Requirements Document and generate task files",
        Example: `  hermes prd docs/PRD.md
  hermes prd requirements.md --dry-run
  hermes prd spec.md --ai claude --timeout 1200`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return prdExecute(args[0], opts)
        },
    }
    
    cmd.Flags().StringVar(&opts.aiProvider, "ai", "auto", "AI provider")
    cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show output without writing files")
    cmd.Flags().IntVar(&opts.timeout, "timeout", 1200, "Timeout in seconds")
    cmd.Flags().IntVar(&opts.maxRetries, "max-retries", 10, "Max retry attempts")
    cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")
    
    return cmd
}

func prdExecute(prdFile string, opts *prdOptions) error {
    ctx := context.Background()
    
    ui.PrintHeader("Hermes PRD Parser")
    
    // Load config
    cfg, err := config.Load(".")
    if err != nil {
        return err
    }
    
    // Read PRD file
    prdContent, err := os.ReadFile(prdFile)
    if err != nil {
        return fmt.Errorf("failed to read PRD: %w", err)
    }
    
    fmt.Printf("PRD file: %s (%d chars)\n", prdFile, len(prdContent))
    
    // Get AI provider
    providerName := config.GetAIForTask("planning", opts.aiProvider, cfg)
    provider := ai.GetProvider(providerName)
    if provider == nil {
        return fmt.Errorf("AI provider not available: %s", providerName)
    }
    
    fmt.Printf("Using AI: %s\n\n", provider.Name())
    
    // Build prompt
    prompt := buildPrdPrompt(string(prdContent))
    
    // Execute with retry
    result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
        Prompt:       prompt,
        Timeout:      opts.timeout,
        StreamOutput: cfg.AI.StreamOutput,
    }, &ai.RetryConfig{
        MaxRetries: opts.maxRetries,
        Delay:      10 * time.Second,
    })
    
    if err != nil {
        return fmt.Errorf("failed to parse PRD: %w", err)
    }
    
    if opts.dryRun {
        fmt.Println("\n--- DRY RUN OUTPUT ---")
        fmt.Println(result.Output)
        return nil
    }
    
    // Write task files
    return writeTaskFiles(result.Output)
}

func buildPrdPrompt(prdContent string) string {
    return fmt.Sprintf(`Parse this PRD into task files.

For each feature, create a markdown file with this format:

# Feature N: Feature Name
**Feature ID:** FXXX
**Status:** NOT_STARTED

### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** file1, file2
**Dependencies:** None
**Success Criteria:**
- Criterion 1
- Criterion 2

---

PRD Content:

%s

Output each file with:
---FILE: XXX-feature-name.md---
<content>
---END_FILE---`, prdContent)
}

func writeTaskFiles(output string) error {
    // Parse FILE markers and write files to .hermes/tasks/
    // Implementation similar to PowerShell Split-AIOutput
    return nil
}
```

### 10.4 Add Command

```go
// internal/cmd/add.go
package cmd

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    "hermes/internal/ai"
    "hermes/internal/analyzer"
    "hermes/internal/config"
    "hermes/internal/ui"
)

type addOptions struct {
    aiProvider string
    dryRun     bool
    timeout    int
    debug      bool
}

func NewAddCmd() *cobra.Command {
    opts := &addOptions{}
    
    cmd := &cobra.Command{
        Use:   "add <feature-description>",
        Short: "Add a single feature",
        Long:  "Add a new feature to the task plan using AI",
        Example: `  hermes add "user authentication with JWT"
  hermes add "dark mode toggle" --dry-run
  hermes add "API rate limiting" --ai claude`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return addExecute(args[0], opts)
        },
    }
    
    cmd.Flags().StringVar(&opts.aiProvider, "ai", "auto", "AI provider")
    cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show output without writing")
    cmd.Flags().IntVar(&opts.timeout, "timeout", 300, "Timeout in seconds")
    cmd.Flags().BoolVar(&opts.debug, "debug", false, "Enable debug output")
    
    return cmd
}

func addExecute(featureDesc string, opts *addOptions) error {
    ctx := context.Background()
    
    ui.PrintHeader("Hermes Feature Add")
    
    fmt.Printf("Adding feature: %s\n\n", featureDesc)
    
    // Load config
    cfg, err := config.Load(".")
    if err != nil {
        return err
    }
    
    // Get next IDs
    featureAnalyzer := analyzer.NewFeatureAnalyzer(".")
    nextFeatureID, nextTaskID, err := featureAnalyzer.GetNextIDs()
    if err != nil {
        return err
    }
    
    fmt.Printf("Next Feature ID: F%03d\n", nextFeatureID)
    fmt.Printf("Next Task ID: T%03d\n\n", nextTaskID)
    
    // Get AI provider
    providerName := config.GetAIForTask("planning", opts.aiProvider, cfg)
    provider := ai.GetProvider(providerName)
    if provider == nil {
        return fmt.Errorf("AI provider not available: %s", providerName)
    }
    
    // Build prompt
    prompt := buildAddPrompt(featureDesc, nextFeatureID, nextTaskID)
    
    // Execute
    result, err := provider.Execute(ctx, &ai.ExecuteOptions{
        Prompt:       prompt,
        Timeout:      opts.timeout,
        StreamOutput: cfg.AI.StreamOutput,
    })
    
    if err != nil {
        return fmt.Errorf("failed to add feature: %w", err)
    }
    
    if opts.dryRun {
        fmt.Println("\n--- DRY RUN OUTPUT ---")
        fmt.Println(result.Output)
        return nil
    }
    
    // Write task file
    return writeFeatureFile(result.Output, nextFeatureID)
}

func buildAddPrompt(desc string, featureID, taskID int) string {
    return fmt.Sprintf(`Create a feature file for: %s

Use Feature ID: F%03d
Start Task IDs from: T%03d

Format:
# Feature N: Feature Name
**Feature ID:** FXXX
**Status:** NOT_STARTED

### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** file1, file2
**Dependencies:** None
**Success Criteria:**
- Criterion 1`, desc, featureID, taskID)
}

func writeFeatureFile(output string, featureID int) error {
    // Write to .hermes/tasks/XXX-feature-name.md
    return nil
}
```

### 10.5 Init Command

```go
// internal/cmd/init.go
package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    "hermes/internal/config"
    "hermes/internal/prompt"
)

func NewInitCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "init [project-name]",
        Short: "Initialize Hermes project",
        Long:  "Create .hermes directory structure and default configuration",
        Example: `  hermes init
  hermes init my-project`,
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            projectPath := "."
            if len(args) > 0 {
                projectPath = args[0]
            }
            return initExecute(projectPath)
        },
    }
    
    return cmd
}

func initExecute(projectPath string) error {
    // Create project directory if needed
    if projectPath != "." {
        if err := os.MkdirAll(projectPath, 0755); err != nil {
            return err
        }
    }
    
    fmt.Printf("Initializing Hermes in: %s\n\n", projectPath)
    
    // Create directory structure
    dirs := []string{
        ".hermes",
        ".hermes/tasks",
        ".hermes/logs",
        ".hermes/docs",
    }
    
    for _, dir := range dirs {
        path := filepath.Join(projectPath, dir)
        if err := os.MkdirAll(path, 0755); err != nil {
            return err
        }
        fmt.Printf("  Created: %s/\n", dir)
    }
    
    // Create default config
    configPath := filepath.Join(projectPath, ".hermes", "config.json")
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        cfg := config.DefaultConfig()
        if err := config.WriteConfig(configPath, cfg); err != nil {
            return err
        }
        fmt.Println("  Created: .hermes/config.json")
    }
    
    // Create default PROMPT.md
    injector := prompt.NewInjector(projectPath)
    if err := injector.CreateDefault(); err != nil {
        return err
    }
    fmt.Println("  Created: .hermes/PROMPT.md")
    
    // Update .gitignore
    appendToGitignore(filepath.Join(projectPath, ".gitignore"))
    fmt.Println("  Updated: .gitignore")
    
    fmt.Println("\nHermes initialized successfully!")
    fmt.Println("\nNext steps:")
    fmt.Println("  1. Add your PRD to .hermes/docs/PRD.md")
    fmt.Println("  2. Run: hermes prd .hermes/docs/PRD.md")
    fmt.Println("  3. Run: hermes run --auto-branch --auto-commit")
    
    return nil
}

func appendToGitignore(path string) {
    entries := []string{
        "\n# Hermes",
        ".hermes/logs/",
        ".hermes/circuit-*.json",
    }
    
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return
    }
    defer f.Close()
    
    for _, entry := range entries {
        f.WriteString(entry + "\n")
    }
}
```

### 10.6 Status Command

```go
// internal/cmd/status.go
package cmd

import (
    "fmt"
    
    "github.com/spf13/cobra"
    "hermes/internal/circuit"
    "hermes/internal/task"
    "hermes/internal/ui"
)

type statusOptions struct {
    filter   string
    priority string
    json     bool
}

func NewStatusCmd() *cobra.Command {
    opts := &statusOptions{}
    
    cmd := &cobra.Command{
        Use:   "status",
        Short: "Show task status",
        Long:  "Display task progress table and statistics",
        Example: `  hermes status
  hermes status --filter IN_PROGRESS
  hermes status --priority P1`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return statusExecute(opts)
        },
    }
    
    cmd.Flags().StringVar(&opts.filter, "filter", "", "Filter by status (NOT_STARTED, IN_PROGRESS, COMPLETED, BLOCKED)")
    cmd.Flags().StringVar(&opts.priority, "priority", "", "Filter by priority (P1, P2, P3, P4)")
    cmd.Flags().BoolVar(&opts.json, "json", false, "Output as JSON")
    
    return cmd
}

func statusExecute(opts *statusOptions) error {
    reader := task.NewReader(".")
    
    if !reader.HasTasks() {
        fmt.Println("No tasks found. Run 'hermes prd <file>' to create tasks.")
        return nil
    }
    
    tasks, err := reader.GetAllTasks()
    if err != nil {
        return err
    }
    
    // Apply filters
    if opts.filter != "" {
        tasks = filterByStatus(tasks, task.Status(opts.filter))
    }
    if opts.priority != "" {
        tasks = filterByPriority(tasks, task.Priority(opts.priority))
    }
    
    // Display table
    fmt.Println(ui.FormatTaskTable(tasks))
    
    // Show progress
    progress, err := reader.GetProgress()
    if err != nil {
        return err
    }
    ui.PrintProgress(progress)
    
    // Show circuit breaker status
    breaker := circuit.New(".")
    state, _ := breaker.GetState()
    if state != nil && state.State != circuit.StateClosed {
        fmt.Println()
        breaker.PrintStatus()
    }
    
    return nil
}

func filterByStatus(tasks []task.Task, status task.Status) []task.Task {
    var filtered []task.Task
    for _, t := range tasks {
        if t.Status == status {
            filtered = append(filtered, t)
        }
    }
    return filtered
}

func filterByPriority(tasks []task.Task, priority task.Priority) []task.Task {
    var filtered []task.Task
    for _, t := range tasks {
        if t.Priority == priority {
            filtered = append(filtered, t)
        }
    }
    return filtered
}
```

### 10.7 TUI Command

```go
// internal/cmd/tui.go
package cmd

import (
    "fmt"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
    "hermes/internal/tui"
)

func NewTuiCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "tui",
        Short: "Launch interactive TUI",
        Long:  "Start the interactive terminal user interface",
        RunE: func(cmd *cobra.Command, args []string) error {
            return tuiExecute()
        },
    }
    
    return cmd
}

func tuiExecute() error {
    app, err := tui.NewApp(".")
    if err != nil {
        return fmt.Errorf("failed to initialize TUI: %w", err)
    }
    
    p := tea.NewProgram(app, tea.WithAltScreen())
    
    if _, err := p.Run(); err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }
    
    return nil
}
```

### 10.8 Reset Command

```go
// internal/cmd/reset.go
package cmd

import (
    "fmt"
    
    "github.com/spf13/cobra"
    "hermes/internal/circuit"
)

func NewResetCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "reset",
        Short: "Reset circuit breaker",
        Long:  "Reset the circuit breaker to allow execution to continue",
        RunE: func(cmd *cobra.Command, args []string) error {
            return resetExecute()
        },
    }
    
    return cmd
}

func resetExecute() error {
    breaker := circuit.New(".")
    
    state, err := breaker.GetState()
    if err != nil {
        return err
    }
    
    if state.State == circuit.StateClosed {
        fmt.Println("Circuit breaker is already closed (normal state).")
        return nil
    }
    
    fmt.Printf("Current state: %s\n", state.State)
    fmt.Printf("Reason: %s\n", state.Reason)
    
    if err := breaker.Reset("Manual reset via CLI"); err != nil {
        return err
    }
    
    fmt.Println("\nCircuit breaker reset successfully.")
    fmt.Println("You can now run 'hermes run' to continue.")
    
    return nil
}
```

## Files to Create

| File | Description |
|------|-------------|
| `cmd/hermes/main.go` | Entry point |
| `internal/cmd/run.go` | Run subcommand |
| `internal/cmd/prd.go` | PRD subcommand |
| `internal/cmd/add.go` | Add subcommand |
| `internal/cmd/init.go` | Init subcommand |
| `internal/cmd/status.go` | Status subcommand |
| `internal/cmd/tui.go` | TUI subcommand |
| `internal/cmd/reset.go` | Reset subcommand |

## Command Summary

```
hermes - Hermes Autonomous Agent

Usage:
  hermes [command]

Available Commands:
  run         Run task execution loop
  prd         Parse PRD to task files
  add         Add a single feature
  setup       Initialize Hermes project
  status      Show task status
  tui         Launch interactive TUI
  reset       Reset circuit breaker
  help        Help about any command

Flags:
  -h, --help      help for hermes
  -v, --version   version for hermes

Use "hermes [command] --help" for more information about a command.
```

## Acceptance Criteria

- [ ] Single `hermes` binary with subcommands
- [ ] `hermes run` executes task loop
- [ ] `hermes prd <file>` parses PRD
- [ ] `hermes add <feature>` adds feature
- [ ] `hermes setup` initializes project
- [ ] `hermes status` shows task table
- [ ] `hermes tui` launches TUI
- [ ] `hermes reset` resets circuit breaker
- [ ] All commands have `--help`
- [ ] Ctrl+C cancels gracefully
