<#
.SYNOPSIS
    Hermes Autonomous Agent - PRD Parser - Convert PRD to task-plan format
.DESCRIPTION
    Reads a PRD file and uses AI to generate task files in task-plan format
.EXAMPLE
    hermes-prd docs/PRD.md
    hermes-prd docs/PRD.md -AI claude
    hermes-prd docs/PRD.md -AI droid -DryRun
    hermes-prd -List
#>

param(
    [Parameter(Position = 0)]
    [string]$PrdFile,
    
    [ValidateSet("claude", "droid", "aider", "auto")]
    [string]$AI = "auto",
    
    [switch]$List,
    
    [switch]$DryRun,
    
    [string]$OutputDir = "tasks",
    
    [int]$Timeout = 1200,
    
    [int]$MaxRetries = 10,
    
    # Incremental update options
    [switch]$Force,     # Overwrite NOT_STARTED features
    [switch]$Clean      # Remove all existing tasks, start fresh
)

$ErrorActionPreference = "Stop"

# Get script directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Import modules
. "$scriptDir\lib\ConfigManager.ps1"
. "$scriptDir\lib\AIProvider.ps1"
. "$scriptDir\lib\TaskReader.ps1"

# Load configuration
$hermesConfig = Get-HermesConfig

# Get prompt template path
$promptTemplatePath = "$scriptDir\lib\prompts\prd-parser.md"

function Get-ExistingTaskState {
    <#
    .SYNOPSIS
        Reads current task state from tasks directory
    #>
    param([string]$TasksDir = "tasks")
    
    $state = @{
        Features = @{}
        HighestFeatureId = 0
        HighestTaskId = 0
        HasTasks = $false
    }
    
    if (-not (Test-Path $TasksDir)) {
        return $state
    }
    
    $files = Get-ChildItem -Path $TasksDir -Filter "*.md" -ErrorAction SilentlyContinue
    if (-not $files -or $files.Count -eq 0) {
        return $state
    }
    
    foreach ($file in $files) {
        # Skip status files
        if ($file.Name -match "status") { continue }
        
        $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
        if (-not $content) { continue }
        
        # Extract feature info
        $featureIdMatch = [regex]::Match($content, "\*\*Feature ID:\*\*\s*(F(\d+))")
        if (-not $featureIdMatch.Success) { continue }
        
        $featureId = $featureIdMatch.Groups[1].Value
        $featureNum = [int]$featureIdMatch.Groups[2].Value
        
        if ($featureNum -gt $state.HighestFeatureId) {
            $state.HighestFeatureId = $featureNum
        }
        
        # Extract feature name
        $nameMatch = [regex]::Match($content, "^# Feature \d+:\s*(.+)$", [System.Text.RegularExpressions.RegexOptions]::Multiline)
        $featureName = if ($nameMatch.Success) { $nameMatch.Groups[1].Value.Trim() } else { "" }
        
        # Extract feature status
        $statusMatch = [regex]::Match($content, "\*\*Status:\*\*\s*(NOT_STARTED|IN_PROGRESS|COMPLETED|BLOCKED)")
        $featureStatus = if ($statusMatch.Success) { $statusMatch.Groups[1].Value } else { "NOT_STARTED" }
        
        # Extract tasks
        $taskMatches = [regex]::Matches($content, "### (T(\d+)):\s*(.+)")
        $tasks = @{}
        
        foreach ($tm in $taskMatches) {
            $taskId = $tm.Groups[1].Value
            $taskNum = [int]$tm.Groups[2].Value
            $taskName = $tm.Groups[3].Value.Trim()
            
            if ($taskNum -gt $state.HighestTaskId) {
                $state.HighestTaskId = $taskNum
            }
            
            # Find task status
            $taskStatusPattern = "### $taskId.*?\*\*Status:\*\*\s*(NOT_STARTED|IN_PROGRESS|COMPLETED|BLOCKED)"
            $taskStatusMatch = [regex]::Match($content, $taskStatusPattern, [System.Text.RegularExpressions.RegexOptions]::Singleline)
            $taskStatus = if ($taskStatusMatch.Success) { $taskStatusMatch.Groups[1].Value } else { "NOT_STARTED" }
            
            $tasks[$taskId] = @{
                Name = $taskName
                Status = $taskStatus
            }
        }
        
        $state.Features[$featureId] = @{
            Name = $featureName
            Status = $featureStatus
            Tasks = $tasks
            FileName = $file.Name
            FilePath = $file.FullName
        }
        
        $state.HasTasks = $true
    }
    
    return $state
}

function Get-FeatureProgress {
    <#
    .SYNOPSIS
        Calculates progress percentage for a feature
    #>
    param([hashtable]$Feature)
    
    $total = $Feature.Tasks.Count
    if ($total -eq 0) { return 0 }
    
    $completed = ($Feature.Tasks.Values | Where-Object { $_.Status -eq "COMPLETED" }).Count
    return [Math]::Round(($completed / $total) * 100)
}

function Test-FeatureHasProgress {
    <#
    .SYNOPSIS
        Checks if any task in feature has been started or completed
    #>
    param([hashtable]$Feature)
    
    foreach ($task in $Feature.Tasks.Values) {
        if ($task.Status -ne "NOT_STARTED") {
            return $true
        }
    }
    return $false
}

function Write-IncrementalSummary {
    <#
    .SYNOPSIS
        Displays summary of incremental changes
    #>
    param(
        [hashtable]$ExistingState,
        [int]$NewFeatureCount,
        [int]$NewTaskCount
    )
    
    Write-Host ""
    Write-Host "Incremental Update Summary:" -ForegroundColor Cyan
    Write-Host ""
    
    # Preserved features
    $preserved = @()
    $completed = @()
    $inProgress = @()
    
    foreach ($fid in $ExistingState.Features.Keys) {
        $f = $ExistingState.Features[$fid]
        $progress = Get-FeatureProgress -Feature $f
        
        if ($f.Status -eq "COMPLETED" -or $progress -eq 100) {
            $completed += "  $fid`: $($f.Name) - 100%"
        }
        elseif (Test-FeatureHasProgress -Feature $f) {
            $inProgress += "  $fid`: $($f.Name) - $progress%"
        }
        else {
            $preserved += "  $fid`: $($f.Name)"
        }
    }
    
    if ($completed.Count -gt 0) {
        Write-Host "Preserved (completed):" -ForegroundColor Green
        $completed | ForEach-Object { Write-Host $_ -ForegroundColor Green }
        Write-Host ""
    }
    
    if ($inProgress.Count -gt 0) {
        Write-Host "Preserved (in progress):" -ForegroundColor Yellow
        $inProgress | ForEach-Object { Write-Host $_ -ForegroundColor Yellow }
        Write-Host ""
    }
    
    if ($preserved.Count -gt 0) {
        Write-Host "Preserved (not started):" -ForegroundColor Gray
        $preserved | ForEach-Object { Write-Host $_ -ForegroundColor Gray }
        Write-Host ""
    }
    
    if ($NewFeatureCount -gt 0) {
        Write-Host "Added:" -ForegroundColor Cyan
        Write-Host "  $NewFeatureCount new feature(s) with $NewTaskCount task(s)" -ForegroundColor Cyan
        Write-Host ""
    }
    else {
        Write-Host "No new features to add." -ForegroundColor Yellow
        Write-Host ""
    }
}

function Show-Usage {
    Write-Host ""
    Write-Host "Hermes Autonomous Agent - PRD Parser" -ForegroundColor Cyan
    Write-Host "================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage:" -ForegroundColor Yellow
    Write-Host "  hermes-prd <prd-file> [-AI <provider>] [-DryRun] [-OutputDir <dir>]"
    Write-Host "  hermes-prd -List"
    Write-Host ""
    Write-Host "Parameters:" -ForegroundColor Yellow
    Write-Host "  <prd-file>     Path to PRD markdown file"
    Write-Host "  -AI            AI provider: claude, droid, aider, auto (default: auto)"
    Write-Host "  -DryRun        Show what would be created without writing files"
    Write-Host "  -OutputDir     Output directory (default: tasks)"
    Write-Host "  -Timeout       AI timeout in seconds (default: 1200)"
    Write-Host "  -MaxRetries    Max retry attempts (default: 10)"
    Write-Host "  -Force         Overwrite NOT_STARTED features with new content"
    Write-Host "  -Clean         Remove all existing tasks, start fresh"
    Write-Host "  -List          List available AI providers"
    Write-Host ""
    Write-Host "Incremental Update:" -ForegroundColor Yellow
    Write-Host "  By default, hermes-prd preserves existing features and only adds new ones."
    Write-Host "  Completed and in-progress features are never overwritten."
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  hermes-prd docs/PRD.md"
    Write-Host "  hermes-prd docs/PRD.md -AI claude"
    Write-Host "  hermes-prd docs/PRD.md -DryRun"
    Write-Host ""
}

function Write-FilePreview {
    param(
        [array]$Files
    )
    
    Write-Host ""
    Write-Host "Files to create:" -ForegroundColor Cyan
    Write-Host ""
    
    foreach ($file in $Files) {
        $lines = ($file.Content -split "`n").Count
        Write-Host "  [+] $($file.FileName)" -ForegroundColor Green -NoNewline
        Write-Host " ($lines lines)" -ForegroundColor Gray
    }
    
    Write-Host ""
}

function Write-TaskFiles {
    param(
        [array]$Files,
        [string]$OutputDir
    )
    
    # Create output directory if not exists
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
        Write-Host "[OK] Created directory: $OutputDir" -ForegroundColor Green
    }
    
    $created = @()
    
    foreach ($file in $Files) {
        # Extract just filename from path like "tasks/001-feature.md"
        $fileName = Split-Path -Leaf $file.FileName
        $filePath = Join-Path $OutputDir $fileName
        
        # Write file
        $file.Content | Out-File -FilePath $filePath -Encoding UTF8 -Force
        
        # Extract stats
        $featureMatch = [regex]::Match($file.Content, "\*\*Feature ID:\*\*\s*(F\d+)")
        $taskMatches = [regex]::Matches($file.Content, "### (T\d+):")
        
        $featureId = if ($featureMatch.Success) { $featureMatch.Groups[1].Value } else { "-" }
        $taskRange = if ($taskMatches.Count -gt 0) { 
            $first = $taskMatches[0].Groups[1].Value
            $last = $taskMatches[$taskMatches.Count - 1].Groups[1].Value
            "$first-$last"
        }
        else { "" }
        
        $created += @{
            FileName  = $fileName
            FeatureId = $featureId
            TaskRange = $taskRange
        }
        
        if ($taskRange) {
            Write-Host "[OK] Created: $fileName ($featureId, $taskRange)" -ForegroundColor Green
        }
        else {
            Write-Host "[OK] Created: $fileName" -ForegroundColor Green
        }
    }
    
    return $created
}

function Get-Summary {
    param(
        [array]$Files
    )
    
    $featureCount = 0
    $taskCount = 0
    $totalEffort = 0
    
    foreach ($file in $Files) {
        if ($file.FileName -notmatch "status") {
            $featureCount++
            
            $taskMatches = [regex]::Matches($file.Content, "### T\d+:")
            $taskCount += $taskMatches.Count
            
            # Extract effort estimates
            $effortMatches = [regex]::Matches($file.Content, "Estimated Effort:\s*([\d.]+)\s*day")
            foreach ($match in $effortMatches) {
                $totalEffort += [double]$match.Groups[1].Value
            }
        }
    }
    
    return @{
        Features = $featureCount
        Tasks    = $taskCount
        Days     = $totalEffort
    }
}

# Main execution

# List mode
if ($List) {
    Write-AIProviderList
    exit 0
}

# Check if PRD file provided
if (-not $PrdFile) {
    Show-Usage
    exit 1
}

# Check if PRD file exists
if (-not (Test-Path $PrdFile)) {
    Write-Error "PRD file not found: $PrdFile"
    exit 1
}

# Check prompt template exists
if (-not (Test-Path $promptTemplatePath)) {
    Write-Error "Prompt template not found: $promptTemplatePath"
    exit 1
}

Write-Host ""
Write-Host "Hermes Autonomous Agent - PRD Parser" -ForegroundColor Cyan
Write-Host "================" -ForegroundColor Cyan
Write-Host ""

# Handle -Clean flag
if ($Clean) {
    if (Test-Path $OutputDir) {
        Write-Host "[WARN] Removing existing tasks directory: $OutputDir" -ForegroundColor Yellow
        Remove-Item -Path $OutputDir -Recurse -Force
        Write-Host "[OK] Tasks directory cleaned" -ForegroundColor Green
    }
}

# Check existing task state for incremental update
$existingState = Get-ExistingTaskState -TasksDir $OutputDir
$isIncremental = $existingState.HasTasks

if ($isIncremental) {
    Write-Host "[INFO] Existing tasks found - running in incremental mode" -ForegroundColor Cyan
    Write-Host "[INFO] Features: $($existingState.Features.Count), Highest ID: F$($existingState.HighestFeatureId)" -ForegroundColor Gray
    Write-Host "[INFO] Tasks: Highest ID: T$($existingState.HighestTaskId)" -ForegroundColor Gray
}

# Read and check PRD size
Write-Host "[INFO] Reading PRD: $PrdFile" -ForegroundColor Cyan
$prdInfo = Test-PrdSize -PrdFile $PrdFile

# Determine AI provider (CLI > config > auto-detect)
$configProvider = Get-ConfigValue -Key "ai.provider"
if ($AI -eq "auto" -and $configProvider -ne "auto") {
    $AI = $configProvider
} elseif ($AI -eq "auto") {
    $AI = Get-AutoProvider
    if (-not $AI) {
        Write-Error "No AI provider found. Install claude, droid, or aider."
        exit 1
    }
}

# Get timeout from config if not overridden (use prdTimeout for PRD parsing)
$configTimeout = Get-ConfigValue -Key "ai.prdTimeout"
if (-not $configTimeout) { $configTimeout = Get-ConfigValue -Key "ai.timeout" }
$configMaxRetries = Get-ConfigValue -Key "ai.maxRetries"
if ($Timeout -eq 1200 -and $configTimeout) { $Timeout = $configTimeout }
if ($MaxRetries -eq 10 -and $configMaxRetries) { $MaxRetries = $configMaxRetries }

# Verify provider is available
if (-not (Test-AIProvider -Provider $AI)) {
    Write-Error "AI provider '$AI' is not installed or not in PATH"
    exit 1
}

Write-Host "[INFO] Using AI: $AI" -ForegroundColor Cyan

# Load prompt template
$promptTemplate = Get-Content $promptTemplatePath -Raw

# Build incremental context if needed
$incrementalContext = ""
if ($isIncremental) {
    $nextFeatureId = $existingState.HighestFeatureId + 1
    $nextTaskId = $existingState.HighestTaskId + 1
    
    $existingFeaturesList = @()
    foreach ($fid in $existingState.Features.Keys | Sort-Object) {
        $f = $existingState.Features[$fid]
        $existingFeaturesList += "- $fid`: $($f.Name)"
    }
    
    $incrementalContext = @"

## INCREMENTAL UPDATE MODE

The following features already exist. DO NOT recreate them:
$($existingFeaturesList -join "`n")

Start numbering from:
- Next Feature ID: F$("{0:D3}" -f $nextFeatureId)
- Next Task ID: T$("{0:D3}" -f $nextTaskId)

ONLY output NEW features that are not listed above.
If there are no new features to add, output: NO_NEW_FEATURES

"@
    Write-Host "[INFO] Incremental context added to prompt" -ForegroundColor Gray
}

# Replace placeholder with PRD content
$fullPrompt = $promptTemplate -replace '\{PRD_CONTENT\}', $prdInfo.Content
$fullPrompt = $fullPrompt -replace '\{INCREMENTAL_CONTEXT\}', $incrementalContext

# Call AI with retry
Write-Host "[INFO] Parsing PRD with $AI..." -ForegroundColor Cyan
Write-Host ""

$result = Invoke-AIWithRetry -Provider $AI `
    -PromptText $fullPrompt `
    -Content $prdInfo.Content `
    -InputFile $PrdFile `
    -MaxRetries $MaxRetries `
    -TimeoutSeconds $Timeout

if (-not $result.Success) {
    Write-Error "Failed to parse PRD: $($result.Error)"
    exit 1
}

Write-Host ""

# Check for NO_NEW_FEATURES response
$noNewFeatures = $false
if ($result.Files.Count -eq 0 -or ($result.RawOutput -and $result.RawOutput -match "NO_NEW_FEATURES")) {
    $noNewFeatures = $true
}

# DryRun mode
if ($DryRun) {
    if ($noNewFeatures) {
        Write-Host "No new features detected in PRD." -ForegroundColor Yellow
        if ($isIncremental) {
            Write-IncrementalSummary -ExistingState $existingState -NewFeatureCount 0 -NewTaskCount 0
        }
    }
    else {
        Write-FilePreview -Files $result.Files
        
        $summary = Get-Summary -Files $result.Files
        
        Write-Host "Summary (DryRun):" -ForegroundColor Yellow
        Write-Host "  New Features: $($summary.Features)"
        Write-Host "  New Tasks: $($summary.Tasks)"
        Write-Host "  Estimated: $($summary.Days) days"
        
        if ($isIncremental) {
            Write-Host ""
            Write-Host "Existing features will be preserved." -ForegroundColor Gray
        }
    }
    Write-Host ""
    Write-Host "Run without -DryRun to create files." -ForegroundColor Cyan
    exit 0
}

# Handle no new features case
if ($noNewFeatures) {
    Write-Host "No new features detected in PRD." -ForegroundColor Yellow
    if ($isIncremental) {
        Write-IncrementalSummary -ExistingState $existingState -NewFeatureCount 0 -NewTaskCount 0
    }
    Write-Host ""
    Write-Host "All features in PRD already exist in tasks directory." -ForegroundColor Cyan
    exit 0
}

# Write files
$created = Write-TaskFiles -Files $result.Files -OutputDir $OutputDir

Write-Host ""

# Show summary
$summary = Get-Summary -Files $result.Files

if ($isIncremental) {
    Write-IncrementalSummary -ExistingState $existingState -NewFeatureCount $summary.Features -NewTaskCount $summary.Tasks
}

Write-Host "Summary:" -ForegroundColor Green
Write-Host "  New Features: $($summary.Features)"
Write-Host "  New Tasks: $($summary.Tasks)"
Write-Host "  Estimated: $($summary.Days) days"
Write-Host "  Attempts: $($result.Attempts)"

if ($isIncremental) {
    $totalFeatures = $existingState.Features.Count + $summary.Features
    $totalTasks = $existingState.HighestTaskId + $summary.Tasks
    Write-Host ""
    Write-Host "Total (including existing):" -ForegroundColor Gray
    Write-Host "  Features: $totalFeatures"
    Write-Host "  Tasks: ~$totalTasks"
}

Write-Host ""
Write-Host "Next: Run 'hermes -TaskMode -AutoBranch -AutoCommit' to start" -ForegroundColor Cyan
Write-Host ""
