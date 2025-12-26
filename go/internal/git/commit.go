package git

import "fmt"

// StageAll stages all changes
func (g *Git) StageAll() error {
	_, err := g.run("add", "-A")
	return err
}

// StageFiles stages specific files
func (g *Git) StageFiles(files ...string) error {
	args := append([]string{"add"}, files...)
	_, err := g.run(args...)
	return err
}

// Unstage unstages all files
func (g *Git) Unstage() error {
	_, err := g.run("reset", "HEAD")
	return err
}

// Commit creates a commit with the given message
func (g *Git) Commit(message string) error {
	_, err := g.run("commit", "-m", message)
	return err
}

// CommitTask creates a commit for a task
func (g *Git) CommitTask(taskID, taskName string) error {
	message := fmt.Sprintf("feat(%s): %s", taskID, taskName)
	return g.Commit(message)
}

// CommitFeature creates a commit for completing a feature
func (g *Git) CommitFeature(featureID, featureName string) error {
	message := fmt.Sprintf("feat(%s): complete %s", featureID, featureName)
	return g.Commit(message)
}

// GetLastCommitMessage returns the last commit message
func (g *Git) GetLastCommitMessage() (string, error) {
	return g.run("log", "-1", "--pretty=%B")
}

// GetLastCommitHash returns the last commit hash
func (g *Git) GetLastCommitHash() (string, error) {
	return g.run("rev-parse", "HEAD")
}

// GetLastCommitShortHash returns the short hash of the last commit
func (g *Git) GetLastCommitShortHash() (string, error) {
	return g.run("rev-parse", "--short", "HEAD")
}

// AmendCommit amends the last commit with staged changes
func (g *Git) AmendCommit() error {
	_, err := g.run("commit", "--amend", "--no-edit")
	return err
}

// GetLog returns recent commit log
func (g *Git) GetLog(count int) (string, error) {
	return g.run("log", fmt.Sprintf("-%d", count), "--oneline")
}
