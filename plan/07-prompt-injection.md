# Phase 07: Prompt Injection

## Goal

Implement PROMPT.md task injection and management.

## PowerShell Reference

```powershell
# From lib/PromptInjector.ps1
# - Get-PromptPath
# - Test-PromptExists
# - Add-TaskToPrompt
# - Remove-TaskFromPrompt
# - Get-TaskPromptSection
# - Backup-Prompt / Restore-Prompt
```

## Go Implementation

### 7.1 Prompt Injector

```go
// internal/prompt/injector.go
package prompt

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"
    
    "hermes/internal/task"
)

const (
    TaskSectionStart = "<!-- HERMES_TASK_START -->"
    TaskSectionEnd   = "<!-- HERMES_TASK_END -->"
)

type Injector struct {
    basePath   string
    promptPath string
}

func NewInjector(basePath string) *Injector {
    return &Injector{
        basePath:   basePath,
        promptPath: filepath.Join(basePath, ".hermes", "PROMPT.md"),
    }
}

func (i *Injector) GetPromptPath() string {
    return i.promptPath
}

func (i *Injector) Exists() bool {
    _, err := os.Stat(i.promptPath)
    return err == nil
}

func (i *Injector) Read() (string, error) {
    data, err := os.ReadFile(i.promptPath)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func (i *Injector) Write(content string) error {
    dir := filepath.Dir(i.promptPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    return os.WriteFile(i.promptPath, []byte(content), 0644)
}

func (i *Injector) AddTask(t *task.Task) error {
    content, err := i.Read()
    if err != nil {
        // Create new prompt if doesn't exist
        content = ""
    }
    
    // Remove existing task section
    content = i.removeTaskSection(content)
    
    // Add new task section
    section := i.generateTaskSection(t)
    content = content + "\n\n" + section
    
    return i.Write(content)
}

func (i *Injector) RemoveTask() error {
    content, err := i.Read()
    if err != nil {
        return err
    }
    
    content = i.removeTaskSection(content)
    return i.Write(strings.TrimSpace(content))
}

func (i *Injector) removeTaskSection(content string) string {
    re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(TaskSectionStart) + `.*?` + regexp.QuoteMeta(TaskSectionEnd))
    content = re.ReplaceAllString(content, "")
    return strings.TrimSpace(content)
}

func (i *Injector) generateTaskSection(t *task.Task) string {
    var sb strings.Builder
    
    sb.WriteString(TaskSectionStart + "\n")
    sb.WriteString(fmt.Sprintf("## Current Task: %s\n\n", t.ID))
    sb.WriteString(fmt.Sprintf("**Task:** %s: %s\n\n", t.ID, t.Name))
    
    if len(t.FilesToTouch) > 0 {
        sb.WriteString("**Files to Touch:**\n")
        for _, f := range t.FilesToTouch {
            sb.WriteString(fmt.Sprintf("- %s\n", f))
        }
        sb.WriteString("\n")
    }
    
    if len(t.SuccessCriteria) > 0 {
        sb.WriteString("**Success Criteria:**\n")
        for _, c := range t.SuccessCriteria {
            sb.WriteString(fmt.Sprintf("- %s\n", c))
        }
        sb.WriteString("\n")
    }
    
    sb.WriteString("**Instructions:**\n")
    sb.WriteString("1. Implement the task requirements\n")
    sb.WriteString("2. Run tests to verify\n")
    sb.WriteString("3. Output status block when complete\n\n")
    
    sb.WriteString("**Status Block (output at end):**\n")
    sb.WriteString("```\n")
    sb.WriteString("---HERMES_STATUS---\n")
    sb.WriteString("STATUS: COMPLETE\n")
    sb.WriteString("EXIT_SIGNAL: true\n")
    sb.WriteString("RECOMMENDATION: Move to next task\n")
    sb.WriteString("---END_HERMES_STATUS---\n")
    sb.WriteString("```\n")
    
    sb.WriteString(TaskSectionEnd)
    
    return sb.String()
}

func (i *Injector) GetCurrentTaskID() (string, error) {
    content, err := i.Read()
    if err != nil {
        return "", err
    }
    
    re := regexp.MustCompile(`## Current Task: (T\d+)`)
    matches := re.FindStringSubmatch(content)
    if len(matches) > 1 {
        return matches[1], nil
    }
    
    return "", nil
}

func (i *Injector) HasTaskSection() (bool, error) {
    content, err := i.Read()
    if err != nil {
        return false, err
    }
    
    return strings.Contains(content, TaskSectionStart), nil
}
```

### 7.2 Backup and Restore

```go
// internal/prompt/backup.go
package prompt

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "time"
)

func (i *Injector) Backup() (string, error) {
    content, err := i.Read()
    if err != nil {
        return "", err
    }
    
    timestamp := time.Now().Format("20060102_150405")
    backupName := fmt.Sprintf("prompt_backup_%s.md", timestamp)
    backupPath := filepath.Join(filepath.Dir(i.promptPath), backupName)
    
    if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
        return "", err
    }
    
    return backupPath, nil
}

func (i *Injector) Restore(backupPath string) error {
    content, err := os.ReadFile(backupPath)
    if err != nil {
        return err
    }
    
    return i.Write(string(content))
}

func (i *Injector) GetLatestBackup() (string, error) {
    dir := filepath.Dir(i.promptPath)
    pattern := filepath.Join(dir, "prompt_backup_*.md")
    
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return "", err
    }
    
    if len(matches) == 0 {
        return "", nil
    }
    
    // Sort by name (which includes timestamp) descending
    sort.Sort(sort.Reverse(sort.StringSlice(matches)))
    
    return matches[0], nil
}

func (i *Injector) RestoreLatest() error {
    backup, err := i.GetLatestBackup()
    if err != nil {
        return err
    }
    
    if backup == "" {
        return fmt.Errorf("no backup found")
    }
    
    return i.Restore(backup)
}
```

### 7.3 Templates

```go
// internal/prompt/templates.go
package prompt

const DefaultPromptTemplate = `# Project Instructions

## Overview

This project uses Hermes for AI-powered autonomous application development.

## Guidelines

1. Follow existing code patterns
2. Write tests for new functionality
3. Use conventional commits
4. Keep changes focused and atomic

## Status Reporting

At the end of each response, output:

` + "```" + `
---HERMES_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
RECOMMENDATION: <next action>
---END_HERMES_STATUS---
` + "```" + `
`

func (i *Injector) CreateDefault() error {
    if i.Exists() {
        return nil
    }
    
    return i.Write(DefaultPromptTemplate)
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/prompt/injector.go` | Task injection |
| `internal/prompt/backup.go` | Backup/restore |
| `internal/prompt/templates.go` | Default templates |
| `internal/prompt/injector_test.go` | Tests |

## Acceptance Criteria

- [ ] Add task section to PROMPT.md
- [ ] Remove task section correctly
- [ ] Create backup before changes
- [ ] Restore from backup
- [ ] Get current task ID from prompt
- [ ] Create default prompt if missing
- [ ] All tests pass
