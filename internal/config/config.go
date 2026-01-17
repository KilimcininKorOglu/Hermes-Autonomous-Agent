package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Load loads configuration with priority: Project config > Global config > Defaults
func Load(basePath string) (*Config, error) {
	cfg := DefaultConfig()

	// Global config: ~/.hermes/config.json
	if homeDir, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(homeDir, ".hermes", "config.json")
		loadFile(globalPath, cfg)
	}

	// Project config: .hermes/config.json
	projectPath := filepath.Join(basePath, ".hermes", "config.json")
	loadFile(projectPath, cfg)

	return cfg, nil
}

// loadFile loads a config file and merges it into cfg
func loadFile(path string, cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	return v.Unmarshal(cfg)
}

// GetAIForTask returns the AI provider for a given task type
func GetAIForTask(taskType string, override string, cfg *Config) string {
	if override != "" && override != "auto" {
		return override
	}
	if taskType == "planning" {
		return cfg.AI.Planning
	}
	return cfg.AI.Coding
}

// Save writes the configuration to a file
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// EnsureDirectories creates all required directories
func (c *Config) EnsureDirectories(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, c.Paths.HermesDir),
		filepath.Join(basePath, c.Paths.TasksDir),
		filepath.Join(basePath, c.Paths.LogsDir),
		filepath.Join(basePath, c.Paths.DocsDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GetTasksPath returns the absolute path to the tasks directory
func (c *Config) GetTasksPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.TasksDir)
}

// GetLogsPath returns the absolute path to the logs directory
func (c *Config) GetLogsPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.LogsDir)
}

// GetHermesPath returns the absolute path to the .hermes directory
func (c *Config) GetHermesPath(basePath string) string {
	return filepath.Join(basePath, c.Paths.HermesDir)
}
