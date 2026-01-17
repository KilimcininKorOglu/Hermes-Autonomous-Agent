package scheduler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ParallelLogger provides thread-safe logging for parallel task execution
type ParallelLogger struct {
	basePath    string
	mainLog     *LogWriter
	workerLogs  map[int]*LogWriter
	mergeLog    *LogWriter
	mu          sync.RWMutex
	startTime   time.Time
}

// LogWriter wraps a file with thread-safe writing
type LogWriter struct {
	file   *os.File
	mu     sync.Mutex
	prefix string
}

// NewLogWriter creates a new log writer
func NewLogWriter(path, prefix string) (*LogWriter, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &LogWriter{
		file:   file,
		prefix: prefix,
	}, nil
}

// Write writes a message to the log
func (w *LogWriter) Write(format string, args ...interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	
	if w.prefix != "" {
		fmt.Fprintf(w.file, "[%s] [%s] %s\n", timestamp, w.prefix, message)
	} else {
		fmt.Fprintf(w.file, "[%s] %s\n", timestamp, message)
	}
}

// WriteRaw writes a raw message without formatting
func (w *LogWriter) WriteRaw(message string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintln(w.file, message)
}

// Close closes the log file
func (w *LogWriter) Close() error {
	return w.file.Close()
}

// GetWriter returns the underlying io.Writer
func (w *LogWriter) GetWriter() io.Writer {
	return w.file
}

// NewParallelLogger creates a new parallel logger
func NewParallelLogger(basePath string, workers int) (*ParallelLogger, error) {
	logDir := filepath.Join(basePath, ".hermes", "logs", "parallel")

	// Clean old parallel logs
	os.RemoveAll(logDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create parallel log directory: %w", err)
	}

	// Create main log
	mainLog, err := NewLogWriter(filepath.Join(logDir, "hermes-parallel.log"), "MAIN")
	if err != nil {
		return nil, err
	}

	// Create worker logs
	workerLogs := make(map[int]*LogWriter)
	for i := 1; i <= workers; i++ {
		logPath := filepath.Join(logDir, fmt.Sprintf("worker-%d.log", i))
		workerLog, err := NewLogWriter(logPath, fmt.Sprintf("W%d", i))
		if err != nil {
			return nil, err
		}
		workerLogs[i] = workerLog
	}

	// Create merge log
	mergeLog, err := NewLogWriter(filepath.Join(logDir, "merge.log"), "MERGE")
	if err != nil {
		return nil, err
	}

	logger := &ParallelLogger{
		basePath:   basePath,
		mainLog:    mainLog,
		workerLogs: workerLogs,
		mergeLog:   mergeLog,
		startTime:  time.Now(),
	}

	// Log startup
	logger.Main("Parallel execution started with %d workers", workers)

	return logger, nil
}

// Main logs to the main log
func (l *ParallelLogger) Main(format string, args ...interface{}) {
	l.mainLog.Write(format, args...)
}

// Worker logs to a specific worker's log
func (l *ParallelLogger) Worker(workerID int, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if log, ok := l.workerLogs[workerID]; ok {
		log.Write(format, args...)
	}
	// Also log to main log with worker prefix
	l.mainLog.Write("[Worker %d] %s", workerID, fmt.Sprintf(format, args...))
}

// Merge logs to the merge log
func (l *ParallelLogger) Merge(format string, args ...interface{}) {
	l.mergeLog.Write(format, args...)
	l.mainLog.Write("[Merge] %s", fmt.Sprintf(format, args...))
}

// TaskStart logs the start of a task
func (l *ParallelLogger) TaskStart(workerID int, taskID, taskName string) {
	l.Worker(workerID, "Starting task %s: %s", taskID, taskName)
}

// TaskComplete logs the completion of a task
func (l *ParallelLogger) TaskComplete(workerID int, taskID string, duration time.Duration) {
	l.Worker(workerID, "Completed task %s in %v", taskID, duration.Round(time.Second))
}

// TaskFailed logs a task failure
func (l *ParallelLogger) TaskFailed(workerID int, taskID string, err error) {
	l.Worker(workerID, "Task %s failed: %v", taskID, err)
}

// BatchStart logs the start of a batch
func (l *ParallelLogger) BatchStart(batchNum, totalBatches, taskCount int) {
	l.Main("Starting batch %d/%d with %d tasks", batchNum, totalBatches, taskCount)
}

// BatchComplete logs the completion of a batch
func (l *ParallelLogger) BatchComplete(batchNum int, duration time.Duration) {
	l.Main("Completed batch %d in %v", batchNum, duration.Round(time.Second))
}

// ConflictDetected logs a conflict
func (l *ParallelLogger) ConflictDetected(file string, tasks []string, conflictType string) {
	l.Merge("Conflict detected in %s (tasks: %v, type: %s)", file, tasks, conflictType)
}

// ConflictResolved logs a resolution
func (l *ParallelLogger) ConflictResolved(file string, strategy string) {
	l.Merge("Resolved conflict in %s using %s", file, strategy)
}

// ExecutionComplete logs the end of parallel execution
func (l *ParallelLogger) ExecutionComplete(successful, failed int) {
	duration := time.Since(l.startTime)
	l.Main("Parallel execution completed in %v (successful: %d, failed: %d)",
		duration.Round(time.Second), successful, failed)
}

// Close closes all log files
func (l *ParallelLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var lastErr error

	if err := l.mainLog.Close(); err != nil {
		lastErr = err
	}

	for _, log := range l.workerLogs {
		if err := log.Close(); err != nil {
			lastErr = err
		}
	}

	if err := l.mergeLog.Close(); err != nil {
		lastErr = err
	}

	return lastErr
}

// GetLogDirectory returns the parallel log directory
func (l *ParallelLogger) GetLogDirectory() string {
	return filepath.Join(l.basePath, ".hermes", "logs", "parallel")
}

// WriteOutput writes task output to a separate file
func (l *ParallelLogger) WriteOutput(taskID, output string) error {
	outputPath := filepath.Join(l.GetLogDirectory(), fmt.Sprintf("output-%s.log", taskID))
	return os.WriteFile(outputPath, []byte(output), 0644)
}

// GetMainLogPath returns the main log file path
func (l *ParallelLogger) GetMainLogPath() string {
	return filepath.Join(l.GetLogDirectory(), "hermes-parallel.log")
}

// GetWorkerLogPath returns a worker's log file path
func (l *ParallelLogger) GetWorkerLogPath(workerID int) string {
	return filepath.Join(l.GetLogDirectory(), fmt.Sprintf("worker-%d.log", workerID))
}

// GetMergeLogPath returns the merge log file path
func (l *ParallelLogger) GetMergeLogPath() string {
	return filepath.Join(l.GetLogDirectory(), "merge.log")
}
