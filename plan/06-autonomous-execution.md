# Plan 06: Autonomous Execution Mode

## Ozet

`hermes -TaskMode` komutuna tam otonom calisma modu ekleme. Kullanici onay beklemeden tum tasklari ve feature'lari bitirene kadar devam etme.

## Mevcut Durum

### Var Olan

- `hermes -TaskMode` task loop'u calistiriyor
- Her loop'ta Claude Code cagriliyor
- Task completion detection var
- Auto-branch ve auto-commit var

### Eksikler

1. Her feature sonunda otomatik sonraki feature'a gecmiyor
2. "Durmadan devam et" davranisi yok
3. Feature summary output formati farkli
4. Tum feature'lar bitene kadar devam mekanizmasi yok

## Hedef

```powershell
hermes -TaskMode -AutoBranch -AutoCommit -Autonomous

# Veya varsayilan olarak autonomous
hermes -TaskMode -AutoBranch -AutoCommit
```

### Beklenen Davranis

```
[INFO] Starting Task Mode (Autonomous)
[INFO] Total Features: 3
[INFO] Total Tasks: 15

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Starting T001: Kayit formu UI (F001)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Claude execution...]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ T001: Kayit formu UI completed (45m)
  Files: 3 created, 1 modified
  Tests: 12 passed
  
  Progress: [████░░░░░░░░░░░░░░░░] 7% (1/15 tasks)
  Next: T002 - Input validation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Immediately continues to T002...]

...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ F001: User Registration - COMPLETED & MERGED

  Tasks: 5/5 completed
  Duration: 2h 15m
  Files: 17 changed
  
  Feature Progress: [████████░░░░░░░░░░░░] 33% (1/3 features)
  Next Feature: F002 - Password Reset
  
  Continuing automatically...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Immediately starts F002...]

...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ ALL TASKS COMPLETED

  Duration: 6h 45m
  Tasks: 15/15 completed
  Features: 3/3 completed
  
  Blocked: 0
  Errors: 2 (recovered)
  
  Git: All branches merged to main
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Teknik Tasarim

### Yeni Parametre

```powershell
# hermes_loop.ps1
param(
    # ... mevcut parametreler ...
    
    [switch]$Autonomous,     # Durmadan devam et
    [int]$MaxErrors = 5      # Max ard arda hata, sonra dur
)
```

### Execution Loop Degisiklikleri

```powershell
function Start-TaskModeLoop {
    # ... mevcut kod ...
    
    $consecutiveErrors = 0
    $startTime = Get-Date
    
    while ($true) {
        # Get next task
        $task = Get-NextTask -BasePath "."
        
        if (-not $task) {
            # ALL DONE
            Show-FinalSummary -StartTime $startTime
            break
        }
        
        # Execute task
        $result = Invoke-TaskExecution -Task $task
        
        if ($result.Success) {
            $consecutiveErrors = 0
            
            # Check if feature completed
            if (Test-FeatureCompleted -FeatureId $task.FeatureId) {
                Complete-Feature -FeatureId $task.FeatureId
                Show-FeatureSummary -FeatureId $task.FeatureId
                
                # IMMEDIATELY continue - no pause
            }
            else {
                Show-TaskSummary -Task $task -Result $result
            }
        }
        else {
            $consecutiveErrors++
            
            if ($consecutiveErrors -ge $MaxErrors) {
                Write-Status -Level "ERROR" -Message "Too many consecutive errors ($consecutiveErrors)"
                break
            }
        }
        
        # NO CONFIRMATION
        # NO "Do you want to continue?"
        # JUST CONTINUE
    }
}
```

### Summary Output Fonksiyonlari

```powershell
function Show-TaskSummary {
    param(
        [hashtable]$Task,
        [hashtable]$Result
    )
    
    $progress = Get-TaskProgress -BasePath "."
    $bar = Get-ProgressBar -Percentage $progress.Percentage -Width 20
    
    Write-Host ""
    Write-Host ("━" * 50) -ForegroundColor Cyan
    Write-Host "✓ $($Task.TaskId): $($Task.Name) completed ($($Result.Duration))" -ForegroundColor Green
    Write-Host "  Files: $($Result.FilesCreated) created, $($Result.FilesModified) modified"
    if ($Result.TestsPassed) {
        Write-Host "  Tests: $($Result.TestsPassed) passed"
    }
    Write-Host ""
    Write-Host "  Progress: $bar $($progress.Percentage)% ($($progress.Completed)/$($progress.Total) tasks)"
    
    $nextTask = Get-NextTask -BasePath "."
    if ($nextTask) {
        Write-Host "  Next: $($nextTask.TaskId) - $($nextTask.Name)"
    }
    Write-Host ("━" * 50) -ForegroundColor Cyan
    Write-Host ""
}

function Show-FeatureSummary {
    param(
        [string]$FeatureId
    )
    
    $feature = Get-FeatureById -FeatureId $FeatureId -BasePath "."
    $fp = Get-FeatureProgress -FeatureId $FeatureId -BasePath "."
    
    $allFeatures = Get-AllFeatures -BasePath "."
    $completedFeatures = @($allFeatures | Where-Object { $_.Status -eq "COMPLETED" }).Count
    $totalFeatures = $allFeatures.Count
    
    $bar = Get-ProgressBar -Percentage ([int](($completedFeatures / $totalFeatures) * 100)) -Width 20
    
    Write-Host ""
    Write-Host ("━" * 50) -ForegroundColor Green
    Write-Host "✓ $FeatureId: $($feature.FeatureName) - COMPLETED & MERGED" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Tasks: $($fp.Completed)/$($fp.Total) completed"
    Write-Host "  Duration: $($fp.Duration)"
    Write-Host "  Files: $($fp.FilesChanged) changed"
    Write-Host ""
    Write-Host "  Feature Progress: $bar $completedFeatures/$totalFeatures features"
    
    $nextFeature = Get-NextFeature -BasePath "."
    if ($nextFeature) {
        Write-Host "  Next Feature: $($nextFeature.FeatureId) - $($nextFeature.FeatureName)"
        Write-Host ""
        Write-Host "  Continuing automatically..." -ForegroundColor Cyan
    }
    Write-Host ("━" * 50) -ForegroundColor Green
    Write-Host ""
}

function Show-FinalSummary {
    param(
        [datetime]$StartTime
    )
    
    $duration = (Get-Date) - $StartTime
    $progress = Get-TaskProgress -BasePath "."
    $allFeatures = Get-AllFeatures -BasePath "."
    
    Write-Host ""
    Write-Host ("━" * 50) -ForegroundColor Green
    Write-Host "✓ ALL TASKS COMPLETED" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Duration: $($duration.ToString('hh\:mm\:ss'))"
    Write-Host "  Tasks: $($progress.Completed)/$($progress.Total) completed"
    Write-Host "  Features: $($allFeatures.Count)/$($allFeatures.Count) completed"
    Write-Host ""
    Write-Host "  Blocked: $($progress.Blocked)"
    Write-Host "  Errors: X (recovered)"
    Write-Host ""
    Write-Host "  Git: All branches merged to main"
    Write-Host ("━" * 50) -ForegroundColor Green
    Write-Host ""
}
```

### Progress Bar Helper

```powershell
function Get-ProgressBar {
    param(
        [int]$Percentage,
        [int]$Width = 20
    )
    
    $filled = [Math]::Floor(($Percentage / 100) * $Width)
    $empty = $Width - $filled
    
    $filledChar = [char]0x2588  # █
    $emptyChar = [char]0x2591   # ░
    
    return "[" + ($filledChar.ToString() * $filled) + ($emptyChar.ToString() * $empty) + "]"
}
```

## Uygulama Adimlari

1. [ ] `-Autonomous` parametresi ekle
2. [ ] `Show-TaskSummary` fonksiyonu yaz
3. [ ] `Show-FeatureSummary` fonksiyonu yaz
4. [ ] `Show-FinalSummary` fonksiyonu yaz
5. [ ] `Get-ProgressBar` helper fonksiyonu ekle
6. [ ] `Start-TaskModeLoop` guncelle (otonom davranis)
7. [ ] Feature completion check ekle
8. [ ] Max errors handling ekle
9. [ ] Unit testler
10. [ ] README.md guncelle

## Bagimliliklari

- `lib/TaskReader.ps1` (mevcut)
- `lib/TaskStatusUpdater.ps1` (mevcut)
- `lib/GitBranchManager.ps1` (mevcut)

## Tahmini Sure

3-4 saat
