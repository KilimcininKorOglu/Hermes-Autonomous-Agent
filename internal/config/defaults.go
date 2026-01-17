package config

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			Planning:     "claude",
			Coding:       "claude",
			Timeout:      300,
			PrdTimeout:   1200,
			MaxRetries:   10,
			StreamOutput: true,
		},
		TaskMode: TaskModeConfig{
			AutoBranch:           true,
			AutoCommit:           true,
			Autonomous:           true,
			MaxConsecutiveErrors: 5,
		},
		Loop: LoopConfig{
			MaxCallsPerHour: 100,
			TimeoutMinutes:  15,
			ErrorDelay:      10,
		},
		Paths: PathsConfig{
			HermesDir: ".hermes",
			TasksDir:  ".hermes/tasks",
			LogsDir:   ".hermes/logs",
			DocsDir:   ".hermes/docs",
		},
		Parallel: ParallelConfig{
			Enabled:            false,
			MaxWorkers:         3,
			Strategy:           "branch-per-task",
			ConflictResolution: "ai-assisted",
			IsolatedWorkspaces: true,
			MergeStrategy:      "sequential",
			MaxCostPerHour:     0, // 0 means no limit
			FailureStrategy:    "continue",
			MaxRetries:         2,
		},
	}
}
