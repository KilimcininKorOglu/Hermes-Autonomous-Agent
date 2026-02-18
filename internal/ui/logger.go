package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
)

// LogLevel represents a logging level
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogSuccess
)

var (
	debugColor   = color.New(color.FgHiBlack)
	infoColor    = color.New(color.FgCyan)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
	successColor = color.New(color.FgGreen)
)

// Logger provides logging to console and file
type Logger struct {
	logFile  *os.File
	logPath  string
	minLevel LogLevel
	debug    bool
	silent   bool // When true, only write to file (for TUI mode)
}

// NewLogger creates a new logger
func NewLogger(basePath string, debug bool) (*Logger, error) {
	logsDir := filepath.Join(basePath, ".hermes", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}

	logPath := filepath.Join(logsDir, "hermes.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	minLevel := LogInfo
	if debug {
		minLevel = LogDebug
	}

	return &Logger{
		logFile:  file,
		logPath:  logPath,
		minLevel: minLevel,
		debug:    debug,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// SetSilent enables or disables console output (for TUI mode)
func (l *Logger) SetSilent(silent bool) {
	l.silent = silent
}

func (l *Logger) log(level LogLevel, levelStr string, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}

	msg := fmt.Sprintf(format, args...)

	// Console output with color (skip if silent mode)
	if !l.silent {
		var c *color.Color
		switch level {
		case LogDebug:
			c = debugColor
		case LogInfo:
			c = infoColor
		case LogWarn:
			c = warnColor
		case LogError:
			c = errorColor
		case LogSuccess:
			c = successColor
		}

		c.Printf("[%s] %s\n", levelStr, msg)
	}

	// File output
	if l.logFile != nil {
		fullTimestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Fprintf(l.logFile, "[%s] [%s] %s\n", fullTimestamp, levelStr, msg)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogDebug, "DEBUG", format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogInfo, "INFO", format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogWarn, "WARN", format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogError, "ERROR", format, args...)
}

// Success logs a success message
func (l *Logger) Success(format string, args ...interface{}) {
	l.log(LogSuccess, "SUCCESS", format, args...)
}

// GetLogPath returns the log file path
func (l *Logger) GetLogPath() string {
	return l.logPath
}
