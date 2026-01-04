# Hermes v2.0 - Parallel Task Execution

## Overview

Hermes v2.0 introduces parallel task execution, allowing multiple AI agents to work on independent tasks simultaneously. This significantly reduces development time for projects with tasks that don't have dependencies on each other.

## Current State (v1.x)

```
Task 1 ──────> Task 2 ──────> Task 3 ──────> Task 4
   30min         30min         30min         30min
                                            Total: 2 hours
```

## Proposed State (v2.0)

```
Task 1 ──────>
Task 2 ──────> ──> Merge ──> Task 4
Task 3 ──────>
   30min          10min       30min
                            Total: ~70min
```

## Key Features

### 1. Dependency Graph Analysis

Build a directed acyclic graph (DAG) from task dependencies to identify parallelizable tasks.

```go
// internal/scheduler/graph.go
type TaskGraph struct {
    nodes map[string]*TaskNode
    edges map[string][]string  // task -> dependencies
}

type TaskNode struct {
    Task       *task.Task
    InDegree   int      // number of dependencies
    Dependents []string // tasks that depend on this
}

func (g *TaskGraph) GetReadyTasks() []*task.Task {
    // Return tasks with InDegree == 0 (no pending dependencies)
}

func (g *TaskGraph) MarkComplete(taskID string) {
    // Decrement InDegree of dependent tasks
}
```

### 2. Worker Pool

Manage multiple AI agent instances with configurable concurrency.

```go
// internal/scheduler/pool.go
type WorkerPool struct {
    workers    int
    taskQueue  chan *task.Task
    results    chan *TaskResult
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

type TaskResult struct {
    TaskID  string
    Success bool
    Output  string
    Error   error
    Branch  string
}

func NewWorkerPool(workers int, provider ai.Provider) *WorkerPool

func (p *WorkerPool) Submit(task *task.Task)
func (p *WorkerPool) Wait() []TaskResult
func (p *WorkerPool) Stop()
```

### 3. Branch Strategy

Each parallel task works on its own branch to avoid conflicts.

```
main
  │
  ├── hermes/T001-database-schema
  ├── hermes/T002-api-endpoints
  └── hermes/T003-frontend-components
  │
  └── merge back to feature branch
```

```go
// internal/git/parallel.go
type ParallelBranchManager struct {
    baseBranch string
    branches   map[string]string // taskID -> branchName
}

func (m *ParallelBranchManager) CreateTaskBranch(taskID string) (string, error)
func (m *ParallelBranchManager) MergeBranches(taskIDs []string) error
func (m *ParallelBranchManager) ResolveConflicts(branch1, branch2 string) error
```

### 4. Conflict Detection & Resolution

Detect and handle file conflicts between parallel tasks.

```go
// internal/merger/conflict.go
type ConflictDetector struct {
    fileChanges map[string][]string // file -> taskIDs that modified it
}

type Conflict struct {
    File     string
    Tasks    []string
    Type     ConflictType // SAME_FILE, SAME_FUNCTION, IMPORT_CONFLICT
    Severity int          // 1-3
}

func (d *ConflictDetector) Analyze(results []TaskResult) []Conflict
func (d *ConflictDetector) CanAutoResolve(c Conflict) bool
```

**Conflict Resolution Strategies:**

| Type | Strategy |
|------|----------|
| Different files | Auto-merge (no conflict) |
| Same file, different sections | Auto-merge with git |
| Same file, same section | AI-assisted resolution |
| Semantic conflict | Queue for sequential re-run |

### 5. AI-Assisted Merge

Use AI to resolve complex merge conflicts.

```go
// internal/merger/ai_merge.go
type AIMerger struct {
    provider ai.Provider
}

func (m *AIMerger) ResolveConflict(conflict Conflict, context MergeContext) (string, error)

type MergeContext struct {
    File           string
    OriginalCode   string
    Task1Changes   string
    Task2Changes   string
    Task1Intent    string // from task description
    Task2Intent    string
}
```

**Merge Prompt Template:**

```
You are merging code changes from two parallel tasks.

File: {file}
Original: {original}

Task 1 ({task1_id}): {task1_intent}
Changes: {task1_changes}

Task 2 ({task2_id}): {task2_intent}
Changes: {task2_changes}

Merge these changes preserving both intents. Output only the merged code.
```

## Architecture

### New Packages

```
internal/
  scheduler/
    graph.go          # Task dependency graph
    pool.go           # Worker pool management
    scheduler.go      # Main scheduling logic
    priority.go       # Task prioritization
  merger/
    conflict.go       # Conflict detection
    resolver.go       # Auto-resolution strategies
    ai_merge.go       # AI-assisted merging
  isolation/
    workspace.go      # Isolated workspaces per task
    sandbox.go        # File system isolation
```

### Modified Packages

```
internal/
  cmd/
    run.go            # Add --parallel, --workers flags
  config/
    types.go          # Add parallel config section
  git/
    parallel.go       # Parallel branch management
  task/
    reader.go         # Add dependency parsing
    graph.go          # Task graph utilities
```

## Configuration

```json
{
  "parallel": {
    "enabled": true,
    "maxWorkers": 3,
    "strategy": "branch-per-task",
    "conflictResolution": "ai-assisted",
    "isolatedWorkspaces": true,
    "mergeStrategy": "sequential"
  }
}
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | false | Enable parallel execution |
| `maxWorkers` | 3 | Maximum concurrent AI agents |
| `strategy` | branch-per-task | Branching strategy |
| `conflictResolution` | ai-assisted | How to handle conflicts |
| `isolatedWorkspaces` | true | Separate workspace per task |
| `mergeStrategy` | sequential | How to merge completed tasks |

## CLI Changes

```bash
# Run with parallel execution
hermes run --parallel

# Specify number of workers
hermes run --parallel --workers 3

# Dry run - show execution plan
hermes run --parallel --dry-run

# Show dependency graph
hermes graph

# Manual merge after parallel run
hermes merge --tasks T001,T002,T003
```

## Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        HERMES RUN --PARALLEL                     │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. PARSE TASKS & BUILD DEPENDENCY GRAPH                        │
│     - Read all task files                                        │
│     - Parse dependencies                                         │
│     - Build DAG                                                  │
│     - Detect cycles (error if found)                            │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. IDENTIFY PARALLEL BATCHES                                    │
│     Batch 1: [T001, T002, T003] (no deps)                       │
│     Batch 2: [T004, T005] (depend on batch 1)                   │
│     Batch 3: [T006] (depends on T004)                           │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. EXECUTE BATCH                                                │
│     ┌──────────┐  ┌──────────┐  ┌──────────┐                    │
│     │ Worker 1 │  │ Worker 2 │  │ Worker 3 │                    │
│     │   T001   │  │   T002   │  │   T003   │                    │
│     │  branch  │  │  branch  │  │  branch  │                    │
│     └────┬─────┘  └────┬─────┘  └────┬─────┘                    │
│          │             │             │                           │
│          ▼             ▼             ▼                           │
│     ┌─────────────────────────────────────┐                     │
│     │         WAIT FOR COMPLETION          │                     │
│     └─────────────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. MERGE & CONFLICT RESOLUTION                                  │
│     - Detect file conflicts                                      │
│     - Auto-merge non-conflicting                                 │
│     - AI-resolve conflicts                                       │
│     - Commit merged result                                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. REPEAT FOR NEXT BATCH                                        │
│     - Update dependency graph                                    │
│     - Get next ready tasks                                       │
│     - Execute until all complete                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Workspace Isolation

Each worker operates in an isolated workspace to prevent file conflicts during execution.

```go
// internal/isolation/workspace.go
type Workspace struct {
    TaskID    string
    BasePath  string
    WorkPath  string // /tmp/hermes-{taskID}/
    Branch    string
}

func (w *Workspace) Setup() error {
    // 1. Create temp directory
    // 2. Clone/copy repository
    // 3. Checkout task branch
    // 4. Return isolated path
}

func (w *Workspace) Cleanup() error {
    // Remove temp directory
}

func (w *Workspace) GetChanges() ([]FileChange, error) {
    // Git diff to get modified files
}
```

**Isolation Strategy Options:**

| Strategy | Disk Usage | Speed | Isolation |
|----------|------------|-------|-----------|
| Full Clone | High | Slow | Complete |
| Worktree | Medium | Fast | Complete |
| Copy-on-Write | Low | Fast | Partial |
| Shared + Locks | None | Fastest | Risky |

Recommended: **Git Worktree** - native git feature, low overhead, complete isolation.

```bash
# Git worktree for each parallel task
git worktree add /tmp/hermes-T001 -b hermes/T001
git worktree add /tmp/hermes-T002 -b hermes/T002
git worktree add /tmp/hermes-T003 -b hermes/T003
```

## Resource Management

### API Rate Limiting

```go
// internal/scheduler/ratelimit.go
type RateLimiter struct {
    callsPerMinute int
    tokens         chan struct{}
    refillTicker   *time.Ticker
}

func (r *RateLimiter) Acquire() {
    <-r.tokens
}

func (r *RateLimiter) Release() {
    r.tokens <- struct{}{}
}
```

### Memory Management

```go
type ResourceMonitor struct {
    maxMemoryMB   int
    maxCPUPercent int
}

func (m *ResourceMonitor) CanStartWorker() bool
func (m *ResourceMonitor) WaitForResources()
```

## Progress Tracking

### TUI Updates

```
┌─────────────────────────────────────────────────────────────────┐
│  HERMES PARALLEL EXECUTION                              v2.0.0  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Batch 1/3                                    [████████░░] 80%  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Worker 1: T001 Database Schema          [████████████] DONE ││
│  │ Worker 2: T002 API Endpoints            [████████░░░░] 75%  ││
│  │ Worker 3: T003 Frontend Components      [██████░░░░░░] 50%  ││
│  └─────────────────────────────────────────────────────────────┘│
│                                                                  │
│  Pending: T004, T005, T006                                      │
│  Completed: 1/6 tasks                                           │
│                                                                  │
│  [s] Stop  [p] Pause  [d] Details  [q] Quit                    │
└─────────────────────────────────────────────────────────────────┘
```

### Logging

```go
// internal/ui/parallel_logger.go
type ParallelLogger struct {
    workerLogs map[string]*log.Logger
    mainLog    *log.Logger
}

func (l *ParallelLogger) WorkerLog(taskID, message string)
func (l *ParallelLogger) MainLog(message string)
```

Log files:

```
.hermes/logs/
  hermes.log           # Main log
  parallel/
    T001.log           # Worker 1 log
    T002.log           # Worker 2 log
    T003.log           # Worker 3 log
    merge.log          # Merge operation log
```

## Error Handling

### Task Failure Strategies

| Strategy | Description |
|----------|-------------|
| `fail-fast` | Stop all workers on first failure |
| `continue` | Continue other tasks, report failures at end |
| `retry` | Retry failed task up to N times |
| `fallback-sequential` | Re-run failed tasks sequentially |

```go
type FailureStrategy struct {
    Mode       string // fail-fast, continue, retry
    MaxRetries int
    RetryDelay time.Duration
}
```

### Rollback

```go
// internal/scheduler/rollback.go
func (s *Scheduler) Rollback(batch []string) error {
    // 1. Delete task branches
    // 2. Reset to pre-batch state
    // 3. Mark tasks as NOT_STARTED
}
```

## Task File Format Changes

Add explicit dependency syntax:

```markdown
### T004: User Dashboard

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 3 days
**Depends On:** T001, T002
**Parallelizable:** true

#### Description
...
```

New fields:

- `**Depends On:**` - Explicit task dependencies
- `**Parallelizable:**` - Can run in parallel (default: true)
- `**Exclusive Files:**` - Files only this task should modify

## Implementation Phases

### Phase 1: Foundation (v2.0.0-alpha)

- [ ] Dependency graph parser
- [ ] Basic worker pool
- [ ] Git worktree integration
- [ ] Sequential merge

### Phase 2: Core Features (v2.0.0-beta)

- [ ] Conflict detection
- [ ] Auto-merge for non-conflicting
- [ ] TUI parallel view
- [ ] Parallel logging

### Phase 3: AI Integration (v2.0.0-rc)

- [ ] AI-assisted merge
- [ ] Smart conflict resolution
- [ ] Semantic conflict detection
- [ ] Merge validation

### Phase 4: Polish (v2.0.0)

- [ ] Resource monitoring
- [ ] Rate limiting
- [ ] Rollback support
- [ ] Documentation

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Merge conflicts | High | Medium | AI-assisted resolution |
| API rate limits | Medium | High | Built-in rate limiter |
| Resource exhaustion | High | Low | Resource monitor |
| Circular dependencies | High | Low | DAG validation |
| Semantic conflicts | High | Medium | Test validation step |
| Data corruption | Critical | Low | Isolated workspaces |

## Performance Expectations

| Scenario | Sequential | Parallel (3 workers) | Speedup |
|----------|------------|----------------------|---------|
| 6 independent tasks | 3 hours | 1 hour | 3x |
| 6 tasks, linear deps | 3 hours | 3 hours | 1x |
| 6 tasks, 2 batches | 3 hours | 1.5 hours | 2x |
| 10 tasks, mixed deps | 5 hours | 2 hours | 2.5x |

## API Cost Considerations

Parallel execution means parallel API calls:

| Workers | Calls/Hour | Estimated Cost* |
|---------|------------|-----------------|
| 1 | 100 | $X |
| 2 | 200 | $2X |
| 3 | 300 | $3X |

*Actual cost depends on AI provider pricing

Configuration option to set budget limits:

```json
{
  "parallel": {
    "maxCostPerHour": 10.00,
    "pauseOnBudgetExceeded": true
  }
}
```

## Open Questions

1. **Worktree vs Clone?** - Worktree is faster but requires git 2.5+
2. **Merge order?** - By completion time or task priority?
3. **Conflict threshold?** - How many conflicts before fallback to sequential?
4. **Test validation?** - Run tests after each merge or at end?
5. **Cross-feature parallelism?** - Parallelize across features or within?

## References

- [Git Worktrees](https://git-scm.com/docs/git-worktree)
- [Go Worker Pool Pattern](https://gobyexample.com/worker-pools)
- [DAG Scheduling](https://en.wikipedia.org/wiki/Directed_acyclic_graph)
