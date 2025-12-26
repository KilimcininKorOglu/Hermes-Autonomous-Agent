package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hermes/internal/task"
)

func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "hermes-ui-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewLogger(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger, err := NewLogger(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// Log file should be created
	logPath := logger.GetLogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file should exist")
	}
}

func TestLoggerDebugLevel(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// With debug=false, debug messages shouldn't be logged
	logger, _ := NewLogger(tmpDir, false)
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Close()

	// Read log file
	content, _ := os.ReadFile(logger.GetLogPath())
	if strings.Contains(string(content), "DEBUG") {
		t.Error("debug messages should not be logged when debug=false")
	}
	if !strings.Contains(string(content), "INFO") {
		t.Error("info messages should be logged")
	}
}

func TestLoggerDebugEnabled(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger, _ := NewLogger(tmpDir, true)
	logger.Debug("debug message")
	logger.Close()

	content, _ := os.ReadFile(logger.GetLogPath())
	if !strings.Contains(string(content), "DEBUG") {
		t.Error("debug messages should be logged when debug=true")
	}
}

func TestFormatTaskTable(t *testing.T) {
	tasks := []task.Task{
		{ID: "T001", Name: "Test task 1", Status: task.StatusNotStarted, Priority: task.PriorityP1, FeatureID: "F001"},
		{ID: "T002", Name: "Test task 2", Status: task.StatusInProgress, Priority: task.PriorityP2, FeatureID: "F001"},
	}

	table := FormatTaskTable(tasks)

	if !strings.Contains(table, "T001") {
		t.Error("table should contain task ID")
	}
	if !strings.Contains(table, "Test task 1") {
		t.Error("table should contain task name")
	}
	if !strings.Contains(table, "NOT_STARTED") {
		t.Error("table should contain task status")
	}
}

func TestFormatTaskTableEmpty(t *testing.T) {
	table := FormatTaskTable(nil)
	if table != "No tasks found." {
		t.Errorf("expected 'No tasks found.', got %s", table)
	}
}

func TestFormatProgressBar(t *testing.T) {
	tests := []struct {
		percentage float64
		width      int
		expected   string
	}{
		{0, 10, "[----------] 0.0%"},
		{50, 10, "[#####-----] 50.0%"},
		{100, 10, "[##########] 100.0%"},
		{25, 8, "[##------] 25.0%"},
	}

	for _, tt := range tests {
		result := FormatProgressBar(tt.percentage, tt.width)
		if result != tt.expected {
			t.Errorf("FormatProgressBar(%.1f, %d) = %q, want %q",
				tt.percentage, tt.width, result, tt.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"short", 10, "short     "},
		{"exactly10", 10, "exactly10 "},
		{"this is too long", 10, "this is..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q",
				tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"abc", 5, "abc  "},
		{"abcde", 5, "abcde"},
		{"abcdef", 5, "abcde"},
	}

	for _, tt := range tests {
		result := padRight(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("padRight(%q, %d) = %q, want %q",
				tt.input, tt.width, result, tt.expected)
		}
	}
}

func TestFilterTasksByStatus(t *testing.T) {
	tasks := []task.Task{
		{ID: "T001", Status: task.StatusCompleted},
		{ID: "T002", Status: task.StatusInProgress},
		{ID: "T003", Status: task.StatusCompleted},
		{ID: "T004", Status: task.StatusNotStarted},
	}

	completed := FilterTasksByStatus(tasks, task.StatusCompleted)
	if len(completed) != 2 {
		t.Errorf("expected 2 completed tasks, got %d", len(completed))
	}

	inProgress := FilterTasksByStatus(tasks, task.StatusInProgress)
	if len(inProgress) != 1 {
		t.Errorf("expected 1 in-progress task, got %d", len(inProgress))
	}
}

func TestFilterTasksByPriority(t *testing.T) {
	tasks := []task.Task{
		{ID: "T001", Priority: task.PriorityP1},
		{ID: "T002", Priority: task.PriorityP2},
		{ID: "T003", Priority: task.PriorityP1},
	}

	p1Tasks := FilterTasksByPriority(tasks, task.PriorityP1)
	if len(p1Tasks) != 2 {
		t.Errorf("expected 2 P1 tasks, got %d", len(p1Tasks))
	}
}

func TestFilterTasksByFeature(t *testing.T) {
	tasks := []task.Task{
		{ID: "T001", FeatureID: "F001"},
		{ID: "T002", FeatureID: "F002"},
		{ID: "T003", FeatureID: "F001"},
	}

	f001Tasks := FilterTasksByFeature(tasks, "F001")
	if len(f001Tasks) != 2 {
		t.Errorf("expected 2 F001 tasks, got %d", len(f001Tasks))
	}
}

func TestLoggerAllLevels(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger, _ := NewLogger(tmpDir, true)
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")
	logger.Success("success")
	logger.Close()

	content, _ := os.ReadFile(logger.GetLogPath())
	logContent := string(content)

	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "SUCCESS"}
	for _, level := range levels {
		if !strings.Contains(logContent, level) {
			t.Errorf("log should contain %s level", level)
		}
	}
}

func TestLogsDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	logger, _ := NewLogger(tmpDir, false)
	logger.Close()

	logsDir := filepath.Join(tmpDir, ".hermes", "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("logs directory should be created")
	}
}
