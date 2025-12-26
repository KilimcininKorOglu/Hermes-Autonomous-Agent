package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewLogCmd creates the log command
func NewLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "View hermes logs",
		Long:  "Display logs from .hermes/logs/hermes.log",
		RunE:  runLog,
	}

	cmd.Flags().IntP("lines", "n", 50, "Number of lines to show")
	cmd.Flags().BoolP("follow", "f", false, "Follow log output (like tail -f)")
	cmd.Flags().String("level", "", "Filter by log level (ERROR, WARN, INFO, DEBUG)")

	return cmd
}

func runLog(cmd *cobra.Command, args []string) error {
	lines, _ := cmd.Flags().GetInt("lines")
	follow, _ := cmd.Flags().GetBool("follow")
	level, _ := cmd.Flags().GetString("level")
	level = strings.ToUpper(level)

	logPath := filepath.Join(".hermes", "logs", "hermes.log")

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logPath)
	}

	if follow {
		return followLog(logPath, level)
	}

	return showLog(logPath, lines, level)
}

func showLog(logPath string, numLines int, level string) error {
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if level == "" || strings.Contains(line, "["+level+"]") {
			allLines = append(allLines, line)
		}
	}

	// Show last N lines
	start := len(allLines) - numLines
	if start < 0 {
		start = 0
	}

	for i := start; i < len(allLines); i++ {
		printColoredLine(allLines[i])
	}

	return nil
}

func followLog(logPath string, level string) error {
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Go to end of file
	file.Seek(0, 2)

	fmt.Println("Following log... (Ctrl+C to stop)")
	fmt.Println()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		line = strings.TrimRight(line, "\r\n")
		if level == "" || strings.Contains(line, "["+level+"]") {
			printColoredLine(line)
		}
	}
}

func printColoredLine(line string) {
	if strings.Contains(line, "[ERROR]") {
		color.Red("%s\n", line)
	} else if strings.Contains(line, "[WARN]") {
		color.Yellow("%s\n", line)
	} else if strings.Contains(line, "[SUCCESS]") {
		color.Green("%s\n", line)
	} else if strings.Contains(line, "[DEBUG]") {
		color.HiBlack("%s\n", line)
	} else {
		fmt.Println(line)
	}
}
