package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hermes/internal/task"
)

func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "hermes-prompt-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewInjector(t *testing.T) {
	i := NewInjector("/test/path")
	expected := filepath.Join("/test/path", ".hermes", "PROMPT.md")
	if i.promptPath != expected {
		t.Errorf("expected promptPath %s, got %s", expected, i.promptPath)
	}
}

func TestExists(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)

	if i.Exists() {
		t.Error("expected Exists = false before creating file")
	}

	// Create the file
	i.Write("test content")

	if !i.Exists() {
		t.Error("expected Exists = true after creating file")
	}
}

func TestReadWrite(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)

	content := "# Test Prompt\n\nSome content here."
	if err := i.Write(content); err != nil {
		t.Fatal(err)
	}

	read, err := i.Read()
	if err != nil {
		t.Fatal(err)
	}

	if read != content {
		t.Errorf("expected content %q, got %q", content, read)
	}
}

func TestAddTask(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)
	i.Write("# Base Prompt\n\nSome instructions.")

	testTask := &task.Task{
		ID:              "T001",
		Name:            "Implement login",
		Priority:        "P1",
		FilesToTouch:    []string{"auth.go", "login.go"},
		SuccessCriteria: []string{"Login works", "Tests pass"},
	}

	if err := i.AddTask(testTask); err != nil {
		t.Fatal(err)
	}

	content, _ := i.Read()

	// Check that task section exists
	if !strings.Contains(content, TaskSectionStart) {
		t.Error("expected task section start marker")
	}
	if !strings.Contains(content, TaskSectionEnd) {
		t.Error("expected task section end marker")
	}

	// Check task content
	if !strings.Contains(content, "T001") {
		t.Error("expected task ID in content")
	}
	if !strings.Contains(content, "Implement login") {
		t.Error("expected task name in content")
	}
	if !strings.Contains(content, "auth.go") {
		t.Error("expected files to touch in content")
	}
	if !strings.Contains(content, "Login works") {
		t.Error("expected success criteria in content")
	}
}

func TestRemoveTask(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)

	// Create prompt with task section
	content := "# Base Prompt\n\n" + TaskSectionStart + "\nTask content\n" + TaskSectionEnd + "\n\nMore content"
	i.Write(content)

	if err := i.RemoveTask(); err != nil {
		t.Fatal(err)
	}

	newContent, _ := i.Read()

	if strings.Contains(newContent, TaskSectionStart) {
		t.Error("expected task section to be removed")
	}
	if !strings.Contains(newContent, "# Base Prompt") {
		t.Error("expected base content to remain")
	}
}

func TestGetCurrentTaskID(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)

	testTask := &task.Task{
		ID:   "T042",
		Name: "Test task",
	}
	i.AddTask(testTask)

	taskID, err := i.GetCurrentTaskID()
	if err != nil {
		t.Fatal(err)
	}

	if taskID != "T042" {
		t.Errorf("expected task ID T042, got %s", taskID)
	}
}

func TestHasTaskSection(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)
	i.Write("# Simple prompt")

	has, _ := i.HasTaskSection()
	if has {
		t.Error("expected no task section")
	}

	testTask := &task.Task{ID: "T001", Name: "Test"}
	i.AddTask(testTask)

	has, _ = i.HasTaskSection()
	if !has {
		t.Error("expected task section after AddTask")
	}
}

func TestBackupRestore(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)

	originalContent := "# Original Content"
	i.Write(originalContent)

	// Create backup
	backupPath, err := i.Backup()
	if err != nil {
		t.Fatal(err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file should exist")
	}

	// Modify prompt
	i.Write("# Modified Content")

	// Restore
	if err := i.Restore(backupPath); err != nil {
		t.Fatal(err)
	}

	content, _ := i.Read()
	if content != originalContent {
		t.Errorf("expected restored content %q, got %q", originalContent, content)
	}
}

func TestGetLatestBackup(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)
	i.Write("# Content")

	// Create multiple backups
	i.Backup()
	i.Write("# Content 2")
	latestPath, _ := i.Backup()

	// Get latest
	latest, err := i.GetLatestBackup()
	if err != nil {
		t.Fatal(err)
	}

	if latest != latestPath {
		t.Errorf("expected latest backup %s, got %s", latestPath, latest)
	}
}

func TestCreateDefault(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)

	if err := i.CreateDefault(); err != nil {
		t.Fatal(err)
	}

	content, err := i.Read()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, "# Project Instructions") {
		t.Error("expected default template content")
	}

	// Should not overwrite existing
	i.Write("# Custom Content")
	i.CreateDefault()

	content, _ = i.Read()
	if !strings.Contains(content, "# Custom Content") {
		t.Error("CreateDefault should not overwrite existing content")
	}
}

func TestAddTaskReplacesExisting(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)
	i.Write("# Base Prompt")

	// Add first task
	task1 := &task.Task{ID: "T001", Name: "First task"}
	i.AddTask(task1)

	// Add second task (should replace first)
	task2 := &task.Task{ID: "T002", Name: "Second task"}
	i.AddTask(task2)

	content, _ := i.Read()

	// Should only have one task section
	count := strings.Count(content, TaskSectionStart)
	if count != 1 {
		t.Errorf("expected 1 task section, got %d", count)
	}

	// Should have second task, not first
	if strings.Contains(content, "T001") {
		t.Error("should not contain first task ID")
	}
	if !strings.Contains(content, "T002") {
		t.Error("should contain second task ID")
	}
}

func TestCleanupBackups(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	i := NewInjector(tmpDir)
	i.Write("# Content")

	// Create 5 backups with unique names in .hermes directory
	hermesDir := filepath.Join(tmpDir, ".hermes")
	for j := 0; j < 5; j++ {
		backupName := filepath.Join(hermesDir, "prompt_backup_20240101_00000"+string(rune('0'+j))+".md")
		os.WriteFile(backupName, []byte("backup"), 0644)
	}

	backups, _ := i.ListBackups()
	if len(backups) != 5 {
		t.Fatalf("expected 5 backups, got %d", len(backups))
	}

	// Keep only 2
	if err := i.CleanupBackups(2); err != nil {
		t.Fatal(err)
	}

	backups, _ = i.ListBackups()
	if len(backups) != 2 {
		t.Errorf("expected 2 backups after cleanup, got %d", len(backups))
	}
}
