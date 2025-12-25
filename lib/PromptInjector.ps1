#Requires -Version 7.0

<#
.SYNOPSIS
    Prompt Injector Module for Hermes
.DESCRIPTION
    Injects current task details into PROMPT.md for Claude execution.
    Manages task context section in prompt file.
#>

# Markers for task section
$script:TaskSectionStart = "<!-- HERMES_TASK_START -->"
$script:TaskSectionEnd = "<!-- HERMES_TASK_END -->"

function Get-PromptPath {
    <#
    .SYNOPSIS
        Gets the PROMPT.md file path
    #>
    param(
        [string]$BasePath = "."
    )
    
    return Join-Path $BasePath "PROMPT.md"
}

function Test-PromptExists {
    <#
    .SYNOPSIS
        Checks if PROMPT.md exists
    #>
    param(
        [string]$BasePath = "."
    )
    
    return Test-Path (Get-PromptPath -BasePath $BasePath)
}

function Get-TaskPromptSection {
    <#
    .SYNOPSIS
        Generates the task context section for PROMPT.md
    .OUTPUTS
        Formatted markdown string
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Task,
        
        [string]$FeatureName = "",
        
        [string]$BranchName = ""
    )
    
    $sb = [System.Text.StringBuilder]::new()
    
    [void]$sb.AppendLine($script:TaskSectionStart)
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("## CURRENT TASK")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("**Task ID:** $($Task.TaskId)")
    [void]$sb.AppendLine("**Task Name:** $($Task.Name)")
    
    if ($Task.FeatureId) {
        $featureDisplay = if ($FeatureName) { "$($Task.FeatureId) - $FeatureName" } else { $Task.FeatureId }
        [void]$sb.AppendLine("**Feature:** $featureDisplay")
    }
    
    if ($Task.Priority) {
        [void]$sb.AppendLine("**Priority:** $($Task.Priority)")
    }
    
    if ($BranchName) {
        [void]$sb.AppendLine("**Branch:** $BranchName")
    }
    
    [void]$sb.AppendLine("")
    
    # Description
    if ($Task.Description) {
        [void]$sb.AppendLine("### Description")
        [void]$sb.AppendLine("")
        [void]$sb.AppendLine($Task.Description)
        [void]$sb.AppendLine("")
    }
    
    # Technical Details
    if ($Task.TechnicalDetails) {
        [void]$sb.AppendLine("### Technical Details")
        [void]$sb.AppendLine("")
        [void]$sb.AppendLine($Task.TechnicalDetails)
        [void]$sb.AppendLine("")
    }
    
    # Files to Touch
    if ($Task.FilesToTouch -and $Task.FilesToTouch.Count -gt 0) {
        [void]$sb.AppendLine("### Files to Touch")
        [void]$sb.AppendLine("")
        foreach ($file in $Task.FilesToTouch) {
            [void]$sb.AppendLine("- ``$file``")
        }
        [void]$sb.AppendLine("")
    }
    
    # Success Criteria
    if ($Task.SuccessCriteria -and $Task.SuccessCriteria.Count -gt 0) {
        [void]$sb.AppendLine("### Success Criteria")
        [void]$sb.AppendLine("")
        foreach ($criteria in $Task.SuccessCriteria) {
            [void]$sb.AppendLine("- [ ] $criteria")
        }
        [void]$sb.AppendLine("")
    }
    
    # Dependencies info
    if ($Task.Dependencies -and $Task.Dependencies.Count -gt 0) {
        [void]$sb.AppendLine("### Dependencies")
        [void]$sb.AppendLine("")
        [void]$sb.AppendLine("Completed dependencies: $($Task.Dependencies -join ', ')")
        [void]$sb.AppendLine("")
    }
    
    # Instructions for Claude
    [void]$sb.AppendLine("---")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("**IMPORTANT:** Focus ONLY on this task. When complete, output:")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("``````")
    [void]$sb.AppendLine("---HERMES_STATUS---")
    [void]$sb.AppendLine("STATUS: COMPLETE")
    [void]$sb.AppendLine("TASK_ID: $($Task.TaskId)")
    [void]$sb.AppendLine("EXIT_SIGNAL: false")
    [void]$sb.AppendLine("RECOMMENDATION: Continue to next task")
    [void]$sb.AppendLine("---END_HERMES_STATUS---")
    [void]$sb.AppendLine("``````")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine($script:TaskSectionEnd)
    
    return $sb.ToString()
}

function Add-TaskToPrompt {
    <#
    .SYNOPSIS
        Adds or updates task section in PROMPT.md
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Task,
        
        [string]$FeatureName = "",
        
        [string]$BranchName = "",
        
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        Write-Host "[ERROR] PROMPT.md not found at: $promptPath" -ForegroundColor Red
        return $false
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    
    # Generate task section
    $taskSection = Get-TaskPromptSection -Task $Task -FeatureName $FeatureName -BranchName $BranchName
    
    # Check if task section already exists
    if ($content -match "$([regex]::Escape($script:TaskSectionStart)).*?$([regex]::Escape($script:TaskSectionEnd))") {
        # Replace existing section
        $content = $content -replace "(?s)$([regex]::Escape($script:TaskSectionStart)).*?$([regex]::Escape($script:TaskSectionEnd))", $taskSection.TrimEnd()
    }
    else {
        # Add at beginning (after first heading if exists)
        if ($content -match "^(#[^\r\n]+\r?\n)") {
            # Insert after first heading
            $content = $Matches[1] + "`n" + $taskSection + "`n" + $content.Substring($Matches[0].Length)
        }
        else {
            # Insert at beginning
            $content = $taskSection + "`n" + $content
        }
    }
    
    try {
        $content | Set-Content $promptPath -Encoding UTF8 -NoNewline
        Write-Host "[OK] Task $($Task.TaskId) injected into PROMPT.md" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "[ERROR] Failed to update PROMPT.md: $_" -ForegroundColor Red
        return $false
    }
}

function Remove-TaskFromPrompt {
    <#
    .SYNOPSIS
        Removes task section from PROMPT.md
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return $true
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    
    # Remove task section
    $pattern = "(?s)\r?\n?$([regex]::Escape($script:TaskSectionStart)).*?$([regex]::Escape($script:TaskSectionEnd))\r?\n?"
    $content = $content -replace $pattern, "`n"
    
    # Clean up multiple newlines
    $content = $content -replace "\r?\n{3,}", "`n`n"
    
    try {
        $content | Set-Content $promptPath -Encoding UTF8 -NoNewline
        Write-Host "[OK] Task section removed from PROMPT.md" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "[ERROR] Failed to update PROMPT.md: $_" -ForegroundColor Red
        return $false
    }
}

function Get-CurrentTaskFromPrompt {
    <#
    .SYNOPSIS
        Extracts current task ID from PROMPT.md
    .OUTPUTS
        Task ID string or empty
    #>
    param(
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return ""
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    
    if ($content -match "\*\*Task ID:\*\*\s*(T\d+)") {
        return $Matches[1]
    }
    
    return ""
}

function Test-TaskSectionExists {
    <#
    .SYNOPSIS
        Checks if task section exists in PROMPT.md
    .OUTPUTS
        Boolean
    #>
    param(
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return $false
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    return $content -match [regex]::Escape($script:TaskSectionStart)
}

function Update-TaskProgress {
    <#
    .SYNOPSIS
        Updates task progress indicator in PROMPT.md
    #>
    param(
        [string]$TaskId,
        
        [int]$LoopCount,
        
        [string]$LastAction = "",
        
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    
    # Look for progress section marker
    $progressMarker = "<!-- HERMES_PROGRESS -->"
    $progressEndMarker = "<!-- HERMES_PROGRESS_END -->"
    
    $timestamp = Get-Date -Format "HH:mm:ss"
    $progressSection = @"
$progressMarker
**Loop:** $LoopCount | **Time:** $timestamp | **Last:** $LastAction
$progressEndMarker
"@
    
    if ($content -match "$([regex]::Escape($progressMarker)).*?$([regex]::Escape($progressEndMarker))") {
        $content = $content -replace "(?s)$([regex]::Escape($progressMarker)).*?$([regex]::Escape($progressEndMarker))", $progressSection
    }
    elseif ($content -match [regex]::Escape($script:TaskSectionEnd)) {
        # Add before task section end
        $content = $content -replace [regex]::Escape($script:TaskSectionEnd), "$progressSection`n$($script:TaskSectionEnd)"
    }
    
    $content | Set-Content $promptPath -Encoding UTF8 -NoNewline
}

function Get-MinimalTaskPrompt {
    <#
    .SYNOPSIS
        Generates a minimal task prompt for constrained contexts
    .OUTPUTS
        Compact markdown string
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Task
    )
    
    $sb = [System.Text.StringBuilder]::new()
    
    [void]$sb.AppendLine("## Task: $($Task.TaskId) - $($Task.Name)")
    [void]$sb.AppendLine("")
    
    if ($Task.Description) {
        # Truncate description to first 500 chars
        $desc = $Task.Description
        if ($desc.Length -gt 500) {
            $desc = $desc.Substring(0, 500) + "..."
        }
        [void]$sb.AppendLine($desc)
        [void]$sb.AppendLine("")
    }
    
    if ($Task.FilesToTouch -and $Task.FilesToTouch.Count -gt 0) {
        [void]$sb.AppendLine("Files: $($Task.FilesToTouch -join ', ')")
        [void]$sb.AppendLine("")
    }
    
    if ($Task.SuccessCriteria -and $Task.SuccessCriteria.Count -gt 0) {
        [void]$sb.AppendLine("Criteria: $($Task.SuccessCriteria -join '; ')")
    }
    
    return $sb.ToString()
}

function Backup-Prompt {
    <#
    .SYNOPSIS
        Creates a backup of PROMPT.md
    .OUTPUTS
        Backup file path or empty string on failure
    #>
    param(
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return ""
    }
    
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupPath = Join-Path $BasePath ".Hermes" "prompt_backup_$timestamp.md"
    
    $backupDir = Split-Path $backupPath -Parent
    if (-not (Test-Path $backupDir)) {
        New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    }
    
    try {
        Copy-Item $promptPath $backupPath
        return $backupPath
    }
    catch {
        return ""
    }
}

function Restore-Prompt {
    <#
    .SYNOPSIS
        Restores PROMPT.md from a backup
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [Parameter(Mandatory)]
        [string]$BackupPath,
        
        [string]$BasePath = "."
    )
    
    if (-not (Test-Path $BackupPath)) {
        return $false
    }
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    try {
        Copy-Item $BackupPath $promptPath -Force
        return $true
    }
    catch {
        return $false
    }
}

function Get-LatestBackup {
    <#
    .SYNOPSIS
        Gets the most recent PROMPT.md backup
    .OUTPUTS
        Backup file path or empty string
    #>
    param(
        [string]$BasePath = "."
    )
    
    $backupDir = Join-Path $BasePath ".Hermes"
    
    if (-not (Test-Path $backupDir)) {
        return ""
    }
    
    $latest = Get-ChildItem -Path $backupDir -Filter "prompt_backup_*.md" |
              Sort-Object LastWriteTime -Descending |
              Select-Object -First 1
    
    if ($latest) {
        return $latest.FullName
    }
    
    return ""
}

function Add-TaskContextComment {
    <#
    .SYNOPSIS
        Adds an HTML comment with task context (hidden from display)
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Task,
        
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return $false
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    
    # JSON context for parsing
    $context = @{
        taskId = $Task.TaskId
        name = $Task.Name
        featureId = $Task.FeatureId
        priority = $Task.Priority
        timestamp = Get-Date -Format "o"
    } | ConvertTo-Json -Compress
    
    $comment = "<!-- HERMES_CONTEXT: $context -->"
    
    # Add at the very beginning
    $content = "$comment`n$content"
    
    $content | Set-Content $promptPath -Encoding UTF8 -NoNewline
    return $true
}

function Get-TaskContextFromComment {
    <#
    .SYNOPSIS
        Extracts task context from HTML comment
    .OUTPUTS
        Hashtable with task context or null
    #>
    param(
        [string]$BasePath = "."
    )
    
    $promptPath = Get-PromptPath -BasePath $BasePath
    
    if (-not (Test-Path $promptPath)) {
        return $null
    }
    
    $content = Get-Content $promptPath -Raw -Encoding UTF8
    
    if ($content -match "<!-- HERMES_CONTEXT: ({.*?}) -->") {
        try {
            return $Matches[1] | ConvertFrom-Json -AsHashtable
        }
        catch {
            return $null
        }
    }
    
    return $null
}

# Export functions for dot-sourcing
