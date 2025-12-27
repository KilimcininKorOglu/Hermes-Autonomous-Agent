package scheduler

import (
	"context"
	"testing"
	"time"

	"hermes/internal/task"
)

func TestGraphIntegration(t *testing.T) {
	// Create a more complex task graph
	tasks := []*task.Task{
		{ID: "T001", Name: "Setup Database", Status: task.StatusNotStarted, Priority: task.PriorityP1},
		{ID: "T002", Name: "Create Models", Status: task.StatusNotStarted, Priority: task.PriorityP1, DependsOn: []string{"T001"}},
		{ID: "T003", Name: "Create API", Status: task.StatusNotStarted, Priority: task.PriorityP2, DependsOn: []string{"T002"}},
		{ID: "T004", Name: "Create UI", Status: task.StatusNotStarted, Priority: task.PriorityP2, DependsOn: []string{"T002"}},
		{ID: "T005", Name: "Integration Tests", Status: task.StatusNotStarted, Priority: task.PriorityP3, DependsOn: []string{"T003", "T004"}},
	}

	graph, err := NewTaskGraph(tasks)
	if err != nil {
		t.Fatalf("Failed to create graph: %v", err)
	}

	// Test initial ready tasks
	ready := graph.GetReadyTasks()
	if len(ready) != 1 || ready[0].ID != "T001" {
		t.Errorf("Expected only T001 to be ready initially, got %v", ready)
	}

	// Simulate execution of T001
	graph.MarkRunning("T001")
	if graph.GetRunningCount() != 1 {
		t.Errorf("Expected 1 running task")
	}

	graph.MarkComplete("T001")

	// Now T002 should be ready
	ready = graph.GetReadyTasks()
	if len(ready) != 1 || ready[0].ID != "T002" {
		t.Errorf("Expected T002 to be ready after T001, got %v", ready)
	}

	// Complete T002
	graph.MarkRunning("T002")
	graph.MarkComplete("T002")

	// Now T003 and T004 should be ready (parallel)
	ready = graph.GetReadyTasks()
	if len(ready) != 2 {
		t.Errorf("Expected 2 parallel tasks (T003, T004), got %d", len(ready))
	}

	// Complete both
	graph.MarkRunning("T003")
	graph.MarkRunning("T004")
	graph.MarkComplete("T003")
	graph.MarkComplete("T004")

	// T005 should be ready
	ready = graph.GetReadyTasks()
	if len(ready) != 1 || ready[0].ID != "T005" {
		t.Errorf("Expected T005 to be ready")
	}

	graph.MarkRunning("T005")
	graph.MarkComplete("T005")

	// All complete
	if !graph.IsComplete() {
		t.Error("Graph should be complete")
	}
	if graph.GetCompletedCount() != 5 {
		t.Errorf("Expected 5 completed tasks, got %d", graph.GetCompletedCount())
	}
}

func TestBatchExecution(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted},
		{ID: "T003", Name: "Task 3", Status: task.StatusNotStarted},
		{ID: "T004", Name: "Task 4", Status: task.StatusNotStarted, DependsOn: []string{"T001", "T002", "T003"}},
	}

	graph, _ := NewTaskGraph(tasks)
	batches, err := graph.GetBatches()
	if err != nil {
		t.Fatalf("Failed to get batches: %v", err)
	}

	// Should have 2 batches: [T001, T002, T003] and [T004]
	if len(batches) != 2 {
		t.Errorf("Expected 2 batches, got %d", len(batches))
	}

	if len(batches[0]) != 3 {
		t.Errorf("First batch should have 3 tasks, got %d", len(batches[0]))
	}

	if len(batches[1]) != 1 || batches[1][0].ID != "T004" {
		t.Errorf("Second batch should have only T004")
	}
}

func TestResourceMonitor(t *testing.T) {
	monitor := NewResourceMonitor(100, 50, 10)
	
	// Should be able to make initial calls
	if !monitor.CanMakeAPICall() {
		t.Error("Should be able to make API call initially")
	}

	// Record some calls
	for i := 0; i < 5; i++ {
		monitor.RecordAPICall(0.01)
	}

	stats := monitor.GetStats()
	if stats.TotalAPICalls != 5 {
		t.Errorf("Expected 5 API calls, got %d", stats.TotalAPICalls)
	}

	// Memory check should work
	if !monitor.CheckMemory() {
		t.Error("Memory check should pass with high limit")
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(60) // 60 per minute = 1 per second

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Should be able to acquire immediately
	if err := limiter.Acquire(ctx); err != nil {
		t.Errorf("First acquire should succeed: %v", err)
	}

	// Try acquire should work for a while
	count := 0
	for limiter.TryAcquire() {
		count++
		if count > 100 {
			break
		}
	}

	// Should have acquired some tokens
	if count < 10 {
		t.Errorf("Expected to acquire at least 10 tokens, got %d", count)
	}
}

func TestConflictDetection(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", FilesToTouch: []string{"main.go", "config.go"}},
		{ID: "T002", Name: "Task 2", FilesToTouch: []string{"config.go", "utils.go"}},
		{ID: "T003", Name: "Task 3", FilesToTouch: []string{"tests/main_test.go"}},
	}

	conflicts := DetectFileConflicts(tasks)

	// Only config.go should be a conflict
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}

	if _, ok := conflicts["config.go"]; !ok {
		t.Error("config.go should be detected as a conflict")
	}
}

func TestGroupByConflicts(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", FilesToTouch: []string{"a.go"}},
		{ID: "T002", FilesToTouch: []string{"a.go"}},
		{ID: "T003", FilesToTouch: []string{"b.go"}},
		{ID: "T004", FilesToTouch: []string{"b.go"}},
	}

	groups := GroupByConflicts(tasks)

	// Should group conflicting tasks together
	// T001 and T002 conflict (a.go), T003 and T004 conflict (b.go)
	// So we need at least 2 groups
	if len(groups) < 2 {
		t.Errorf("Expected at least 2 groups, got %d", len(groups))
	}
}

func TestRollback(t *testing.T) {
	// Skip if not in a git repo
	rollback := NewRollback(".")
	if rollback.GetBaseBranch() == "" {
		t.Skip("Not in a git repository")
	}

	// Test snapshot saving
	err := rollback.SaveSnapshot("TEST-001")
	if err != nil {
		t.Errorf("Failed to save snapshot: %v", err)
	}

	if !rollback.HasSnapshots() {
		t.Error("Should have snapshots after saving")
	}

	commit, ok := rollback.GetSnapshot("TEST-001")
	if !ok || commit == "" {
		t.Error("Should be able to retrieve snapshot")
	}
}
