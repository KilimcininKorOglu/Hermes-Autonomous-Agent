package task

import (
	"os"
	"path/filepath"
	"testing"
)

const testFeatureContent = `# Feature 1: User Authentication

**Feature ID:** F001
**Priority:** P1 - CRITICAL
**Target Version:** v1.0.0
**Estimated Duration:** 2-3 weeks
**Status:** IN_PROGRESS

## Overview

This feature implements user authentication for the application.
It includes login, password hashing, and JWT token management.

## Goals

- Enable secure user authentication
- Implement industry-standard password hashing
- Provide stateless session management with JWT

## Tasks

### T001: Create login endpoint

**Status:** COMPLETED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description

Create the login endpoint that accepts user credentials and returns a JWT token.

#### Technical Details

Use Go standard library for HTTP handling. Follow REST conventions.

#### Files to Touch

- api/auth.go
- handlers/login.go

#### Dependencies

- None

#### Success Criteria

- Endpoint accepts POST /api/login
- Returns JWT token on success

---

### T002: Add password hashing

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

Implement secure password hashing using bcrypt algorithm.

#### Technical Details

Use golang.org/x/crypto/bcrypt package with cost factor of 12.

#### Files to Touch

- utils/crypto.go

#### Dependencies

- T001

#### Success Criteria

- Use bcrypt for hashing
- Hash stored in database

---

### T003: Implement JWT tokens

**Status:** BLOCKED
**Priority:** P2
**Estimated Effort:** 1.5 days

#### Description

Generate and verify JWT tokens for session management.

#### Technical Details

Use github.com/golang-jwt/jwt/v5 for token operations.

#### Files to Touch

- auth/jwt.go

#### Dependencies

- T001, T002

#### Success Criteria

- Generate valid JWT
- Verify JWT signature

## Performance Targets

- Login response time: < 200ms
- Token verification: < 10ms

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Brute force attacks | Medium | High | Rate limiting |
`

func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "hermes-task-test-*")
	if err != nil {
		t.Fatal(err)
	}

	tasksDir := filepath.Join(tmpDir, ".hermes", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test feature file
	featurePath := filepath.Join(tasksDir, "001-user-auth.md")
	if err := os.WriteFile(featurePath, []byte(testFeatureContent), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func TestParseFeature(t *testing.T) {
	feature, err := ParseFeature(testFeatureContent, "test.md")
	if err != nil {
		t.Fatal(err)
	}

	if feature.ID != "F001" {
		t.Errorf("expected ID = F001, got %s", feature.ID)
	}
	if feature.Name != "User Authentication" {
		t.Errorf("expected Name = 'User Authentication', got %s", feature.Name)
	}
	if feature.Status != StatusInProgress {
		t.Errorf("expected Status = IN_PROGRESS, got %s", feature.Status)
	}
	if feature.Priority != PriorityP1 {
		t.Errorf("expected Priority = P1, got %s", feature.Priority)
	}
	if feature.TargetVersion != "v1.0.0" {
		t.Errorf("expected TargetVersion = v1.0.0, got %s", feature.TargetVersion)
	}
	if feature.EstimatedDuration != "2-3 weeks" {
		t.Errorf("expected EstimatedDuration = '2-3 weeks', got %s", feature.EstimatedDuration)
	}
	if feature.Overview == "" {
		t.Error("expected Overview to be populated")
	}
	if len(feature.Goals) != 3 {
		t.Errorf("expected 3 goals, got %d", len(feature.Goals))
	}
	if feature.PerformanceTarget == "" {
		t.Error("expected PerformanceTarget to be populated")
	}
	if len(feature.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(feature.Tasks))
	}
}

func TestParseTasks(t *testing.T) {
	feature, _ := ParseFeature(testFeatureContent, "test.md")

	// Test first task
	task1 := feature.Tasks[0]
	if task1.ID != "T001" {
		t.Errorf("expected ID = T001, got %s", task1.ID)
	}
	if task1.Name != "Create login endpoint" {
		t.Errorf("expected Name = 'Create login endpoint', got %s", task1.Name)
	}
	if task1.Status != StatusCompleted {
		t.Errorf("expected Status = COMPLETED, got %s", task1.Status)
	}
	if task1.Priority != PriorityP1 {
		t.Errorf("expected Priority = P1, got %s", task1.Priority)
	}
	if task1.EstimatedEffort != "2 days" {
		t.Errorf("expected EstimatedEffort = '2 days', got %s", task1.EstimatedEffort)
	}
	if task1.Description == "" {
		t.Error("expected Description to be populated")
	}
	if task1.TechnicalDetails == "" {
		t.Error("expected TechnicalDetails to be populated")
	}
	if len(task1.FilesToTouch) != 2 {
		t.Errorf("expected 2 files, got %d", len(task1.FilesToTouch))
	}
	if len(task1.SuccessCriteria) != 2 {
		t.Errorf("expected 2 criteria, got %d", len(task1.SuccessCriteria))
	}

	// Test second task with dependencies
	task2 := feature.Tasks[1]
	if len(task2.Dependencies) != 1 || task2.Dependencies[0] != "T001" {
		t.Errorf("expected dependency T001, got %v", task2.Dependencies)
	}

	// Test third task with multiple dependencies
	task3 := feature.Tasks[2]
	if len(task3.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(task3.Dependencies))
	}
	if task3.Status != StatusBlocked {
		t.Errorf("expected Status = BLOCKED, got %s", task3.Status)
	}
}

func TestReader(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	reader := NewReader(tmpDir)

	// Test HasTasks
	if !reader.HasTasks() {
		t.Error("expected HasTasks = true")
	}

	// Test GetFeatureFiles
	files, err := reader.GetFeatureFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	// Test GetAllTasks
	tasks, err := reader.GetAllTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}

	// Test GetTaskByID
	task, err := reader.GetTaskByID("T002")
	if err != nil {
		t.Fatal(err)
	}
	if task == nil {
		t.Error("expected to find T002")
	}
	if task.Name != "Add password hashing" {
		t.Errorf("expected name 'Add password hashing', got %s", task.Name)
	}
}

func TestGetNextTask(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	reader := NewReader(tmpDir)

	// T001 is completed, T002 depends on T001 (satisfied), T003 depends on T001+T002 (not satisfied)
	// So next task should be T002
	next, err := reader.GetNextTask()
	if err != nil {
		t.Fatal(err)
	}
	if next == nil {
		t.Error("expected to find next task")
	}
	if next.ID != "T002" {
		t.Errorf("expected next task = T002, got %s", next.ID)
	}
}

func TestGetProgress(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	reader := NewReader(tmpDir)
	progress, err := reader.GetProgress()
	if err != nil {
		t.Fatal(err)
	}

	if progress.Total != 3 {
		t.Errorf("expected Total = 3, got %d", progress.Total)
	}
	if progress.Completed != 1 {
		t.Errorf("expected Completed = 1, got %d", progress.Completed)
	}
	if progress.NotStarted != 1 {
		t.Errorf("expected NotStarted = 1, got %d", progress.NotStarted)
	}
	if progress.Blocked != 1 {
		t.Errorf("expected Blocked = 1, got %d", progress.Blocked)
	}
}

func TestStatusUpdater(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	updater := NewStatusUpdater(tmpDir)

	// Update T002 status
	err := updater.UpdateTaskStatus("T002", StatusInProgress)
	if err != nil {
		t.Fatal(err)
	}

	// Verify update
	reader := NewReader(tmpDir)
	task, err := reader.GetTaskByID("T002")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusInProgress {
		t.Errorf("expected Status = IN_PROGRESS, got %s", task.Status)
	}
}

func TestCanStart(t *testing.T) {
	completed := map[string]bool{"T001": true}

	task1 := Task{ID: "T002", Status: StatusNotStarted, Dependencies: []string{"T001"}}
	if !task1.CanStart(completed) {
		t.Error("T002 should be able to start (T001 is complete)")
	}

	task2 := Task{ID: "T003", Status: StatusNotStarted, Dependencies: []string{"T001", "T002"}}
	if task2.CanStart(completed) {
		t.Error("T003 should NOT be able to start (T002 is not complete)")
	}

	task3 := Task{ID: "T004", Status: StatusInProgress, Dependencies: nil}
	if task3.CanStart(completed) {
		t.Error("T004 should NOT be able to start (already in progress)")
	}
}
