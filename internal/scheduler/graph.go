package scheduler

import (
	"fmt"

	"hermes/internal/task"
)

// NodeStatus represents the execution status of a task node
type NodeStatus int

const (
	NodePending NodeStatus = iota
	NodeReady
	NodeRunning
	NodeCompleted
	NodeFailed
)

// String returns the string representation of NodeStatus
func (s NodeStatus) String() string {
	switch s {
	case NodePending:
		return "PENDING"
	case NodeReady:
		return "READY"
	case NodeRunning:
		return "RUNNING"
	case NodeCompleted:
		return "COMPLETED"
	case NodeFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// TaskNode represents a node in the task dependency graph
type TaskNode struct {
	Task       *task.Task
	InDegree   int      // number of unfinished dependencies
	Dependents []string // task IDs that depend on this task
	Status     NodeStatus
}

// TaskGraph represents a directed acyclic graph of task dependencies
type TaskGraph struct {
	nodes map[string]*TaskNode
	edges map[string][]string // task -> its dependencies
}

// NewTaskGraph creates a new task graph from a list of tasks
func NewTaskGraph(tasks []*task.Task) (*TaskGraph, error) {
	g := &TaskGraph{
		nodes: make(map[string]*TaskNode),
		edges: make(map[string][]string),
	}

	// Create nodes for all tasks
	for _, t := range tasks {
		g.nodes[t.ID] = &TaskNode{
			Task:       t,
			InDegree:   0,
			Dependents: []string{},
			Status:     NodePending,
		}
	}

	// Build edges and calculate in-degrees
	for _, t := range tasks {
		deps := t.DependsOn
		// Also include legacy Dependencies field for backward compatibility
		if len(deps) == 0 {
			deps = t.Dependencies
		}

		g.edges[t.ID] = deps

		for _, depID := range deps {
			if _, exists := g.nodes[depID]; !exists {
				return nil, fmt.Errorf("task %s depends on non-existent task %s", t.ID, depID)
			}
			g.nodes[t.ID].InDegree++
			g.nodes[depID].Dependents = append(g.nodes[depID].Dependents, t.ID)
		}
	}

	// Check for cycles
	if g.HasCycle() {
		return nil, fmt.Errorf("circular dependency detected in task graph")
	}

	// Mark tasks with no dependencies as ready
	for _, node := range g.nodes {
		if node.InDegree == 0 && node.Task.Status != task.StatusCompleted {
			node.Status = NodeReady
		} else if node.Task.Status == task.StatusCompleted {
			node.Status = NodeCompleted
		}
	}

	return g, nil
}

// GetReadyTasks returns tasks that are ready to be executed (no pending dependencies)
func (g *TaskGraph) GetReadyTasks() []*task.Task {
	var ready []*task.Task
	for _, node := range g.nodes {
		if node.Status == NodeReady {
			ready = append(ready, node.Task)
		}
	}
	return ready
}

// GetPendingCount returns the number of pending tasks
func (g *TaskGraph) GetPendingCount() int {
	count := 0
	for _, node := range g.nodes {
		if node.Status == NodePending || node.Status == NodeReady {
			count++
		}
	}
	return count
}

// GetRunningCount returns the number of running tasks
func (g *TaskGraph) GetRunningCount() int {
	count := 0
	for _, node := range g.nodes {
		if node.Status == NodeRunning {
			count++
		}
	}
	return count
}

// GetCompletedCount returns the number of completed tasks
func (g *TaskGraph) GetCompletedCount() int {
	count := 0
	for _, node := range g.nodes {
		if node.Status == NodeCompleted {
			count++
		}
	}
	return count
}

// MarkRunning marks a task as running
func (g *TaskGraph) MarkRunning(taskID string) error {
	node, exists := g.nodes[taskID]
	if !exists {
		return fmt.Errorf("task %s not found in graph", taskID)
	}
	if node.Status != NodeReady {
		return fmt.Errorf("task %s is not ready (status: %s)", taskID, node.Status)
	}
	node.Status = NodeRunning
	return nil
}

// MarkComplete marks a task as completed and updates dependent tasks
func (g *TaskGraph) MarkComplete(taskID string) error {
	node, exists := g.nodes[taskID]
	if !exists {
		return fmt.Errorf("task %s not found in graph", taskID)
	}

	node.Status = NodeCompleted

	// Decrement in-degree of dependent tasks
	for _, depID := range node.Dependents {
		depNode := g.nodes[depID]
		depNode.InDegree--
		if depNode.InDegree == 0 && depNode.Status == NodePending {
			depNode.Status = NodeReady
		}
	}

	return nil
}

// MarkFailed marks a task as failed
func (g *TaskGraph) MarkFailed(taskID string) error {
	node, exists := g.nodes[taskID]
	if !exists {
		return fmt.Errorf("task %s not found in graph", taskID)
	}
	node.Status = NodeFailed
	return nil
}

// HasCycle detects circular dependencies using DFS
func (g *TaskGraph) HasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycleDFS func(taskID string) bool
	hasCycleDFS = func(taskID string) bool {
		visited[taskID] = true
		recStack[taskID] = true

		for _, depID := range g.edges[taskID] {
			if !visited[depID] {
				if hasCycleDFS(depID) {
					return true
				}
			} else if recStack[depID] {
				return true
			}
		}

		recStack[taskID] = false
		return false
	}

	for taskID := range g.nodes {
		if !visited[taskID] {
			if hasCycleDFS(taskID) {
				return true
			}
		}
	}

	return false
}

// TopologicalSort returns tasks in valid execution order
func (g *TaskGraph) TopologicalSort() ([]*task.Task, error) {
	inDegree := make(map[string]int)
	for id, node := range g.nodes {
		inDegree[id] = node.InDegree
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []*task.Task
	for len(queue) > 0 {
		// Pop front
		taskID := queue[0]
		queue = queue[1:]

		sorted = append(sorted, g.nodes[taskID].Task)

		// Reduce in-degree of dependents
		for _, depID := range g.nodes[taskID].Dependents {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, depID)
			}
		}
	}

	if len(sorted) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected: could not complete topological sort")
	}

	return sorted, nil
}

// GetBatches returns tasks grouped by execution batch (parallelizable groups)
func (g *TaskGraph) GetBatches() ([][]*task.Task, error) {
	inDegree := make(map[string]int)
	for id, node := range g.nodes {
		inDegree[id] = node.InDegree
	}

	var batches [][]*task.Task
	remaining := len(g.nodes)

	for remaining > 0 {
		var batch []*task.Task

		// Find all tasks with in-degree 0
		for id, deg := range inDegree {
			if deg == 0 && g.nodes[id].Status != NodeCompleted {
				batch = append(batch, g.nodes[id].Task)
			}
		}

		if len(batch) == 0 && remaining > 0 {
			return nil, fmt.Errorf("cycle detected or all tasks blocked")
		}

		batches = append(batches, batch)

		// Remove this batch from consideration
		for _, t := range batch {
			inDegree[t.ID] = -1 // Mark as processed
			remaining--

			// Reduce in-degree of dependents
			for _, depID := range g.nodes[t.ID].Dependents {
				if inDegree[depID] > 0 {
					inDegree[depID]--
				}
			}
		}
	}

	return batches, nil
}

// GetNode returns a node by task ID
func (g *TaskGraph) GetNode(taskID string) (*TaskNode, bool) {
	node, exists := g.nodes[taskID]
	return node, exists
}

// GetAllNodes returns all nodes in the graph
func (g *TaskGraph) GetAllNodes() map[string]*TaskNode {
	return g.nodes
}

// IsComplete returns true if all tasks are completed
func (g *TaskGraph) IsComplete() bool {
	for _, node := range g.nodes {
		if node.Status != NodeCompleted {
			return false
		}
	}
	return true
}

// HasFailures returns true if any task has failed
func (g *TaskGraph) HasFailures() bool {
	for _, node := range g.nodes {
		if node.Status == NodeFailed {
			return true
		}
	}
	return false
}
