#Requires -Version 7.0

<#
.SYNOPSIS
    Task Reader Module for Ralph
.DESCRIPTION
    Reads and parses task files from tasks/ directory.
    Supports task-plan.md format with Features (FXXX) and Tasks (TXXX).
#>

# Configuration
$script:TasksDir = "tasks"
$script:TaskFilePattern = "^\d{3}-.*\.md$"
$script:StatusFile = "tasks-status.md"

function Get-TasksDirectory {
    <#
    .SYNOPSIS
        Gets the tasks directory path, creates if not exists
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksPath = Join-Path $BasePath $script:TasksDir
    return $tasksPath
}

function Test-TasksDirectoryExists {
    <#
    .SYNOPSIS
        Checks if tasks directory exists
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksPath = Get-TasksDirectory -BasePath $BasePath
    return Test-Path $tasksPath
}

function Get-FeatureFiles {
    <#
    .SYNOPSIS
        Gets all feature files from tasks directory
    .OUTPUTS
        Array of file info objects
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksPath = Get-TasksDirectory -BasePath $BasePath
    
    if (-not (Test-Path $tasksPath)) {
        return @()
    }
    
    $files = Get-ChildItem -Path $tasksPath -Filter "*.md" | 
             Where-Object { $_.Name -match $script:TaskFilePattern } |
             Sort-Object Name
    
    return @($files)
}

function Read-FeatureFile {
    <#
    .SYNOPSIS
        Reads and parses a single feature file
    .OUTPUTS
        Hashtable with feature and tasks data
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FilePath
    )
    
    if (-not (Test-Path $FilePath)) {
        return $null
    }
    
    $content = Get-Content $FilePath -Raw -Encoding UTF8
    $lines = Get-Content $FilePath -Encoding UTF8
    
    $feature = @{
        FilePath = $FilePath
        FileName = Split-Path $FilePath -Leaf
        FeatureId = ""
        FeatureName = ""
        Priority = ""
        Status = "NOT_STARTED"
        TargetVersion = ""
        EstimatedDuration = ""
        Overview = ""
        Tasks = @()
    }
    
    # Parse feature header
    if ($content -match "(?m)^#\s+Feature\s+(\d+):\s*(.+)$") {
        $feature.FeatureId = "F" + $Matches[1].PadLeft(3, '0')
        $feature.FeatureName = $Matches[2].Trim()
    }
    
    if ($content -match "(?m)\*\*Feature ID:\*\*\s*(F?\d+)") {
        $feature.FeatureId = $Matches[1]
        if ($feature.FeatureId -notmatch "^F") {
            $feature.FeatureId = "F" + $feature.FeatureId.PadLeft(3, '0')
        }
    }
    
    if ($content -match "(?m)\*\*Feature Name:\*\*\s*(.+)$") {
        $feature.FeatureName = $Matches[1].Trim()
    }
    
    if ($content -match "(?m)\*\*Priority:\*\*\s*(P[1-4][^\r\n]*)") {
        $feature.Priority = $Matches[1].Trim()
    }
    
    if ($content -match "(?m)\*\*Status:\*\*\s*(\w+)") {
        $feature.Status = $Matches[1].Trim()
    }
    
    if ($content -match "(?m)\*\*Target Version:\*\*\s*(.+)$") {
        $feature.TargetVersion = $Matches[1].Trim()
    }
    
    if ($content -match "(?m)\*\*Estimated Duration:\*\*\s*(.+)$") {
        $feature.EstimatedDuration = $Matches[1].Trim()
    }
    
    # Parse tasks
    $feature.Tasks = @(Read-TasksFromContent -Content $content -FeatureId $feature.FeatureId -FeatureFile $FilePath)
    
    return $feature
}

function Read-TasksFromContent {
    <#
    .SYNOPSIS
        Extracts tasks from feature file content
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Content,
        
        [string]$FeatureId = "",
        
        [string]$FeatureFile = ""
    )
    
    $tasks = @()
    
    # Split by task headers (### TXXX: or ### T001:)
    $taskPattern = "(?m)^###\s+(T\d+):\s*(.+)$"
    $taskMatches = [regex]::Matches($Content, $taskPattern)
    
    for ($i = 0; $i -lt $taskMatches.Count; $i++) {
        $match = $taskMatches[$i]
        $taskId = $match.Groups[1].Value
        $taskName = $match.Groups[2].Value.Trim()
        
        # Get content until next task or end
        $startIndex = $match.Index + $match.Length
        $endIndex = if ($i -lt $taskMatches.Count - 1) {
            $taskMatches[$i + 1].Index
        } else {
            $Content.Length
        }
        
        $taskContent = $Content.Substring($startIndex, $endIndex - $startIndex)
        
        $task = @{
            TaskId = $taskId
            Name = $taskName
            Status = "NOT_STARTED"
            Priority = "P2"
            Effort = ""
            Description = ""
            TechnicalDetails = ""
            FilesToTouch = @()
            Dependencies = @()
            SuccessCriteria = @()
            FeatureId = $FeatureId
            FeatureFile = $FeatureFile
        }
        
        # Parse task properties
        if ($taskContent -match "(?m)\*\*Status:\*\*\s*(\w+)") {
            $task.Status = $Matches[1].Trim()
        }
        
        if ($taskContent -match "(?m)\*\*Priority:\*\*\s*(P[1-4])") {
            $task.Priority = $Matches[1].Trim()
        }
        
        if ($taskContent -match "(?m)\*\*Estimated Effort:\*\*\s*(.+)$") {
            $task.Effort = $Matches[1].Trim()
        }
        
        # Parse Description section
        if ($taskContent -match "(?ms)####\s*Description\s*\r?\n(.+?)(?=####|$)") {
            $task.Description = $Matches[1].Trim()
        }
        
        # Parse Technical Details section
        if ($taskContent -match "(?ms)####\s*Technical Details\s*\r?\n(.+?)(?=####|$)") {
            $task.TechnicalDetails = $Matches[1].Trim()
        }
        
        # Parse Files to Touch
        if ($taskContent -match "(?ms)####\s*Files to Touch\s*\r?\n(.+?)(?=####|$)") {
            $filesSection = $Matches[1]
            $fileMatches = [regex]::Matches($filesSection, "[-*]\s*`"?([^`"\r\n]+)`"?\s*\(?(new|update|delete)?\)?")
            foreach ($fileMatch in $fileMatches) {
                $task.FilesToTouch += $fileMatch.Groups[1].Value.Trim().Trim('`')
            }
        }
        
        # Parse Dependencies
        if ($taskContent -match "(?ms)####\s*Dependencies\s*\r?\n(.+?)(?=####|$)") {
            $depsSection = $Matches[1]
            $depMatches = [regex]::Matches($depsSection, "(T\d+)")
            foreach ($depMatch in $depMatches) {
                $task.Dependencies += $depMatch.Groups[1].Value
            }
            # Remove "None" or empty
            $task.Dependencies = @($task.Dependencies | Where-Object { $_ -and $_ -ne "None" })
        }
        
        # Parse Success Criteria
        if ($taskContent -match "(?ms)####\s*Success Criteria\s*\r?\n(.+?)(?=####|---|$)") {
            $criteriaSection = $Matches[1]
            $criteriaMatches = [regex]::Matches($criteriaSection, "[-*]\s*\[[ x]\]\s*(.+?)(?=\r?\n|$)")
            foreach ($criteriaMatch in $criteriaMatches) {
                $task.SuccessCriteria += $criteriaMatch.Groups[1].Value.Trim()
            }
        }
        
        $tasks += $task
    }
    
    return $tasks
}

function Get-AllTasks {
    <#
    .SYNOPSIS
        Gets all tasks from all feature files
    .OUTPUTS
        Array of task hashtables
    #>
    param(
        [string]$BasePath = "."
    )
    
    $allTasks = @()
    $featureFiles = Get-FeatureFiles -BasePath $BasePath
    
    foreach ($file in $featureFiles) {
        $feature = Read-FeatureFile -FilePath $file.FullName
        if ($feature -and $feature.Tasks) {
            $allTasks += $feature.Tasks
        }
    }
    
    return $allTasks
}

function Get-AllFeatures {
    <#
    .SYNOPSIS
        Gets all features from tasks directory
    .OUTPUTS
        Array of feature hashtables
    #>
    param(
        [string]$BasePath = "."
    )
    
    $features = @()
    $featureFiles = Get-FeatureFiles -BasePath $BasePath
    
    foreach ($file in $featureFiles) {
        $feature = Read-FeatureFile -FilePath $file.FullName
        if ($feature) {
            $features += $feature
        }
    }
    
    return $features
}

function Get-TaskById {
    <#
    .SYNOPSIS
        Gets a specific task by ID
    .OUTPUTS
        Task hashtable or null
    #>
    param(
        [Parameter(Mandatory)]
        [string]$TaskId,
        
        [string]$BasePath = "."
    )
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    return $allTasks | Where-Object { $_.TaskId -eq $TaskId } | Select-Object -First 1
}

function Get-FeatureById {
    <#
    .SYNOPSIS
        Gets a specific feature by ID
    .OUTPUTS
        Feature hashtable or null
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [string]$BasePath = "."
    )
    
    $features = Get-AllFeatures -BasePath $BasePath
    return $features | Where-Object { $_.FeatureId -eq $FeatureId } | Select-Object -First 1
}

function Get-TasksByStatus {
    <#
    .SYNOPSIS
        Gets tasks filtered by status
    .OUTPUTS
        Array of task hashtables
    #>
    param(
        [Parameter(Mandatory)]
        [ValidateSet("NOT_STARTED", "IN_PROGRESS", "COMPLETED", "BLOCKED", "AT_RISK", "PAUSED")]
        [string]$Status,
        
        [string]$BasePath = "."
    )
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    return @($allTasks | Where-Object { $_.Status -eq $Status })
}

function Get-TasksByFeature {
    <#
    .SYNOPSIS
        Gets tasks for a specific feature
    .OUTPUTS
        Array of task hashtables
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [string]$BasePath = "."
    )
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    return @($allTasks | Where-Object { $_.FeatureId -eq $FeatureId })
}

function Test-TaskDependenciesMet {
    <#
    .SYNOPSIS
        Checks if all dependencies of a task are completed
    .OUTPUTS
        Boolean
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Task,
        
        [string]$BasePath = "."
    )
    
    if (-not $Task.Dependencies -or $Task.Dependencies.Count -eq 0) {
        return $true
    }
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    $taskMap = @{}
    foreach ($t in $allTasks) {
        $taskMap[$t.TaskId] = $t
    }
    
    foreach ($depId in $Task.Dependencies) {
        if ($taskMap.ContainsKey($depId)) {
            if ($taskMap[$depId].Status -ne "COMPLETED") {
                return $false
            }
        }
    }
    
    return $true
}

function Get-NextTask {
    <#
    .SYNOPSIS
        Gets the next task to work on based on priority and dependencies
    .DESCRIPTION
        Finds tasks that are NOT_STARTED, have all dependencies met,
        and returns the highest priority one.
    .OUTPUTS
        Task hashtable or null
    #>
    param(
        [string]$BasePath = "."
    )
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    
    # Filter available tasks (NOT_STARTED with dependencies met)
    $availableTasks = @()
    foreach ($task in $allTasks) {
        if ($task.Status -eq "NOT_STARTED") {
            if (Test-TaskDependenciesMet -Task $task -BasePath $BasePath) {
                $availableTasks += $task
            }
        }
    }
    
    if ($availableTasks.Count -eq 0) {
        # Check for IN_PROGRESS tasks
        $inProgress = @($allTasks | Where-Object { $_.Status -eq "IN_PROGRESS" })
        if ($inProgress.Count -gt 0) {
            return $inProgress[0]
        }
        return $null
    }
    
    # Sort by priority (P1 > P2 > P3 > P4)
    $sorted = $availableTasks | Sort-Object { 
        switch ($_.Priority) {
            "P1" { 1 }
            "P2" { 2 }
            "P3" { 3 }
            "P4" { 4 }
            default { 5 }
        }
    }
    
    return $sorted | Select-Object -First 1
}

function Get-TaskProgress {
    <#
    .SYNOPSIS
        Gets overall task progress statistics
    .OUTPUTS
        Hashtable with progress data
    #>
    param(
        [string]$BasePath = "."
    )
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    $total = $allTasks.Count
    
    if ($total -eq 0) {
        return @{
            Total = 0
            Completed = 0
            InProgress = 0
            NotStarted = 0
            Blocked = 0
            Percentage = 0
        }
    }
    
    $completed = @($allTasks | Where-Object { $_.Status -eq "COMPLETED" }).Count
    $inProgress = @($allTasks | Where-Object { $_.Status -eq "IN_PROGRESS" }).Count
    $notStarted = @($allTasks | Where-Object { $_.Status -eq "NOT_STARTED" }).Count
    $blocked = @($allTasks | Where-Object { $_.Status -eq "BLOCKED" }).Count
    
    return @{
        Total = $total
        Completed = $completed
        InProgress = $inProgress
        NotStarted = $notStarted
        Blocked = $blocked
        Percentage = [Math]::Round(($completed / $total) * 100, 1)
    }
}

function Get-FeatureProgress {
    <#
    .SYNOPSIS
        Gets progress for a specific feature
    .OUTPUTS
        Hashtable with feature progress data
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [string]$BasePath = "."
    )
    
    $tasks = Get-TasksByFeature -FeatureId $FeatureId -BasePath $BasePath
    $total = $tasks.Count
    
    if ($total -eq 0) {
        return @{
            FeatureId = $FeatureId
            Total = 0
            Completed = 0
            Percentage = 0
            IsComplete = $false
        }
    }
    
    $completed = @($tasks | Where-Object { $_.Status -eq "COMPLETED" }).Count
    
    return @{
        FeatureId = $FeatureId
        Total = $total
        Completed = $completed
        Percentage = [Math]::Round(($completed / $total) * 100, 1)
        IsComplete = ($completed -eq $total)
    }
}

function Test-FeatureComplete {
    <#
    .SYNOPSIS
        Checks if all tasks in a feature are completed
    .OUTPUTS
        Boolean
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [string]$BasePath = "."
    )
    
    $progress = Get-FeatureProgress -FeatureId $FeatureId -BasePath $BasePath
    return $progress.IsComplete
}

function Get-HighestTaskId {
    <#
    .SYNOPSIS
        Gets the highest task ID number
    .OUTPUTS
        Integer
    #>
    param(
        [string]$BasePath = "."
    )
    
    $allTasks = Get-AllTasks -BasePath $BasePath
    $highest = 0
    
    foreach ($task in $allTasks) {
        if ($task.TaskId -match "T(\d+)") {
            $num = [int]$Matches[1]
            if ($num -gt $highest) {
                $highest = $num
            }
        }
    }
    
    return $highest
}

function Get-HighestFeatureId {
    <#
    .SYNOPSIS
        Gets the highest feature ID number
    .OUTPUTS
        Integer
    #>
    param(
        [string]$BasePath = "."
    )
    
    $features = Get-AllFeatures -BasePath $BasePath
    $highest = 0
    
    foreach ($feature in $features) {
        if ($feature.FeatureId -match "F(\d+)") {
            $num = [int]$Matches[1]
            if ($num -gt $highest) {
                $highest = $num
            }
        }
    }
    
    return $highest
}

# Export functions for dot-sourcing
