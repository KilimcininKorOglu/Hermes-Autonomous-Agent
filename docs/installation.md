# Hermes Installation Guide

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
git clone https://github.com/frankbria/hermes-claude-code.git
cd hermes-claude-code

# Install globally
.\install.ps1

# Verify
Hermes -Help
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

### 4. Install Hermes

```powershell
# Clone the repository
git clone https://github.com/frankbria/hermes-claude-code.git
cd hermes-claude-code

# Run installer
.\install.ps1
```

### 5. Restart Terminal

Close and reopen your terminal, or update PATH manually:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Hermes\bin;$env:PATH"
```

### 6. Verify Installation

```powershell
Hermes -Help
hermes-prd -List
```

---

## Installation Paths

| Component  | Path                                 |
|------------|--------------------------------------|
| Commands   | `$env:LOCALAPPDATA\Hermes\bin\`       |
| Scripts    | `$env:LOCALAPPDATA\Hermes\`           |
| Templates  | `$env:LOCALAPPDATA\Hermes\templates\` |

Typically resolves to:

```
C:\Users\<username>\AppData\Local\Hermes\
```

---

## Uninstallation

```powershell
cd hermes-claude-code
.\install.ps1 -Uninstall
```

This removes:

- All Hermes commands from PATH
- Scripts from AppData
- Templates directory

---

## Troubleshooting

### "Running scripts is disabled"

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### "Hermes is not recognized"

Restart terminal or manually add to PATH:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Hermes\bin;$env:PATH"
```

### "claude/droid/aider is not recognized"

Ensure at least one AI CLI is installed:

```powershell
# Check available providers
hermes-prd -List
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
Hermes -Help
```

---

## Post-Installation

### Create Your First Project

```powershell
hermes-setup my-project
cd my-project
```

### Parse a PRD

```powershell
hermes-prd docs/PRD.md -DryRun
hermes-prd docs/PRD.md
```

### Start Task Mode

```powershell
Hermes -TaskMode -AutoBranch -AutoCommit
```

---

## Updating Hermes

```powershell
cd hermes-claude-code
git pull origin main
.\install.ps1
```

---

**Next:** See [USER-GUIDE.md](./USER-GUIDE.md) for complete usage documentation.
