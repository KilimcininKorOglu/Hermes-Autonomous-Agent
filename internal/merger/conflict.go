package merger

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ConflictType represents the type of conflict between parallel tasks
type ConflictType int

const (
	ConflictNone ConflictType = iota
	ConflictSameFile           // Both tasks modified the same file
	ConflictSameFunction       // Both tasks modified the same function
	ConflictImport             // Import conflicts
	ConflictSemantic           // Semantic conflicts (e.g., incompatible changes)
)

// String returns the string representation of ConflictType
func (c ConflictType) String() string {
	switch c {
	case ConflictNone:
		return "NONE"
	case ConflictSameFile:
		return "SAME_FILE"
	case ConflictSameFunction:
		return "SAME_FUNCTION"
	case ConflictImport:
		return "IMPORT"
	case ConflictSemantic:
		return "SEMANTIC"
	default:
		return "UNKNOWN"
	}
}

// Severity levels for conflicts
const (
	SeverityLow    = 1 // Can be auto-resolved
	SeverityMedium = 2 // May need review
	SeverityHigh   = 3 // Requires manual resolution
)

// Conflict represents a conflict between parallel task changes
type Conflict struct {
	File            string       // File path with conflict
	Tasks           []string     // Task IDs involved
	Type            ConflictType // Type of conflict
	Severity        int          // 1-3 severity level
	Description     string       // Human-readable description
	LineStart       int          // Starting line of conflict
	LineEnd         int          // Ending line of conflict
	CanAutoResolve  bool         // Whether this can be auto-resolved
	OriginalContent string       // Original file content before changes
	Task1Content    string       // Content after task 1 changes
	Task2Content    string       // Content after task 2 changes
}

// ConflictDetector analyzes changes for potential conflicts
type ConflictDetector struct {
	fileChanges   map[string][]TaskChange // file -> changes by tasks
	taskChanges   map[string][]string     // taskID -> files changed
	conflicts     []Conflict
}

// TaskChange represents changes made by a task to a file
type TaskChange struct {
	TaskID    string
	File      string
	Added     []string // Lines added
	Removed   []string // Lines removed
	Modified  []string // Lines modified
	Functions []string // Functions modified
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		fileChanges: make(map[string][]TaskChange),
		taskChanges: make(map[string][]string),
		conflicts:   make([]Conflict, 0),
	}
}

// AddTaskChanges adds changes made by a task
func (d *ConflictDetector) AddTaskChanges(taskID string, files []string, diffs map[string]string) {
	d.taskChanges[taskID] = files

	for _, file := range files {
		change := TaskChange{
			TaskID: taskID,
			File:   file,
		}

		// Parse diff if available
		if diff, ok := diffs[file]; ok {
			change.Added, change.Removed, change.Modified = parseDiff(diff)
			change.Functions = extractModifiedFunctions(diff)
		}

		d.fileChanges[file] = append(d.fileChanges[file], change)
	}
}

// Analyze detects conflicts between all registered task changes
func (d *ConflictDetector) Analyze() []Conflict {
	d.conflicts = make([]Conflict, 0)

	// Check each file for conflicts
	for file, changes := range d.fileChanges {
		if len(changes) > 1 {
			conflict := d.analyzeFileConflict(file, changes)
			if conflict.Type != ConflictNone {
				d.conflicts = append(d.conflicts, conflict)
			}
		}
	}

	return d.conflicts
}

// analyzeFileConflict analyzes conflicts for a single file
func (d *ConflictDetector) analyzeFileConflict(file string, changes []TaskChange) Conflict {
	taskIDs := make([]string, len(changes))
	for i, c := range changes {
		taskIDs[i] = c.TaskID
	}

	conflict := Conflict{
		File:  file,
		Tasks: taskIDs,
	}

	// Check for function-level conflicts
	functionConflicts := d.detectFunctionConflicts(changes)
	if len(functionConflicts) > 0 {
		conflict.Type = ConflictSameFunction
		conflict.Severity = SeverityHigh
		conflict.Description = fmt.Sprintf("Multiple tasks modified the same functions: %v", functionConflicts)
		conflict.CanAutoResolve = false
		return conflict
	}

	// Check for overlapping line modifications
	if d.hasOverlappingChanges(changes) {
		conflict.Type = ConflictSameFile
		conflict.Severity = SeverityMedium
		conflict.Description = "Multiple tasks modified overlapping sections of the file"
		conflict.CanAutoResolve = false
		return conflict
	}

	// Non-overlapping changes in same file - can be auto-merged
	conflict.Type = ConflictSameFile
	conflict.Severity = SeverityLow
	conflict.Description = "Multiple tasks modified different sections of the file"
	conflict.CanAutoResolve = true

	return conflict
}

// detectFunctionConflicts checks if multiple tasks modified the same functions
func (d *ConflictDetector) detectFunctionConflicts(changes []TaskChange) []string {
	funcCount := make(map[string]int)

	for _, change := range changes {
		for _, fn := range change.Functions {
			funcCount[fn]++
		}
	}

	var conflicts []string
	for fn, count := range funcCount {
		if count > 1 {
			conflicts = append(conflicts, fn)
		}
	}

	return conflicts
}

// hasOverlappingChanges checks if changes overlap
func (d *ConflictDetector) hasOverlappingChanges(changes []TaskChange) bool {
	// Simplified check: if any modified lines appear in multiple changes
	modifiedLines := make(map[string]int)

	for _, change := range changes {
		for _, line := range change.Modified {
			modifiedLines[line]++
		}
	}

	for _, count := range modifiedLines {
		if count > 1 {
			return true
		}
	}

	return false
}

// GetConflicts returns all detected conflicts
func (d *ConflictDetector) GetConflicts() []Conflict {
	return d.conflicts
}

// HasConflicts returns true if there are any conflicts
func (d *ConflictDetector) HasConflicts() bool {
	return len(d.conflicts) > 0
}

// GetHighSeverityConflicts returns conflicts that require manual resolution
func (d *ConflictDetector) GetHighSeverityConflicts() []Conflict {
	var high []Conflict
	for _, c := range d.conflicts {
		if c.Severity == SeverityHigh {
			high = append(high, c)
		}
	}
	return high
}

// GetAutoResolvableConflicts returns conflicts that can be auto-resolved
func (d *ConflictDetector) GetAutoResolvableConflicts() []Conflict {
	var auto []Conflict
	for _, c := range d.conflicts {
		if c.CanAutoResolve {
			auto = append(auto, c)
		}
	}
	return auto
}

// CanAutoResolve checks if a specific conflict can be auto-resolved
func (d *ConflictDetector) CanAutoResolve(c Conflict) bool {
	return c.CanAutoResolve
}

// GetConflictsByFile returns conflicts for a specific file
func (d *ConflictDetector) GetConflictsByFile(file string) []Conflict {
	var fileConflicts []Conflict
	for _, c := range d.conflicts {
		if c.File == file {
			fileConflicts = append(fileConflicts, c)
		}
	}
	return fileConflicts
}

// GetConflictsByTask returns conflicts involving a specific task
func (d *ConflictDetector) GetConflictsByTask(taskID string) []Conflict {
	var taskConflicts []Conflict
	for _, c := range d.conflicts {
		for _, id := range c.Tasks {
			if id == taskID {
				taskConflicts = append(taskConflicts, c)
				break
			}
		}
	}
	return taskConflicts
}

// PrintConflictSummary prints a summary of all conflicts
func (d *ConflictDetector) PrintConflictSummary() {
	if len(d.conflicts) == 0 {
		fmt.Println("[OK] No conflicts detected")
		return
	}

	fmt.Printf("\n[!] %d conflict(s) detected:\n", len(d.conflicts))
	fmt.Println("=======================================")

	for i, c := range d.conflicts {
		status := "[!]"
		if c.CanAutoResolve {
			status = "[OK]"
		} else if c.Severity == SeverityHigh {
			status = "[X]"
		}

		fmt.Printf("\n%d. %s %s\n", i+1, status, filepath.Base(c.File))
		fmt.Printf("   Type: %s | Severity: %d\n", c.Type, c.Severity)
		fmt.Printf("   Tasks: %v\n", c.Tasks)
		fmt.Printf("   %s\n", c.Description)
		if c.CanAutoResolve {
			fmt.Println("   -> Can be auto-resolved")
		}
	}
	fmt.Println("=======================================")
}

// Helper functions

// parseDiff parses a unified diff and extracts added, removed, and modified lines
func parseDiff(diff string) (added, removed, modified []string) {
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added = append(added, strings.TrimPrefix(line, "+"))
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed = append(removed, strings.TrimPrefix(line, "-"))
		}
	}

	// Modified lines are those that appear in both added and removed (after trimming whitespace)
	removedSet := make(map[string]bool)
	for _, r := range removed {
		removedSet[strings.TrimSpace(r)] = true
	}

	for _, a := range added {
		trimmed := strings.TrimSpace(a)
		if removedSet[trimmed] {
			modified = append(modified, trimmed)
		}
	}

	return
}

// extractModifiedFunctions extracts function names from a diff
func extractModifiedFunctions(diff string) []string {
	var functions []string
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		// Look for function declarations in Go
		if strings.Contains(line, "func ") {
			// Extract function name
			parts := strings.Split(line, "func ")
			if len(parts) > 1 {
				funcPart := parts[1]
				if idx := strings.Index(funcPart, "("); idx > 0 {
					funcName := strings.TrimSpace(funcPart[:idx])
					if funcName != "" {
						functions = append(functions, funcName)
					}
				}
			}
		}
	}

	return functions
}
