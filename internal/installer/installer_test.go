package installer

import (
	"runtime"
	"testing"
)

func TestGetInstallDir(t *testing.T) {
	inst := &Installer{sourcePath: "/tmp/hermes"}

	dir := inst.GetInstallDir()

	if runtime.GOOS == "windows" {
		if dir != windowsInstallDir {
			t.Errorf("GetInstallDir() = %s, want %s", dir, windowsInstallDir)
		}
	} else {
		if dir != unixInstallDir {
			t.Errorf("GetInstallDir() = %s, want %s", dir, unixInstallDir)
		}
	}
}

func TestGetInstallPath(t *testing.T) {
	inst := &Installer{sourcePath: "/tmp/hermes"}

	path := inst.GetInstallPath()

	if path == "" {
		t.Error("GetInstallPath() returned empty string")
	}

	if runtime.GOOS == "windows" {
		expected := windowsInstallDir + "\\hermes.exe"
		if path != expected {
			t.Errorf("GetInstallPath() = %s, want %s", path, expected)
		}
	} else {
		expected := unixInstallDir + "/hermes"
		if path != expected {
			t.Errorf("GetInstallPath() = %s, want %s", path, expected)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	inst := &Installer{sourcePath: "/tmp/hermes"}

	// Should return false for non-existent path
	// This test depends on hermes not being installed
	// Just verify it doesn't panic
	_ = inst.IsInstalled()
}
