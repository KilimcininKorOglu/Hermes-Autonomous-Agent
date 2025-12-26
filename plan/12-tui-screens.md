# Phase 12: TUI Screens (Terminal User Interface)

## Goal

Interactive terminal UI using `bubbletea` and `lipgloss` with dynamic resizing.

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/lipgloss` | Styling and layout |
| `github.com/charmbracelet/bubbles` | Pre-built components (table, viewport, spinner) |

## Screen Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  HERMES AUTONOMOUS AGENT                              v1.0.0    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [1] Dashboard    - Overview and status                         │
│  [2] Tasks        - Task list and management                    │
│  [3] Logs         - Real-time log viewer                        │
│  [4] Settings     - Configuration                               │
│                                                                 │
│  Press number to switch, q to quit, ? for help                  │
└─────────────────────────────────────────────────────────────────┘
```

## Architecture

```go
// internal/tui/app.go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type Screen int

const (
    ScreenDashboard Screen = iota
    ScreenTasks
    ScreenTaskDetail
    ScreenLogs
    ScreenSettings
    ScreenHelp
)

type Model struct {
    screen       Screen
    width        int
    height       int
    ready        bool
    
    // Sub-models
    dashboard    DashboardModel
    tasks        TasksModel
    taskDetail   TaskDetailModel
    logs         LogsModel
    settings     SettingsModel
    
    // Shared state
    config       *config.Config
    taskReader   *task.Reader
    logger       *Logger
}

func NewApp(basePath string) (*Model, error) {
    cfg, err := config.Load(basePath)
    if err != nil {
        return nil, err
    }
    
    return &Model{
        screen:     ScreenDashboard,
        config:     cfg,
        taskReader: task.NewReader(basePath),
        dashboard:  NewDashboardModel(),
        tasks:      NewTasksModel(),
        logs:       NewLogsModel(),
        settings:   NewSettingsModel(),
    }, nil
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        tea.EnterAltScreen,
        m.dashboard.Init(),
        tickEvery(time.Second), // Refresh data
    )
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.ready = true
        // Propagate to all screens
        m.dashboard = m.dashboard.SetSize(msg.Width, msg.Height)
        m.tasks = m.tasks.SetSize(msg.Width, msg.Height)
        m.logs = m.logs.SetSize(msg.Width, msg.Height)
        
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "1":
            m.screen = ScreenDashboard
        case "2":
            m.screen = ScreenTasks
        case "3":
            m.screen = ScreenLogs
        case "4":
            m.screen = ScreenSettings
        case "?":
            m.screen = ScreenHelp
        case "esc":
            if m.screen == ScreenTaskDetail {
                m.screen = ScreenTasks
            }
        }
    }
    
    // Update active screen
    var cmd tea.Cmd
    switch m.screen {
    case ScreenDashboard:
        m.dashboard, cmd = m.dashboard.Update(msg)
    case ScreenTasks:
        m.tasks, cmd = m.tasks.Update(msg)
    case ScreenTaskDetail:
        m.taskDetail, cmd = m.taskDetail.Update(msg)
    case ScreenLogs:
        m.logs, cmd = m.logs.Update(msg)
    }
    
    return m, cmd
}

func (m Model) View() string {
    if !m.ready {
        return "Initializing..."
    }
    
    var content string
    switch m.screen {
    case ScreenDashboard:
        content = m.dashboard.View()
    case ScreenTasks:
        content = m.tasks.View()
    case ScreenTaskDetail:
        content = m.taskDetail.View()
    case ScreenLogs:
        content = m.logs.View()
    case ScreenSettings:
        content = m.settings.View()
    case ScreenHelp:
        content = m.helpView()
    }
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.headerView(),
        content,
        m.footerView(),
    )
}
```

---

## Screen 1: Dashboard

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  HERMES DASHBOARD                                            Loop #42       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─ Progress ────────────────────────┐  ┌─ Circuit Breaker ───────────────┐ │
│  │                                   │  │                                 │ │
│  │  [████████████░░░░░░░░] 62%       │  │  State: CLOSED [OK]             │ │
│  │                                   │  │  Loops since progress: 0        │ │
│  │  Total:       24 tasks            │  │  Last progress: Loop #42        │ │
│  │  Completed:   15                  │  │                                 │ │
│  │  In Progress:  1                  │  └─────────────────────────────────┘ │
│  │  Blocked:      2                  │                                      │
│  │  Not Started:  6                  │  ┌─ Current Task ──────────────────┐ │
│  │                                   │  │                                 │ │
│  └───────────────────────────────────┘  │  T016: Add user validation      │ │
│                                         │  Feature: F003 - User Auth      │ │
│  ┌─ AI Status ───────────────────────┐  │  Priority: P1                   │ │
│  │                                   │  │  Status: IN_PROGRESS            │ │
│  │  Provider: claude                 │  │                                 │ │
│  │  Last call: 2m ago               │  │  Files:                         │ │
│  │  Cost: $0.0234                   │  │  - api/validation.go            │ │
│  │  Tokens: 1,234 / 4,567           │  │  - handlers/user.go             │ │
│  │                                   │  │                                 │ │
│  └───────────────────────────────────┘  └─────────────────────────────────┘ │
│                                                                             │
│  ┌─ Recent Activity ────────────────────────────────────────────────────┐   │
│  │  [20:15:32] Task T015 completed                                      │   │
│  │  [20:14:18] Branch created: feature/F003-user-auth                   │   │
│  │  [20:12:45] AI response received (1.2s, $0.0089)                     │   │
│  │  [20:10:22] Task T015 started                                        │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  [1]Dashboard [2]Tasks [3]Logs [4]Settings    [Space]Pause [R]Reset  [Q]uit │
└─────────────────────────────────────────────────────────────────────────────┘
```

```go
// internal/tui/dashboard.go
package tui

import (
    "fmt"
    "strings"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

var (
    // Adaptive styles
    boxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(0, 1)
    
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("39"))
    
    progressFullStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("42"))
    
    progressEmptyStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("238"))
)

type DashboardModel struct {
    width      int
    height     int
    progress   *task.Progress
    circuit    *circuit.BreakerState
    currentTask *task.Task
    activities []Activity
    aiStatus   AIStatus
    loopNumber int
}

type Activity struct {
    Time    string
    Message string
    Type    string // "success", "info", "warning", "error"
}

type AIStatus struct {
    Provider   string
    LastCall   string
    Cost       float64
    TokensIn   int
    TokensOut  int
}

func NewDashboardModel() DashboardModel {
    return DashboardModel{
        activities: make([]Activity, 0, 10),
    }
}

func (m DashboardModel) SetSize(w, h int) DashboardModel {
    m.width = w
    m.height = h
    return m
}

func (m DashboardModel) Init() tea.Cmd {
    return m.loadData
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
    switch msg := msg.(type) {
    case DataLoadedMsg:
        m.progress = msg.Progress
        m.circuit = msg.Circuit
        m.currentTask = msg.CurrentTask
    case ActivityMsg:
        m.activities = append([]Activity{msg.Activity}, m.activities...)
        if len(m.activities) > 10 {
            m.activities = m.activities[:10]
        }
    }
    return m, nil
}

func (m DashboardModel) View() string {
    if m.width == 0 {
        return ""
    }
    
    // Calculate panel widths based on terminal size
    contentWidth := m.width - 4
    leftWidth := contentWidth / 2
    rightWidth := contentWidth - leftWidth - 1
    
    // Build panels
    progressPanel := m.progressPanel(leftWidth)
    circuitPanel := m.circuitPanel(rightWidth)
    aiPanel := m.aiPanel(leftWidth)
    taskPanel := m.currentTaskPanel(rightWidth)
    activityPanel := m.activityPanel(contentWidth)
    
    // Layout
    topRow := lipgloss.JoinHorizontal(lipgloss.Top, progressPanel, " ", circuitPanel)
    midRow := lipgloss.JoinHorizontal(lipgloss.Top, aiPanel, " ", taskPanel)
    
    return lipgloss.JoinVertical(lipgloss.Left, topRow, midRow, activityPanel)
}

func (m DashboardModel) progressPanel(width int) string {
    if m.progress == nil {
        return boxStyle.Width(width).Render("Loading...")
    }
    
    // Progress bar
    barWidth := width - 10
    filled := int(m.progress.Percentage / 100 * float64(barWidth))
    bar := progressFullStyle.Render(strings.Repeat("█", filled)) +
           progressEmptyStyle.Render(strings.Repeat("░", barWidth-filled))
    
    content := fmt.Sprintf(
        "%s\n%s %.0f%%\n\n"+
        "Total:       %d tasks\n"+
        "Completed:   %d\n"+
        "In Progress: %d\n"+
        "Blocked:     %d\n"+
        "Not Started: %d",
        titleStyle.Render("Progress"),
        bar, m.progress.Percentage,
        m.progress.Total,
        m.progress.Completed,
        m.progress.InProgress,
        m.progress.Blocked,
        m.progress.NotStarted,
    )
    
    return boxStyle.Width(width).Render(content)
}

func (m DashboardModel) circuitPanel(width int) string {
    state := "CLOSED"
    stateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
    icon := "[OK]"
    
    if m.circuit != nil {
        state = string(m.circuit.State)
        switch m.circuit.State {
        case circuit.StateHalfOpen:
            stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
            icon = "[!!]"
        case circuit.StateOpen:
            stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
            icon = "[XX]"
        }
    }
    
    content := fmt.Sprintf(
        "%s\n\nState: %s %s\nLoops since progress: %d\nLast progress: Loop #%d",
        titleStyle.Render("Circuit Breaker"),
        stateStyle.Render(state), icon,
        m.circuit.ConsecutiveNoProgress,
        m.circuit.LastProgress,
    )
    
    return boxStyle.Width(width).Render(content)
}

func (m DashboardModel) currentTaskPanel(width int) string {
    if m.currentTask == nil {
        content := titleStyle.Render("Current Task") + "\n\nNo active task"
        return boxStyle.Width(width).Render(content)
    }
    
    t := m.currentTask
    content := fmt.Sprintf(
        "%s\n\n%s: %s\nFeature: %s\nPriority: %s\nStatus: %s\n\nFiles:\n%s",
        titleStyle.Render("Current Task"),
        t.ID, truncateText(t.Name, width-15),
        t.FeatureID,
        t.Priority,
        t.Status,
        formatFiles(t.FilesToTouch, width-4),
    )
    
    return boxStyle.Width(width).Render(content)
}

func (m DashboardModel) aiPanel(width int) string {
    content := fmt.Sprintf(
        "%s\n\nProvider: %s\nLast call: %s\nCost: $%.4f\nTokens: %d / %d",
        titleStyle.Render("AI Status"),
        m.aiStatus.Provider,
        m.aiStatus.LastCall,
        m.aiStatus.Cost,
        m.aiStatus.TokensIn,
        m.aiStatus.TokensOut,
    )
    
    return boxStyle.Width(width).Render(content)
}

func (m DashboardModel) activityPanel(width int) string {
    var lines []string
    lines = append(lines, titleStyle.Render("Recent Activity"))
    lines = append(lines, "")
    
    maxLines := 5
    if m.height > 40 {
        maxLines = 8
    }
    
    for i, a := range m.activities {
        if i >= maxLines {
            break
        }
        line := fmt.Sprintf("[%s] %s", a.Time, a.Message)
        if len(line) > width-4 {
            line = line[:width-7] + "..."
        }
        lines = append(lines, line)
    }
    
    return boxStyle.Width(width).Render(strings.Join(lines, "\n"))
}
```

---

## Screen 2: Task List

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  TASK LIST                                        Filter: [All ▼] [P1-P4 ▼] │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─ Feature: F001 - Project Setup ──────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  ● T001  Initialize Go module              COMPLETED   P1           │   │
│  │  ● T002  Create directory structure        COMPLETED   P1           │   │
│  │  ● T003  Add Makefile                      COMPLETED   P2           │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─ Feature: F002 - Configuration ──────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  ● T004  Create config types               COMPLETED   P1           │   │
│  │  ● T005  Implement config loading          COMPLETED   P1           │   │
│  │  ○ T006  Add config validation             NOT_STARTED P2           │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─ Feature: F003 - User Authentication ────────────────────────────────┐   │
│  │                                                                      │   │
│  │  ◐ T007  Create login endpoint             IN_PROGRESS P1      ←    │   │
│  │  ○ T008  Add password hashing              NOT_STARTED P1           │   │
│  │  ○ T009  Implement JWT tokens              NOT_STARTED P1           │   │
│  │  ✕ T010  Add rate limiting                 BLOCKED     P2           │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                                                    Showing 10 of 24 tasks   │
├─────────────────────────────────────────────────────────────────────────────┤
│  [↑↓]Navigate [Enter]Details [F]Filter [S]Status [Space]Toggle  [Esc]Back   │
└─────────────────────────────────────────────────────────────────────────────┘
```

```go
// internal/tui/tasks.go
package tui

import (
    "fmt"
    "strings"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/list"
)

var (
    statusIcons = map[task.Status]string{
        task.StatusCompleted:  "●",
        task.StatusInProgress: "◐",
        task.StatusNotStarted: "○",
        task.StatusBlocked:    "✕",
    }
    
    statusColors = map[task.Status]lipgloss.Color{
        task.StatusCompleted:  lipgloss.Color("42"),  // Green
        task.StatusInProgress: lipgloss.Color("214"), // Yellow
        task.StatusNotStarted: lipgloss.Color("245"), // Gray
        task.StatusBlocked:    lipgloss.Color("196"), // Red
    }
    
    selectedStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("62")).
        Foreground(lipgloss.Color("230"))
    
    featureHeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("39")).
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(0, 1)
)

type TasksModel struct {
    width        int
    height       int
    features     []task.Feature
    cursor       int
    offset       int
    filter       task.Status // empty = all
    priorityFilter task.Priority
    expanded     map[string]bool // feature ID -> expanded
}

func NewTasksModel() TasksModel {
    return TasksModel{
        expanded: make(map[string]bool),
    }
}

func (m TasksModel) SetSize(w, h int) TasksModel {
    m.width = w
    m.height = h
    return m
}

func (m TasksModel) Update(msg tea.Msg) (TasksModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            m.cursor++
        case "enter":
            // Open task detail
            return m, openTaskDetail(m.getSelectedTask())
        case "f":
            // Cycle filter
            m.cycleFilter()
        case "space":
            // Toggle feature expand
            m.toggleFeature()
        }
    case FeaturesLoadedMsg:
        m.features = msg.Features
    }
    return m, nil
}

func (m TasksModel) View() string {
    if m.width == 0 {
        return ""
    }
    
    contentWidth := m.width - 4
    contentHeight := m.height - 6 // header + footer
    
    var lines []string
    lineIdx := 0
    
    for _, feature := range m.features {
        // Feature header
        header := m.renderFeatureHeader(feature, contentWidth)
        lines = append(lines, header)
        lineIdx++
        
        // Feature tasks (if expanded)
        if m.expanded[feature.ID] || len(m.expanded) == 0 {
            for _, t := range feature.Tasks {
                if !m.matchesFilter(t) {
                    continue
                }
                
                line := m.renderTaskLine(t, contentWidth, lineIdx == m.cursor)
                lines = append(lines, line)
                lineIdx++
            }
        }
        
        lines = append(lines, "") // Spacing
    }
    
    // Apply scrolling
    visibleLines := contentHeight
    if m.offset > len(lines)-visibleLines {
        m.offset = max(0, len(lines)-visibleLines)
    }
    if m.cursor < m.offset {
        m.offset = m.cursor
    }
    if m.cursor >= m.offset+visibleLines {
        m.offset = m.cursor - visibleLines + 1
    }
    
    end := min(m.offset+visibleLines, len(lines))
    visible := lines[m.offset:end]
    
    return strings.Join(visible, "\n")
}

func (m TasksModel) renderFeatureHeader(f task.Feature, width int) string {
    completed := 0
    total := len(f.Tasks)
    for _, t := range f.Tasks {
        if t.Status == task.StatusCompleted {
            completed++
        }
    }
    
    progress := ""
    if total > 0 {
        progress = fmt.Sprintf(" [%d/%d]", completed, total)
    }
    
    title := fmt.Sprintf("Feature: %s - %s%s", f.ID, f.Name, progress)
    return featureHeaderStyle.Width(width).Render(title)
}

func (m TasksModel) renderTaskLine(t task.Task, width int, selected bool) string {
    icon := statusIcons[t.Status]
    color := statusColors[t.Status]
    
    iconStyled := lipgloss.NewStyle().Foreground(color).Render(icon)
    
    // Calculate widths
    idWidth := 6
    statusWidth := 12
    prioWidth := 4
    nameWidth := width - idWidth - statusWidth - prioWidth - 10
    
    name := t.Name
    if len(name) > nameWidth {
        name = name[:nameWidth-3] + "..."
    }
    
    line := fmt.Sprintf("  %s %-*s %-*s %-*s %s",
        iconStyled,
        idWidth, t.ID,
        nameWidth, name,
        statusWidth, t.Status,
        t.Priority,
    )
    
    if selected {
        line = selectedStyle.Render(line) + " ←"
    }
    
    return line
}

func (m *TasksModel) cycleFilter() {
    switch m.filter {
    case "":
        m.filter = task.StatusNotStarted
    case task.StatusNotStarted:
        m.filter = task.StatusInProgress
    case task.StatusInProgress:
        m.filter = task.StatusBlocked
    case task.StatusBlocked:
        m.filter = task.StatusCompleted
    default:
        m.filter = ""
    }
}

func (m TasksModel) matchesFilter(t task.Task) bool {
    if m.filter == "" {
        return true
    }
    return t.Status == m.filter
}
```

---

## Screen 3: Task Detail

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  TASK DETAIL                                                    [Esc] Back  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─ T007: Create login endpoint ────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Feature:     F003 - User Authentication                            │   │
│  │  Status:      IN_PROGRESS                                           │   │
│  │  Priority:    P1                                                    │   │
│  │  Dependencies: T004, T005                                           │   │
│  │                                                                      │   │
│  │  ─────────────────────────────────────────────────────────────────  │   │
│  │                                                                      │   │
│  │  Files to Touch:                                                    │   │
│  │    • api/auth.go                                                    │   │
│  │    • handlers/login.go                                              │   │
│  │    • models/user.go                                                 │   │
│  │                                                                      │   │
│  │  ─────────────────────────────────────────────────────────────────  │   │
│  │                                                                      │   │
│  │  Success Criteria:                                                  │   │
│  │    ✓ Endpoint accepts POST /api/login                               │   │
│  │    ✓ Validates username and password                                │   │
│  │    ○ Returns JWT token on success                                   │   │
│  │    ○ Returns 401 on invalid credentials                             │   │
│  │                                                                      │   │
│  │  ─────────────────────────────────────────────────────────────────  │   │
│  │                                                                      │   │
│  │  Notes:                                                             │   │
│  │    Use bcrypt for password hashing. JWT secret should be            │   │
│  │    configurable via environment variable.                           │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  [S]et Status  [E]dit  [D]ependencies  [↑↓]Scroll                [Esc]Back  │
└─────────────────────────────────────────────────────────────────────────────┘
```

```go
// internal/tui/task_detail.go
package tui

import (
    "fmt"
    "strings"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/viewport"
)

type TaskDetailModel struct {
    width    int
    height   int
    task     *task.Task
    feature  *task.Feature
    viewport viewport.Model
}

func NewTaskDetailModel(t *task.Task, f *task.Feature) TaskDetailModel {
    return TaskDetailModel{
        task:    t,
        feature: f,
    }
}

func (m TaskDetailModel) SetSize(w, h int) TaskDetailModel {
    m.width = w
    m.height = h
    m.viewport = viewport.New(w-4, h-8)
    m.viewport.SetContent(m.buildContent())
    return m
}

func (m TaskDetailModel) Update(msg tea.Msg) (TaskDetailModel, tea.Cmd) {
    var cmd tea.Cmd
    
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "s":
            // Open status picker
            return m, openStatusPicker(m.task)
        case "e":
            // Edit task (future)
        }
    }
    
    m.viewport, cmd = m.viewport.Update(msg)
    return m, cmd
}

func (m TaskDetailModel) View() string {
    return m.viewport.View()
}

func (m TaskDetailModel) buildContent() string {
    t := m.task
    if t == nil {
        return "No task selected"
    }
    
    var sb strings.Builder
    
    // Header
    header := fmt.Sprintf("%s: %s", t.ID, t.Name)
    sb.WriteString(titleStyle.Render(header))
    sb.WriteString("\n\n")
    
    // Metadata
    sb.WriteString(fmt.Sprintf("Feature:      %s - %s\n", m.feature.ID, m.feature.Name))
    sb.WriteString(fmt.Sprintf("Status:       %s\n", m.renderStatus(t.Status)))
    sb.WriteString(fmt.Sprintf("Priority:     %s\n", t.Priority))
    
    if len(t.Dependencies) > 0 {
        sb.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(t.Dependencies, ", ")))
    }
    
    sb.WriteString("\n" + strings.Repeat("─", m.width-8) + "\n\n")
    
    // Files
    if len(t.FilesToTouch) > 0 {
        sb.WriteString(subtitleStyle.Render("Files to Touch:"))
        sb.WriteString("\n")
        for _, f := range t.FilesToTouch {
            sb.WriteString(fmt.Sprintf("  • %s\n", f))
        }
        sb.WriteString("\n" + strings.Repeat("─", m.width-8) + "\n\n")
    }
    
    // Success criteria
    if len(t.SuccessCriteria) > 0 {
        sb.WriteString(subtitleStyle.Render("Success Criteria:"))
        sb.WriteString("\n")
        for _, c := range t.SuccessCriteria {
            icon := "○"
            if strings.HasPrefix(c, "[x]") || strings.HasPrefix(c, "[X]") {
                icon = "✓"
                c = strings.TrimPrefix(strings.TrimPrefix(c, "[x]"), "[X]")
            }
            sb.WriteString(fmt.Sprintf("  %s %s\n", icon, strings.TrimSpace(c)))
        }
    }
    
    return sb.String()
}

func (m TaskDetailModel) renderStatus(s task.Status) string {
    color := statusColors[s]
    return lipgloss.NewStyle().Foreground(color).Render(string(s))
}
```

---

## Screen 4: Log Viewer

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  LOG VIEWER                                    Level: [INFO ▼] [Auto-scroll]│
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  2024-01-15 20:15:32 [INFO]    Loop #42 starting...                         │
│  2024-01-15 20:15:32 [INFO]    Working on task: T016 - Add user validation  │
│  2024-01-15 20:15:33 [DEBUG]   Injecting task into PROMPT.md                │
│  2024-01-15 20:15:33 [INFO]    Calling AI provider: claude                  │
│  2024-01-15 20:15:34 [DEBUG]   Request size: 2,456 tokens                   │
│  2024-01-15 20:16:45 [INFO]    AI response received                         │
│  2024-01-15 20:16:45 [DEBUG]   Response size: 1,234 tokens                  │
│  2024-01-15 20:16:45 [DEBUG]   Cost: $0.0089                                │
│  2024-01-15 20:16:46 [INFO]    Analyzing response...                        │
│  2024-01-15 20:16:46 [SUCCESS] Task T015 marked complete                    │
│  2024-01-15 20:16:46 [INFO]    Committing changes...                        │
│  2024-01-15 20:16:47 [SUCCESS] Committed: feat(T015): Add input validation  │
│  2024-01-15 20:16:47 [INFO]    Moving to next task...                       │
│  2024-01-15 20:16:48 [INFO]    Task T016 started                            │
│  2024-01-15 20:16:48 [DEBUG]   Files to touch: api/validation.go            │
│  2024-01-15 20:16:49 [INFO]    Calling AI provider: claude                  │
│  2024-01-15 20:17:52 [INFO]    AI response received                         │
│  2024-01-15 20:17:52 [WARNING] Response analysis: low confidence (0.4)      │
│  2024-01-15 20:17:53 [INFO]    Continuing loop...                           │
│                                                                             │
│                                                         ▼ Following latest  │
├─────────────────────────────────────────────────────────────────────────────┤
│  [L]evel [F]ilter [C]lear [/]Search  [Space]Pause [G]o to end    [Esc]Back  │
└─────────────────────────────────────────────────────────────────────────────┘
```

```go
// internal/tui/logs.go
package tui

import (
    "fmt"
    "strings"
    "time"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/bubbles/viewport"
)

var logLevelStyles = map[string]lipgloss.Style{
    "DEBUG":   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
    "INFO":    lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
    "SUCCESS": lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
    "WARNING": lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
    "ERROR":   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
}

type LogEntry struct {
    Timestamp time.Time
    Level     string
    Message   string
}

type LogsModel struct {
    width      int
    height     int
    entries    []LogEntry
    viewport   viewport.Model
    autoScroll bool
    level      string // filter level
    search     string
    following  bool
}

func NewLogsModel() LogsModel {
    return LogsModel{
        autoScroll: true,
        level:      "INFO",
        following:  true,
    }
}

func (m LogsModel) SetSize(w, h int) LogsModel {
    m.width = w
    m.height = h
    m.viewport = viewport.New(w-4, h-8)
    m.viewport.SetContent(m.buildContent())
    if m.following {
        m.viewport.GotoBottom()
    }
    return m
}

func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "l":
            m.cycleLevel()
            m.viewport.SetContent(m.buildContent())
        case "c":
            m.entries = nil
            m.viewport.SetContent("")
        case "space":
            m.following = !m.following
        case "G":
            m.viewport.GotoBottom()
            m.following = true
        case "/":
            // Open search (future)
        }
    case LogEntryMsg:
        m.entries = append(m.entries, msg.Entry)
        m.viewport.SetContent(m.buildContent())
        if m.following {
            m.viewport.GotoBottom()
        }
    }
    
    var cmd tea.Cmd
    m.viewport, cmd = m.viewport.Update(msg)
    
    // Check if user scrolled
    if m.viewport.AtBottom() {
        m.following = true
    } else if msg, ok := msg.(tea.KeyMsg); ok {
        if msg.String() == "up" || msg.String() == "k" {
            m.following = false
        }
    }
    
    return m, cmd
}

func (m LogsModel) View() string {
    status := "Following latest"
    if !m.following {
        status = "Paused - press G to follow"
    }
    
    header := fmt.Sprintf("Level: [%s]  %s", m.level, status)
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        m.viewport.View(),
        lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(header),
    )
}

func (m LogsModel) buildContent() string {
    var lines []string
    
    for _, entry := range m.entries {
        if !m.matchesLevel(entry.Level) {
            continue
        }
        if m.search != "" && !strings.Contains(entry.Message, m.search) {
            continue
        }
        
        line := m.formatEntry(entry)
        lines = append(lines, line)
    }
    
    return strings.Join(lines, "\n")
}

func (m LogsModel) formatEntry(e LogEntry) string {
    timestamp := e.Timestamp.Format("2006-01-02 15:04:05")
    style := logLevelStyles[e.Level]
    
    levelStr := fmt.Sprintf("[%-7s]", e.Level)
    
    return fmt.Sprintf("%s %s %s",
        lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(timestamp),
        style.Render(levelStr),
        e.Message,
    )
}

func (m *LogsModel) cycleLevel() {
    levels := []string{"DEBUG", "INFO", "WARNING", "ERROR"}
    for i, l := range levels {
        if l == m.level {
            m.level = levels[(i+1)%len(levels)]
            return
        }
    }
    m.level = "INFO"
}

func (m LogsModel) matchesLevel(level string) bool {
    order := map[string]int{"DEBUG": 0, "INFO": 1, "SUCCESS": 1, "WARNING": 2, "ERROR": 3}
    return order[level] >= order[m.level]
}
```

---

## Screen 5: Settings

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  SETTINGS                                                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─ AI Configuration ───────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Planning AI:        [claude     ▼]                                 │   │
│  │  Coding AI:          [droid      ▼]                                 │   │
│  │  Timeout:            [300] seconds                                  │   │
│  │  Max Retries:        [10]                                           │   │
│  │  Stream Output:      [●] Enabled                                    │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─ Git Configuration ──────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Auto Branch:        [●] Enabled                                    │   │
│  │  Auto Commit:        [●] Enabled                                    │   │
│  │  Commit Prefix:      [feat]                                         │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─ Circuit Breaker ────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Half-Open Threshold: [2] loops                                     │   │
│  │  Open Threshold:      [3] loops                                     │   │
│  │                                                                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                                         [Save] [Reset to Defaults] [Cancel] │
├─────────────────────────────────────────────────────────────────────────────┤
│  [↑↓]Navigate [Enter]Edit [Tab]Next Section                      [Esc]Back  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Responsive Layout

```go
// internal/tui/layout.go
package tui

import "github.com/charmbracelet/lipgloss"

type Layout struct {
    Width  int
    Height int
}

func NewLayout(w, h int) *Layout {
    return &Layout{Width: w, Height: h}
}

// Breakpoints
func (l *Layout) IsSmall() bool  { return l.Width < 80 }
func (l *Layout) IsMedium() bool { return l.Width >= 80 && l.Width < 120 }
func (l *Layout) IsLarge() bool  { return l.Width >= 120 }

// Panel calculations
func (l *Layout) ContentWidth() int {
    return l.Width - 4 // margins
}

func (l *Layout) ContentHeight() int {
    return l.Height - 6 // header + footer
}

func (l *Layout) HalfWidth() int {
    return (l.ContentWidth() - 1) / 2
}

func (l *Layout) ThirdWidth() int {
    return (l.ContentWidth() - 2) / 3
}

// Adaptive panel count
func (l *Layout) DashboardColumns() int {
    if l.IsSmall() {
        return 1 // Stack vertically
    }
    return 2 // Side by side
}

// Render helpers
func (l *Layout) RenderColumns(panels ...string) string {
    if l.IsSmall() {
        return lipgloss.JoinVertical(lipgloss.Left, panels...)
    }
    return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}
```

---

## Files to Create

| File | Description |
|------|-------------|
| `internal/tui/app.go` | Main TUI app and routing |
| `internal/tui/styles.go` | Shared styles |
| `internal/tui/layout.go` | Responsive layout |
| `internal/tui/dashboard.go` | Dashboard screen |
| `internal/tui/tasks.go` | Task list screen |
| `internal/tui/task_detail.go` | Task detail screen |
| `internal/tui/logs.go` | Log viewer screen |
| `internal/tui/settings.go` | Settings screen |
| `internal/tui/help.go` | Help screen |
| `internal/tui/messages.go` | Tea messages |
| `internal/tui/commands.go` | Tea commands |

## Acceptance Criteria

- [ ] All screens render correctly
- [ ] Dynamic resize works smoothly
- [ ] Keyboard navigation works
- [ ] Log viewer auto-scrolls
- [ ] Task filtering works
- [ ] Settings save correctly
- [ ] Small terminal layout works
- [ ] Large terminal uses space well
