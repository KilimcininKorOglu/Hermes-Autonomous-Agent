# Phase 03: Task File Parsing

## Goal

Implement task file reading, parsing, and status updates.

## PowerShell Reference

```powershell
# From lib/TaskReader.ps1
# Task file format:
# tasks/001-feature-name.md
#
# # Feature 1: Feature Name
# **Feature ID:** F001
# **Status:** NOT_STARTED
#
# ### T001: Task Name
# **Status:** NOT_STARTED
# **Priority:** P1
# **Files to Touch:** file1.go, file2.go
# **Dependencies:** T000
# **Success Criteria:**
# - Criterion 1
# - Criterion 2
```

## Go Implementation

### 3.1 Types

```go
// internal/task/types.go
package task

type Status string

const (
    StatusNotStarted Status = "NOT_STARTED"
    StatusInProgress Status = "IN_PROGRESS"
    StatusCompleted  Status = "COMPLETED"
    StatusBlocked    Status = "BLOCKED"
)

type Priority string

const (
    PriorityP1 Priority = "P1"
    PriorityP2 Priority = "P2"
    PriorityP3 Priority = "P3"
    PriorityP4 Priority = "P4"
)

type Feature struct {
    ID          string   `json:"id"`          // F001
    Name        string   `json:"name"`
    Status      Status   `json:"status"`
    Description string   `json:"description"`
    Tasks       []Task   `json:"tasks"`
    FilePath    string   `json:"filePath"`
}

type Task struct {
    ID              string   `json:"id"`              // T001
    Name            string   `json:"name"`
    Status          Status   `json:"status"`
    Priority        Priority `json:"priority"`
    FilesToTouch    []string `json:"filesToTouch"`
    Dependencies    []string `json:"dependencies"`
    SuccessCriteria []string `json:"successCriteria"`
    FeatureID       string   `json:"featureId"`
}

type Progress struct {
    Total       int     `json:"total"`
    Completed   int     `json:"completed"`
    InProgress  int     `json:"inProgress"`
    NotStarted  int     `json:"notStarted"`
    Blocked     int     `json:"blocked"`
    Percentage  float64 `json:"percentage"`
}
```

### 3.2 Reader

```go
// internal/task/reader.go
package task

import (
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

type Reader struct {
    basePath string
    tasksDir string
}

func NewReader(basePath string) *Reader {
    return &Reader{
        basePath: basePath,
        tasksDir: filepath.Join(basePath, ".hermes", "tasks"),
    }
}

func (r *Reader) GetFeatureFiles() ([]string, error) {
    pattern := filepath.Join(r.tasksDir, "[0-9][0-9][0-9]-*.md")
    return filepath.Glob(pattern)
}

func (r *Reader) ReadFeature(filePath string) (*Feature, error) {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    return parseFeature(string(content), filePath)
}

func (r *Reader) GetAllTasks() ([]Task, error) {
    files, err := r.GetFeatureFiles()
    if err != nil {
        return nil, err
    }
    
    var tasks []Task
    for _, file := range files {
        feature, err := r.ReadFeature(file)
        if err != nil {
            continue
        }
        tasks = append(tasks, feature.Tasks...)
    }
    return tasks, nil
}

func (r *Reader) GetTaskByID(id string) (*Task, error) {
    tasks, err := r.GetAllTasks()
    if err != nil {
        return nil, err
    }
    for _, t := range tasks {
        if t.ID == id {
            return &t, nil
        }
    }
    return nil, nil
}

func (r *Reader) GetTasksByStatus(status Status) ([]Task, error) {
    tasks, err := r.GetAllTasks()
    if err != nil {
        return nil, err
    }
    var filtered []Task
    for _, t := range tasks {
        if t.Status == status {
            filtered = append(filtered, t)
        }
    }
    return filtered, nil
}

func (r *Reader) GetNextTask() (*Task, error) {
    tasks, err := r.GetTasksByStatus(StatusNotStarted)
    if err != nil {
        return nil, err
    }
    if len(tasks) == 0 {
        return nil, nil
    }
    // Sort by priority, return first
    // TODO: Check dependencies
    return &tasks[0], nil
}

func (r *Reader) GetProgress() (*Progress, error) {
    tasks, err := r.GetAllTasks()
    if err != nil {
        return nil, err
    }
    
    p := &Progress{Total: len(tasks)}
    for _, t := range tasks {
        switch t.Status {
        case StatusCompleted:
            p.Completed++
        case StatusInProgress:
            p.InProgress++
        case StatusNotStarted:
            p.NotStarted++
        case StatusBlocked:
            p.Blocked++
        }
    }
    if p.Total > 0 {
        p.Percentage = float64(p.Completed) / float64(p.Total) * 100
    }
    return p, nil
}
```

### 3.3 Parser

```go
// internal/task/parser.go
package task

import (
    "regexp"
    "strings"
)

var (
    featureIDRegex     = regexp.MustCompile(`\*\*Feature ID:\*\*\s*(F\d+)`)
    featureNameRegex   = regexp.MustCompile(`^#\s*Feature\s*\d+:\s*(.+)`)
    featureStatusRegex = regexp.MustCompile(`\*\*Status:\*\*\s*(\w+)`)
    taskHeaderRegex    = regexp.MustCompile(`^###\s*(T\d+):\s*(.+)`)
    priorityRegex      = regexp.MustCompile(`\*\*Priority:\*\*\s*(P[1-4])`)
    filesToTouchRegex  = regexp.MustCompile(`\*\*Files to Touch:\*\*\s*(.+)`)
    dependenciesRegex  = regexp.MustCompile(`\*\*Dependencies:\*\*\s*(.+)`)
)

func parseFeature(content, filePath string) (*Feature, error) {
    feature := &Feature{FilePath: filePath}
    
    // Parse feature metadata
    if m := featureIDRegex.FindStringSubmatch(content); len(m) > 1 {
        feature.ID = m[1]
    }
    if m := featureNameRegex.FindStringSubmatch(content); len(m) > 1 {
        feature.Name = strings.TrimSpace(m[1])
    }
    if m := featureStatusRegex.FindStringSubmatch(content); len(m) > 1 {
        feature.Status = Status(m[1])
    }
    
    // Parse tasks
    feature.Tasks = parseTasks(content, feature.ID)
    
    return feature, nil
}

func parseTasks(content, featureID string) []Task {
    var tasks []Task
    lines := strings.Split(content, "\n")
    
    var currentTask *Task
    var inSuccessCriteria bool
    
    for _, line := range lines {
        // Check for task header
        if m := taskHeaderRegex.FindStringSubmatch(line); len(m) > 2 {
            if currentTask != nil {
                tasks = append(tasks, *currentTask)
            }
            currentTask = &Task{
                ID:        m[1],
                Name:      strings.TrimSpace(m[2]),
                FeatureID: featureID,
                Status:    StatusNotStarted,
                Priority:  PriorityP2,
            }
            inSuccessCriteria = false
            continue
        }
        
        if currentTask == nil {
            continue
        }
        
        // Parse task attributes
        if m := featureStatusRegex.FindStringSubmatch(line); len(m) > 1 {
            currentTask.Status = Status(m[1])
        }
        if m := priorityRegex.FindStringSubmatch(line); len(m) > 1 {
            currentTask.Priority = Priority(m[1])
        }
        if m := filesToTouchRegex.FindStringSubmatch(line); len(m) > 1 {
            currentTask.FilesToTouch = parseList(m[1])
        }
        if m := dependenciesRegex.FindStringSubmatch(line); len(m) > 1 {
            currentTask.Dependencies = parseList(m[1])
        }
        
        // Success criteria
        if strings.Contains(line, "**Success Criteria:**") {
            inSuccessCriteria = true
            continue
        }
        if inSuccessCriteria && strings.HasPrefix(strings.TrimSpace(line), "-") {
            criterion := strings.TrimPrefix(strings.TrimSpace(line), "- ")
            currentTask.SuccessCriteria = append(currentTask.SuccessCriteria, criterion)
        }
    }
    
    if currentTask != nil {
        tasks = append(tasks, *currentTask)
    }
    
    return tasks
}

func parseList(s string) []string {
    var items []string
    for _, item := range strings.Split(s, ",") {
        item = strings.TrimSpace(item)
        if item != "" && item != "None" {
            items = append(items, item)
        }
    }
    return items
}
```

### 3.4 Status Updater

```go
// internal/task/status.go
package task

import (
    "os"
    "strings"
)

type StatusUpdater struct {
    basePath string
}

func NewStatusUpdater(basePath string) *StatusUpdater {
    return &StatusUpdater{basePath: basePath}
}

func (u *StatusUpdater) UpdateTaskStatus(taskID string, newStatus Status) error {
    reader := NewReader(u.basePath)
    files, err := reader.GetFeatureFiles()
    if err != nil {
        return err
    }
    
    for _, file := range files {
        content, err := os.ReadFile(file)
        if err != nil {
            continue
        }
        
        if !strings.Contains(string(content), taskID) {
            continue
        }
        
        // Update status in file
        updated := updateTaskStatusInContent(string(content), taskID, newStatus)
        return os.WriteFile(file, []byte(updated), 0644)
    }
    
    return nil
}

func updateTaskStatusInContent(content, taskID string, newStatus Status) string {
    // Find task section and update status
    lines := strings.Split(content, "\n")
    var result []string
    inTask := false
    
    for _, line := range lines {
        if strings.Contains(line, taskID+":") {
            inTask = true
        } else if strings.HasPrefix(line, "### T") {
            inTask = false
        }
        
        if inTask && strings.Contains(line, "**Status:**") {
            line = "**Status:** " + string(newStatus)
        }
        
        result = append(result, line)
    }
    
    return strings.Join(result, "\n")
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/task/types.go` | Type definitions |
| `internal/task/reader.go` | Task reading |
| `internal/task/parser.go` | Markdown parsing |
| `internal/task/status.go` | Status updates |
| `internal/task/reader_test.go` | Reader tests |
| `internal/task/parser_test.go` | Parser tests |

## Acceptance Criteria

- [ ] Parse feature files correctly
- [ ] Extract all task metadata
- [ ] Get tasks by status
- [ ] Get next available task
- [ ] Update task status in file
- [ ] Calculate progress correctly
- [ ] All tests pass
