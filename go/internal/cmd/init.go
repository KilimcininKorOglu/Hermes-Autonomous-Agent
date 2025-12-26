package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"hermes/internal/config"
	"hermes/internal/prompt"
)

// NewInitCmd creates the init subcommand
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize Hermes project",
		Long:  "Create .hermes directory structure and default configuration",
		Example: `  hermes init
  hermes init my-project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}
			return initExecute(projectPath)
		},
	}

	return cmd
}

func initExecute(projectPath string) error {
	// Create project directory if needed
	if projectPath != "." {
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			return err
		}
	}

	absPath, _ := filepath.Abs(projectPath)
	fmt.Printf("Initializing Hermes in: %s\n\n", absPath)

	// Initialize git if not already a git repo
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := initGit(projectPath); err != nil {
			fmt.Printf("  Warning: Could not init git: %v\n", err)
		} else {
			fmt.Println("  Initialized: git repository")
		}
	}

	// Create directory structure
	dirs := []string{
		".hermes",
		".hermes/tasks",
		".hermes/logs",
		".hermes/docs",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
		fmt.Printf("  Created: %s/\n", dir)
	}

	// Create default config
	configPath := filepath.Join(projectPath, ".hermes", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		if err := config.Save(configPath, cfg); err != nil {
			return err
		}
		fmt.Println("  Created: .hermes/config.json")
	}

	// Create default PROMPT.md
	injector := prompt.NewInjector(projectPath)
	if err := injector.CreateDefault(); err != nil {
		return err
	}
	fmt.Println("  Created: .hermes/PROMPT.md")

	// Update .gitignore
	appendToGitignore(filepath.Join(projectPath, ".gitignore"))
	fmt.Println("  Updated: .gitignore")

	fmt.Println("\nHermes initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your PRD to .hermes/docs/PRD.md")
	fmt.Println("  2. Run: hermes prd .hermes/docs/PRD.md")
	fmt.Println("  3. Run: hermes run --auto-branch --auto-commit")

	return nil
}

func appendToGitignore(path string) {
	entries := []string{
		"\n# Hermes",
		".hermes/logs/",
		".hermes/circuit-*.json",
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	for _, entry := range entries {
		f.WriteString(entry + "\n")
	}
}

func initGit(projectPath string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	return cmd.Run()
}
