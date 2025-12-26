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
	}
}
