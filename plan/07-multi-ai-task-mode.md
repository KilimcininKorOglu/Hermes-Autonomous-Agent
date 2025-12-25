# Plan 07: Multi-AI Support for Task Mode

## Overview

Task Mode (`hermes -TaskMode`) currently uses only Claude CLI. This plan adds support for Droid and Aider CLIs, allowing users to choose their preferred AI provider for task execution.

## Current State

| Component | AI Support |
|-----------|------------|
| `hermes-prd.ps1` | claude, droid, aider (via AIProvider.ps1) |
| `hermes-add.ps1` | claude, droid, aider (via AIProvider.ps1) |
| `hermes_loop.ps1` | **claude only** (hard-coded) |

### Hard-coded Claude in hermes_loop.ps1

```powershell
# Line 108
$script:Config = @{
    ClaudeCommand = "claude"
    # ...
}

# Line 497
$result = $content | & $claudeCmd 2>&1
```

## Proposed Changes

### 1. Add -AI Parameter to hermes_loop.ps1

```powershell
param(
    # ... existing params
    [ValidateSet("claude", "droid", "aider", "auto")]
    [string]$AI = "auto"
)
```

### 2. Update Config Section

```powershell
# Load AIProvider module
. "$PSScriptRoot\lib\AIProvider.ps1"

# Resolve AI provider
$resolvedProvider = if ($AI -eq "auto") {
    Get-AutoProvider
} else {
    $AI
}

if (-not $resolvedProvider) {
    Write-Error "No AI provider found. Install claude, droid, or aider."
    exit 1
}

if (-not (Test-AIProvider -Provider $resolvedProvider)) {
    Write-Error "AI provider '$AI' not found."
    exit 1
}

$script:Config = @{
    # ...existing
    AIProvider = $resolvedProvider
    AICommand = $script:Providers[$resolvedProvider].Command
    # Remove: ClaudeCommand = "claude"
}
```

### 3. Update Invoke-ClaudeCode Function

Rename to `Invoke-AIExecution` and use provider-specific execution:

```powershell
function Invoke-AIExecution {
    param([int]$timeoutSeconds)
    
    $promptContent = Get-Content $script:Config.PromptFile -Raw
    $provider = $script:Config.AIProvider
    
    $job = Start-Job -ScriptBlock {
        param($content, $provider)
        
        switch ($provider) {
            "claude" {
                $content | claude 2>&1
            }
            "droid" {
                $content | droid exec --auto low 2>&1
            }
            "aider" {
                # Aider needs file-based approach
                $tempFile = [System.IO.Path]::GetTempFileName() + ".md"
                $content | Set-Content $tempFile
                aider --yes --no-auto-commits --message "Execute the instructions in this file" $tempFile 2>&1
                Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
            }
        }
    } -ArgumentList $promptContent, $provider
    
    # ... rest of timeout/monitoring logic
}
```

### 4. Update Help Text

```powershell
function Show-Help {
    Write-Host @"
Hermes Loop - Autonomous AI Development

USAGE:
    Hermes [-Monitor] [-Calls <int>] [-AI <provider>] ...

OPTIONS:
    -AI <provider>    AI provider: claude, droid, aider, auto (default: auto)
    # ... existing options
"@
}
```

### 5. Update Status Display

```powershell
Write-Host "[INFO] AI Provider: $($script:Config.AIProvider)" -ForegroundColor Cyan
```

## Files to Modify

| File | Changes |
|------|---------|
| `hermes_loop.ps1` | Add -AI param, use AIProvider.ps1, update execution |
| `lib/AIProvider.ps1` | Add `Invoke-TaskExecution` function (simpler than PRD) |

## New Function in AIProvider.ps1

```powershell
function Invoke-TaskExecution {
    <#
    .SYNOPSIS
        Execute AI for task mode (simpler than PRD parsing)
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptContent,
        
        [int]$TimeoutSeconds = 900
    )
    
    $job = Start-Job -ScriptBlock {
        param($content, $provider)
        
        switch ($provider) {
            "claude" {
                $content | claude 2>&1
            }
            "droid" {
                $content | droid exec --auto low 2>&1
            }
            "aider" {
                $tempFile = [System.IO.Path]::GetTempFileName() + ".md"
                $content | Set-Content $tempFile -Encoding UTF8
                try {
                    aider --yes --no-auto-commits --message "Execute the task described in this file" $tempFile 2>&1
                }
                finally {
                    Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
                }
            }
        }
    } -ArgumentList $PromptContent, $Provider
    
    $completed = Wait-Job $job -Timeout $TimeoutSeconds
    
    if (-not $completed) {
        Stop-Job $job -ErrorAction SilentlyContinue
        Remove-Job $job -Force -ErrorAction SilentlyContinue
        return @{
            Success = $false
            Error = "Timeout after $TimeoutSeconds seconds"
            Output = $null
        }
    }
    
    $output = Receive-Job $job
    $exitCode = $job.ChildJobs[0].JobStateInfo.Reason
    Remove-Job $job -Force
    
    return @{
        Success = $true
        Output = $output
        ExitCode = $exitCode
    }
}
```

## Usage Examples

```powershell
# Auto-detect (default, backward compatible)
hermes -TaskMode -AutoBranch -AutoCommit

# Explicit Claude
hermes -TaskMode -AI claude -AutoBranch -AutoCommit

# Use Droid
hermes -TaskMode -AI droid -AutoBranch -AutoCommit

# Use Aider
hermes -TaskMode -AI aider -AutoBranch -AutoCommit

# Autonomous with specific provider
hermes -TaskMode -AI droid -Autonomous
```

## Implementation Steps

1. **Update AIProvider.ps1**
   - Add `Invoke-TaskExecution` function
   - Export the new function

2. **Update hermes_loop.ps1**
   - Add `-AI` parameter with validation
   - Load AIProvider.ps1 at startup
   - Resolve provider (auto-detect or explicit)
   - Replace `Invoke-ClaudeCode` with new execution logic
   - Update progress/status messages to show provider name
   - Update help text

3. **Update Documentation**
   - README.md - update AI Integration section
   - CLAUDE.md - add -AI parameter to commands

4. **Add Unit Tests**
   - Test provider selection
   - Test auto-detection fallback
   - Test timeout handling per provider

## Backward Compatibility

- Default `-AI auto` ensures existing scripts work
- `claude` remains highest priority in auto-detection
- No breaking changes to existing parameters

## Estimated Effort

| Task | Estimate |
|------|----------|
| AIProvider.ps1 update | 1 hour |
| hermes_loop.ps1 refactor | 2 hours |
| Testing | 1 hour |
| Documentation | 30 min |
| **Total** | **4.5 hours** |

## Risks

| Risk | Mitigation |
|------|------------|
| Aider file-based approach complexity | Test thoroughly, document limitations |
| Different output formats per provider | ResponseAnalyzer may need updates |
| Provider-specific timeouts | Allow per-provider timeout config |

## Success Criteria

- [ ] `hermes -TaskMode -AI droid` works end-to-end
- [ ] `hermes -TaskMode -AI aider` works end-to-end
- [ ] `hermes -TaskMode` (auto) maintains backward compatibility
- [ ] Status display shows active provider
- [ ] All existing tests pass
- [ ] New unit tests for provider selection
