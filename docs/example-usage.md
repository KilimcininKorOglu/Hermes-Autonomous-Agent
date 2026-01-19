# Hermes Autonomous Agent - Example Usage (v2.3)

Step-by-step walkthrough of using Hermes to build a project from PRD to completion.

---

## Scenario: Building an E-Commerce API

We will build a simple e-commerce REST API with user authentication, product catalog, and shopping cart features.

---

## Step 1: Initialize Project

```bash
# Create and initialize a new project
hermes init ecommerce-api
cd ecommerce-api
```

**Output:**

```
Initializing Hermes in: C:\Projects\ecommerce-api

  Initialized: git repository
  Created: .hermes/
  Created: .hermes/tasks/
  Created: .hermes/logs/
  Created: .hermes/docs/
  Created: .hermes/config.json
  Created: .hermes/PROMPT.md
  Created: .gitignore
  Created: initial commit on main branch

Hermes initialized successfully!

Next steps:
  1. Add your PRD to .hermes/docs/PRD.md
  2. Run: hermes prd .hermes/docs/PRD.md
  3. Run: hermes run --auto-branch --auto-commit
```

---

## Step 2: Generate PRD from Idea (v1.1.0)

Instead of manually writing a PRD, you can generate one from a simple idea:

```bash
hermes idea "e-commerce REST API with user authentication, product catalog, and shopping cart"
```

**Output:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      AI-Powered Application Development

Idea to PRD Generator
=====================

Idea: e-commerce REST API with user authentication, product catalog, and shopping cart
AI: claude
Language: en

Generating PRD...

[SUCCESS] PRD generated: .hermes/docs/PRD.md

Next steps:
  1. Review: cat .hermes/docs/PRD.md
  2. Parse:  hermes prd .hermes/docs/PRD.md
```

### Interactive Mode

For more detailed PRD, use interactive mode:

```bash
hermes idea "e-commerce API" --interactive
```

This will ask additional questions about target audience, tech stack, scale, and timeline.

---

## Step 3: Review or Create PRD Manually

You can review the generated PRD or create `.hermes/docs/PRD.md` manually with your requirements:

```markdown
# E-Commerce API - Product Requirements Document

## Overview
Build a REST API for an e-commerce platform using Go and PostgreSQL.

## Features

### Feature 1: User Authentication
- User registration with email/password
- Email verification
- Login with JWT tokens
- Password reset functionality

### Feature 2: Product Catalog
- CRUD operations for products
- Category management
- Product search and filtering
- Pagination support

### Feature 3: Shopping Cart
- Add/remove items
- Update quantities
- Calculate totals
- Persist cart for logged-in users

## Technical Requirements
- Go 1.21+
- PostgreSQL 15+
- JWT for authentication
- RESTful API design
- Input validation
- Error handling
```

---

## Step 4: Parse PRD into Tasks

```bash
hermes prd .hermes/docs/PRD.md
```

**Output:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      AI-Powered Application Development

PRD Parser
==========

PRD file: .hermes/docs/PRD.md (1247 chars)
Using AI: claude

Created: .hermes/tasks/001-user-authentication.md
Created: .hermes/tasks/002-product-catalog.md
Created: .hermes/tasks/003-shopping-cart.md

Created 3 task files in .hermes/tasks
```

---

## Step 5: Review Generated Tasks

```bash
hermes status
```

**Output:**

```
+-------+--------------------------------+--------------+----------+---------+
| ID    | Name                           | Status       | Priority | Feature |
+-------+--------------------------------+--------------+----------+---------+
| T001  | Database Schema for Users      | NOT_STARTED  | P1       | F001    |
| T002  | User Registration Endpoint     | NOT_STARTED  | P1       | F001    |
| T003  | Email Verification System      | NOT_STARTED  | P1       | F001    |
| T004  | Login Endpoint with JWT        | NOT_STARTED  | P1       | F001    |
| T005  | Password Reset Flow            | NOT_STARTED  | P2       | F001    |
| T006  | Database Schema for Products   | NOT_STARTED  | P1       | F002    |
| T007  | Product CRUD Endpoints         | NOT_STARTED  | P1       | F002    |
| T008  | Category Management            | NOT_STARTED  | P2       | F002    |
| T009  | Product Search and Filtering   | NOT_STARTED  | P2       | F002    |
| T010  | Pagination Implementation      | NOT_STARTED  | P2       | F002    |
| T011  | Cart Database Schema           | NOT_STARTED  | P1       | F003    |
| T012  | Add/Remove Cart Items          | NOT_STARTED  | P1       | F003    |
| T013  | Update Cart Quantities         | NOT_STARTED  | P1       | F003    |
| T014  | Cart Total Calculation         | NOT_STARTED  | P1       | F003    |
+-------+--------------------------------+--------------+----------+---------+

Task Progress
----------------------------------------
[------------------------------] 0.0%

Total:       14
Completed:   0
In Progress: 0
Not Started: 14
Blocked:     0
----------------------------------------
```

---

## Step 6: View Task Details

```bash
hermes task T001
```

**Output:**

```
Task: T001
--------------------------------------------------
Name:     Database Schema for Users
Status:   NOT_STARTED
Priority: P1
Feature:  F001

Files to Touch:
  - db/migrations/001_create_users.sql
  - internal/models/user.go

Dependencies:
  - None

Success Criteria:
  - Users table created with id, email, password_hash, created_at
  - Email column has unique constraint
  - Migration runs without errors
  - Rollback works correctly
```

---

## Step 7: Start Task Execution

### Option A: Full Automation

```bash
hermes run --auto-branch --auto-commit
```

### Option B: Interactive Mode

```bash
hermes run --auto-branch --auto-commit --autonomous=false
```

### Option C: Use Specific AI Provider

```bash
hermes run --ai claude --auto-branch --auto-commit
hermes run --ai droid --auto-branch --auto-commit
hermes run --ai opencode --auto-branch --auto-commit
hermes run --ai gemini --auto-branch --auto-commit
```

### Option D: Parallel Execution (v2.0.0)

```bash
# Preview execution plan first
hermes run --dry-run

# Run with 3 parallel workers
hermes run --parallel --workers 3 --auto-commit
```

**Output:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      AI-Powered Application Development

Task Execution Loop
===================

[INFO] Using AI provider: claude

========================================
Loop #1
========================================

Task: T001 - Database Schema for Users
Feature: F001 - User Authentication
Priority: P1
Status: NOT_STARTED

[INFO] Working on task: T001 - Database Schema for Users
[INFO] On branch: feature/F001-user-authentication

... AI execution output ...

[SUCCESS] Task T001 completed
[SUCCESS] Committed task T001

Progress: [###---------------------------] 7.1%

========================================
Loop #2
========================================

Task: T002 - User Registration Endpoint
...
```

---

## Step 8: Monitor Progress

### Check Status

```bash
hermes status
```

**Output after some tasks complete:**

```
+-------+--------------------------------+--------------+----------+---------+
| ID    | Name                           | Status       | Priority | Feature |
+-------+--------------------------------+--------------+----------+---------+
| T001  | Database Schema for Users      | COMPLETED    | P1       | F001    |
| T002  | User Registration Endpoint     | COMPLETED    | P1       | F001    |
| T003  | Email Verification System      | IN_PROGRESS  | P1       | F001    |
| T004  | Login Endpoint with JWT        | NOT_STARTED  | P1       | F001    |
...
+-------+--------------------------------+--------------+----------+---------+

Task Progress
----------------------------------------
[######------------------------] 21.4%

Total:       14
Completed:   3
In Progress: 1
Not Started: 10
Blocked:     0
----------------------------------------
```

### View Logs

```bash
# Last 50 lines
hermes log

# Follow in real-time
hermes log -f

# Only errors
hermes log --level ERROR
```

### Use Interactive TUI

```bash
hermes tui
```

Navigate with:

- `1` - Dashboard
- `2` - Tasks list
- `3` - Logs
- `?` - Help

---

## Step 9: Handle Issues

### If Circuit Breaker Opens

```bash
# Check status
hermes status

# Reset circuit breaker
hermes reset

# Continue execution
hermes run --auto-branch --auto-commit
```

### If You Need to Stop

Press `Ctrl+C` during execution. Progress is saved automatically.

### Resume After Interruption

```bash
# Simply run again - Hermes resumes from last incomplete task
hermes run --auto-branch --auto-commit
```

---

## Step 10: Add New Feature

After initial development, add a new feature:

```bash
hermes add "order management with checkout flow"
```

**Output:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      AI-Powered Application Development

Feature Add
===========

Adding feature: order management with checkout flow

Next Feature ID: F004
Next Task ID: T015

Using AI: claude

Created: .hermes/tasks/004-order-management-with-che.md
```

---

## Step 11: Complete Project

Continue running until all tasks complete:

```bash
hermes run --auto-branch --auto-commit
```

**Final Output:**

```
[SUCCESS] All tasks completed!

Task Progress
----------------------------------------
[##############################] 100.0%

Total:       18
Completed:   18
In Progress: 0
Not Started: 0
Blocked:     0
----------------------------------------
```

---

## Git History

After completion, your git history looks like:

```
* feat(T018): Order confirmation email
* feat(T017): Payment processing integration
* feat(T016): Checkout endpoint
* feat(T015): Order database schema
* feat(T014): Cart total calculation
* feat(T013): Update cart quantities
* feat(T012): Add/Remove cart items
* feat(T011): Cart database schema
* feat(T010): Pagination implementation
* feat(T009): Product search and filtering
* feat(T008): Category management
* feat(T007): Product CRUD endpoints
* feat(T006): Database schema for products
* feat(T005): Password reset flow
* feat(T004): Login endpoint with JWT
* feat(T003): Email verification system
* feat(T002): User registration endpoint
* feat(T001): Database schema for users
* chore: Initialize project with Hermes
```

---

## Command Reference

| Step | Command                                  | Description                 |
|------|------------------------------------------|-----------------------------|
| 1    | `hermes init <name>`                     | Initialize project          |
| 2    | `hermes idea "<description>"`            | Generate PRD from idea      |
| 3    | Review/edit `.hermes/docs/PRD.md`        | Review requirements         |
| 4    | `hermes prd .hermes/docs/PRD.md`         | Parse PRD to tasks          |
| 5    | `hermes convertprd`                      | Convert PRD between formats |
| 6    | `hermes status`                          | View all tasks              |
| 7    | `hermes task <id>`                       | View task details           |
| 8    | `hermes run --auto-branch --auto-commit` | Execute tasks               |
| 9    | `hermes run --parallel --workers 3`      | Parallel execution          |
| 10   | `hermes run --dry-run`                   | Preview execution plan      |
| 11   | `hermes log -f`                          | Monitor logs                |
| 12   | `hermes reset`                           | Reset circuit breaker       |
| 13   | `hermes add "<feature>"`                 | Add new feature             |
| 14   | `hermes tui`                             | Interactive interface       |
| 15   | `hermes install`                         | Install to system PATH      |
| 16   | `hermes update`                          | Check and install updates   |

---

## Tips

1. **Start Small**: Begin with a focused PRD for better results
2. **Review Tasks**: Check generated tasks before running
3. **Use Branches**: Always use `--auto-branch` for clean history
4. **Monitor Progress**: Use `hermes tui` or `hermes log -f`
5. **Iterate**: Add features incrementally with `hermes add`
6. **Preview First**: Use `--dry-run` to see execution plan before running
7. **Parallel Mode**: Use `--parallel` for independent tasks to speed up
8. **Install System-wide**: Run `hermes install` for global access
