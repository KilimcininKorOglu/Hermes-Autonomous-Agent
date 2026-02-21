# Changelog

All notable changes to Hermes Autonomous Agent are documented in this file.

## v2.5.3

- Fixed parallel execution creating tags on wrong commit - tags now created immediately after merge

## v2.5.2

- Mandatory HERMES_STATUS block enforcement - AI responses without status block trigger retry
- Implicit documentation dependencies - doc tasks automatically deferred to end of execution
- Added `implicitDocDependencies` config option (default: true)
- Fixed task status not updating to COMPLETED in parallel execution with isolated workspaces
- Task status now updated after successful merge (not in worktree where .hermes is gitignored)
- Added circuit breaker state change logging to hermes.log
- Added merge operation logging to merge.log (BatchStart, BatchComplete, ConflictDetected, ConflictResolved)
- Added MergeFeatureBranch for sequential mode - feature branches now merge to main on completion
- Fixed TUI parallel execution not logging (scheduler logger was nil)
- Added execution start/stop/interrupt logging

## v2.5.1

- TUI now matches CLI run behavior: IN_PROGRESS status, auto-branch, auto-commit, feature tags
- Added BLOCKED/AT_RISK/PAUSED status handling in TUI
- Circuit breaker now opens after consecutive errors (MaxConsecutiveErrors config, default: 5)
- Task timeout prevents infinite execution (AI.Timeout config, default: 300s)
- Logger silent mode for TUI prevents console output corruption
- Fixed duplicate status lines in TUI Run screen
- All critical events now logged to file (circuit breaker state, execution times)
- Task completion, error, pause, resume, and stop events logged in TUI mode
- Fixed Recent Activity alignment in TUI

## v2.5.0

- TUI now matches CLI run behavior: IN_PROGRESS status, auto-branch, auto-commit, feature tags
- Added BLOCKED/AT_RISK/PAUSED status handling in TUI

- Worktrees now cleaned up on execution cancel or completion
- Workers displayed on separate lines in TUI Run screen
- Worker status shows task name: T028 (Database Schema): started

## v2.4.15

- Fixed parallel logs being deleted on restart (removed os.RemoveAll in logger)
- TUI footer now shows consistent navigation menu on all screens during execution
- Removed duplicate help text from individual screens (run.go, settings.go)
- Install command now updates existing installation instead of refusing

## v2.4.14

- Fixed 'depends on non-existent task' error when resuming parallel execution
- Scheduler now receives all tasks for correct dependency resolution

## v2.4.13

- TUI screens refactored with common styles (styles.go)
- Consistent visual design across all 11 TUI screens
- Fixed tasks table format string issues

## v2.4.12

- CRITICAL FIX: Task status now correctly updated to COMPLETED in parallel execution
- Previously tasks remained IN_PROGRESS even after successful completion
- This fix resolves progress bar staying at 0% and circuit breaker false triggers

## v2.4.11

- PROMPT.md now strongly emphasizes HERMES_STATUS block is MANDATORY
- Task completion now checks Success Criteria from task definition
- Improved analysis logs show criteria met/total count
- Better keyword matching for success criteria validation

## v2.4.10

- Run screen shows red warning banner when circuit breaker is OPEN
- Press 'x' in Run screen to reset circuit breaker without leaving TUI

## v2.4.9

- TUI screen navigation now works during any background execution
- Dashboard shows live run status with progress bar during parallel execution
- Fixed progress bar not updating (completedTasks now incremented via callback)
- All screens accessible via 1-9 keys while tasks are running

## v2.4.8

- Real-time progress updates in TUI parallel execution
- Worker status now shows task ID and status (started/completed/failed)
- Fixed TUI screen corruption from CLI-style logging
- Batch progress displayed in status line

## v2.4.7

- Added parallel execution mode to TUI Run screen
- Workers status display during parallel execution
- Sequential and parallel mode toggle with visual feedback

## v2.4.6

- Fixed TUI dashboard showing "No pending tasks" when tasks are IN_PROGRESS
- GetNextTask now returns IN_PROGRESS tasks first before NOT_STARTED

## v2.4.5

- Fixed PRD parsing when AI uses Create tool instead of file markers
- Build timestamp now uses local time instead of UTC

## v2.4.4

- Fixed tool result display for all AI providers (Droid, Claude, Gemini, OpenCode)
- Tool results now show correct tool name instead of empty brackets
- Each tool call shows on separate line with proper formatting

## v2.4.3

- Show provider name prefix for AI text output ([Droid], [Claude], [Gemini])
- Provider name shown in cyan bold at start of each text block

## v2.4.2

- Fixed version display - now correctly shows git tag version
- Improved tool display during AI execution (shows file paths, commands)
- Removed 001-tasks.md fallback - PRD parsing now requires proper file markers
- Tool errors now shown in red

## v2.4.1

- Fixed TUI logging - all TUI operations now write to .hermes/logs/
- Improved PRD parsing prompts to prevent invalid dependencies
- Dependencies must now be valid task IDs (T001, etc.) or "None"

## v2.4.0

- New TUI Run screen with progress bar and pause/resume controls
- Expanded TUI with 11 screens (Idea, PRD, Add, Settings, Circuit, Update, Init, Run)
- TUI Settings screen with all 27 configuration options
- Added retryDelay configuration option
- All settings now properly read from config (no hardcoded values)
- Hot reload support for settings changes

## v2.3.5

- Added convertprd command for PRD format conversion
- Updated dependencies and stability improvements

## v2.2.3

- Windows version info in Task Manager/Properties
- Dynamic version from git tags
- GoReleaser integration for releases

## v2.1.x

- Full status support (BLOCKED, AT_RISK, PAUSED)
- Enhanced prompt templates
- PRD filename sanitization

## v2.0.0

- Parallel task execution with worker pools
- Git worktree isolation
- AI-assisted merge conflict resolution
- Dependency graph scheduling
