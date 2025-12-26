# Phase 10: CLI Commands

## Goal

Implement all CLI commands using Cobra framework.

## Commands Overview

| Command | Description |
|---------|-------------|
| `hermes` | Main loop, task mode, status |
| `hermes-prd` | Parse PRD to tasks |
| `hermes-add` | Add single feature |
| `hermes-setup` | Initialize project |

## Go Implementation

### 10.1 Main CLI (hermes)

```go
// cmd/hermes/main.go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/spf13/cobra"
    "hermes/internal/config"
    "hermes/internal/task"
    "hermes/internal/ui"
)

var (
    version = "dev"
    
    // Flags
    taskMode        bool
    autoBranch      bool
    autoCommit      bool
    autonomous      bool
    showStatus      bool
    resetCircuit    bool
    aiProvider      string
    maxCalls        int
    timeoutMinutes  int
    debug           bool
)

func main() {
    rootCmd := &cobra.Command{
        Use:     "hermes",
        Short:   "Hermes Autonomous Agent",
        Version: version,
        RunE:    run,
    }
    
    // Flags
    rootCmd.Flags().BoolVar(&taskMode, "task-mode", false, "Run in task mode")
    rootCmd.Flags().BoolVar(&autoBranch, "auto-branch", false, "Create feature branches")
    rootCmd.Flags().BoolVar(&autoCommit, "auto-commit", false, "Auto-commit on completion")
    rootCmd.Flags().BoolVar(&autonomous, "autonomous", false, "Run without pausing")
    rootCmd.Flags().BoolVar(&showStatus, "status", false, "Show task status")
    rootCmd.Flags().BoolVar(&resetCircuit, "reset-circuit", false, "Reset circuit breaker")
    rootCmd.Flags().StringVar(&aiProvider, "ai", "auto", "AI provider (claude, auto)")
    rootCmd.Flags().IntVar(&maxCalls, "calls", 100, "Max API calls per hour")
    rootCmd.Flags().IntVar(&timeoutMinutes, "timeout", 15, "AI timeout in minutes")
    rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func run(cmd *cobra.Command, args []string) error {
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
    logger, err := ui.NewLogger(".", debug)
    if err != nil {
        return err
    }
    defer logger.Close()
    
    // Handle different modes
    if showStatus {
        return showTaskStatus(cfg)
    }
    
    if resetCircuit {
        return resetCircuitBreaker()
    }
    
    // Default: check for tasks and run task mode
    reader := task.NewReader(".")
    if reader.HasTasks() {
        return runTaskMode(ctx, cfg, logger)
    }
    
    // Show help if no tasks
    return cmd.Help()
}

func showTaskStatus(cfg *config.Config) error {
    reader := task.NewReader(".")
    tasks, err := reader.GetAllTasks()
    if err != nil {
        return err
    }
    
    fmt.Println(ui.FormatTaskTable(tasks))
    
    progress, err := reader.GetProgress()
    if err != nil {
        return err
    }
    
    ui.PrintProgress(progress)
    return nil
}

func resetCircuitBreaker() error {
    breaker := circuit.New(".")
    return breaker.Reset("Manual reset via CLI")
}

func runTaskMode(ctx context.Context, cfg *config.Config, logger *ui.Logger) error {
    // Main task loop implementation
    ui.PrintHeader("Hermes Autonomous Agent - Task Mode")
    
    // Initialize components
    reader := task.NewReader(".")
    breaker := circuit.New(".")
    gitOps := git.New(".")
    promptInjector := prompt.NewInjector(".")
    analyzer := analyzer.NewResponseAnalyzer()
    
    // Get AI provider
    provider := ai.GetProvider(config.GetAIForTask("coding", aiProvider, cfg))
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
        if autoBranch && gitOps.IsRepository() {
            feature, _ := reader.GetFeatureByID(nextTask.FeatureID)
            if feature != nil {
                gitOps.EnsureOnFeatureBranch(feature.ID, feature.Name)
            }
        }
        
        // Inject task into prompt
        promptInjector.AddTask(nextTask)
        
        // Execute AI
        result, err := provider.Execute(ctx, &ai.ExecuteOptions{
            Prompt:       promptInjector.Read(),
            Timeout:      cfg.AI.Timeout,
            StreamOutput: cfg.AI.StreamOutput,
        })
        
        if err != nil {
            logger.Error("AI execution failed: %v", err)
            breaker.AddLoopResult(false, true, loopNumber)
            continue
        }
        
        // Analyze response
        analysis := analyzer.Analyze(result.Output)
        
        // Update circuit breaker
        breaker.AddLoopResult(analysis.HasProgress, false, loopNumber)
        
        // Update task status if complete
        if analysis.IsComplete {
            statusUpdater := task.NewStatusUpdater(".")
            statusUpdater.UpdateTaskStatus(nextTask.ID, task.StatusCompleted)
            
            // Auto-commit
            if autoCommit && gitOps.HasUncommittedChanges() {
                gitOps.StageAll()
                gitOps.CommitTask(nextTask.ID, nextTask.Name)
            }
            
            logger.Success("Task %s completed", nextTask.ID)
        }
        
        // Pause between tasks if not autonomous
        if !autonomous && analysis.IsComplete {
            logger.Info("Press Enter to continue or Ctrl+C to stop...")
            fmt.Scanln()
        }
    }
}
```

### 10.2 PRD Parser (hermes-prd)

```go
// cmd/hermes-prd/main.go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "hermes/internal/ai"
    "hermes/internal/config"
    "hermes/internal/ui"
)

var (
    version    = "dev"
    aiProvider string
    dryRun     bool
    timeout    int
    maxRetries int
    debug      bool
)

func main() {
    rootCmd := &cobra.Command{
        Use:     "hermes-prd <prd-file>",
        Short:   "Parse PRD to task files",
        Version: version,
        Args:    cobra.ExactArgs(1),
        RunE:    run,
    }
    
    rootCmd.Flags().StringVar(&aiProvider, "ai", "auto", "AI provider")
    rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show output without writing")
    rootCmd.Flags().IntVar(&timeout, "timeout", 1200, "Timeout in seconds")
    rootCmd.Flags().IntVar(&maxRetries, "max-retries", 10, "Max retry attempts")
    rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func run(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    prdFile := args[0]
    
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
    
    fmt.Printf("PRD size: %d characters\n", len(prdContent))
    
    // Get AI provider
    providerName := config.GetAIForTask("planning", aiProvider, cfg)
    provider := ai.GetProvider(providerName)
    if provider == nil {
        return fmt.Errorf("AI provider not available: %s", providerName)
    }
    
    fmt.Printf("Using AI: %s\n", provider.Name())
    
    // Build prompt
    prompt := buildPrdPrompt(string(prdContent))
    
    // Execute with retry
    result, err := ai.ExecuteWithRetry(ctx, provider, &ai.ExecuteOptions{
        Prompt:       prompt,
        Timeout:      timeout,
        StreamOutput: cfg.AI.StreamOutput,
    }, &ai.RetryConfig{
        MaxRetries: maxRetries,
        Delay:      10 * time.Second,
    })
    
    if err != nil {
        return fmt.Errorf("failed to parse PRD: %w", err)
    }
    
    // Parse and write output
    if dryRun {
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
    // Parse FILE markers and write files
    // Implementation similar to PowerShell Split-AIOutput
    return nil
}
```

### 10.3 Feature Add (hermes-add)

```go
// cmd/hermes-add/main.go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
)

var (
    version    = "dev"
    aiProvider string
    dryRun     bool
    timeout    int
    debug      bool
)

func main() {
    rootCmd := &cobra.Command{
        Use:     "hermes-add <feature>",
        Short:   "Add a single feature",
        Version: version,
        Args:    cobra.ExactArgs(1),
        RunE:    run,
    }
    
    rootCmd.Flags().StringVar(&aiProvider, "ai", "auto", "AI provider")
    rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show output without writing")
    rootCmd.Flags().IntVar(&timeout, "timeout", 300, "Timeout in seconds")
    rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug output")
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func run(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    featureDesc := args[0]
    
    ui.PrintHeader("Hermes Feature Add")
    
    fmt.Printf("Adding feature: %s\n", featureDesc)
    
    // Similar implementation to hermes-prd
    // but for single feature
    
    return nil
}
```

### 10.4 Project Setup (hermes-setup)

```go
// cmd/hermes-setup/main.go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    "hermes/internal/config"
    "hermes/internal/prompt"
)

var version = "dev"

func main() {
    rootCmd := &cobra.Command{
        Use:     "hermes-setup [project-name]",
        Short:   "Initialize Hermes project",
        Version: version,
        Args:    cobra.MaximumNArgs(1),
        RunE:    run,
    }
    
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func run(cmd *cobra.Command, args []string) error {
    var projectPath string
    
    if len(args) > 0 {
        projectPath = args[0]
        if err := os.MkdirAll(projectPath, 0755); err != nil {
            return err
        }
    } else {
        projectPath = "."
    }
    
    fmt.Printf("Initializing Hermes in: %s\n", projectPath)
    
    // Create .hermes directory structure
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
        fmt.Printf("Created: %s\n", dir)
    }
    
    // Create default config
    configPath := filepath.Join(projectPath, ".hermes", "config.json")
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        cfg := config.DefaultConfig()
        if err := config.WriteConfig(configPath, cfg); err != nil {
            return err
        }
        fmt.Println("Created: .hermes/config.json")
    }
    
    // Create default PROMPT.md
    injector := prompt.NewInjector(projectPath)
    if err := injector.CreateDefault(); err != nil {
        return err
    }
    fmt.Println("Created: .hermes/PROMPT.md")
    
    // Add .hermes to .gitignore
    gitignorePath := filepath.Join(projectPath, ".gitignore")
    appendToGitignore(gitignorePath)
    
    fmt.Println("\nHermes initialized successfully!")
    fmt.Println("\nNext steps:")
    fmt.Println("  1. Add your PRD to .hermes/docs/PRD.md")
    fmt.Println("  2. Run: hermes-prd .hermes/docs/PRD.md")
    fmt.Println("  3. Run: hermes --task-mode --auto-branch --auto-commit")
    
    return nil
}

func appendToGitignore(path string) {
    entries := []string{
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

## Files to Create

| File | Description |
|------|-------------|
| `cmd/hermes/main.go` | Main CLI |
| `cmd/hermes-prd/main.go` | PRD parser |
| `cmd/hermes-add/main.go` | Feature add |
| `cmd/hermes-setup/main.go` | Project setup |

## Acceptance Criteria

- [ ] `hermes --status` shows task table
- [ ] `hermes --task-mode` runs task loop
- [ ] `hermes-prd` parses PRD files
- [ ] `hermes-add` adds features
- [ ] `hermes-setup` initializes projects
- [ ] Ctrl+C cancels gracefully
- [ ] All flags work correctly
