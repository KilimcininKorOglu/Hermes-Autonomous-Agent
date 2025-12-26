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

	// Create/update .gitignore
	createGitignore(filepath.Join(projectPath, ".gitignore"))
	fmt.Println("  Created: .gitignore")

	// Create initial commit
	if err := initialCommit(projectPath); err != nil {
		fmt.Printf("  Warning: Could not create initial commit: %v\n", err)
	} else {
		fmt.Println("  Created: initial commit on main branch")
	}

	fmt.Println("\nHermes initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add your PRD to .hermes/docs/PRD.md")
	fmt.Println("  2. Run: hermes prd .hermes/docs/PRD.md")
	fmt.Println("  3. Run: hermes run --auto-branch --auto-commit")

	return nil
}

func createGitignore(path string) {
	// Check if file exists and has content
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		// Append to existing
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString("\n# Hermes\n.hermes/\n")
		return
	}

	// Create comprehensive .gitignore
	content := `# Hermes
.hermes/

# Dependencies
node_modules/
vendor/
.venv/
venv/
__pycache__/
*.pyc

# Build outputs
dist/
build/
out/
bin/
*.exe
*.dll
*.so
*.dylib

# Environment
.env
.env.local
.env.*.local
*.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/

# Testing
coverage/
.coverage
.nyc_output/

# Misc
*.tmp
*.temp
.cache/
`

	os.WriteFile(path, []byte(content), 0644)
}

func initGit(projectPath string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectPath
	return cmd.Run()
}

func initialCommit(projectPath string) error {
	// Add all files
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = projectPath
	if err := addCmd.Run(); err != nil {
		return err
	}

	// Create initial commit
	commitCmd := exec.Command("git", "commit", "-m", "chore: Initialize project with Hermes")
	commitCmd.Dir = projectPath
	return commitCmd.Run()
}
