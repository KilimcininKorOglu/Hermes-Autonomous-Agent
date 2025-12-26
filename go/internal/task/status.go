package task

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// StatusUpdater updates task status in files
type StatusUpdater struct {
	basePath string
}

// NewStatusUpdater creates a new status updater
func NewStatusUpdater(basePath string) *StatusUpdater {
	return &StatusUpdater{basePath: basePath}
}

// UpdateTaskStatus updates the status of a task in its feature file
func (u *StatusUpdater) UpdateTaskStatus(taskID string, newStatus Status) error {
	reader := NewReader(u.basePath)
	files, err := reader.GetFeatureFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, taskID+":") {
			continue
		}

		updated := updateTaskStatusInContent(contentStr, taskID, newStatus)
		return os.WriteFile(file, []byte(updated), 0644)
	}

	return fmt.Errorf("task %s not found", taskID)
}

// UpdateFeatureStatus updates the status of a feature
func (u *StatusUpdater) UpdateFeatureStatus(featureID string, newStatus Status) error {
	reader := NewReader(u.basePath)
	features, err := reader.GetAllFeatures()
	if err != nil {
		return err
	}

	for _, f := range features {
		if f.ID != featureID {
			continue
		}

		content, err := os.ReadFile(f.FilePath)
		if err != nil {
			return err
		}

		updated := updateFeatureStatusInContent(string(content), newStatus)
		return os.WriteFile(f.FilePath, []byte(updated), 0644)
	}

	return fmt.Errorf("feature %s not found", featureID)
}

func updateTaskStatusInContent(content, taskID string, newStatus Status) string {
	lines := strings.Split(content, "\n")
	var result []string
	inTask := false
	statusUpdated := false

	taskPattern := regexp.MustCompile(`^###\s*` + regexp.QuoteMeta(taskID) + `:`)

	for _, line := range lines {
		// Check if we're entering the target task
		if taskPattern.MatchString(line) {
			inTask = true
			statusUpdated = false
		} else if strings.HasPrefix(line, "### T") {
			// Entering a different task
			inTask = false
		}

		// Update status line if in target task
		if inTask && !statusUpdated && strings.Contains(line, "**Status:**") {
			line = "**Status:** " + string(newStatus)
			statusUpdated = true
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func updateFeatureStatusInContent(content string, newStatus Status) string {
	// Find the first **Status:** line (feature status, not task status)
	lines := strings.Split(content, "\n")
	var result []string
	statusUpdated := false

	for _, line := range lines {
		// Only update before first task
		if strings.HasPrefix(line, "### T") {
			statusUpdated = true // Don't update after tasks start
		}

		if !statusUpdated && strings.Contains(line, "**Status:**") {
			line = "**Status:** " + string(newStatus)
			statusUpdated = true
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// MarkTaskInProgress marks a task as in progress
func (u *StatusUpdater) MarkTaskInProgress(taskID string) error {
	return u.UpdateTaskStatus(taskID, StatusInProgress)
}

// MarkTaskCompleted marks a task as completed
func (u *StatusUpdater) MarkTaskCompleted(taskID string) error {
	return u.UpdateTaskStatus(taskID, StatusCompleted)
}

// MarkTaskBlocked marks a task as blocked
func (u *StatusUpdater) MarkTaskBlocked(taskID string) error {
	return u.UpdateTaskStatus(taskID, StatusBlocked)
}
