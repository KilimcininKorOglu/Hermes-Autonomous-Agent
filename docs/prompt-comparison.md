# Prompt Comparison: task-plan.md vs Hermes

This document compares the prompt structure defined in `task-plan.md` with the prompts Hermes uses internally.

## Overview

| Aspect                 | task-plan.md                          | Hermes Prompts                    |
|------------------------|---------------------------------------|-----------------------------------|
| Purpose                | Standalone skill/command spec         | Internal AI execution             |
| Complexity             | High (detailed)                       | Low (minimal)                     |
| File Count             | 3 (feature, status, execution-plan)   | 1 (feature file only)             |
| Git Strategy           | Detailed branch policy                | Simple auto-branch/commit flags   |

## PRD Parsing Differences

### task-plan.md Expected Format

```markdown
# Feature XXX: [Feature Name]

**Feature ID:** FXXX
**Feature Name:** [Descriptive Name]
**Priority:** P[1-4] - [CRITICAL/HIGH/MEDIUM/LOW]
**Target Version:** vX.Y.Z
**Estimated Duration:** X-Y weeks
**Status:** NOT_STARTED

## Overview
[2-3 paragraph description]

## Goals
- [Measurable goals]

## Success Criteria
- [ ] All tasks completed (TXXX-TYYY)
- [ ] [Specific criteria]
- [ ] Tests passing

## Tasks

### TXXX: [Task Name]

**Status:** NOT_STARTED
**Priority:** P[1-4]
**Estimated Effort:** X days

#### Description
[Clear task description]

#### Technical Details
[Code snippets or technical details]

#### Files to Touch
- `src/path/file.ts` (new/update)

#### Dependencies
- TYYY (must complete first)

#### Success Criteria
- [ ] [Deliverable 1]
- [ ] [Deliverable 2]

## Performance Targets
[Performance metrics]

## Risk Assessment
[Risks and mitigation strategies]
```

### Hermes PRD Prompt (internal/cmd/prd.go)

```go
func buildPrdPrompt(prdContent string) string {
    return fmt.Sprintf(`Parse this PRD into task files.

For each feature, create a markdown file with this format:

# Feature N: Feature Name
**Feature ID:** FXXX
**Status:** NOT_STARTED

### TXXX: Task Name
**Status:** NOT_STARTED
**Priority:** P1
**Files to Touch:** file1, file2
**Dependencies:** None
**Success Criteria:**
- Criterion 1
- Criterion 2

---

PRD Content:

%s

Output each file with:
---FILE: XXX-feature-name.md---
<content>
---END_FILE---`, prdContent)
}
```

## Task Status Differences

| Status         | task-plan.md | Hermes |
|----------------|--------------|--------|
| NOT_STARTED    | Yes          | Yes    |
| IN_PROGRESS    | Yes          | Yes    |
| COMPLETED      | Yes          | Yes    |
| BLOCKED        | Yes          | Yes    |
| AT_RISK        | Yes          | No     |
| PAUSED         | Yes          | No     |

## Git Strategy Differences

### task-plan.md

- Separate branch per feature: `feature/FXXX-description`
- Commit per task: `feat(TXXX): Task name completed`
- On feature completion: `--no-ff` merge
- Branches are never deleted (history preservation)
- Detailed merge commit messages

### Hermes

- Optional branch creation with `--auto-branch` flag
- Optional commits with `--auto-commit` flag
- Simple branch/commit logic
- No detailed branch policy

## Execution Differences

### task-plan.md

- Dependency + Priority algorithm
- Checkpoint system (`run-state.md`)
- Resume mechanism (after context compaction)
- Aggressive "NEVER STOP" rules
- BLOCKED after 3 retries

### Hermes

- Sequential task execution
- Circuit breaker (stagnation detection)
- Simple retry mechanism
- `maxConsecutiveErrors` limit

## File Structure Differences

### task-plan.md

```
tasks/
  001-feature-name.md       # Feature file
  002-another-feature.md
  tasks-status.md           # Overall status tracking
  task-execution-plan.md    # Execution plan
  run-state.md              # Runtime state (during run)
```

### Hermes

```
.hermes/
  tasks/
    001-feature-name.md     # Feature file
    002-another-feature.md
  config.json               # Configuration
  PROMPT.md                 # AI prompt (auto-managed)
  logs/hermes.log           # Logs
  circuit-state.json        # Circuit breaker state
```

## Task Injection Differences

### task-plan.md

Reads directly from task files, updates execution plan.

### Hermes (internal/prompt/injector.go)

```go
func (i *Injector) generateTaskSection(t *task.Task) string {
    // HERMES_TASK_START/END markers
    // Task ID, Name, Priority, Files, Dependencies, Success Criteria
    // Expects HERMES_STATUS block output
}
```

Hermes expects AI output in this format:

```
---HERMES_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: true
RECOMMENDATION: Move to next task
---END_HERMES_STATUS---
```

## Idea-to-PRD Differences

### task-plan.md

No idea-to-PRD support.

### Hermes (internal/idea/prompt.go)

```go
func BuildPrompt(idea, language, additionalContext string) string {
    // Senior PM role
    // 6 sections: Overview, Features, Technical, Non-Functional, Metrics, Timeline
    // Turkish/English language support
}
```

## Conclusion

The `task-plan.md` file defines a more comprehensive task planning system that is independent of Hermes. Hermes does not use this file - it uses its own minimal prompts internally.

If the `task-plan.md` approach is to be adopted, the following Hermes files would need to be updated:

1. `internal/cmd/prd.go` - buildPrdPrompt function
2. `internal/cmd/add.go` - buildAddPrompt function
3. `internal/prompt/injector.go` - generateTaskSection function
4. `internal/task/reader.go` - To parse new fields
5. `internal/cmd/run.go` - For Dependency + Priority algorithm
