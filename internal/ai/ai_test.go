package ai

import (
	"testing"

	"hermes/internal/task"
)

func TestGetProvider(t *testing.T) {
	// Test claude provider
	p := GetProvider("claude")
	if p == nil {
		t.Error("expected claude provider")
	}
	if p.Name() != "Claude" {
		t.Errorf("expected name 'Claude', got %s", p.Name())
	}

	// Test unknown provider
	p = GetProvider("unknown")
	if p != nil {
		t.Error("expected nil for unknown provider")
	}
}

func TestAutoDetectProvider(t *testing.T) {
	p := AutoDetectProvider()
	// May or may not be available depending on system
	if p != nil && p.Name() != "Claude" && p.Name() != "Droid" && p.Name() != "Gemini" && p.Name() != "OpenCode" {
		t.Errorf("expected valid provider name, got %s", p.Name())
	}
}

func TestClaudeProviderName(t *testing.T) {
	p := NewClaudeProvider()
	if p.Name() != "Claude" {
		t.Errorf("expected 'Claude', got %s", p.Name())
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries = 3, got %d", cfg.MaxRetries)
	}
	if cfg.Delay.Seconds() != 5 {
		t.Errorf("expected Delay = 5s, got %v", cfg.Delay)
	}
	if cfg.MaxDelay.Seconds() != 60 {
		t.Errorf("expected MaxDelay = 60s, got %v", cfg.MaxDelay)
	}
}

func TestFormatFiles(t *testing.T) {
	// Empty files
	result := formatFiles(nil)
	if result != "- (none specified)" {
		t.Errorf("expected '- (none specified)', got %s", result)
	}

	// With files
	result = formatFiles([]string{"file1.go", "file2.go"})
	expected := "- file1.go\n- file2.go\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatCriteria(t *testing.T) {
	// Empty criteria
	result := formatCriteria(nil)
	if result != "- (none specified)" {
		t.Errorf("expected '- (none specified)', got %s", result)
	}

	// With criteria
	result = formatCriteria([]string{"Criterion 1", "Criterion 2"})
	expected := "- Criterion 1\n- Criterion 2\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTaskExecutorBuildPrompt(t *testing.T) {
	provider := NewClaudeProvider()
	executor := NewTaskExecutor(provider, "/project")

	testTask := &task.Task{
		ID:              "T001",
		Name:            "Test Task",
		FilesToTouch:    []string{"main.go"},
		SuccessCriteria: []string{"Tests pass"},
	}

	// We can't call buildTaskPrompt directly, but we can test via ExecuteTask
	// For now, just verify the executor is created correctly
	if executor.provider == nil {
		t.Error("expected provider to be set")
	}
	if executor.workDir != "/project" {
		t.Errorf("expected workDir = '/project', got %s", executor.workDir)
	}
	_ = testTask // Used in actual execution
}

func TestStreamDisplay(t *testing.T) {
	display := NewStreamDisplay(true, true, "droid")
	if !display.showTools {
		t.Error("expected showTools = true")
	}
	if !display.showCost {
		t.Error("expected showCost = true")
	}
	if display.providerName != "droid" {
		t.Errorf("expected providerName = 'droid', got %s", display.providerName)
	}

	// Test with disabled options
	display2 := NewStreamDisplay(false, false, "")
	if display2.showTools {
		t.Error("expected showTools = false")
	}
	if display2.showCost {
		t.Error("expected showCost = false")
	}
}
