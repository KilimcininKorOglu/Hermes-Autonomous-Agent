package scheduler

import (
	"sort"

	"hermes/internal/task"
)

// PriorityOrder defines the order of priorities (lower number = higher priority)
var PriorityOrder = map[task.Priority]int{
	task.PriorityP1: 1,
	task.PriorityP2: 2,
	task.PriorityP3: 3,
	task.PriorityP4: 4,
}

// SortByPriority sorts tasks by priority (P1 first, then P2, etc.)
func SortByPriority(tasks []*task.Task) []*task.Task {
	sorted := make([]*task.Task, len(tasks))
	copy(sorted, tasks)

	sort.Slice(sorted, func(i, j int) bool {
		pi := PriorityOrder[sorted[i].Priority]
		pj := PriorityOrder[sorted[j].Priority]
		if pi != pj {
			return pi < pj
		}
		// If same priority, sort by ID for consistency
		return sorted[i].ID < sorted[j].ID
	})

	return sorted
}

// FilterParallelizable filters tasks that can run in parallel
func FilterParallelizable(tasks []*task.Task) []*task.Task {
	var parallelizable []*task.Task
	for _, t := range tasks {
		// Default to true if not explicitly set
		if t.Parallelizable || len(t.ExclusiveFiles) == 0 {
			parallelizable = append(parallelizable, t)
		}
	}
	return parallelizable
}

// FilterNonParallelizable filters tasks that cannot run in parallel
func FilterNonParallelizable(tasks []*task.Task) []*task.Task {
	var nonParallelizable []*task.Task
	for _, t := range tasks {
		if !t.Parallelizable {
			nonParallelizable = append(nonParallelizable, t)
		}
	}
	return nonParallelizable
}

// SplitByParallelizability splits tasks into parallelizable and non-parallelizable groups
func SplitByParallelizability(tasks []*task.Task) (parallelizable, nonParallelizable []*task.Task) {
	for _, t := range tasks {
		if t.Parallelizable {
			parallelizable = append(parallelizable, t)
		} else {
			nonParallelizable = append(nonParallelizable, t)
		}
	}
	return
}

// DetectFileConflicts detects potential file conflicts between tasks
func DetectFileConflicts(tasks []*task.Task) map[string][]string {
	fileToTasks := make(map[string][]string)

	for _, t := range tasks {
		// Check FilesToTouch
		for _, file := range t.FilesToTouch {
			fileToTasks[file] = append(fileToTasks[file], t.ID)
		}
		// Check ExclusiveFiles
		for _, file := range t.ExclusiveFiles {
			fileToTasks[file] = append(fileToTasks[file], t.ID)
		}
	}

	// Filter to only include files touched by multiple tasks
	conflicts := make(map[string][]string)
	for file, taskIDs := range fileToTasks {
		if len(taskIDs) > 1 {
			conflicts[file] = taskIDs
		}
	}

	return conflicts
}

// GroupByConflicts groups tasks that have file conflicts together
// Returns groups where tasks within each group should not run in parallel
func GroupByConflicts(tasks []*task.Task) [][]*task.Task {
	conflicts := DetectFileConflicts(tasks)
	if len(conflicts) == 0 {
		// No conflicts, all tasks can run together
		return [][]*task.Task{tasks}
	}

	// Build conflict graph
	conflictsWith := make(map[string]map[string]bool)
	for _, t := range tasks {
		conflictsWith[t.ID] = make(map[string]bool)
	}

	for _, taskIDs := range conflicts {
		for i := 0; i < len(taskIDs); i++ {
			for j := i + 1; j < len(taskIDs); j++ {
				conflictsWith[taskIDs[i]][taskIDs[j]] = true
				conflictsWith[taskIDs[j]][taskIDs[i]] = true
			}
		}
	}

	// Greedy coloring to group non-conflicting tasks
	taskByID := make(map[string]*task.Task)
	for _, t := range tasks {
		taskByID[t.ID] = t
	}

	var groups [][]*task.Task
	assigned := make(map[string]bool)

	// Sort tasks by number of conflicts (most conflicting first)
	sortedTasks := make([]*task.Task, len(tasks))
	copy(sortedTasks, tasks)
	sort.Slice(sortedTasks, func(i, j int) bool {
		return len(conflictsWith[sortedTasks[i].ID]) > len(conflictsWith[sortedTasks[j].ID])
	})

	for _, t := range sortedTasks {
		if assigned[t.ID] {
			continue
		}

		// Try to add to existing group
		addedToGroup := false
		for i := range groups {
			canAdd := true
			for _, existing := range groups[i] {
				if conflictsWith[t.ID][existing.ID] {
					canAdd = false
					break
				}
			}
			if canAdd {
				groups[i] = append(groups[i], t)
				assigned[t.ID] = true
				addedToGroup = true
				break
			}
		}

		// Create new group if couldn't add to existing
		if !addedToGroup {
			groups = append(groups, []*task.Task{t})
			assigned[t.ID] = true
		}
	}

	return groups
}

// EstimateParallelTime estimates execution time with parallel execution
func EstimateParallelTime(tasks []*task.Task, workers int) string {
	if len(tasks) == 0 {
		return "0s"
	}

	// Simple estimation: assume each task takes 10 minutes on average
	avgTaskTime := 10 // minutes
	batches := (len(tasks) + workers - 1) / workers
	totalMinutes := batches * avgTaskTime

	if totalMinutes < 60 {
		return string(rune(totalMinutes)) + "m"
	}
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	return string(rune(hours)) + "h " + string(rune(minutes)) + "m"
}
