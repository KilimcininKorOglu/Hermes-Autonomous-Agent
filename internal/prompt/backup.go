package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Backup creates a backup of the prompt file
func (i *Injector) Backup() (string, error) {
	content, err := i.Read()
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("prompt_backup_%s.md", timestamp)
	backupPath := filepath.Join(filepath.Dir(i.promptPath), backupName)

	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		return "", err
	}

	return backupPath, nil
}

// Restore restores the prompt from a backup
func (i *Injector) Restore(backupPath string) error {
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	return i.Write(string(content))
}

// GetLatestBackup returns the path to the latest backup
func (i *Injector) GetLatestBackup() (string, error) {
	dir := filepath.Dir(i.promptPath)
	pattern := filepath.Join(dir, "prompt_backup_*.md")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", nil
	}

	// Sort by name (which includes timestamp) descending
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	return matches[0], nil
}

// RestoreLatest restores from the latest backup
func (i *Injector) RestoreLatest() error {
	backup, err := i.GetLatestBackup()
	if err != nil {
		return err
	}

	if backup == "" {
		return fmt.Errorf("no backup found")
	}

	return i.Restore(backup)
}

// ListBackups returns all backups
func (i *Injector) ListBackups() ([]string, error) {
	dir := filepath.Dir(i.promptPath)
	pattern := filepath.Join(dir, "prompt_backup_*.md")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Sort descending (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	return matches, nil
}

// CleanupBackups removes old backups, keeping the specified count
func (i *Injector) CleanupBackups(keepCount int) error {
	backups, err := i.ListBackups()
	if err != nil {
		return err
	}

	if len(backups) <= keepCount {
		return nil
	}

	for _, backup := range backups[keepCount:] {
		if err := os.Remove(backup); err != nil {
			return err
		}
	}

	return nil
}
