package task

import (
	"os"
	"path/filepath"
	"sort"
)

// Reader reads and parses task files
type Reader struct {
	basePath string
	tasksDir string
}

// NewReader creates a new task reader
func NewReader(basePath string) *Reader {
	return &Reader{
		basePath: basePath,
		tasksDir: filepath.Join(basePath, ".hermes", "tasks"),
	}
}

// HasTasks returns true if tasks directory exists and has files
func (r *Reader) HasTasks() bool {
	files, err := r.GetFeatureFiles()
	return err == nil && len(files) > 0
}

// GetFeatureFiles returns all feature files sorted by name
func (r *Reader) GetFeatureFiles() ([]string, error) {
	// Try both patterns: XXX-*.md and FXXX-*.md
	pattern1 := filepath.Join(r.tasksDir, "[0-9][0-9][0-9]-*.md")
	pattern2 := filepath.Join(r.tasksDir, "F[0-9][0-9][0-9]-*.md")

	files1, _ := filepath.Glob(pattern1)
	files2, _ := filepath.Glob(pattern2)

	// Merge and deduplicate
	fileSet := make(map[string]bool)
	for _, f := range files1 {
		fileSet[f] = true
	}
	for _, f := range files2 {
		fileSet[f] = true
	}

	var files []string
	for f := range fileSet {
		files = append(files, f)
	}

	sort.Strings(files)
	return files, nil
}

// ReadFeature reads and parses a single feature file
func (r *Reader) ReadFeature(filePath string) (*Feature, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseFeature(string(content), filePath)
}

// GetAllFeatures returns all features
func (r *Reader) GetAllFeatures() ([]Feature, error) {
	files, err := r.GetFeatureFiles()
	if err != nil {
		return nil, err
	}

	var features []Feature
	for _, file := range files {
		feature, err := r.ReadFeature(file)
		if err != nil {
			continue
		}
		features = append(features, *feature)
	}
	return features, nil
}

// GetAllTasks returns all tasks from all features
func (r *Reader) GetAllTasks() ([]Task, error) {
	features, err := r.GetAllFeatures()
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, feature := range features {
		tasks = append(tasks, feature.Tasks...)
	}
	return tasks, nil
}

// GetTaskByID finds a task by its ID
func (r *Reader) GetTaskByID(id string) (*Task, error) {
	tasks, err := r.GetAllTasks()
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}

// GetFeatureByID finds a feature by its ID
func (r *Reader) GetFeatureByID(id string) (*Feature, error) {
	features, err := r.GetAllFeatures()
	if err != nil {
		return nil, err
	}
	for _, f := range features {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, nil
}

// IsFeatureComplete checks if all tasks in a feature are completed
func (r *Reader) IsFeatureComplete(featureID string) (bool, error) {
	feature, err := r.GetFeatureByID(featureID)
	if err != nil {
		return false, err
	}
	if feature == nil {
		return false, nil
	}

	if len(feature.Tasks) == 0 {
		return false, nil
	}

	for _, t := range feature.Tasks {
		if t.Status != StatusCompleted {
			return false, nil
		}
	}
	return true, nil
}

// GetTasksByStatus returns all tasks with the given status
func (r *Reader) GetTasksByStatus(status Status) ([]Task, error) {
	tasks, err := r.GetAllTasks()
	if err != nil {
		return nil, err
	}

	var filtered []Task
	for _, t := range tasks {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// GetNextTask returns the next task to work on
func (r *Reader) GetNextTask() (*Task, error) {
	tasks, err := r.GetAllTasks()
	if err != nil {
		return nil, err
	}

	// Build completed tasks map
	completed := make(map[string]bool)
	for _, t := range tasks {
		if t.Status == StatusCompleted {
			completed[t.ID] = true
		}
	}

	// Find first available task by priority
	var candidates []Task
	for _, t := range tasks {
		if t.CanStart(completed) {
			candidates = append(candidates, t)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Sort by priority
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})

	return &candidates[0], nil
}

// GetProgress calculates overall progress
func (r *Reader) GetProgress() (*Progress, error) {
	tasks, err := r.GetAllTasks()
	if err != nil {
		return nil, err
	}

	p := &Progress{Total: len(tasks)}
	for _, t := range tasks {
		switch t.Status {
		case StatusCompleted:
			p.Completed++
		case StatusInProgress:
			p.InProgress++
		case StatusNotStarted:
			p.NotStarted++
		case StatusBlocked:
			p.Blocked++
		}
	}

	if p.Total > 0 {
		p.Percentage = float64(p.Completed) / float64(p.Total) * 100
	}

	return p, nil
}
