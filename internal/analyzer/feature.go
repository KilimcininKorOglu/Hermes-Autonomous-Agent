package analyzer

import (
	"regexp"
	"strconv"

	"hermes/internal/task"
)

// FeatureAnalyzer analyzes features and tasks
type FeatureAnalyzer struct {
	basePath string
}

// NewFeatureAnalyzer creates a new feature analyzer
func NewFeatureAnalyzer(basePath string) *FeatureAnalyzer {
	return &FeatureAnalyzer{basePath: basePath}
}

// GetHighestFeatureID returns the highest feature ID number
func (a *FeatureAnalyzer) GetHighestFeatureID() (int, error) {
	reader := task.NewReader(a.basePath)
	files, err := reader.GetFeatureFiles()
	if err != nil {
		return 0, err
	}

	highest := 0
	re := regexp.MustCompile(`F(\d+)`)

	for _, file := range files {
		feature, err := reader.ReadFeature(file)
		if err != nil {
			continue
		}

		if m := re.FindStringSubmatch(feature.ID); len(m) > 1 {
			if n, _ := strconv.Atoi(m[1]); n > highest {
				highest = n
			}
		}
	}

	return highest, nil
}

// GetHighestTaskID returns the highest task ID number
func (a *FeatureAnalyzer) GetHighestTaskID() (int, error) {
	reader := task.NewReader(a.basePath)
	tasks, err := reader.GetAllTasks()
	if err != nil {
		return 0, err
	}

	highest := 0
	re := regexp.MustCompile(`T(\d+)`)

	for _, t := range tasks {
		if m := re.FindStringSubmatch(t.ID); len(m) > 1 {
			if n, _ := strconv.Atoi(m[1]); n > highest {
				highest = n
			}
		}
	}

	return highest, nil
}

// GetNextIDs returns the next available feature and task IDs
func (a *FeatureAnalyzer) GetNextIDs() (featureID int, taskID int, err error) {
	fid, err := a.GetHighestFeatureID()
	if err != nil {
		return 0, 0, err
	}

	tid, err := a.GetHighestTaskID()
	if err != nil {
		return 0, 0, err
	}

	return fid + 1, tid + 1, nil
}

// GetProgress returns the task completion progress
func (a *FeatureAnalyzer) GetProgress() (completed, total int, err error) {
	reader := task.NewReader(a.basePath)
	tasks, err := reader.GetAllTasks()
	if err != nil {
		return 0, 0, err
	}

	total = len(tasks)
	for _, t := range tasks {
		if t.Status == task.StatusCompleted {
			completed++
		}
	}

	return completed, total, nil
}

// GetProgressPercentage returns the completion percentage
func (a *FeatureAnalyzer) GetProgressPercentage() (float64, error) {
	completed, total, err := a.GetProgress()
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 0, nil
	}

	return float64(completed) / float64(total) * 100, nil
}
