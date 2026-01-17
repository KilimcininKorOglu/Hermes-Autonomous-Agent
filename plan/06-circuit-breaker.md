# Phase 06: Circuit Breaker

## Goal

Implement circuit breaker pattern to detect stagnation and prevent infinite loops.

## PowerShell Reference

```powershell
# From lib/CircuitBreaker.ps1
# States: CLOSED (normal), HALF_OPEN (monitoring), OPEN (halted)
# Opens after 3 consecutive no-progress loops
# Progress = files changed, tests passed, tasks completed
```

## Go Implementation

### 6.1 State Types

```go
// internal/circuit/state.go
package circuit

import (
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

type State string

const (
    StateClosed   State = "CLOSED"
    StateHalfOpen State = "HALF_OPEN"
    StateOpen     State = "OPEN"
)

type BreakerState struct {
    State                State     `json:"state"`
    ConsecutiveNoProgress int       `json:"consecutive_no_progress"`
    ConsecutiveErrors    int       `json:"consecutive_errors"`
    LastProgress         int       `json:"last_progress"`
    CurrentLoop          int       `json:"current_loop"`
    TotalOpens           int       `json:"total_opens"`
    LastUpdated          time.Time `json:"last_updated"`
    Reason               string    `json:"reason"`
}

type HistoryEntry struct {
    Timestamp  time.Time `json:"timestamp"`
    LoopNumber int       `json:"loop_number"`
    FromState  State     `json:"from_state"`
    ToState    State     `json:"to_state"`
    Reason     string    `json:"reason"`
    Progress   bool      `json:"progress"`
    HasError   bool      `json:"has_error"`
}
```

### 6.2 Circuit Breaker

```go
// internal/circuit/breaker.go
package circuit

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

const (
    HalfOpenThreshold = 2  // Loops without progress before HALF_OPEN
    OpenThreshold     = 3  // Loops without progress before OPEN
)

type Breaker struct {
    basePath    string
    stateFile   string
    historyFile string
}

func New(basePath string) *Breaker {
    hermesDir := filepath.Join(basePath, ".hermes")
    return &Breaker{
        basePath:    basePath,
        stateFile:   filepath.Join(hermesDir, "circuit-state.json"),
        historyFile: filepath.Join(hermesDir, "circuit-history.json"),
    }
}

func (b *Breaker) Initialize() error {
    // Ensure directory exists
    dir := filepath.Dir(b.stateFile)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    // Create initial state if not exists
    if _, err := os.Stat(b.stateFile); os.IsNotExist(err) {
        state := &BreakerState{
            State:       StateClosed,
            LastUpdated: time.Now(),
        }
        return b.saveState(state)
    }
    
    return nil
}

func (b *Breaker) GetState() (*BreakerState, error) {
    data, err := os.ReadFile(b.stateFile)
    if err != nil {
        if os.IsNotExist(err) {
            return &BreakerState{State: StateClosed}, nil
        }
        return nil, err
    }
    
    var state BreakerState
    if err := json.Unmarshal(data, &state); err != nil {
        return &BreakerState{State: StateClosed}, nil
    }
    
    return &state, nil
}

func (b *Breaker) saveState(state *BreakerState) error {
    state.LastUpdated = time.Now()
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(b.stateFile, data, 0644)
}

func (b *Breaker) CanExecute() (bool, error) {
    state, err := b.GetState()
    if err != nil {
        return false, err
    }
    return state.State != StateOpen, nil
}

func (b *Breaker) AddLoopResult(hasProgress, hasError bool, loopNumber int) (bool, error) {
    state, err := b.GetState()
    if err != nil {
        return false, err
    }
    
    oldState := state.State
    state.CurrentLoop = loopNumber
    
    if hasProgress {
        // Progress detected - reset counters and close circuit
        state.ConsecutiveNoProgress = 0
        state.LastProgress = loopNumber
        if state.State != StateClosed {
            state.State = StateClosed
            state.Reason = "Progress detected, circuit recovered"
        }
    } else {
        // No progress
        state.ConsecutiveNoProgress++
        
        if state.ConsecutiveNoProgress >= OpenThreshold {
            if state.State != StateOpen {
                state.State = StateOpen
                state.TotalOpens++
                state.Reason = fmt.Sprintf("No recovery, opening circuit after %d loops", state.ConsecutiveNoProgress)
            }
        } else if state.ConsecutiveNoProgress >= HalfOpenThreshold {
            if state.State == StateClosed {
                state.State = StateHalfOpen
                state.Reason = fmt.Sprintf("Monitoring: %d loops without progress", state.ConsecutiveNoProgress)
            }
        }
    }
    
    if hasError {
        state.ConsecutiveErrors++
    } else {
        state.ConsecutiveErrors = 0
    }
    
    // Log state transition
    if oldState != state.State {
        b.addHistory(&HistoryEntry{
            Timestamp:  time.Now(),
            LoopNumber: loopNumber,
            FromState:  oldState,
            ToState:    state.State,
            Reason:     state.Reason,
            Progress:   hasProgress,
            HasError:   hasError,
        })
    }
    
    if err := b.saveState(state); err != nil {
        return false, err
    }
    
    // Return true if can continue
    return state.State != StateOpen, nil
}

func (b *Breaker) Reset(reason string) error {
    state := &BreakerState{
        State:       StateClosed,
        Reason:      reason,
        LastUpdated: time.Now(),
    }
    
    b.addHistory(&HistoryEntry{
        Timestamp: time.Now(),
        FromState: StateOpen,
        ToState:   StateClosed,
        Reason:    reason,
    })
    
    return b.saveState(state)
}

func (b *Breaker) ShouldHalt() (bool, error) {
    state, err := b.GetState()
    if err != nil {
        return false, err
    }
    return state.State == StateOpen, nil
}

func (b *Breaker) addHistory(entry *HistoryEntry) error {
    var history []HistoryEntry
    
    data, err := os.ReadFile(b.historyFile)
    if err == nil {
        json.Unmarshal(data, &history)
    }
    
    history = append(history, *entry)
    
    // Keep last 100 entries
    if len(history) > 100 {
        history = history[len(history)-100:]
    }
    
    data, err = json.MarshalIndent(history, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(b.historyFile, data, 0644)
}
```

### 6.3 Status Display

```go
// internal/circuit/display.go
package circuit

import (
    "fmt"
    "github.com/fatih/color"
)

func (b *Breaker) PrintStatus() error {
    state, err := b.GetState()
    if err != nil {
        return err
    }
    
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println("           Circuit Breaker Status")
    fmt.Println(strings.Repeat("=", 60))
    
    stateColor := color.New(color.FgGreen)
    stateIcon := "[OK]"
    
    switch state.State {
    case StateHalfOpen:
        stateColor = color.New(color.FgYellow)
        stateIcon = "[!!]"
    case StateOpen:
        stateColor = color.New(color.FgRed)
        stateIcon = "[XX]"
    }
    
    fmt.Printf("State:                 ")
    stateColor.Printf("%s %s\n", stateIcon, state.State)
    fmt.Printf("Reason:                %s\n", state.Reason)
    fmt.Printf("Loops since progress:  %d\n", state.ConsecutiveNoProgress)
    fmt.Printf("Last progress:         Loop #%d\n", state.LastProgress)
    fmt.Printf("Current loop:          #%d\n", state.CurrentLoop)
    fmt.Printf("Total opens:           %d\n", state.TotalOpens)
    fmt.Println(strings.Repeat("=", 60))
    
    return nil
}

func (b *Breaker) PrintHaltMessage() {
    red := color.New(color.FgRed, color.Bold)
    
    fmt.Println()
    fmt.Println(strings.Repeat("=", 60))
    red.Println("  EXECUTION HALTED: Circuit Breaker Opened")
    fmt.Println(strings.Repeat("=", 60))
    fmt.Println()
    fmt.Println("Hermes has detected that no progress is being made.")
    fmt.Println()
    fmt.Println("Possible reasons:")
    fmt.Println("  - Project may be complete")
    fmt.Println("  - AI may be stuck on an error")
    fmt.Println("  - PROMPT.md may need clarification")
    fmt.Println()
    fmt.Println("To continue:")
    fmt.Println("  1. Review recent logs")
    fmt.Println("  2. Check AI output")
    fmt.Println("  3. Reset circuit breaker:")
    fmt.Println("     hermes --reset-circuit")
}
```

## Files to Create

| File | Description |
|------|-------------|
| `internal/circuit/state.go` | State types |
| `internal/circuit/breaker.go` | Breaker logic |
| `internal/circuit/display.go` | Status display |
| `internal/circuit/breaker_test.go` | Tests |

## Acceptance Criteria

- [ ] Initialize creates state file
- [ ] CLOSED -> HALF_OPEN after 2 no-progress
- [ ] HALF_OPEN -> OPEN after 3 no-progress
- [ ] HALF_OPEN -> CLOSED on progress
- [ ] Reset works correctly
- [ ] History is logged
- [ ] Status display shows correct info
- [ ] All tests pass
