# Ralph Installation Guide

## Requirements

| Requirement    | Minimum Version | Notes                      |
|----------------|-----------------|----------------------------|
| PowerShell     | 7.0+            | NOT Windows PowerShell 5.1 |
| Git            | 2.30+           | For version control        |
| AI CLI         | -               | At least one required      |

### Supported AI CLIs

| CLI    | Installation Command                    |
|--------|-----------------------------------------|
| Claude | `npm install -g @anthropic-ai/claude-code` |
| Droid  | Available via Factory                   |
| Aider  | `pip install aider-chat`                |

---

## Quick Installation

```powershell
# Clone repository
git clone https://github.com/frankbria/ralph-claude-code.git
cd ralph-claude-code

# Install globally
.\install.ps1

# Verify
ralph -Help
```

---

## Step-by-Step Installation

### 1. Install PowerShell 7+

**Using winget:**

```powershell
winget install Microsoft.PowerShell
```

**Using Chocolatey:**

```powershell
choco install powershell-core
```

**Manual download:**

- Visit: <https://github.com/PowerShell/PowerShell/releases>
- Download the MSI installer for Windows

Verify installation:

```powershell
pwsh --version
# Should show: PowerShell 7.x.x
```

### 2. Install Git

**Using winget:**

```powershell
winget install Git.Git
```

**Using Chocolatey:**

```powershell
choco install git
```

Verify installation:

```powershell
git --version
```

### 3. Install AI CLI (at least one)

**Claude Code:**

```powershell
npm install -g @anthropic-ai/claude-code
claude --version
```

**Aider:**

```powershell
pip install aider-chat
aider --version
```

### 4. Install Ralph

```powershell
# Clone the repository
git clone https://github.com/frankbria/ralph-claude-code.git
cd ralph-claude-code

# Run installer
.\install.ps1
```

### 5. Restart Terminal

Close and reopen your terminal, or update PATH manually:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Ralph\bin;$env:PATH"
```

### 6. Verify Installation

```powershell
ralph -Help
ralph-prd -List
```

---

## Installation Paths

| Component  | Path                                 |
|------------|--------------------------------------|
| Commands   | `$env:LOCALAPPDATA\Ralph\bin\`       |
| Scripts    | `$env:LOCALAPPDATA\Ralph\`           |
| Templates  | `$env:LOCALAPPDATA\Ralph\templates\` |

Typically resolves to:

```
C:\Users\<username>\AppData\Local\Ralph\
```

---

## Uninstallation

```powershell
cd ralph-claude-code
.\install.ps1 -Uninstall
```

This removes:

- All Ralph commands from PATH
- Scripts from AppData
- Templates directory

---

## Troubleshooting

### "Running scripts is disabled"

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### "ralph is not recognized"

Restart terminal or manually add to PATH:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Ralph\bin;$env:PATH"
```

### "claude/droid/aider is not recognized"

Ensure at least one AI CLI is installed:

```powershell
# Check available providers
ralph-prd -List
```

### PowerShell version too old

Check your version:

```powershell
$PSVersionTable.PSVersion
```

If below 7.0, install PowerShell 7+:

```powershell
winget install Microsoft.PowerShell
```

Then use `pwsh` instead of `powershell`:

```powershell
pwsh
ralph -Help
```

---

## Post-Installation

### Create Your First Project

```powershell
ralph-setup my-project
cd my-project
```

### Parse a PRD

```powershell
ralph-prd docs/PRD.md -DryRun
ralph-prd docs/PRD.md
```

### Start Task Mode

```powershell
ralph -TaskMode -AutoBranch -AutoCommit
```

---

## Updating Ralph

```powershell
cd ralph-claude-code
git pull origin main
.\install.ps1
```

---

**Next:** See [USER-GUIDE.md](./USER-GUIDE.md) for complete usage documentation.
