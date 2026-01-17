package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	windowsInstallDir = "C:\\Program Files\\Hermes"
	unixInstallDir    = "/usr/local/bin"
)

// Installer handles binary installation
type Installer struct {
	sourcePath string
}

// New creates a new Installer
func New() (*Installer, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	return &Installer{sourcePath: execPath}, nil
}

// GetInstallDir returns the installation directory for current OS
func (i *Installer) GetInstallDir() string {
	if runtime.GOOS == "windows" {
		return windowsInstallDir
	}
	return unixInstallDir
}

// GetInstallPath returns the full installation path
func (i *Installer) GetInstallPath() string {
	dir := i.GetInstallDir()
	name := "hermes"
	if runtime.GOOS == "windows" {
		name = "hermes.exe"
	}
	return filepath.Join(dir, name)
}

// IsInstalled checks if hermes is already installed
func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.GetInstallPath())
	return err == nil
}

// Install copies the binary to system path
func (i *Installer) Install() error {
	installDir := i.GetInstallDir()
	installPath := i.GetInstallPath()

	// Create install directory if needed (Windows)
	if runtime.GOOS == "windows" {
		if err := os.MkdirAll(installDir, 0755); err != nil {
			return fmt.Errorf("failed to create install directory: %w", err)
		}
	}

	// Copy binary
	if err := i.copyFile(i.sourcePath, installPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// Set executable permission on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(installPath, 0755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	// Add to PATH on Windows
	if runtime.GOOS == "windows" {
		if err := i.addToWindowsPath(installDir); err != nil {
			return fmt.Errorf("failed to add to PATH: %w", err)
		}
	}

	return nil
}

// Uninstall removes the binary from system path
func (i *Installer) Uninstall() error {
	installPath := i.GetInstallPath()

	if !i.IsInstalled() {
		return fmt.Errorf("hermes is not installed")
	}

	if err := os.Remove(installPath); err != nil {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	// Remove directory on Windows if empty
	if runtime.GOOS == "windows" {
		installDir := i.GetInstallDir()
		entries, _ := os.ReadDir(installDir)
		if len(entries) == 0 {
			os.Remove(installDir)
		}

		// Remove from PATH
		i.removeFromWindowsPath(installDir)
	}

	return nil
}

func (i *Installer) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (i *Installer) addToWindowsPath(dir string) error {
	// Get current user PATH
	cmd := exec.Command("powershell", "-Command",
		"[Environment]::GetEnvironmentVariable('Path', 'User')")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	currentPath := strings.TrimSpace(string(output))

	// Check if already in PATH
	paths := strings.Split(currentPath, ";")
	for _, p := range paths {
		if strings.EqualFold(strings.TrimSpace(p), dir) {
			return nil // Already in PATH
		}
	}

	// Add to PATH
	newPath := currentPath
	if newPath != "" {
		newPath += ";"
	}
	newPath += dir

	cmd = exec.Command("powershell", "-Command",
		fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', '%s', 'User')", newPath))
	return cmd.Run()
}

func (i *Installer) removeFromWindowsPath(dir string) error {
	// Get current user PATH
	cmd := exec.Command("powershell", "-Command",
		"[Environment]::GetEnvironmentVariable('Path', 'User')")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	currentPath := strings.TrimSpace(string(output))
	paths := strings.Split(currentPath, ";")

	// Filter out the directory
	var newPaths []string
	for _, p := range paths {
		if !strings.EqualFold(strings.TrimSpace(p), dir) && strings.TrimSpace(p) != "" {
			newPaths = append(newPaths, p)
		}
	}

	newPath := strings.Join(newPaths, ";")

	cmd = exec.Command("powershell", "-Command",
		fmt.Sprintf("[Environment]::SetEnvironmentVariable('Path', '%s', 'User')", newPath))
	return cmd.Run()
}

// NeedsElevation checks if installation requires admin/root privileges
func (i *Installer) NeedsElevation() bool {
	if runtime.GOOS == "windows" {
		// Try to create a test file in Program Files
		testPath := filepath.Join(windowsInstallDir, ".hermes-test")
		os.MkdirAll(windowsInstallDir, 0755)
		f, err := os.Create(testPath)
		if err != nil {
			return true
		}
		f.Close()
		os.Remove(testPath)
		return false
	}

	// Unix: check if we can write to /usr/local/bin
	testPath := filepath.Join(unixInstallDir, ".hermes-test")
	f, err := os.Create(testPath)
	if err != nil {
		return true
	}
	f.Close()
	os.Remove(testPath)
	return false
}
