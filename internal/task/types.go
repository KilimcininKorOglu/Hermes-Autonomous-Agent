package task

// Status represents the status of a task or feature
type Status string

const (
	StatusNotStarted Status = "NOT_STARTED"
	StatusInProgress Status = "IN_PROGRESS"
	StatusCompleted  Status = "COMPLETED"
	StatusBlocked    Status = "BLOCKED"
	StatusAtRisk     Status = "AT_RISK"
	StatusPaused     Status = "PAUSED"
)

// Priority represents task priority
type Priority string

const (
	PriorityP1 Priority = "P1" // Critical
	PriorityP2 Priority = "P2" // High
	PriorityP3 Priority = "P3" // Medium
	PriorityP4 Priority = "P4" // Low
)

// Feature represents a feature with its tasks
type Feature struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Status            Status   `json:"status"`
	Priority          Priority `json:"priority"`
	Description       string   `json:"description"`
	Overview          string   `json:"overview"`
	Goals             []string `json:"goals"`
	TargetVersion     string   `json:"targetVersion"`
	EstimatedDuration string   `json:"estimatedDuration"`
	PerformanceTarget string   `json:"performanceTarget"`
	RiskAssessment    string   `json:"riskAssessment"`
	Tasks             []Task   `json:"tasks"`
	FilePath          string   `json:"filePath"`
}

// Task represents a single task within a feature
type Task struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Status           Status   `json:"status"`
	Priority         Priority `json:"priority"`
	EstimatedEffort  string   `json:"estimatedEffort"`
	Description      string   `json:"description"`
	TechnicalDetails string   `json:"technicalDetails"`
	FilesToTouch     []string `json:"filesToTouch"`
	Dependencies     []string `json:"dependencies"`
	SuccessCriteria  []string `json:"successCriteria"`
	FeatureID        string   `json:"featureId"`
}

// Progress represents overall task progress
type Progress struct {
	Total      int     `json:"total"`
	Completed  int     `json:"completed"`
	InProgress int     `json:"inProgress"`
	NotStarted int     `json:"notStarted"`
	Blocked    int     `json:"blocked"`
	Percentage float64 `json:"percentage"`
}

// IsComplete returns true if task is completed
func (t *Task) IsComplete() bool {
	return t.Status == StatusCompleted
}

// IsBlocked returns true if task is blocked
func (t *Task) IsBlocked() bool {
	return t.Status == StatusBlocked
}

// CanStart returns true if task can be started
func (t *Task) CanStart(completedTasks map[string]bool) bool {
	if t.Status != StatusNotStarted {
		return false
	}
	for _, dep := range t.Dependencies {
		if !completedTasks[dep] {
			return false
		}
	}
	return true
}
