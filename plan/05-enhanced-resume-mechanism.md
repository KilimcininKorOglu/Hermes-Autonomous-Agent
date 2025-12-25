# Plan 05: Enhanced Resume Mechanism

## Ozet

Task Mode'da otomatik resume mekanizmasini gelistirme. Context compaction veya beklenmedik durus sonrasi otomatik devam etme.

## Mevcut Durum

### Var Olan

```powershell
# lib/TaskStatusUpdater.ps1
Update-RunState       # run-state.md'yi gunceller
Get-RunState          # run-state.md'yi okur
Set-RunStateCompleted # Tamamlandi olarak isaretle
```

### Eksikler

1. Otomatik resume tetikleme yok
2. Error log detayli degil
3. Quick resume triggers (devam, continue) yok
4. Execution queue gosterimi yok

## Hedef

### 1. Otomatik Resume

```powershell
# Session baslangicinda
hermes -TaskMode

# Otomatik tespit:
# "run-state.md found with IN_PROGRESS status"
# "Resuming from T003..."
```

### 2. Quick Resume Triggers

```powershell
# Kullanici yazarsa:
devam
continue
devam et

# Otomatik resume baslar
```

### 3. Detayli run-state.md

```markdown
# Task Plan Run State

**Started:** 2024-01-15T10:00:00Z
**Last Updated:** 2024-01-15T14:30:00Z
**Status:** IN_PROGRESS

## Current Position

- **Current Feature:** F001
- **Current Branch:** feature/F001-user-registration
- **Current Task:** T003
- **Next Task:** T004

## Progress

| Task | Feature | Status | Started | Completed | Duration |
|------|---------|--------|---------|-----------|----------|
| T001 | F001 | COMPLETED | 10:00 | 10:45 | 45m |
| T002 | F001 | COMPLETED | 10:45 | 11:30 | 45m |
| T003 | F001 | IN_PROGRESS | 11:30 | - | - |

## Execution Queue

Priority-sorted remaining tasks:
1. T004 (P1, F001) - blocked by T003
2. T005 (P2, F001) - blocked by T004
3. T006 (P2, F002) - no deps, new feature branch needed

## Error Log

| Task | Attempt | Error | Timestamp |
|------|---------|-------|-----------|
| T003 | 1 | Build failed: missing dependency | 11:35 |
| T003 | 2 | Build failed: missing dependency | 11:40 |

## Summary

- **Total Features:** 2
- **Total Tasks:** 10
- **Completed:** 2
- **In Progress:** 1
- **Remaining:** 7
- **Blocked:** 0
- **Errors:** 2 (recovered)
```

## Teknik Tasarim

### Yeni Fonksiyonlar

```powershell
function Test-ShouldResume {
    <#
    .SYNOPSIS
        Checks if there's an active run state to resume
    .OUTPUTS
        Boolean
    #>
    param(
        [string]$BasePath = "."
    )
    
    $state = Get-RunState -BasePath $BasePath
    return ($state -and $state.Status -eq "IN_PROGRESS")
}

function Resume-TaskMode {
    <#
    .SYNOPSIS
        Resumes task mode from last checkpoint
    #>
    param(
        [string]$BasePath = "."
    )
    
    $state = Get-RunState -BasePath $BasePath
    
    if (-not $state) {
        Write-Host "[ERROR] No run state found" -ForegroundColor Red
        return $false
    }
    
    Write-Host "[INFO] Resuming from $($state.CurrentTaskId)..." -ForegroundColor Cyan
    
    # Switch to correct branch if needed
    if ($state.CurrentBranch) {
        Switch-ToFeatureBranch -BranchName $state.CurrentBranch
    }
    
    # Continue with StartFrom
    $script:Config.StartFromTask = $state.NextTaskId
    
    return $true
}

function Add-ErrorLogEntry {
    <#
    .SYNOPSIS
        Adds an error entry to run-state.md
    #>
    param(
        [string]$TaskId,
        [int]$Attempt,
        [string]$ErrorMessage,
        [string]$BasePath = "."
    )
}

function Get-ExecutionQueue {
    <#
    .SYNOPSIS
        Gets priority-sorted remaining tasks
    .OUTPUTS
        Array of tasks with dependency info
    #>
    param(
        [string]$BasePath = "."
    )
}

function Update-RunStateProgress {
    <#
    .SYNOPSIS
        Updates the progress table in run-state.md
    #>
    param(
        [string]$TaskId,
        [string]$Status,
        [datetime]$StartTime,
        [datetime]$EndTime,
        [string]$BasePath = "."
    )
}
```

### hermes_loop.ps1 Degisiklikleri

```powershell
# Task Mode baslarken
if ($TaskMode) {
    # Check for resume
    if (Test-ShouldResume -BasePath ".") {
        Write-Status -Level "INFO" -Message "Previous run detected. Resuming..."
        $resumed = Resume-TaskMode -BasePath "."
        
        if ($resumed) {
            Write-Status -Level "SUCCESS" -Message "Resumed from checkpoint"
        }
    }
    
    Start-TaskModeLoop
}
```

### Quick Resume Trigger (Ayri Script)

```powershell
# hermes-resume.ps1
<#
.SYNOPSIS
    Quick resume trigger for interrupted task mode
#>

$triggers = @("devam", "continue", "devam et", "resume")

# Check if user input matches trigger
if ($args -and $triggers -contains $args[0].ToLower()) {
    if (Test-ShouldResume -BasePath ".") {
        Write-Host "[INFO] Resuming task mode..." -ForegroundColor Cyan
        & "$PSScriptRoot\hermes_loop.ps1" -TaskMode -AutoBranch -AutoCommit
    }
    else {
        Write-Host "[INFO] No active run to resume" -ForegroundColor Yellow
    }
}
```

## Uygulama Adimlari

1. [ ] `Test-ShouldResume` fonksiyonu ekle
2. [ ] `Resume-TaskMode` fonksiyonu ekle
3. [ ] `Add-ErrorLogEntry` fonksiyonu ekle
4. [ ] `Update-RunStateProgress` fonksiyonu ekle
5. [ ] `hermes_loop.ps1` basinda resume check ekle
6. [ ] `run-state.md` formatini zenginlestir
7. [ ] `hermes-resume.ps1` script'i olustur (opsiyonel)
8. [ ] Unit testler
9. [ ] README.md guncelle

## Bagimliliklari

- `lib/TaskStatusUpdater.ps1` (mevcut)
- `lib/TaskReader.ps1` (mevcut)

## Tahmini Sure

2-3 saat
