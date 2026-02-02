package cmd

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"hermes/internal/installer"
)

// NewInstallCmd creates the install command
func NewInstallCmd() *cobra.Command {
	var uninstall bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Hermes to system PATH",
		Long: `Install Hermes binary to system PATH for global access.

Windows: Installs to C:\Program Files\Hermes and adds to user PATH
Linux/macOS: Installs to /usr/local/bin (requires sudo)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if uninstall {
				return uninstallExecute()
			}
			return installExecute()
		},
	}

	cmd.Flags().BoolVar(&uninstall, "uninstall", false, "Uninstall Hermes from system")

	return cmd
}

func installExecute() error {
	inst, err := installer.New()
	if err != nil {
		return err
	}

	// Check for elevation
	if inst.NeedsElevation() {
		if runtime.GOOS == "windows" {
			color.Red("Administrator privileges required!")
			fmt.Println("Please run this command as Administrator.")
		} else {
			color.Red("Root privileges required!")
			fmt.Println("Please run: sudo hermes install")
		}
		return fmt.Errorf("insufficient privileges")
	}

	updating := inst.IsInstalled()
	if updating {
		fmt.Printf("Updating Hermes at: %s\n", inst.GetInstallPath())
	} else {
		fmt.Printf("Installing Hermes to: %s\n", inst.GetInstallPath())
	}

	if err := inst.Install(); err != nil {
		return err
	}

	if updating {
		color.Green("Hermes updated successfully!")
	} else {
		color.Green("Hermes installed successfully!")
	}

	if runtime.GOOS == "windows" {
		fmt.Println("\nPATH has been updated. Please restart your terminal to use 'hermes' globally.")
	} else {
		fmt.Println("\nYou can now use 'hermes' from anywhere.")
	}

	return nil
}

func uninstallExecute() error {
	inst, err := installer.New()
	if err != nil {
		return err
	}

	if !inst.IsInstalled() {
		color.Yellow("Hermes is not installed.")
		return nil
	}

	// Check for elevation
	if inst.NeedsElevation() {
		if runtime.GOOS == "windows" {
			color.Red("Administrator privileges required!")
			fmt.Println("Please run this command as Administrator.")
		} else {
			color.Red("Root privileges required!")
			fmt.Println("Please run: sudo hermes install --uninstall")
		}
		return fmt.Errorf("insufficient privileges")
	}

	fmt.Printf("Uninstalling Hermes from: %s\n", inst.GetInstallPath())

	if err := inst.Uninstall(); err != nil {
		return err
	}

	color.Green("Hermes uninstalled successfully!")

	if runtime.GOOS == "windows" {
		fmt.Println("\nPATH has been updated. Please restart your terminal.")
	}

	return nil
}
