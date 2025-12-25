<#
.SYNOPSIS
    Feature Analyzer Module for Hermes
.DESCRIPTION
    Analyzes feature descriptions and creates task breakdowns.
    Used by Hermes-add command.
#>

function Get-HighestFeatureId {
    <#
    .SYNOPSIS
        Gets the highest Feature ID from existing task files
    .OUTPUTS
        Integer (e.g., 1 for F001, 0 if none exist)
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksDir = Join-Path $BasePath "tasks"
    
    if (-not (Test-Path $tasksDir)) {
        return 0
    }
    
    $highest = 0
    $files = Get-ChildItem -Path $tasksDir -Filter "*.md" -ErrorAction SilentlyContinue
    
    foreach ($file in $files) {
        $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
        
        if ($content -match "\*\*Feature ID:\*\*\s*F(\d+)") {
            $num = [int]$Matches[1]
            if ($num -gt $highest) {
                $highest = $num
            }
        }
    }
    
    return $highest
}

function Get-HighestTaskId {
    <#
    .SYNOPSIS
        Gets the highest Task ID from existing task files
    .OUTPUTS
        Integer (e.g., 5 for T005, 0 if none exist)
    #>
    param(
        [string]$BasePath = "."
    )
    
    $tasksDir = Join-Path $BasePath "tasks"
    
    if (-not (Test-Path $tasksDir)) {
        return 0
    }
    
    $highest = 0
    $files = Get-ChildItem -Path $tasksDir -Filter "*.md" -ErrorAction SilentlyContinue
    
    foreach ($file in $files) {
        $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
        
        $matches = [regex]::Matches($content, "###\s+T(\d+):")
        foreach ($match in $matches) {
            $num = [int]$match.Groups[1].Value
            if ($num -gt $highest) {
                $highest = $num
            }
        }
    }
    
    return $highest
}

function Get-NextIds {
    <#
    .SYNOPSIS
        Gets the next available Feature ID and Task ID
    .OUTPUTS
        Hashtable with NextFeatureId and NextTaskId
    #>
    param(
        [string]$BasePath = "."
    )
    
    $highestFeature = Get-HighestFeatureId -BasePath $BasePath
    $highestTask = Get-HighestTaskId -BasePath $BasePath
    
    return @{
        NextFeatureId = $highestFeature + 1
        NextTaskId = $highestTask + 1
        NextFeatureIdPadded = ($highestFeature + 1).ToString().PadLeft(3, '0')
        NextTaskIdPadded = ($highestTask + 1).ToString().PadLeft(3, '0')
    }
}

function Read-FeatureInput {
    <#
    .SYNOPSIS
        Reads feature input from inline string or file
    .PARAMETER InputText
        Either inline text or @filepath
    .OUTPUTS
        String content
    #>
    param(
        [Parameter(Mandatory)]
        [string]$InputText
    )
    
    # Check if it's a file reference
    if ($InputText.StartsWith("@")) {
        $filePath = $InputText.Substring(1)
        
        if (-not (Test-Path $filePath)) {
            throw "File not found: $filePath"
        }
        
        $content = Get-Content $filePath -Raw -Encoding UTF8
        return @{
            Type = "file"
            Path = $filePath
            Content = $content
        }
    }
    else {
        return @{
            Type = "inline"
            Path = $null
            Content = $InputText
        }
    }
}

function ConvertTo-KebabCase {
    <#
    .SYNOPSIS
        Converts string to kebab-case for filenames
    #>
    param(
        [string]$Text
    )
    
    # Turkish characters
    $text = $Text.ToLower()
    $text = $text -replace 'ı', 'i'
    $text = $text -replace 'ğ', 'g'
    $text = $text -replace 'ü', 'u'
    $text = $text -replace 'ş', 's'
    $text = $text -replace 'ö', 'o'
    $text = $text -replace 'ç', 'c'
    $text = $text -replace 'İ', 'i'
    $text = $text -replace 'Ğ', 'g'
    $text = $text -replace 'Ü', 'u'
    $text = $text -replace 'Ş', 's'
    $text = $text -replace 'Ö', 'o'
    $text = $text -replace 'Ç', 'c'
    
    # Replace spaces and special chars with hyphens
    $text = $text -replace '[^a-z0-9]', '-'
    $text = $text -replace '-+', '-'
    $text = $text.Trim('-')
    
    return $text
}

function Get-FeatureFileName {
    <#
    .SYNOPSIS
        Generates feature file name
    #>
    param(
        [int]$FeatureNumber,
        [string]$FeatureName
    )
    
    $num = $FeatureNumber.ToString().PadLeft(3, '0')
    $name = ConvertTo-KebabCase -Text $FeatureName
    
    # Limit length
    if ($name.Length -gt 40) {
        $name = $name.Substring(0, 40).TrimEnd('-')
    }
    
    return "$num-$name.md"
}

function Build-FeaturePrompt {
    <#
    .SYNOPSIS
        Builds the prompt for AI feature analysis
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureDescription,
        
        [Parameter(Mandatory)]
        [int]$NextFeatureId,
        
        [Parameter(Mandatory)]
        [int]$NextTaskId,
        
        [string]$PriorityOverride
    )
    
    $scriptDir = Split-Path -Parent $PSScriptRoot
    $promptPath = Join-Path $scriptDir "lib\prompts\feature-analyzer.md"
    
    if (-not (Test-Path $promptPath)) {
        throw "Prompt template not found: $promptPath"
    }
    
    $template = Get-Content $promptPath -Raw -Encoding UTF8
    
    # Replace placeholders
    $template = $template -replace '\{FEATURE_DESCRIPTION\}', $FeatureDescription
    $template = $template -replace '\{NEXT_FEATURE_ID\}', $NextFeatureId.ToString().PadLeft(3, '0')
    $template = $template -replace '\{NEXT_TASK_ID\}', $NextTaskId.ToString().PadLeft(3, '0')
    $template = $template -replace '\{FILE_NUMBER\}', $NextFeatureId.ToString().PadLeft(3, '0')
    
    if ($PriorityOverride) {
        $template = $template -replace '\{PRIORITY_INSTRUCTION\}', "Priority Override: Use $PriorityOverride for all tasks."
    }
    else {
        $template = $template -replace '\{PRIORITY_INSTRUCTION\}', ""
    }
    
    return $template
}

function Parse-FeatureOutput {
    <#
    .SYNOPSIS
        Parses AI output to extract feature file
    .OUTPUTS
        Hashtable with FileName, Content, and metadata
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Output
    )
    
    # Use Split-AIOutput from AIProvider
    $files = @(Split-AIOutput -Output $Output)
    
    if ($files.Count -eq 0) {
        return $null
    }
    
    # Get the first (and should be only) feature file
    $file = $files[0]
    
    # Extract metadata from content
    $content = $file.Content
    
    $featureId = ""
    $featureName = ""
    $priority = ""
    $taskCount = 0
    $taskRange = ""
    $totalEffort = 0
    
    if ($content -match "\*\*Feature ID:\*\*\s*(F\d+)") {
        $featureId = $Matches[1]
    }
    
    if ($content -match "\*\*Feature Name:\*\*\s*(.+)") {
        $featureName = $Matches[1].Trim()
    }
    
    if ($content -match "\*\*Priority:\*\*\s*(P\d[^\r\n]*)") {
        $priority = $Matches[1].Trim()
    }
    
    $taskMatches = [regex]::Matches($content, "###\s+(T\d+):")
    $taskCount = $taskMatches.Count
    
    if ($taskCount -gt 0) {
        $firstTask = $taskMatches[0].Groups[1].Value
        $lastTask = $taskMatches[$taskCount - 1].Groups[1].Value
        $taskRange = "$firstTask-$lastTask"
    }
    
    # Calculate total effort
    $effortMatches = [regex]::Matches($content, "\*\*Estimated Effort:\*\*\s*([\d.]+)\s*day")
    foreach ($match in $effortMatches) {
        $totalEffort += [double]$match.Groups[1].Value
    }
    
    return @{
        FileName = $file.FileName
        Content = $content
        FeatureId = $featureId
        FeatureName = $featureName
        Priority = $priority
        TaskCount = $taskCount
        TaskRange = $taskRange
        TotalEffort = $totalEffort
    }
}

function Write-FeatureFile {
    <#
    .SYNOPSIS
        Writes feature file to tasks directory
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Feature,
        
        [string]$OutputDir = "tasks"
    )
    
    # Create directory if not exists
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    }
    
    # Extract filename from path
    $fileName = Split-Path -Leaf $Feature.FileName
    $filePath = Join-Path $OutputDir $fileName
    
    # Write file
    $Feature.Content | Set-Content $filePath -Encoding UTF8 -Force
    
    return $filePath
}

function Update-TasksStatus {
    <#
    .SYNOPSIS
        Updates tasks-status.md with new feature
    #>
    param(
        [Parameter(Mandatory)]
        [hashtable]$Feature,
        
        [string]$OutputDir = "tasks"
    )
    
    $statusFile = Join-Path $OutputDir "tasks-status.md"
    
    # Read existing or create new
    if (Test-Path $statusFile) {
        $content = Get-Content $statusFile -Raw -Encoding UTF8
    }
    else {
        $content = @"
# Task Status Tracker

**Last Updated:** $(Get-Date -Format "yyyy-MM-dd HH:mm")
**Total Features:** 0
**Total Tasks:** 0

## Progress Overview

| Feature | ID | Tasks | Completed | Progress |
|---------|-----|-------|-----------|----------|

## Task List

| Task | Name | Feature | Status | Priority |
|------|------|---------|--------|----------|

"@
    }
    
    # Update Last Updated
    $content = $content -replace "\*\*Last Updated:\*\*[^\r\n]+", "**Last Updated:** $(Get-Date -Format 'yyyy-MM-dd HH:mm')"
    
    # Update counts (simple increment)
    if ($content -match "\*\*Total Features:\*\*\s*(\d+)") {
        $currentFeatures = [int]$Matches[1]
        $content = $content -replace "\*\*Total Features:\*\*\s*\d+", "**Total Features:** $($currentFeatures + 1)"
    }
    
    if ($content -match "\*\*Total Tasks:\*\*\s*(\d+)") {
        $currentTasks = [int]$Matches[1]
        $content = $content -replace "\*\*Total Tasks:\*\*\s*\d+", "**Total Tasks:** $($currentTasks + $Feature.TaskCount)"
    }
    
    # Add feature to Progress Overview table
    $featureRow = "| $($Feature.FeatureName) | $($Feature.FeatureId) | $($Feature.TaskCount) | 0 | 0% |"
    
    # Find the table and add row
    if ($content -match "(## Progress Overview\r?\n\r?\n\|[^\r\n]+\|\r?\n\|[-\s|]+\|)") {
        $tableHeader = $Matches[1]
        $content = $content.Replace($tableHeader, "$tableHeader`n$featureRow")
    }
    
    $content | Set-Content $statusFile -Encoding UTF8 -Force
}
