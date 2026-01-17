package updater

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current  string
		remote   string
		expected bool
	}{
		{"1.0.0", "v1.0.1", true},
		{"1.0.0", "v1.1.0", true},
		{"1.0.0", "v2.0.0", true},
		{"1.2.0", "v1.2.1", true},
		{"1.2.1", "v1.2.1", false},
		{"1.2.2", "v1.2.1", false},
		{"2.0.0", "v1.9.9", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.2.1", "v1.2.1", false},
	}

	for _, tt := range tests {
		u := New(tt.current)
		result := u.compareVersions(tt.remote)
		if result != tt.expected {
			t.Errorf("compareVersions(%s, %s) = %v, want %v",
				tt.current, tt.remote, result, tt.expected)
		}
	}
}

func TestGetAssetName(t *testing.T) {
	u := New("1.0.0")
	name := u.GetAssetName()

	if name == "" {
		t.Error("GetAssetName returned empty string")
	}

	// Should contain hermes
	if len(name) < 6 || name[:6] != "hermes" {
		t.Errorf("GetAssetName = %s, should start with 'hermes'", name)
	}
}

func TestFindAsset(t *testing.T) {
	u := New("1.0.0")
	expectedName := u.GetAssetName()

	release := &Release{
		TagName: "v1.0.1",
		Assets: []Asset{
			{Name: "hermes-linux-amd64", BrowserDownloadURL: "https://example.com/linux"},
			{Name: "hermes-windows-amd64.exe", BrowserDownloadURL: "https://example.com/windows"},
			{Name: "hermes-darwin-arm64", BrowserDownloadURL: "https://example.com/darwin"},
		},
	}

	asset := u.FindAsset(release)

	// Should find matching asset for current platform
	if asset != nil && asset.Name != expectedName {
		t.Errorf("FindAsset returned wrong asset: got %s, want %s", asset.Name, expectedName)
	}
}

func TestFindAssetNotFound(t *testing.T) {
	u := New("1.0.0")

	release := &Release{
		TagName: "v1.0.1",
		Assets: []Asset{
			{Name: "some-other-file.zip", BrowserDownloadURL: "https://example.com/other"},
		},
	}

	asset := u.FindAsset(release)

	if asset != nil {
		t.Error("FindAsset should return nil when no matching asset")
	}
}
