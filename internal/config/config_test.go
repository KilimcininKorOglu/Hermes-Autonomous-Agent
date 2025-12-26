package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AI.Planning != "claude" {
		t.Errorf("expected AI.Planning = claude, got %s", cfg.AI.Planning)
	}
	if cfg.AI.Coding != "claude" {
		t.Errorf("expected AI.Coding = claude, got %s", cfg.AI.Coding)
	}
	if cfg.AI.Timeout != 300 {
		t.Errorf("expected AI.Timeout = 300, got %d", cfg.AI.Timeout)
	}
	if cfg.AI.PrdTimeout != 1200 {
		t.Errorf("expected AI.PrdTimeout = 1200, got %d", cfg.AI.PrdTimeout)
	}
	if cfg.AI.MaxRetries != 10 {
		t.Errorf("expected AI.MaxRetries = 10, got %d", cfg.AI.MaxRetries)
	}
	if !cfg.AI.StreamOutput {
		t.Error("expected AI.StreamOutput = true")
	}
	if !cfg.TaskMode.AutoBranch {
		t.Error("expected TaskMode.AutoBranch = true")
	}
	if cfg.Paths.TasksDir != ".hermes/tasks" {
		t.Errorf("expected Paths.TasksDir = .hermes/tasks, got %s", cfg.Paths.TasksDir)
	}
}

func TestGetAIForTask(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AI.Planning = "claude"
	cfg.AI.Coding = "claude"

	tests := []struct {
		taskType string
		override string
		expected string
	}{
		{"planning", "", "claude"},
		{"coding", "", "claude"},
		{"planning", "claude", "claude"},
		{"coding", "claude", "claude"},
		{"planning", "auto", "claude"},
		{"coding", "auto", "claude"},
	}

	for _, tt := range tests {
		result := GetAIForTask(tt.taskType, tt.override, cfg)
		if result != tt.expected {
			t.Errorf("GetAIForTask(%s, %s) = %s, want %s",
				tt.taskType, tt.override, result, tt.expected)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "hermes-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .hermes directory
	hermesDir := filepath.Join(tmpDir, ".hermes")
	if err := os.MkdirAll(hermesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config file with custom timeout
	configContent := `{"ai": {"timeout": 600}}`
	configPath := filepath.Join(hermesDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Check that timeout was overridden
	if cfg.AI.Timeout != 600 {
		t.Errorf("expected AI.Timeout = 600, got %d", cfg.AI.Timeout)
	}

	// Check that defaults are still present
	if cfg.AI.Planning != "claude" {
		t.Errorf("expected AI.Planning = claude, got %s", cfg.AI.Planning)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	cfg.AI.Timeout = 999

	configPath := filepath.Join(tmpDir, ".hermes", "config.json")
	if err := Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load and verify
	loadedCfg, err := Load(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if loadedCfg.AI.Timeout != 999 {
		t.Errorf("expected AI.Timeout = 999, got %d", loadedCfg.AI.Timeout)
	}
}

func TestEnsureDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := DefaultConfig()
	if err := cfg.EnsureDirectories(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Check directories exist
	dirs := []string{
		cfg.GetHermesPath(tmpDir),
		cfg.GetTasksPath(tmpDir),
		cfg.GetLogsPath(tmpDir),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("directory not created: %s", dir)
		}
	}
}
