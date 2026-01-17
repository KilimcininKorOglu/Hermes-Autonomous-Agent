package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"hermes/internal/updater"
)

var updateVersion string

// SetUpdateVersion sets the current version for update command
func SetUpdateVersion(v string) {
	updateVersion = v
}

// GetVersion returns the current version
func GetVersion() string {
	return updateVersion
}

// NewUpdateCmd creates the update command
func NewUpdateCmd() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for updates and update Hermes",
		Long:  "Check GitHub releases for a new version and optionally update the binary.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateExecute(checkOnly)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't download")

	return cmd
}

func updateExecute(checkOnly bool) error {
	u := updater.New(updateVersion)

	fmt.Printf("Current version: %s\n", updateVersion)
	fmt.Println("Checking for updates...")

	release, hasUpdate, err := u.CheckUpdate()
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	if !hasUpdate {
		color.Green("You are running the latest version!")
		return nil
	}

	color.Yellow("New version available: %s", release.TagName)
	fmt.Printf("Release: %s\n", release.Name)
	fmt.Printf("URL: %s\n", release.HTMLURL)

	if checkOnly {
		fmt.Println("\nRun 'hermes update' to download and install the update.")
		return nil
	}

	asset := u.FindAsset(release)
	if asset == nil {
		return fmt.Errorf("no binary found for your platform (%s)", u.GetAssetName())
	}

	fmt.Printf("\nDownloading %s...\n", asset.Name)

	if err := u.DownloadAndReplace(asset); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	color.Green("Successfully updated to %s!", release.TagName)
	fmt.Println("Please restart Hermes to use the new version.")

	return nil
}
