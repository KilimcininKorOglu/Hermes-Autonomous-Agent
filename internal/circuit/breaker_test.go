package circuit

import (
	"os"
	"testing"
)

func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "hermes-circuit-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewBreaker(t *testing.T) {
	b := New("/test/path")
	if b.basePath != "/test/path" {
		t.Errorf("expected basePath '/test/path', got %s", b.basePath)
	}
}

func TestInitialize(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	if err := b.Initialize(); err != nil {
		t.Fatal(err)
	}

	// State file should exist
	if _, err := os.Stat(b.stateFile); os.IsNotExist(err) {
		t.Error("state file should exist after initialize")
	}
}

func TestGetStateDefault(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	state, err := b.GetState()
	if err != nil {
		t.Fatal(err)
	}

	if state.State != StateClosed {
		t.Errorf("expected default state CLOSED, got %s", state.State)
	}
}

func TestCanExecute(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	can, err := b.CanExecute()
	if err != nil {
		t.Fatal(err)
	}
	if !can {
		t.Error("expected CanExecute = true for CLOSED state")
	}
}

func TestStateTransitions(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Initial state should be CLOSED
	state, _ := b.GetState()
	if state.State != StateClosed {
		t.Errorf("expected CLOSED, got %s", state.State)
	}

	// Add no-progress result - should stay CLOSED
	b.AddLoopResult(false, false, 1)
	state, _ = b.GetState()
	if state.State != StateClosed {
		t.Errorf("after 1 no-progress: expected CLOSED, got %s", state.State)
	}

	// Add second no-progress - should go to HALF_OPEN
	b.AddLoopResult(false, false, 2)
	state, _ = b.GetState()
	if state.State != StateHalfOpen {
		t.Errorf("after 2 no-progress: expected HALF_OPEN, got %s", state.State)
	}

	// Add third no-progress - should go to OPEN
	b.AddLoopResult(false, false, 3)
	state, _ = b.GetState()
	if state.State != StateOpen {
		t.Errorf("after 3 no-progress: expected OPEN, got %s", state.State)
	}

	// Can't execute when OPEN
	can, _ := b.CanExecute()
	if can {
		t.Error("should not be able to execute when OPEN")
	}
}

func TestProgressRecovery(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Get to HALF_OPEN
	b.AddLoopResult(false, false, 1)
	b.AddLoopResult(false, false, 2)

	state, _ := b.GetState()
	if state.State != StateHalfOpen {
		t.Fatalf("expected HALF_OPEN, got %s", state.State)
	}

	// Progress should recover to CLOSED
	b.AddLoopResult(true, false, 3)
	state, _ = b.GetState()
	if state.State != StateClosed {
		t.Errorf("after progress: expected CLOSED, got %s", state.State)
	}
	if state.ConsecutiveNoProgress != 0 {
		t.Errorf("expected ConsecutiveNoProgress = 0, got %d", state.ConsecutiveNoProgress)
	}
}

func TestReset(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Get to OPEN
	b.AddLoopResult(false, false, 1)
	b.AddLoopResult(false, false, 2)
	b.AddLoopResult(false, false, 3)

	state, _ := b.GetState()
	if state.State != StateOpen {
		t.Fatalf("expected OPEN, got %s", state.State)
	}

	// Reset
	if err := b.Reset("Manual reset"); err != nil {
		t.Fatal(err)
	}

	state, _ = b.GetState()
	if state.State != StateClosed {
		t.Errorf("after reset: expected CLOSED, got %s", state.State)
	}
	if state.Reason != "Manual reset" {
		t.Errorf("expected reason 'Manual reset', got %s", state.Reason)
	}
}

func TestShouldHalt(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	halt, _ := b.ShouldHalt()
	if halt {
		t.Error("should not halt when CLOSED")
	}

	// Get to OPEN
	b.AddLoopResult(false, false, 1)
	b.AddLoopResult(false, false, 2)
	b.AddLoopResult(false, false, 3)

	halt, _ = b.ShouldHalt()
	if !halt {
		t.Error("should halt when OPEN")
	}
}

func TestHistory(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Trigger state transitions
	b.AddLoopResult(false, false, 1)
	b.AddLoopResult(false, false, 2) // CLOSED -> HALF_OPEN
	b.AddLoopResult(false, false, 3) // HALF_OPEN -> OPEN

	history, err := b.GetHistory()
	if err != nil {
		t.Fatal(err)
	}

	if len(history) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(history))
	}

	// First transition: CLOSED -> HALF_OPEN
	if history[0].FromState != StateClosed || history[0].ToState != StateHalfOpen {
		t.Errorf("expected CLOSED->HALF_OPEN, got %s->%s", history[0].FromState, history[0].ToState)
	}

	// Second transition: HALF_OPEN -> OPEN
	if history[1].FromState != StateHalfOpen || history[1].ToState != StateOpen {
		t.Errorf("expected HALF_OPEN->OPEN, got %s->%s", history[1].FromState, history[1].ToState)
	}
}

func TestErrorTracking(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Add results with errors (no progress)
	b.AddLoopResult(false, true, 1)
	state, _ := b.GetState()
	if state.ConsecutiveErrors != 1 {
		t.Errorf("expected ConsecutiveErrors = 1, got %d", state.ConsecutiveErrors)
	}

	b.AddLoopResult(false, true, 2)
	state, _ = b.GetState()
	if state.ConsecutiveErrors != 2 {
		t.Errorf("expected ConsecutiveErrors = 2, got %d", state.ConsecutiveErrors)
	}

	// No error resets counter
	b.AddLoopResult(false, false, 3)
	state, _ = b.GetState()
	if state.ConsecutiveErrors != 0 {
		t.Errorf("expected ConsecutiveErrors = 0, got %d", state.ConsecutiveErrors)
	}

	// Progress also resets error counter
	b.AddLoopResult(false, true, 4)
	b.AddLoopResult(true, false, 5)
	state, _ = b.GetState()
	if state.ConsecutiveErrors != 0 {
		t.Errorf("expected ConsecutiveErrors = 0 after progress, got %d", state.ConsecutiveErrors)
	}
}

func TestTotalOpens(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Open circuit
	b.AddLoopResult(false, false, 1)
	b.AddLoopResult(false, false, 2)
	b.AddLoopResult(false, false, 3)

	state, _ := b.GetState()
	if state.TotalOpens != 1 {
		t.Errorf("expected TotalOpens = 1, got %d", state.TotalOpens)
	}

	// Reset and open again
	b.Reset("test reset")
	b.AddLoopResult(false, false, 4)
	b.AddLoopResult(false, false, 5)
	b.AddLoopResult(false, false, 6)

	state, _ = b.GetState()
	if state.TotalOpens != 2 {
		t.Errorf("expected TotalOpens = 2, got %d", state.TotalOpens)
	}
}

func TestConsecutiveErrorLimit(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	b := New(tmpDir)
	b.Initialize()

	// Add errors up to limit (maxConsecutiveErrors = 3)
	b.AddLoopResultWithErrorLimit(false, true, 1, 3)
	state, _ := b.GetState()
	if state.State != StateClosed {
		t.Errorf("expected state CLOSED after 1 error, got %s", state.State)
	}

	b.AddLoopResultWithErrorLimit(false, true, 2, 3)
	state, _ = b.GetState()
	if state.State != StateHalfOpen {
		t.Errorf("expected state HALF_OPEN after 2 errors (no progress), got %s", state.State)
	}

	b.AddLoopResultWithErrorLimit(false, true, 3, 3)
	state, _ = b.GetState()
	if state.State != StateOpen {
		t.Errorf("expected state OPEN after 3 consecutive errors, got %s", state.State)
	}
	if state.ConsecutiveErrors != 3 {
		t.Errorf("expected ConsecutiveErrors = 3, got %d", state.ConsecutiveErrors)
	}
}
