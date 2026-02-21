package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hermes/internal/scheduler"
	"hermes/internal/task"
)

// WorkerStatus represents the status of a parallel worker
type WorkerStatus struct {
	ID        int
	TaskID    string
	TaskName  string
	Status    string // "idle", "running", "completed", "failed"
	Progress  int    // 0-100
	StartTime time.Time
	Duration  time.Duration
}

// ParallelModel is the parallel execution TUI model
type ParallelModel struct {
	basePath    string
	width       int
	height      int
	workers     []WorkerStatus
	maxWorkers  int
	currentBatch int
	totalBatches int
	completed   int
	failed      int
	total       int
	startTime   time.Time
	graph       *scheduler.TaskGraph
	results     []*scheduler.TaskResult
	mu          sync.Mutex
	done        bool
}

// NewParallelModel creates a new parallel execution model
func NewParallelModel(basePath string, maxWorkers int) *ParallelModel {
	workers := make([]WorkerStatus, maxWorkers)
	for i := range workers {
		workers[i] = WorkerStatus{
			ID:     i + 1,
			Status: "idle",
		}
	}

	return &ParallelModel{
		basePath:   basePath,
		maxWorkers: maxWorkers,
		workers:    workers,
		startTime:  time.Now(),
	}
}

// SetSize updates the terminal size
func (m *ParallelModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetGraph sets the task graph for visualization
func (m *ParallelModel) SetGraph(graph *scheduler.TaskGraph) {
	m.graph = graph
	m.total = len(graph.GetAllNodes())
}

// SetBatchInfo sets batch information
func (m *ParallelModel) SetBatchInfo(current, total int) {
	m.currentBatch = current
	m.totalBatches = total
}

// UpdateWorker updates a worker's status
func (m *ParallelModel) UpdateWorker(workerID int, taskID, taskName, status string, progress int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if workerID > 0 && workerID <= len(m.workers) {
		w := &m.workers[workerID-1]
		w.TaskID = taskID
		w.TaskName = taskName
		w.Status = status
		w.Progress = progress
		if status == "running" && w.StartTime.IsZero() {
			w.StartTime = time.Now()
		}
		if status == "completed" || status == "failed" {
			w.Duration = time.Since(w.StartTime)
		}
	}
}

// AddResult adds a task result
func (m *ParallelModel) AddResult(result *scheduler.TaskResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.results = append(m.results, result)
	if result.Success {
		m.completed++
	} else {
		m.failed++
	}
}

// SetDone marks the execution as complete
func (m *ParallelModel) SetDone() {
	m.done = true
}

// Init initializes the model
func (m *ParallelModel) Init() tea.Cmd {
	return tickCmd()
}

// Update handles messages
func (m *ParallelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "p":
			// Pause (future feature)
		}
	case tickMsg:
		// Update durations
		m.mu.Lock()
		for i := range m.workers {
			if m.workers[i].Status == "running" {
				m.workers[i].Duration = time.Since(m.workers[i].StartTime)
			}
		}
		m.mu.Unlock()
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}
	return m, nil
}

// View renders the parallel execution view
func (m *ParallelModel) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var sb strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(0, 1)

	header := headerStyle.Render("HERMES PARALLEL EXECUTION")
	version := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("v2.5.3")
	headerLine := fmt.Sprintf("%s %s", header, version)

	sb.WriteString(headerLine)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", m.width-2))
	sb.WriteString("\n\n")

	// Batch progress
	if m.totalBatches > 0 {
		batchPct := float64(m.currentBatch) / float64(m.totalBatches) * 100
		sb.WriteString(fmt.Sprintf("  Batch %d/%d", m.currentBatch, m.totalBatches))
		sb.WriteString(strings.Repeat(" ", 30))
		sb.WriteString(m.progressBar(batchPct, 20))
		sb.WriteString(fmt.Sprintf(" %.0f%%\n\n", batchPct))
	}

	// Worker status box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(m.width - 6)

	var workerContent strings.Builder
	for _, w := range m.workers {
		icon := "⏸️"
		statusStyle := lipgloss.NewStyle()

		switch w.Status {
		case "idle":
			icon = "⏸️"
			statusStyle = statusStyle.Foreground(lipgloss.Color("241"))
		case "running":
			icon = "🔄"
			statusStyle = statusStyle.Foreground(lipgloss.Color("226"))
		case "completed":
			icon = "✅"
			statusStyle = statusStyle.Foreground(lipgloss.Color("42"))
		case "failed":
			icon = "❌"
			statusStyle = statusStyle.Foreground(lipgloss.Color("196"))
		}

		workerLine := fmt.Sprintf("  %s Worker %d: ", icon, w.ID)

		if w.TaskID != "" {
			taskInfo := fmt.Sprintf("%s - %s", w.TaskID, w.TaskName)
			if len(taskInfo) > 40 {
				taskInfo = taskInfo[:37] + "..."
			}
			workerLine += taskInfo
		} else {
			workerLine += statusStyle.Render("idle")
		}

		// Progress bar for running tasks
		if w.Status == "running" {
			workerLine += "  " + m.progressBar(float64(w.Progress), 15)
			workerLine += fmt.Sprintf(" %d%%", w.Progress)
		}

		// Duration
		if w.Duration > 0 {
			workerLine += fmt.Sprintf("  (%s)", w.Duration.Round(time.Second))
		} else if w.Status == "running" && !w.StartTime.IsZero() {
			workerLine += fmt.Sprintf("  (%s)", time.Since(w.StartTime).Round(time.Second))
		}

		workerContent.WriteString(workerLine)
		workerContent.WriteString("\n")
	}

	sb.WriteString(boxStyle.Render(workerContent.String()))
	sb.WriteString("\n\n")

	// Summary stats
	elapsed := time.Since(m.startTime).Round(time.Second)
	sb.WriteString(fmt.Sprintf("  Completed: %d/%d", m.completed, m.total))
	if m.failed > 0 {
		sb.WriteString(fmt.Sprintf(" | Failed: %d", m.failed))
	}
	sb.WriteString(fmt.Sprintf(" | Elapsed: %s\n", elapsed))

	// Overall progress
	if m.total > 0 {
		overallPct := float64(m.completed) / float64(m.total) * 100
		sb.WriteString("\n  Overall: ")
		sb.WriteString(m.progressBar(overallPct, 30))
		sb.WriteString(fmt.Sprintf(" %.0f%%\n", overallPct))
	}

	// Controls
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", m.width-2))
	sb.WriteString("\n")
	controlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	sb.WriteString(controlStyle.Render("  [q] Quit  [p] Pause"))

	if m.done {
		sb.WriteString("\n\n")
		if m.failed == 0 {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render("  ✓ All tasks completed successfully!"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(fmt.Sprintf("  ✗ Completed with %d failures", m.failed)))
		}
	}

	return sb.String()
}

// progressBar renders a progress bar
func (m *ParallelModel) progressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return "[" + filledStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", empty)) + "]"
}

// GetCompletedCount returns the number of completed tasks
func (m *ParallelModel) GetCompletedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.completed
}

// GetFailedCount returns the number of failed tasks
func (m *ParallelModel) GetFailedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.failed
}

// PrintExecutionPlan prints the execution plan in TUI format
func PrintExecutionPlan(plan *scheduler.ExecutionPlan, maxWorkers int) {
	fmt.Println()
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("📋 EXECUTION PLAN")

	fmt.Println(header)
	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("Total Tasks: %d\n", plan.TotalTasks)
	fmt.Printf("Batches: %d\n", len(plan.Batches))
	fmt.Printf("Max Workers: %d\n\n", maxWorkers)

	for i, batch := range plan.Batches {
		batchHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Render(fmt.Sprintf("Batch %d (%d tasks)", i+1, len(batch)))

		fmt.Println(batchHeader)

		for _, t := range batch {
			priorityStyle := lipgloss.NewStyle()
			switch t.Priority {
			case task.PriorityP1:
				priorityStyle = priorityStyle.Foreground(lipgloss.Color("196"))
			case task.PriorityP2:
				priorityStyle = priorityStyle.Foreground(lipgloss.Color("226"))
			case task.PriorityP3:
				priorityStyle = priorityStyle.Foreground(lipgloss.Color("86"))
			default:
				priorityStyle = priorityStyle.Foreground(lipgloss.Color("241"))
			}

			parallel := "✓"
			if !t.Parallelizable {
				parallel = "✗"
			}

			fmt.Printf("  [%s] %s - %s (parallel: %s)\n",
				t.ID,
				t.Name,
				priorityStyle.Render(string(t.Priority)),
				parallel,
			)

			if len(t.DependsOn) > 0 {
				fmt.Printf("       └─ depends on: %v\n", t.DependsOn)
			}
		}

		if i < len(plan.Batches)-1 {
			fmt.Println("  ↓")
		}
	}
	fmt.Println(strings.Repeat("═", 50))
}
