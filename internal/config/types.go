package config

// Config represents the complete Hermes configuration
type Config struct {
	AI       AIConfig       `json:"ai" mapstructure:"ai"`
	TaskMode TaskModeConfig `json:"taskMode" mapstructure:"taskMode"`
	Loop     LoopConfig     `json:"loop" mapstructure:"loop"`
	Paths    PathsConfig    `json:"paths" mapstructure:"paths"`
	Parallel ParallelConfig `json:"parallel" mapstructure:"parallel"`
}

// AIConfig contains AI provider settings
type AIConfig struct {
	Planning     string `json:"planning" mapstructure:"planning"`
	Coding       string `json:"coding" mapstructure:"coding"`
	Timeout      int    `json:"timeout" mapstructure:"timeout"`
	PrdTimeout   int    `json:"prdTimeout" mapstructure:"prdTimeout"`
	MaxRetries   int    `json:"maxRetries" mapstructure:"maxRetries"`
	StreamOutput bool   `json:"streamOutput" mapstructure:"streamOutput"`
}

// TaskModeConfig contains task execution settings
type TaskModeConfig struct {
	AutoBranch           bool `json:"autoBranch" mapstructure:"autoBranch"`
	AutoCommit           bool `json:"autoCommit" mapstructure:"autoCommit"`
	Autonomous           bool `json:"autonomous" mapstructure:"autonomous"`
	MaxConsecutiveErrors int  `json:"maxConsecutiveErrors" mapstructure:"maxConsecutiveErrors"`
}

// LoopConfig contains loop execution settings
type LoopConfig struct {
	MaxCallsPerHour int `json:"maxCallsPerHour" mapstructure:"maxCallsPerHour"`
	TimeoutMinutes  int `json:"timeoutMinutes" mapstructure:"timeoutMinutes"`
	ErrorDelay      int `json:"errorDelay" mapstructure:"errorDelay"`
}

// PathsConfig contains directory paths
type PathsConfig struct {
	HermesDir string `json:"hermesDir" mapstructure:"hermesDir"`
	TasksDir  string `json:"tasksDir" mapstructure:"tasksDir"`
	LogsDir   string `json:"logsDir" mapstructure:"logsDir"`
	DocsDir   string `json:"docsDir" mapstructure:"docsDir"`
}

// ParallelConfig contains parallel execution settings
type ParallelConfig struct {
	Enabled            bool    `json:"enabled" mapstructure:"enabled"`
	MaxWorkers         int     `json:"maxWorkers" mapstructure:"maxWorkers"`
	Strategy           string  `json:"strategy" mapstructure:"strategy"`
	ConflictResolution string  `json:"conflictResolution" mapstructure:"conflictResolution"`
	IsolatedWorkspaces bool    `json:"isolatedWorkspaces" mapstructure:"isolatedWorkspaces"`
	MergeStrategy      string  `json:"mergeStrategy" mapstructure:"mergeStrategy"`
	MaxCostPerHour     float64 `json:"maxCostPerHour" mapstructure:"maxCostPerHour"`
	FailureStrategy    string  `json:"failureStrategy" mapstructure:"failureStrategy"`
	MaxRetries         int     `json:"maxRetries" mapstructure:"maxRetries"`
}
