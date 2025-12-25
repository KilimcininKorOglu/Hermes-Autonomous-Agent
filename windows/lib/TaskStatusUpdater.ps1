#Requires -Version 7.0

<#
.SYNOPSIS
    Task Status Updater Module for Ralph
.DESCRIPTION
    Updates task and feature statuses in task files.
    Maintains tasks-status.md tracking file.
#>

# Status values
$script:ValidStatuses = @(
    "NOT_STARTED",
    "IN_PROGRESS",
    "COMPLETED",
    "BLOCKED",
    "AT_RISK",
    "PAUSED"
)

function Set-TaskStatus {
    <#
    .SYNOPSIS
        Updates the status of a task in its feature file
    .PARAMETER TaskId
        Task ID (e.g., T001)
    .PARAMETER Status
        New status value
    .PARAMETER BasePath
        Base path for tasks directory
    #>
    param(
        [Parameter(Mandatory)]
        [string]$TaskId,
        
        [Parameter(Mandatory)]
        [ValidateSet("NOT_STARTED", "IN_PROGRESS", "COMPLETED", "BLOCKED", "AT_RISK", "PAUSED")]
        [string]$Status,
        
        [string]$BasePath = "."
    )
    
    # Find the task
    $task = Get-TaskById -TaskId $TaskId -BasePath $BasePath
    
    if (-not $task) {
        Write-Host "[ERROR] Task not found: $TaskId" -ForegroundColor Red
        return $false
    }
    
    # Read the feature file
    $filePath = $task.FeatureFile
    if (-not (Test-Path $filePath)) {
        Write-Host "[ERROR] Feature file not found: $filePath" -ForegroundColor Red
        return $false
    }
    
    $content = Get-Content $filePath -Raw -Encoding UTF8
    
    # Find and replace the status for this specific task
    # Pattern: ### TXXX: ... followed by **Status:** VALUE
    $taskPattern = "(###\s+$TaskId\s*:[^\r\n]*(?:\r?\n(?!###).*?)*?\*\*Status:\*\*\s*)(\w+)"
    
    if ($content -match $taskPattern) {
        $content = $content -replace $taskPattern, "`${1}$Status"
        $content | Set-Content $filePath -Encoding UTF8 -NoNewline
        Write-Host "[OK] Task $TaskId status updated to $Status" -ForegroundColor Green
        
        # Update tasks-status.md
        Update-TasksStatusFile -BasePath $BasePath
        
        return $true
    }
    else {
        Write-Host "[ERROR] Could not find status field for task $TaskId" -ForegroundColor Red
        return $false
    }
}

function Set-FeatureStatus {
    <#
    .SYNOPSIS
        Updates the status of a feature in its file
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [Parameter(Mandatory)]
        [ValidateSet("NOT_STARTED", "IN_PROGRESS", "COMPLETED", "BLOCKED", "AT_RISK", "PAUSED")]
        [string]$Status,
        
        [string]$BasePath = "."
    )
    
    $feature = Get-FeatureById -FeatureId $FeatureId -BasePath $BasePath
    
    if (-not $feature) {
        Write-Host "[ERROR] Feature not found: $FeatureId" -ForegroundColor Red
        return $false
    }
    
    $filePath = $feature.FilePath
    if (-not (Test-Path $filePath)) {
        Write-Host "[ERROR] Feature file not found: $filePath" -ForegroundColor Red
        return $false
    }
    
    $content = Get-Content $filePath -Raw -Encoding UTF8
    
    # Replace feature status (near the top of file)
    $pattern = "(\*\*Status:\*\*\s*)(\w+)"
    
    # Only replace the first occurrence (feature status, not task status)
    $match = [regex]::Match($content, $pattern)
    if ($match.Success) {
        $content = $content.Substring(0, $match.Index) + 
                   $match.Groups[1].Value + $Status +
                   $content.Substring($match.Index + $match.Length)
        
        $content | Set-Content $filePath -Encoding UTF8 -NoNewline
        Write-Host "[OK] Feature $FeatureId status updated to $Status" -ForegroundColor Green
        
        Update-TasksStatusFile -BasePath $BasePath
        
        return $true
    }
    
    return $false
}

function Update-TasksStatusFile {
    <#
    .SYNOPSIS
        Updates or creates tasks/tasks-status.md with current status
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksDir = Join-Path $BasePath "tasks"
    $statusFile = Join-Path $tasksDir "tasks-status.md"
    
    if (-not (Test-Path $tasksDir)) {
        return
    }
    
    # Get all data
    $features = Get-AllFeatures -BasePath $BasePath
    $allTasks = Get-AllTasks -BasePath $BasePath
    $progress = Get-TaskProgress -BasePath $BasePath
    
    # Build status content
    $sb = [System.Text.StringBuilder]::new()
    
    [void]$sb.AppendLine("# Task Status Tracker")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("**Last Updated:** $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
    [void]$sb.AppendLine("**Total Tasks:** $($progress.Total)")
    [void]$sb.AppendLine("**Completed:** $($progress.Completed)")
    [void]$sb.AppendLine("**In Progress:** $($progress.InProgress)")
    [void]$sb.AppendLine("**Not Started:** $($progress.NotStarted)")
    [void]$sb.AppendLine("**Blocked:** $($progress.Blocked)")
    [void]$sb.AppendLine("")
    
    # Progress bar
    $barWidth = 25
    $filled = [Math]::Floor(($progress.Percentage / 100) * $barWidth)
    $empty = $barWidth - $filled
    $bar = "[" + ("█" * $filled) + ("░" * $empty) + "]"
    [void]$sb.AppendLine("**Progress:** $bar $($progress.Percentage)%")
    [void]$sb.AppendLine("")
    
    # Feature table
    [void]$sb.AppendLine("## By Feature")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("| Feature | ID | Tasks | Completed | Progress |")
    [void]$sb.AppendLine("|---------|----|----|----------|----------|")
    
    foreach ($feature in $features) {
        $featureProgress = Get-FeatureProgress -FeatureId $feature.FeatureId -BasePath $BasePath
        $pct = "$($featureProgress.Percentage)%"
        [void]$sb.AppendLine("| $($feature.FeatureName) | $($feature.FeatureId) | $($featureProgress.Total) | $($featureProgress.Completed) | $pct |")
    }
    [void]$sb.AppendLine("")
    
    # Task list by status
    [void]$sb.AppendLine("## Task Details")
    [void]$sb.AppendLine("")
    
    # In Progress
    $inProgress = @($allTasks | Where-Object { $_.Status -eq "IN_PROGRESS" })
    if ($inProgress.Count -gt 0) {
        [void]$sb.AppendLine("### In Progress")
        foreach ($task in $inProgress) {
            [void]$sb.AppendLine("- **$($task.TaskId)**: $($task.Name) ($($task.FeatureId))")
        }
        [void]$sb.AppendLine("")
    }
    
    # Blocked
    $blocked = @($allTasks | Where-Object { $_.Status -eq "BLOCKED" })
    if ($blocked.Count -gt 0) {
        [void]$sb.AppendLine("### Blocked")
        foreach ($task in $blocked) {
            $deps = if ($task.Dependencies) { $task.Dependencies -join ", " } else { "N/A" }
            [void]$sb.AppendLine("- **$($task.TaskId)**: $($task.Name) - Blocked by: $deps")
        }
        [void]$sb.AppendLine("")
    }
    
    # Completed (last 10)
    $completed = @($allTasks | Where-Object { $_.Status -eq "COMPLETED" })
    if ($completed.Count -gt 0) {
        [void]$sb.AppendLine("### Recently Completed")
        $recent = $completed | Select-Object -Last 10
        foreach ($task in $recent) {
            [void]$sb.AppendLine("- [x] **$($task.TaskId)**: $($task.Name)")
        }
        [void]$sb.AppendLine("")
    }
    
    # Priority breakdown
    [void]$sb.AppendLine("## By Priority")
    [void]$sb.AppendLine("")
    $p1 = @($allTasks | Where-Object { $_.Priority -eq "P1" -and $_.Status -ne "COMPLETED" }).Count
    $p2 = @($allTasks | Where-Object { $_.Priority -eq "P2" -and $_.Status -ne "COMPLETED" }).Count
    $p3 = @($allTasks | Where-Object { $_.Priority -eq "P3" -and $_.Status -ne "COMPLETED" }).Count
    $p4 = @($allTasks | Where-Object { $_.Priority -eq "P4" -and $_.Status -ne "COMPLETED" }).Count
    [void]$sb.AppendLine("- **P1 (Critical):** $p1 remaining")
    [void]$sb.AppendLine("- **P2 (High):** $p2 remaining")
    [void]$sb.AppendLine("- **P3 (Medium):** $p3 remaining")
    [void]$sb.AppendLine("- **P4 (Low):** $p4 remaining")
    [void]$sb.AppendLine("")
    
    # Write file
    $sb.ToString() | Set-Content $statusFile -Encoding UTF8
}

function Add-TaskCompletionLog {
    <#
    .SYNOPSIS
        Adds a completion entry to the run state log
    #>
    param(
        [Parameter(Mandatory)]
        [string]$TaskId,
        
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [string]$Duration = "",
        
        [string]$CommitHash = "",
        
        [string]$BasePath = "."
    )
    
    $tasksDir = Join-Path $BasePath "tasks"
    $runStateFile = Join-Path $tasksDir "run-state.md"
    
    if (-not (Test-Path $tasksDir)) {
        New-Item -ItemType Directory -Path $tasksDir -Force | Out-Null
    }
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm"
    $logEntry = "| $TaskId | $FeatureId | COMPLETED | $timestamp | $Duration | $CommitHash |"
    
    if (Test-Path $runStateFile) {
        $content = Get-Content $runStateFile -Raw -Encoding UTF8
        
        # Find the Progress table and add entry
        if ($content -match "(\| Task \| Feature \| Status \| Started \| Completed \| Duration \|[\r\n]+\|[-|]+\|)") {
            # Add after header
            $insertPoint = $content.IndexOf($Matches[0]) + $Matches[0].Length
            $content = $content.Insert($insertPoint, "`n$logEntry")
            $content | Set-Content $runStateFile -Encoding UTF8 -NoNewline
        }
    }
    else {
        # Create new run state file
        $content = @"
# Task Plan Run State

**Started:** $(Get-Date -Format "o")
**Last Updated:** $(Get-Date -Format "o")
**Status:** IN_PROGRESS

## Progress

| Task | Feature | Status | Started | Completed | Duration |
|------|---------|--------|---------|-----------|----------|
$logEntry

## Execution Queue

See tasks/*.md for remaining tasks.

"@
        $content | Set-Content $runStateFile -Encoding UTF8
    }
}

function Update-RunState {
    <#
    .SYNOPSIS
        Updates the run-state.md file with current position
    #>
    param(
        [string]$CurrentTaskId,
        [string]$CurrentFeatureId,
        [string]$CurrentBranch,
        [string]$NextTaskId,
        [string]$Status = "IN_PROGRESS",
        [string]$BasePath = "."
    )
    
    $tasksDir = Join-Path $BasePath "tasks"
    $runStateFile = Join-Path $tasksDir "run-state.md"
    
    if (-not (Test-Path $tasksDir)) {
        New-Item -ItemType Directory -Path $tasksDir -Force | Out-Null
    }
    
    $progress = Get-TaskProgress -BasePath $BasePath
    
    $content = @"
# Task Plan Run State

**Started:** $(Get-Date -Format "o")
**Last Updated:** $(Get-Date -Format "o")
**Status:** $Status

## Current Position

- **Current Feature:** $CurrentFeatureId
- **Current Branch:** $CurrentBranch
- **Current Task:** $CurrentTaskId
- **Next Task:** $NextTaskId

## Summary

- **Total Features:** $((@(Get-AllFeatures -BasePath $BasePath)).Count)
- **Total Tasks:** $($progress.Total)
- **Completed:** $($progress.Completed)
- **In Progress:** $($progress.InProgress)
- **Remaining:** $($progress.NotStarted)
- **Blocked:** $($progress.Blocked)
- **Progress:** $($progress.Percentage)%

"@
    
    $content | Set-Content $runStateFile -Encoding UTF8
}

function Get-RunState {
    <#
    .SYNOPSIS
        Gets the current run state from file
    .OUTPUTS
        Hashtable with run state or null
    #>
    param(
        [string]$BasePath = "."
    )
    
    $runStateFile = Join-Path $BasePath "tasks" "run-state.md"
    
    if (-not (Test-Path $runStateFile)) {
        return $null
    }
    
    $content = Get-Content $runStateFile -Raw -Encoding UTF8
    
    $state = @{
        Status = "UNKNOWN"
        CurrentTaskId = ""
        CurrentFeatureId = ""
        CurrentBranch = ""
        NextTaskId = ""
    }
    
    if ($content -match "\*\*Status:\*\*\s*(\w+)") {
        $state.Status = $Matches[1]
    }
    
    if ($content -match "\*\*Current Task:\*\*\s*(T\d+)") {
        $state.CurrentTaskId = $Matches[1]
    }
    
    if ($content -match "\*\*Current Feature:\*\*\s*(F\d+)") {
        $state.CurrentFeatureId = $Matches[1]
    }
    
    if ($content -match "\*\*Current Branch:\*\*\s*(\S+)") {
        $state.CurrentBranch = $Matches[1]
    }
    
    if ($content -match "\*\*Next Task:\*\*\s*(T\d+)") {
        $state.NextTaskId = $Matches[1]
    }
    
    return $state
}

function Set-RunStateCompleted {
    <#
    .SYNOPSIS
        Marks the run state as completed
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksDir = Join-Path $BasePath "tasks"
    $runStateFile = Join-Path $tasksDir "run-state.md"
    
    if (Test-Path $runStateFile) {
        $content = Get-Content $runStateFile -Raw -Encoding UTF8
        $content = $content -replace "\*\*Status:\*\*\s*\w+", "**Status:** COMPLETED"
        $content = $content -replace "\*\*Last Updated:\*\*[^\r\n]+", "**Last Updated:** $(Get-Date -Format 'o')"
        $content | Set-Content $runStateFile -Encoding UTF8 -NoNewline
    }
    
    Update-TasksStatusFile -BasePath $BasePath
}

function Update-SuccessCriteria {
    <#
    .SYNOPSIS
        Marks success criteria as completed in task file
    #>
    param(
        [Parameter(Mandatory)]
        [string]$TaskId,
        
        [Parameter(Mandatory)]
        [int]$CriteriaIndex,
        
        [string]$BasePath = "."
    )
    
    $task = Get-TaskById -TaskId $TaskId -BasePath $BasePath
    
    if (-not $task) {
        return $false
    }
    
    $filePath = $task.FeatureFile
    $content = Get-Content $filePath -Raw -Encoding UTF8
    
    # Find the task section and its success criteria
    $taskPattern = "###\s+$TaskId\s*:[^\r\n]*"
    $taskMatch = [regex]::Match($content, $taskPattern)
    
    if (-not $taskMatch.Success) {
        return $false
    }
    
    # Find success criteria section within this task
    $taskStart = $taskMatch.Index
    $nextTaskMatch = [regex]::Match($content.Substring($taskStart + 1), "###\s+T\d+:")
    $taskEnd = if ($nextTaskMatch.Success) { $taskStart + 1 + $nextTaskMatch.Index } else { $content.Length }
    
    $taskSection = $content.Substring($taskStart, $taskEnd - $taskStart)
    
    # Find and update the specific checkbox
    $checkboxPattern = "- \[ \]"
    $matches = [regex]::Matches($taskSection, $checkboxPattern)
    
    if ($CriteriaIndex -lt $matches.Count) {
        $checkboxPos = $taskStart + $matches[$CriteriaIndex].Index
        $content = $content.Substring(0, $checkboxPos) + "- [x]" + $content.Substring($checkboxPos + 5)
        $content | Set-Content $filePath -Encoding UTF8 -NoNewline
        return $true
    }
    
    return $false
}

function Complete-AllSuccessCriteria {
    <#
    .SYNOPSIS
        Marks all success criteria as completed for a task
    #>
    param(
        [Parameter(Mandatory)]
        [string]$TaskId,
        
        [string]$BasePath = "."
    )
    
    $task = Get-TaskById -TaskId $TaskId -BasePath $BasePath
    
    if (-not $task) {
        return $false
    }
    
    $filePath = $task.FeatureFile
    $content = Get-Content $filePath -Raw -Encoding UTF8
    
    # Find the task section
    $taskPattern = "(###\s+$TaskId\s*:[^\r\n]*(?:\r?\n(?!###).*?)*)"
    
    if ($content -match $taskPattern) {
        $taskSection = $Matches[1]
        $updatedSection = $taskSection -replace "- \[ \]", "- [x]"
        $content = $content.Replace($taskSection, $updatedSection)
        $content | Set-Content $filePath -Encoding UTF8 -NoNewline
        return $true
    }
    
    return $false
}

# Export functions for dot-sourcing
