# Ralph Example Usage Guide

This document shows step-by-step how to work with Ralph using the `sample-prd.md` file.

---

## Scenario

We will perform autonomous development using the e-commerce platform PRD (`sample-prd.md`).

---

## Step 1: Create Project

```powershell
# Create new project
ralph-setup ecommerce-platform
cd ecommerce-platform
```

**Created Structure:**
```
ecommerce-platform/
├── PROMPT.md
├── @fix_plan.md
├── @AGENT.md
├── specs/
├── src/
├── logs/
└── README.md
```

---

## Step 2: Copy PRD File

```powershell
# Copy PRD file into project
mkdir docs
copy "C:\path\to\ralph-claude-code\docs\sample-prd.md" "docs\PRD.md"
```

---

## Step 3: Convert PRD to Tasks

### Preview (DryRun)

```powershell
ralph-prd docs/PRD.md -DryRun
```

**Expected Output:**
```
Ralph PRD Parser
================

[INFO] Reading PRD: docs/PRD.md
[INFO] PRD size: 5200 characters, 180 lines
[INFO] Using AI: claude
[INFO] Attempt 1/10...
[OK] AI completed successfully

Files to create:

  [+] tasks/001-user-authentication.md (45 lines)
  [+] tasks/002-product-catalog.md (52 lines)
  [+] tasks/003-shopping-cart.md (48 lines)
  [+] tasks/004-checkout-orders.md (55 lines)
  [+] tasks/005-admin-panel.md (42 lines)
  [+] tasks/tasks-status.md (35 lines)

Summary (DryRun):
  New Features: 5
  New Tasks: 18
  Estimated: 25 days

Run without -DryRun to create files.
```

### Actual Creation

```powershell
ralph-prd docs/PRD.md
```

**Created Files:**
```
tasks/
├── 001-user-authentication.md    # F001, T001-T004
├── 002-product-catalog.md        # F002, T005-T008
├── 003-shopping-cart.md          # F003, T009-T011
├── 004-checkout-orders.md        # F004, T012-T015
├── 005-admin-panel.md            # F005, T016-T018
└── tasks-status.md               # Status tracking
```

---

## Step 4: Check Task Status

```powershell
ralph -TaskStatus
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
ralph -TaskStatus -PriorityFilter P1

# Tasks for specific feature
ralph -TaskStatus -FeatureFilter F001

# Completed tasks
ralph -TaskStatus -StatusFilter COMPLETED
```

---

## Step 5: Start Task Mode

### Basic Usage

```powershell
ralph -TaskMode
```

This command:
1. Finds the next task (T001)
2. Injects task details into PROMPT.md
3. Runs AI
4. Updates status on completion
5. Waits for user confirmation

### Full Automation

```powershell
ralph -TaskMode -AutoBranch -AutoCommit
```

This command additionally:
- Creates `feature/F001-user-authentication` branch
- Auto-commits after each task
- Merges to main when feature is complete

### Autonomous Mode

```powershell
ralph -TaskMode -AutoBranch -AutoCommit -Autonomous
```

This command:
- Completes all tasks without user intervention
- Automatically switches between features
- Continues on errors when possible

---

## Step 6: Monitor Progress

### In Separate Terminal Window

```powershell
# Terminal 1: Run Ralph
ralph -TaskMode -AutoBranch -AutoCommit -Monitor

# Terminal 2: Monitoring panel opens automatically
```

### Manual Monitoring

```powershell
# In another terminal
ralph-monitor
```

### Examine Log Files

```powershell
# Latest log
Get-Content logs/ralph.log -Tail 50

# Latest AI output
Get-ChildItem logs/*_output_*.log | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 1 | 
    Get-Content
```

---

## Step 7: Start from Specific Task

If you want to start from a specific task:

```powershell
# Start from T005
ralph -TaskMode -StartFrom T005 -AutoBranch -AutoCommit
```

---

## Step 8: Resume After Interruption

Ralph automatically resumes from where it left off:

```powershell
# First run - interrupted at T003
ralph -TaskMode -AutoBranch -AutoCommit
# Ctrl+C or context limit

# Second run - automatically continues from T004
ralph -TaskMode -AutoBranch -AutoCommit
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
notepad docs/PRD.md

# Re-run - only new features are added
ralph-prd docs/PRD.md
```

### Method 2: Add Single Feature

```powershell
# Inline description
ralph-add "User profile page and avatar upload"

# From file
ralph-add @docs/new-feature-spec.md
```

---

## Step 10: Use Different AI Provider

```powershell
# Parse PRD with Droid
ralph-prd docs/PRD.md -AI droid

# Task Mode with Aider
ralph -TaskMode -AI aider -AutoBranch -AutoCommit

# List available providers
ralph-prd -List
```

---

## Workflow Summary

```
1. ralph-setup ecommerce-platform
2. cd ecommerce-platform
3. # Copy PRD file to docs/PRD.md
4. ralph-prd docs/PRD.md -DryRun          # Preview
5. ralph-prd docs/PRD.md                   # Create tasks
6. ralph -TaskStatus                       # View status
7. ralph -TaskMode -AutoBranch -AutoCommit -Autonomous  # Start
8. # ... Ralph is working ...
9. ralph -TaskStatus                       # Check progress
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
ralph-prd docs/PRD.md -Timeout 1800

# Try different AI
ralph-prd docs/PRD.md -AI droid
```

### Task Blocked

```powershell
# View blocked tasks
ralph -TaskStatus -StatusFilter BLOCKED

# Manually skip to next task
ralph -TaskMode -StartFrom T006
```

### Circuit Breaker Opened

```powershell
# Check status
ralph -CircuitStatus

# Reset
ralph -ResetCircuit

# Restart
ralph -TaskMode -AutoBranch -AutoCommit
```

---

## Tips

1. **For large PRDs:** Split PRD into separate files by feature
2. **Error tracking:** Regularly check the `logs/` directory
3. **Branch cleanup:** Merged branches are automatically deleted
4. **Incremental:** When you update and re-run PRD, only new features are added
5. **DryRun:** Always check with `-DryRun` first

---

**Ready!** You can now perform autonomous development with Ralph.
