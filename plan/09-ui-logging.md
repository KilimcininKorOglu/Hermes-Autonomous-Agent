# Phase 09: UI and Logging

## Goal

Implement colored console output, table formatting, and logging.

## PowerShell Reference

```powershell
# From lib/Logger.ps1, lib/TableFormatter.ps1
# - Write-Log with levels
# - Format-TaskTable
# - Get-StatusColor
# - Progress display
```

## Go Implementation

### 9.1 Logger

```go
// internal/ui/logger.go
package ui

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    
    "github.com/fatih/color"
)

type LogLevel int

const (
    LogDebug LogLevel = iota
    LogInfo
    LogWarn
    LogError
    LogSuccess
)

var (
    debugColor   = color.New(color.FgHiBlack)
    infoColor    = color.New(color.FgCyan)
    warnColor    = color.New(color.FgYellow)
    errorColor   = color.New(color.FgRed)
    successColor = color.New(color.FgGreen)
)

type Logger struct {
    logFile  *os.File
    logPath  string
    minLevel LogLevel
    debug    bool
}

func NewLogger(basePath string, debug bool) (*Logger, error) {
    logsDir := filepath.Join(basePath, ".hermes", "logs")
    if err := os.MkdirAll(logsDir, 0755); err != nil {
        return nil, err
    }
    
    logPath := filepath.Join(logsDir, "hermes.log")
    file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    minLevel := LogInfo
    if debug {
        minLevel = LogDebug
    }
    
    return &Logger{
        logFile:  file,
        logPath:  logPath,
        minLevel: minLevel,
        debug:    debug,
    }, nil
}

func (l *Logger) Close() {
    if l.logFile != nil {
        l.logFile.Close()
    }
}

func (l *Logger) log(level LogLevel, levelStr string, format string, args ...interface{}) {
    if level < l.minLevel {
        return
    }
    
    msg := fmt.Sprintf(format, args...)
    timestamp := time.Now().Format("15:04:05")
    
    // Console output with color
    var c *color.Color
    switch level {
    case LogDebug:
        c = debugColor
    case LogInfo:
        c = infoColor
    case LogWarn:
        c = warnColor
    case LogError:
        c = errorColor
    case LogSuccess:
        c = successColor
    }
    
    c.Printf("[%s] %s\n", levelStr, msg)
    
    // File output
    if l.logFile != nil {
        fullTimestamp := time.Now().Format("2006-01-02 15:04:05")
        fmt.Fprintf(l.logFile, "[%s] [%s] %s\n", fullTimestamp, levelStr, msg)
    }
}

func (l *Logger) Debug(format string, args ...interface{}) {
    l.log(LogDebug, "DEBUG", format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
    l.log(LogInfo, "INFO", format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
    l.log(LogWarn, "WARN", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
    l.log(LogError, "ERROR", format, args...)
}

func (l *Logger) Success(format string, args ...interface{}) {
    l.log(LogSuccess, "SUCCESS", format, args...)
}
```

### 9.2 Table Formatter

```go
// internal/ui/table.go
package ui

import (
    "fmt"
    "strings"
    
    "github.com/fatih/color"
    "hermes/internal/task"
)

type Column struct {
    Name  string
    Width int
}

var taskColumns = []Column{
    {"ID", 6},
    {"Name", 35},
    {"Status", 12},
    {"Priority", 8},
    {"Feature", 6},
}

func GetStatusColor(status task.Status) *color.Color {
    switch status {
    case task.StatusCompleted:
        return color.New(color.FgGreen)
    case task.StatusInProgress:
        return color.New(color.FgYellow)
    case task.StatusBlocked:
        return color.New(color.FgRed)
    case task.StatusNotStarted:
        return color.New(color.FgHiBlack)
    default:
        return color.New(color.FgWhite)
    }
}

func FormatTaskTable(tasks []task.Task) string {
    if len(tasks) == 0 {
        return "No tasks found."
    }
    
    var sb strings.Builder
    
    // Header
    sb.WriteString(formatSeparator("top"))
    sb.WriteString(formatHeader())
    sb.WriteString(formatSeparator("middle"))
    
    // Rows
    for _, t := range tasks {
        sb.WriteString(formatTaskRow(t))
    }
    
    sb.WriteString(formatSeparator("bottom"))
    
    return sb.String()
}

func formatSeparator(position string) string {
    var left, mid, right, line string
    switch position {
    case "top":
        left, mid, right, line = "┌", "┬", "┐", "─"
    case "middle":
        left, mid, right, line = "├", "┼", "┤", "─"
    case "bottom":
        left, mid, right, line = "└", "┴", "┘", "─"
    }
    
    var parts []string
    for _, col := range taskColumns {
        parts = append(parts, strings.Repeat(line, col.Width+2))
    }
    
    return left + strings.Join(parts, mid) + right + "\n"
}

func formatHeader() string {
    var parts []string
    for _, col := range taskColumns {
        parts = append(parts, padRight(col.Name, col.Width))
    }
    return "│ " + strings.Join(parts, " │ ") + " │\n"
}

func formatTaskRow(t task.Task) string {
    values := []string{
        padRight(t.ID, taskColumns[0].Width),
        truncate(t.Name, taskColumns[1].Width),
        padRight(string(t.Status), taskColumns[2].Width),
        padRight(string(t.Priority), taskColumns[3].Width),
        padRight(t.FeatureID, taskColumns[4].Width),
    }
    
    // Color the status
    statusColor := GetStatusColor(t.Status)
    values[2] = statusColor.Sprint(values[2])
    
    return "│ " + strings.Join(values, " │ ") + " │\n"
}

func padRight(s string, width int) string {
    if len(s) >= width {
        return s[:width]
    }
    return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, width int) string {
    if len(s) <= width {
        return padRight(s, width)
    }
    return s[:width-3] + "..."
}
```

### 9.3 Progress Display

```go
// internal/ui/progress.go
package ui

import (
    "fmt"
    "strings"
    
    "github.com/fatih/color"
    "hermes/internal/task"
)

func FormatProgressBar(percentage float64, width int) string {
    filled := int(percentage / 100 * float64(width))
    empty := width - filled
    
    bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
    return fmt.Sprintf("[%s] %.1f%%", bar, percentage)
}

func PrintProgress(progress *task.Progress) {
    fmt.Println()
    fmt.Println("Task Progress")
    fmt.Println(strings.Repeat("─", 40))
    
    bar := FormatProgressBar(progress.Percentage, 30)
    fmt.Println(bar)
    
    fmt.Printf("\nTotal:       %d\n", progress.Total)
    
    green := color.New(color.FgGreen)
    yellow := color.New(color.FgYellow)
    gray := color.New(color.FgHiBlack)
    red := color.New(color.FgRed)
    
    fmt.Print("Completed:   ")
    green.Printf("%d\n", progress.Completed)
    
    fmt.Print("In Progress: ")
    yellow.Printf("%d\n", progress.InProgress)
    
    fmt.Print("Not Started: ")
    gray.Printf("%d\n", progress.NotStarted)
    
    fmt.Print("Blocked:     ")
    red.Printf("%d\n", progress.Blocked)
    
    fmt.Println(strings.Repeat("─", 40))
}

func PrintHeader(title string) {
    cyan := color.New(color.FgCyan, color.Bold)
    fmt.Println()
    cyan.Println(title)
    fmt.Println(strings.Repeat("=", len(title)))
    fmt.Println()
}

func PrintSection(title string) {
    yellow := color.New(color.FgYellow)
    fmt.Println()
    yellow.Println(title)
    fmt.Println(strings.Repeat("-", len(title)))
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/ui/logger.go` | Logging |
| `internal/ui/table.go` | Table formatting |
| `internal/ui/progress.go` | Progress display |
| `internal/ui/logger_test.go` | Tests |

## Acceptance Criteria

- [ ] Log to console with colors
- [ ] Log to file with timestamps
- [ ] Format task tables correctly
- [ ] Display progress bar
- [ ] Filter tasks by status/priority
- [ ] All tests pass
