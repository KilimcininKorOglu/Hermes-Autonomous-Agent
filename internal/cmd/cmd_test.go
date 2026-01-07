package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateGitignore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-cmd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	// Test creating new .gitignore
	createGitignore(gitignorePath)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "# Hermes") {
		t.Error("expected .gitignore to contain '# Hermes'")
	}
	if !strings.Contains(string(content), ".hermes/") {
		t.Error("expected .gitignore to contain '.hermes/'")
	}
	if !strings.Contains(string(content), "node_modules/") {
		t.Error("expected .gitignore to contain 'node_modules/'")
	}
	if !strings.Contains(string(content), ".env") {
		t.Error("expected .gitignore to contain '.env'")
	}
}

func TestCreateGitignoreAppend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-cmd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	// Create existing .gitignore
	os.WriteFile(gitignorePath, []byte("existing content\n"), 0644)

	// Test appending to existing .gitignore
	createGitignore(gitignorePath)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "existing content") {
		t.Error("expected .gitignore to contain original content")
	}
	if !strings.Contains(string(content), "# Hermes") {
		t.Error("expected .gitignore to contain '# Hermes'")
	}
}

func TestBuildPrdPrompt(t *testing.T) {
	prdContent := "This is my PRD content"
	prompt := buildPrdPrompt(prdContent)

	if !strings.Contains(prompt, prdContent) {
		t.Error("expected prompt to contain PRD content")
	}
	if !strings.Contains(prompt, "Feature ID:") {
		t.Error("expected prompt to contain 'Feature ID:' instruction")
	}
	if !strings.Contains(prompt, ".hermes/tasks/") {
		t.Error("expected prompt to contain .hermes/tasks/ path instruction")
	}
}

func TestBuildAddPrompt(t *testing.T) {
	prompt := buildAddPrompt("user authentication", 5, 42)

	if !strings.Contains(prompt, "user authentication") {
		t.Error("expected prompt to contain feature description")
	}
	if !strings.Contains(prompt, "F005") {
		t.Error("expected prompt to contain feature ID F005")
	}
	if !strings.Contains(prompt, "T042") {
		t.Error("expected prompt to contain task ID T042")
	}
}

func TestWriteTaskFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-cmd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Test with file markers
	output := `---FILE: 001-test.md---
# Feature 1: Test
**Feature ID:** F001
---END_FILE---

---FILE: 002-test.md---
# Feature 2: Test
**Feature ID:** F002
---END_FILE---`

	err = writeTaskFiles(output)
	if err != nil {
		t.Fatal(err)
	}

	// Check files were created
	files, _ := filepath.Glob(filepath.Join(tmpDir, ".hermes", "tasks", "*.md"))
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestWriteTaskFilesNoMarkers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-cmd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Test without file markers
	output := `# Feature 1: Test
**Feature ID:** F001`

	err = writeTaskFiles(output)
	if err != nil {
		t.Fatal(err)
	}

	// Check single file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, ".hermes", "tasks", "001-tasks.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Feature ID:") {
		t.Error("expected content to contain Feature ID")
	}
}

func TestWriteFeatureFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-cmd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	output := "# Feature 5: User Auth\n**Feature ID:** F005"
	err = writeFeatureFile(output, 5, "user authentication with jwt")
	if err != nil {
		t.Fatal(err)
	}

	// Check file was created with sanitized name
	files, _ := filepath.Glob(filepath.Join(tmpDir, ".hermes", "tasks", "005-*.md"))
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	content, _ := os.ReadFile(files[0])
	if !strings.Contains(string(content), "F005") {
		t.Error("expected content to contain F005")
	}
}

func TestWriteTaskFilesConsecutiveMarkers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hermes-cmd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Test with consecutive FILE markers (no END_FILE)
	output := `Some intro text from AI---FILE: 001-feature.md---
# Feature 1: Test
**Feature ID:** F001
**Status:** NOT_STARTED

## Tasks

### T001: First Task
---FILE: 002-feature.md---
# Feature 2: Another
**Feature ID:** F002
**Status:** NOT_STARTED

## Tasks

### T002: Second Task`

	err = writeTaskFiles(output)
	if err != nil {
		t.Fatal(err)
	}

	// Check files were created
	files, _ := filepath.Glob(filepath.Join(tmpDir, ".hermes", "tasks", "*.md"))
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	// Check first file content
	content1, _ := os.ReadFile(filepath.Join(tmpDir, ".hermes", "tasks", "001-feature.md"))
	if !strings.Contains(string(content1), "F001") {
		t.Error("expected first file to contain F001")
	}
	if !strings.Contains(string(content1), "T001") {
		t.Error("expected first file to contain T001")
	}

	// Check second file content
	content2, _ := os.ReadFile(filepath.Join(tmpDir, ".hermes", "tasks", "002-feature.md"))
	if !strings.Contains(string(content2), "F002") {
		t.Error("expected second file to contain F002")
	}
}
