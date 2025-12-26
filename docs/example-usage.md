# Hermes Example Usage Guide

This document shows step-by-step how to work with Hermes using the `sample-prd.md` file.

---

## Scenario

We will perform autonomous development using the e-commerce platform PRD (`sample-prd.md`).

---

## Step 1: Create Project

```powershell
# Create new project
hermes-setup ecommerce-platform
cd ecommerce-platform
```

**Created Structure:**

```
ecommerce-platform/
├── .gitignore          # Contains ".hermes/"
├── .hermes/            # All Hermes files (gitignored)
│   ├── config.json     # Project configuration
│   ├── PROMPT.md       # AI prompt file
│   ├── tasks/          # Task files
│   ├── logs/           # Execution logs
│   └── docs/           # PRD and documentation
├── README.md
└── (src/, tests/ created by AI)
```

---

## Step 2: Copy PRD File

```powershell
# Copy PRD file into .hermes/docs/
copy "C:\path\to\hermes-claude-code\docs\sample-prd.md" ".hermes\docs\PRD.md"
```

---

## Step 3: Convert PRD to Tasks

### Preview (DryRun)

```powershell
hermes-prd .hermes/docs/PRD.md -DryRun
```

**Expected Output:**

```
Hermes Autonomous Agent - PRD Parser
================

[INFO] Reading PRD: .hermes/docs/PRD.md
[INFO] PRD size: 5946 characters, 264 lines
[INFO] Using AI: claude (timeout: 1200s, retries: 10)
[INFO] Parsing PRD with claude...

[INFO] Attempt 1/10...
[DEBUG] Starting claude execution...
[DEBUG] claude completed in 166.5 seconds
[OK] AI completed successfully

Files to create:

  [+] 001-user-authentication.md (F001, T001-T007)
  [+] 002-product-catalog.md (F002, T008-T014)
  [+] 003-shopping-cart.md (F003, T015-T020)
  [+] 004-checkout-orders.md (F004, T021-T028)
  [+] 005-admin-panel.md (F005, T029-T035)
  [+] tasks-status.md

Summary (DryRun):
  New Features: 5
  New Tasks: 35
  Estimated: 0 days

Run without -DryRun to create files.
```

### Actual Creation

```powershell
hermes-prd .hermes/docs/PRD.md
```

**Created Files:**

```
.hermes/tasks/
├── 001-user-authentication.md    # F001, T001-T007
├── 002-product-catalog.md        # F002, T008-T014
├── 003-shopping-cart.md          # F003, T015-T020
├── 004-checkout-orders.md        # F004, T021-T028
├── 005-admin-panel.md            # F005, T029-T035
└── tasks-status.md               # Status tracking
```

---

## Step 4: Check Task Status

```powershell
hermes -TaskStatus
```

**Expected Output:**

```
============================================================
  TASK STATUS
============================================================

┌──────────┬─────────────────────────────────────┬──────────────┬──────────┬──────────┐
│ Task ID  │ Task Name                           │ Status       │ Priority │ Feature  │
├──────────┼─────────────────────────────────────┼──────────────┼──────────┼──────────┤
│ T001     │ Database Schema                     │ NOT_STARTED  │ P1       │ F001     │
│ T002     │ Registration API                    │ NOT_STARTED  │ P1       │ F001     │
│ T003     │ Login API                           │ NOT_STARTED  │ P1       │ F001     │
│ T004     │ Email Verification                  │ NOT_STARTED  │ P2       │ F001     │
│ T005     │ Product List API                    │ NOT_STARTED  │ P1       │ F002     │
│ ...      │ ...                                 │ ...          │ ...      │ ...      │
└──────────┴─────────────────────────────────────┴──────────────┴──────────┴──────────┘

Summary:
  Total: 18 tasks
  COMPLETED:    0 (0%)
  IN_PROGRESS:  0 (0%)
  NOT_STARTED:  18 (100%)
  BLOCKED:      0 (0%)

Progress: [░░░░░░░░░░░░░░░░░░░░░░░░░] 0%

Next Task: T001 - Database Schema

============================================================
```

### Filtering Examples

```powershell
# Only P1 priority tasks
hermes -TaskStatus -PriorityFilter P1

# Tasks for specific feature
hermes -TaskStatus -FeatureFilter F001

# Completed tasks
hermes -TaskStatus -StatusFilter COMPLETED
```

---

## Step 5: Start Task Mode

### Basic Usage

```powershell
hermes -TaskMode
```

This command:

1. Finds the next task (T001)
2. Injects task details into PROMPT.md
3. Runs AI
4. Updates status on completion
5. Waits for user confirmation

### Full Automation

```powershell
hermes -TaskMode -AutoBranch -AutoCommit
```

This command additionally:

- Creates `feature/F001-user-authentication` branch
- Auto-commits after each task
- Merges to main when feature is complete

### Autonomous Mode

```powershell
hermes -TaskMode -AutoBranch -AutoCommit -Autonomous
```

This command:

- Completes all tasks without user intervention
- Automatically switches between features
- Continues on errors when possible

---

## Step 6: Monitor Progress

### In Separate Terminal Window

```powershell
# Terminal 1: Run Hermes
hermes -TaskMode -AutoBranch -AutoCommit -Monitor

# Terminal 2: Monitoring panel opens automatically
```

### Manual Monitoring

```powershell
# In another terminal
hermes-monitor
```

### Examine Log Files

```powershell
# Latest log
Get-Content .hermes/logs/hermes-loop-*.log -Tail 50

# Latest AI output
Get-ChildItem .hermes/logs/*_output_*.log | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 1 | 
    Get-Content
```

---

## Step 7: Start from Specific Task

If you want to start from a specific task:

```powershell
# Start from T005
hermes -TaskMode -StartFrom T005 -AutoBranch -AutoCommit
```

---

## Step 8: Resume After Interruption

Hermes automatically resumes from where it left off:

```powershell
# First run - interrupted at T003
hermes -TaskMode -AutoBranch -AutoCommit
# Ctrl+C or context limit

# Second run - automatically continues from T004
hermes -TaskMode -AutoBranch -AutoCommit
```

**Output:**

```
============================================================
  Previous run detected - Resuming
============================================================
  Resume Task: T004
  Branch: feature/F001-user-authentication
============================================================
```

---

## Step 9: Add New Feature

If new features are added to PRD:

### Method 1: Re-run PRD (Incremental)

```powershell
# Update PRD
notepad .hermes/docs/PRD.md

# Re-run - only new features are added
hermes-prd .hermes/docs/PRD.md
```

### Method 2: Add Single Feature

```powershell
# Inline description
hermes-add "User profile page and avatar upload"

# From file
hermes-add @.hermes/docs/new-feature-spec.md
```

---

## Step 10: AI Provider Selection

Hermes uses **task-based AI selection** by default:
- **Planning tasks** (PRD parsing, feature addition): `claude`
- **Coding tasks** (task execution): `droid`

```powershell
# Default: uses claude for PRD parsing (planning task)
hermes-prd .hermes/docs/PRD.md

# Default: uses droid for task execution (coding task)
hermes -TaskMode -AutoBranch -AutoCommit

# Override with specific provider
hermes-prd .hermes/docs/PRD.md -AI droid
hermes -TaskMode -AI claude -AutoBranch -AutoCommit

# List available providers
hermes-prd -List
```

Configure in `.hermes/config.json`:

```json
{
  "ai": {
    "planning": "claude",
    "coding": "droid"
  }
}
```

---

## Workflow Summary

```
1. hermes-setup ecommerce-platform
2. cd ecommerce-platform
3. copy sample-prd.md .hermes/docs/PRD.md   # Copy PRD file
4. hermes-prd .hermes/docs/PRD.md -DryRun   # Preview
5. hermes-prd .hermes/docs/PRD.md           # Create tasks
6. hermes -TaskStatus                       # View status
7. hermes -TaskMode -AutoBranch -AutoCommit -Autonomous  # Start
8. # ... Hermes is working (uses droid for coding) ...
9. hermes -TaskStatus                       # Check progress
```

---

## Expected Git History

After Task Mode completes:

```
* abc1234 (HEAD -> main) Merge feature F005 - Admin Panel
|\
| * def5678 feat(T018): Admin Reports completed
| * ghi9012 feat(T017): Order Management completed
| * jkl3456 feat(T016): Product CRUD completed
|/
* mno7890 Merge feature F004 - Checkout Orders
|\
| * ...
|/
* pqr1234 Merge feature F003 - Shopping Cart
* stu5678 Merge feature F002 - Product Catalog
* vwx9012 Merge feature F001 - User Authentication
* yza3456 Initial commit
```

---

## Troubleshooting

### PRD Parse Failed

```powershell
# Increase timeout
hermes-prd .hermes/docs/PRD.md -Timeout 1800

# Try different AI
hermes-prd .hermes/docs/PRD.md -AI droid
```

### Task Blocked

```powershell
# View blocked tasks
hermes -TaskStatus -StatusFilter BLOCKED

# Manually skip to next task
hermes -TaskMode -StartFrom T006
```

### Circuit Breaker Opened

```powershell
# Check status
hermes -CircuitStatus

# Reset
hermes -ResetCircuit

# Restart
hermes -TaskMode -AutoBranch -AutoCommit
```

---

## Tips

1. **For large PRDs:** Split PRD into separate files by feature
2. **Error tracking:** Regularly check the `.hermes/logs/` directory
3. **Branch cleanup:** Merged branches are automatically deleted
4. **Incremental:** When you update and re-run PRD, only new features are added
5. **DryRun:** Always check with `-DryRun` first
6. **AI Selection:** Planning=claude (PRD), Coding=droid (tasks)
7. **Gitignored:** All `.hermes/` files are gitignored by default

---

**Ready!** You can now perform autonomous development with Hermes.

---

**Version:** 1.1  
**Last Updated:** 2025-12-26
