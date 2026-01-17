# Phase 02: Configuration Management

## Goal

Implement configuration loading with priority: CLI flags > Project config > Global config > Defaults.

## PowerShell Reference

```powershell
# From lib/ConfigManager.ps1
$script:DefaultConfig = @{
    ai = @{
        planning = "claude"
        coding = "droid"
        timeout = 300
        prdTimeout = 1200
        maxRetries = 10
        streamOutput = $true
    }
    taskMode = @{
        autoBranch = $true
        autoCommit = $true
        autonomous = $true
        maxConsecutiveErrors = 5
    }
    loop = @{
        maxCallsPerHour = 100
        timeoutMinutes = 15
    }
    paths = @{
        hermesDir = ".hermes"
        tasksDir = ".hermes\tasks"
        logsDir = ".hermes\logs"
        docsDir = ".hermes\docs"
    }
}
```

## Go Implementation

### 2.1 Config Types

```go
// internal/config/types.go
package config

type Config struct {
    AI       AIConfig       `json:"ai" mapstructure:"ai"`
    TaskMode TaskModeConfig `json:"taskMode" mapstructure:"taskMode"`
    Loop     LoopConfig     `json:"loop" mapstructure:"loop"`
    Paths    PathsConfig    `json:"paths" mapstructure:"paths"`
}

type AIConfig struct {
    Planning     string `json:"planning" mapstructure:"planning"`
    Coding       string `json:"coding" mapstructure:"coding"`
    Timeout      int    `json:"timeout" mapstructure:"timeout"`
    PrdTimeout   int    `json:"prdTimeout" mapstructure:"prdTimeout"`
    MaxRetries   int    `json:"maxRetries" mapstructure:"maxRetries"`
    StreamOutput bool   `json:"streamOutput" mapstructure:"streamOutput"`
}

type TaskModeConfig struct {
    AutoBranch           bool `json:"autoBranch" mapstructure:"autoBranch"`
    AutoCommit           bool `json:"autoCommit" mapstructure:"autoCommit"`
    Autonomous           bool `json:"autonomous" mapstructure:"autonomous"`
    MaxConsecutiveErrors int  `json:"maxConsecutiveErrors" mapstructure:"maxConsecutiveErrors"`
}

type LoopConfig struct {
    MaxCallsPerHour int `json:"maxCallsPerHour" mapstructure:"maxCallsPerHour"`
    TimeoutMinutes  int `json:"timeoutMinutes" mapstructure:"timeoutMinutes"`
}

type PathsConfig struct {
    HermesDir string `json:"hermesDir" mapstructure:"hermesDir"`
    TasksDir  string `json:"tasksDir" mapstructure:"tasksDir"`
    LogsDir   string `json:"logsDir" mapstructure:"logsDir"`
    DocsDir   string `json:"docsDir" mapstructure:"docsDir"`
}
```

### 2.2 Default Configuration

```go
// internal/config/defaults.go
package config

func DefaultConfig() *Config {
    return &Config{
        AI: AIConfig{
            Planning:     "claude",
            Coding:       "droid",
            Timeout:      300,
            PrdTimeout:   1200,
            MaxRetries:   10,
            StreamOutput: true,
        },
        TaskMode: TaskModeConfig{
            AutoBranch:           true,
            AutoCommit:           true,
            Autonomous:           true,
            MaxConsecutiveErrors: 5,
        },
        Loop: LoopConfig{
            MaxCallsPerHour: 100,
            TimeoutMinutes:  15,
        },
        Paths: PathsConfig{
            HermesDir: ".hermes",
            TasksDir:  ".hermes/tasks",
            LogsDir:   ".hermes/logs",
            DocsDir:   ".hermes/docs",
        },
    }
}
```

### 2.3 Config Loading

```go
// internal/config/config.go
package config

import (
    "os"
    "path/filepath"
    "github.com/spf13/viper"
)

func Load(basePath string) (*Config, error) {
    cfg := DefaultConfig()
    
    // Global config: ~/.hermes/config.json
    homeDir, _ := os.UserHomeDir()
    globalPath := filepath.Join(homeDir, ".hermes", "config.json")
    loadFile(globalPath, cfg)
    
    // Project config: .hermes/config.json
    projectPath := filepath.Join(basePath, ".hermes", "config.json")
    loadFile(projectPath, cfg)
    
    return cfg, nil
}

func loadFile(path string, cfg *Config) {
    v := viper.New()
    v.SetConfigFile(path)
    if err := v.ReadInConfig(); err == nil {
        v.Unmarshal(cfg)
    }
}

func GetAIForTask(taskType string, override string, cfg *Config) string {
    if override != "" && override != "auto" {
        return override
    }
    if taskType == "planning" {
        return cfg.AI.Planning
    }
    return cfg.AI.Coding
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/config/types.go` | Config struct definitions |
| `internal/config/defaults.go` | Default values |
| `internal/config/config.go` | Loading and merging logic |
| `internal/config/config_test.go` | Unit tests |

## Acceptance Criteria

- [ ] Default config returns expected values
- [ ] Global config overrides defaults
- [ ] Project config overrides global
- [ ] GetAIForTask returns correct provider
- [ ] All tests pass
