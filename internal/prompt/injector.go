package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"hermes/internal/task"
)

const (
	TaskSectionStart = "<!-- HERMES_TASK_START -->"
	TaskSectionEnd   = "<!-- HERMES_TASK_END -->"
)

// Injector manages PROMPT.md task injection
type Injector struct {
	basePath   string
	promptPath string
}

// NewInjector creates a new prompt injector
func NewInjector(basePath string) *Injector {
	return &Injector{
		basePath:   basePath,
		promptPath: filepath.Join(basePath, ".hermes", "PROMPT.md"),
	}
}

// GetPromptPath returns the path to PROMPT.md
func (i *Injector) GetPromptPath() string {
	return i.promptPath
}

// Exists checks if PROMPT.md exists
func (i *Injector) Exists() bool {
	_, err := os.Stat(i.promptPath)
	return err == nil
}

// Read reads the prompt content
func (i *Injector) Read() (string, error) {
	data, err := os.ReadFile(i.promptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Write writes the prompt content
func (i *Injector) Write(content string) error {
	dir := filepath.Dir(i.promptPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(i.promptPath, []byte(content), 0644)
}

// AddTask adds a task section to the prompt
func (i *Injector) AddTask(t *task.Task) error {
	content, err := i.Read()
	if err != nil {
		content = ""
	}

	// Remove existing task section
	content = i.removeTaskSection(content)

	// Add new task section
	section := i.generateTaskSection(t)
	if content != "" {
		content = content + "\n\n" + section
	} else {
		content = section
	}

	return i.Write(content)
}

// RemoveTask removes the task section from the prompt
func (i *Injector) RemoveTask() error {
	content, err := i.Read()
	if err != nil {
		return err
	}

	content = i.removeTaskSection(content)
	return i.Write(strings.TrimSpace(content))
}

func (i *Injector) removeTaskSection(content string) string {
	re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(TaskSectionStart) + `.*?` + regexp.QuoteMeta(TaskSectionEnd))
	content = re.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

func (i *Injector) generateTaskSection(t *task.Task) string {
	var sb strings.Builder

	sb.WriteString(TaskSectionStart + "\n")
	sb.WriteString(fmt.Sprintf("## Current Task: %s\n\n", t.ID))
	sb.WriteString(fmt.Sprintf("**Task:** %s: %s\n\n", t.ID, t.Name))

	if t.Priority != "" {
		sb.WriteString(fmt.Sprintf("**Priority:** %s\n\n", t.Priority))
	}

	if len(t.FilesToTouch) > 0 {
		sb.WriteString("**Files to Touch:**\n")
		for _, f := range t.FilesToTouch {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(t.Dependencies) > 0 {
		sb.WriteString("**Dependencies:**\n")
		for _, d := range t.Dependencies {
			sb.WriteString(fmt.Sprintf("- %s\n", d))
		}
		sb.WriteString("\n")
	}

	if len(t.SuccessCriteria) > 0 {
		sb.WriteString("**Success Criteria:**\n")
		for _, c := range t.SuccessCriteria {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("**Instructions:**\n")
	sb.WriteString("1. Implement the task requirements\n")
	sb.WriteString("2. Run tests to verify\n")
	sb.WriteString("3. Output status block when complete\n\n")

	sb.WriteString("**Status Block (output at end):**\n")
	sb.WriteString("```\n")
	sb.WriteString("---HERMES_STATUS---\n")
	sb.WriteString("STATUS: COMPLETE\n")
	sb.WriteString("EXIT_SIGNAL: true\n")
	sb.WriteString("RECOMMENDATION: Move to next task\n")
	sb.WriteString("---END_HERMES_STATUS---\n")
	sb.WriteString("```\n")

	sb.WriteString(TaskSectionEnd)

	return sb.String()
}

// GetCurrentTaskID returns the task ID from the prompt
func (i *Injector) GetCurrentTaskID() (string, error) {
	content, err := i.Read()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`## Current Task: (T\d+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", nil
}

// HasTaskSection checks if the prompt has a task section
func (i *Injector) HasTaskSection() (bool, error) {
	content, err := i.Read()
	if err != nil {
		return false, err
	}

	return strings.Contains(content, TaskSectionStart), nil
}
