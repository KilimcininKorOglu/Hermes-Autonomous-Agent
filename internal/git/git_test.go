package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "hermes-git-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	os.WriteFile(testFile, []byte("# Test"), 0644)

	cmd = exec.Command("git", "add", "-A")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestIsRepository(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)
	if !g.IsRepository() {
		t.Error("expected IsRepository = true")
	}

	// Test non-repo directory
	tmpDir, _ := os.MkdirTemp("", "hermes-non-git-*")
	defer os.RemoveAll(tmpDir)

	g2 := New(tmpDir)
	if g2.IsRepository() {
		t.Error("expected IsRepository = false for non-git directory")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)
	branch, err := g.GetCurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	// Should be main or master
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master, got %s", branch)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User Authentication", "user-authentication"},
		{"Add API endpoint", "add-api-endpoint"},
		{"Fix bug #123", "fix-bug-123"},
		{"Feature: New UI", "feature-new-ui"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"special@#$chars", "specialchars"},
	}

	for _, tt := range tests {
		result := SanitizeBranchName(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeBranchName(%q) = %q, want %q",
				tt.input, result, tt.expected)
		}
	}
}

func TestGetTaskBranchName(t *testing.T) {
	tests := []struct {
		taskID   string
		taskName string
		expected string
	}{
		{"T001", "User Authentication", "hermes/T001-user-authentication"},
		{"T002", "Add API endpoint", "hermes/T002-add-api-endpoint"},
		{"T003", "Fix bug #123", "hermes/T003-fix-bug-123"},
	}

	for _, tt := range tests {
		result := GetTaskBranchName(tt.taskID, tt.taskName)
		if result != tt.expected {
			t.Errorf("GetTaskBranchName(%q, %q) = %q, want %q",
				tt.taskID, tt.taskName, result, tt.expected)
		}
	}
}

func TestGetFeatureBranchName(t *testing.T) {
	g := New(".")

	name := g.GetFeatureBranchName("F001", "User Authentication")
	expected := "feature/F001-user-authentication"

	if name != expected {
		t.Errorf("got %q, want %q", name, expected)
	}
}

func TestGetFeatureBranchNameTruncation(t *testing.T) {
	g := New(".")

	// Very long name should be truncated
	longName := "This is a very long feature name that should be truncated"
	name := g.GetFeatureBranchName("F001", longName)

	if len(name) > 50 { // feature/F001- (12) + 30 max
		t.Errorf("branch name too long: %d chars", len(name))
	}

	if !strings.HasPrefix(name, "feature/F001-") {
		t.Errorf("expected prefix 'feature/F001-', got %s", name)
	}
}

func TestIsWorkingTreeClean(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)

	// Should be clean initially
	if !g.IsWorkingTreeClean() {
		t.Error("expected clean working tree")
	}

	// Add untracked file
	testFile := filepath.Join(repoDir, "newfile.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	if g.IsWorkingTreeClean() {
		t.Error("expected dirty working tree after adding file")
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)

	if g.HasUncommittedChanges() {
		t.Error("expected no uncommitted changes initially")
	}

	// Modify tracked file
	testFile := filepath.Join(repoDir, "README.md")
	os.WriteFile(testFile, []byte("# Modified"), 0644)

	if !g.HasUncommittedChanges() {
		t.Error("expected uncommitted changes after modifying file")
	}
}

func TestStageAndCommit(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)

	// Create a new file
	testFile := filepath.Join(repoDir, "newfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Stage all
	if err := g.StageAll(); err != nil {
		t.Fatal(err)
	}

	if !g.HasStagedChanges() {
		t.Error("expected staged changes after StageAll")
	}

	// Commit
	if err := g.Commit("Test commit"); err != nil {
		t.Fatal(err)
	}

	// Should be clean now
	if g.HasUncommittedChanges() {
		t.Error("expected clean after commit")
	}

	// Check commit message
	msg, err := g.GetLastCommitMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "Test commit") {
		t.Errorf("expected commit message 'Test commit', got %s", msg)
	}
}

func TestCommitTask(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)

	// Create and stage a file
	testFile := filepath.Join(repoDir, "task.txt")
	os.WriteFile(testFile, []byte("task content"), 0644)
	g.StageAll()

	// Commit task
	if err := g.CommitTask("T001", "Add login endpoint"); err != nil {
		t.Fatal(err)
	}

	msg, _ := g.GetLastCommitMessage()
	expected := "feat(T001): Add login endpoint"
	if !strings.Contains(msg, expected) {
		t.Errorf("expected message containing %q, got %s", expected, msg)
	}
}

func TestBranchOperations(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)

	// Create feature branch
	branchName, err := g.CreateFeatureBranch("F001", "User Auth")
	if err != nil {
		t.Fatal(err)
	}

	expected := "feature/F001-user-auth"
	if branchName != expected {
		t.Errorf("expected branch %s, got %s", expected, branchName)
	}

	// Verify we're on the branch
	current, _ := g.GetCurrentBranch()
	if current != branchName {
		t.Errorf("expected current branch %s, got %s", branchName, current)
	}

	// Branch should exist
	if !g.BranchExists(branchName) {
		t.Error("expected branch to exist")
	}
}

func TestListBranches(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoDir)

	branches, err := g.ListBranches()
	if err != nil {
		t.Fatal(err)
	}

	if len(branches) == 0 {
		t.Error("expected at least one branch")
	}
}
