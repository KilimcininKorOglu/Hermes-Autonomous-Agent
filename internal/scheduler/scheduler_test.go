package scheduler

import (
	"testing"

	"hermes/internal/task"
)

func TestNewTaskGraph(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted},
		{ID: "T003", Name: "Task 3", Status: task.StatusNotStarted, DependsOn: []string{"T001"}},
	}

	graph, err := NewTaskGraph(tasks)
	if err != nil {
		t.Fatalf("Failed to create graph: %v", err)
	}

	// Check nodes were created
	if len(graph.nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.nodes))
	}

	// Check in-degrees
	node1, _ := graph.GetNode("T001")
	node3, _ := graph.GetNode("T003")

	if node1.InDegree != 0 {
		t.Errorf("T001 should have in-degree 0, got %d", node1.InDegree)
	}
	if node3.InDegree != 1 {
		t.Errorf("T003 should have in-degree 1, got %d", node3.InDegree)
	}
}

func TestTaskGraphGetReadyTasks(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted},
		{ID: "T003", Name: "Task 3", Status: task.StatusNotStarted, DependsOn: []string{"T001", "T002"}},
	}

	graph, _ := NewTaskGraph(tasks)
	readyTasks := graph.GetReadyTasks()

	// T001 and T002 should be ready, T003 should not
	if len(readyTasks) != 2 {
		t.Errorf("Expected 2 ready tasks, got %d", len(readyTasks))
	}

	readyIDs := make(map[string]bool)
	for _, task := range readyTasks {
		readyIDs[task.ID] = true
	}

	if !readyIDs["T001"] || !readyIDs["T002"] {
		t.Error("T001 and T002 should be ready")
	}
	if readyIDs["T003"] {
		t.Error("T003 should not be ready")
	}
}

func TestTaskGraphMarkComplete(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted, DependsOn: []string{"T001"}},
	}

	graph, _ := NewTaskGraph(tasks)

	// Mark T001 as running, then complete
	graph.MarkRunning("T001")
	graph.MarkComplete("T001")

	// T002 should now be ready
	readyTasks := graph.GetReadyTasks()
	if len(readyTasks) != 1 {
		t.Errorf("Expected 1 ready task, got %d", len(readyTasks))
	}
	if readyTasks[0].ID != "T002" {
		t.Errorf("Expected T002 to be ready, got %s", readyTasks[0].ID)
	}
}

func TestTaskGraphCycleDetection(t *testing.T) {
	// Create cyclic dependency: T001 -> T002 -> T001
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted, DependsOn: []string{"T002"}},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted, DependsOn: []string{"T001"}},
	}

	_, err := NewTaskGraph(tasks)
	if err == nil {
		t.Error("Expected error for cyclic dependency")
	}
}

func TestTaskGraphTopologicalSort(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted, DependsOn: []string{"T001"}},
		{ID: "T003", Name: "Task 3", Status: task.StatusNotStarted, DependsOn: []string{"T002"}},
	}

	graph, _ := NewTaskGraph(tasks)
	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// Check order: T001 should come before T002, T002 before T003
	positions := make(map[string]int)
	for i, task := range sorted {
		positions[task.ID] = i
	}

	if positions["T001"] > positions["T002"] {
		t.Error("T001 should come before T002")
	}
	if positions["T002"] > positions["T003"] {
		t.Error("T002 should come before T003")
	}
}

func TestTaskGraphGetBatches(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Status: task.StatusNotStarted},
		{ID: "T002", Name: "Task 2", Status: task.StatusNotStarted},
		{ID: "T003", Name: "Task 3", Status: task.StatusNotStarted, DependsOn: []string{"T001", "T002"}},
		{ID: "T004", Name: "Task 4", Status: task.StatusNotStarted, DependsOn: []string{"T003"}},
	}

	graph, _ := NewTaskGraph(tasks)
	batches, err := graph.GetBatches()
	if err != nil {
		t.Fatalf("GetBatches failed: %v", err)
	}

	// Should have 3 batches: [T001, T002], [T003], [T004]
	if len(batches) != 3 {
		t.Errorf("Expected 3 batches, got %d", len(batches))
	}

	// First batch should have 2 tasks
	if len(batches[0]) != 2 {
		t.Errorf("First batch should have 2 tasks, got %d", len(batches[0]))
	}
}

func TestSortByPriority(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", Priority: task.PriorityP3},
		{ID: "T002", Name: "Task 2", Priority: task.PriorityP1},
		{ID: "T003", Name: "Task 3", Priority: task.PriorityP2},
	}

	sorted := SortByPriority(tasks)

	if sorted[0].Priority != task.PriorityP1 {
		t.Error("First task should be P1")
	}
	if sorted[1].Priority != task.PriorityP2 {
		t.Error("Second task should be P2")
	}
	if sorted[2].Priority != task.PriorityP3 {
		t.Error("Third task should be P3")
	}
}

func TestDetectFileConflicts(t *testing.T) {
	tasks := []*task.Task{
		{ID: "T001", Name: "Task 1", FilesToTouch: []string{"file1.go", "file2.go"}},
		{ID: "T002", Name: "Task 2", FilesToTouch: []string{"file2.go", "file3.go"}},
		{ID: "T003", Name: "Task 3", FilesToTouch: []string{"file4.go"}},
	}

	conflicts := DetectFileConflicts(tasks)

	// Only file2.go should be a conflict
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}

	if tasks, ok := conflicts["file2.go"]; ok {
		if len(tasks) != 2 {
			t.Errorf("file2.go should have 2 conflicting tasks, got %d", len(tasks))
		}
	} else {
		t.Error("file2.go should be a conflict")
	}
}
