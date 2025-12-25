#Requires -Version 7.0

<#
.SYNOPSIS
    Ralph Monitor Dashboard - Windows PowerShell Version
.DESCRIPTION
    Live monitoring dashboard for Ralph Task Mode.
    Displays real-time information about task progress, current execution,
    circuit breaker state, and recent logs.
.PARAMETER RefreshInterval
    Refresh interval in seconds (default: 2)
.PARAMETER Help
    Show help message
.EXAMPLE
    .\ralph_monitor.ps1
.EXAMPLE
    .\ralph_monitor.ps1 -RefreshInterval 5
#>

[CmdletBinding()]
param(
    [int]$RefreshInterval = 2,
    
    [Alias('h')]
    [switch]$Help
)

# Configuration
$script:StatusFile = "status.json"
$script:ProgressFile = "progress.json"
$script:LogFile = "logs\ralph.log"
$script:CallCountFile = ".call_count"
$script:CircuitBreakerFile = ".circuit_breaker_state"
$script:RunStateFile = "tasks\run-state.md"
$script:TasksDir = "tasks"

# Import TaskReader module
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$taskReaderPath = Join-Path $scriptDir "lib\TaskReader.ps1"
if (Test-Path $taskReaderPath) {
    . $taskReaderPath
}

function Show-Help {
    Write-Host ""
    Write-Host "Ralph Monitor Dashboard" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: ralph-monitor [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Yellow
    Write-Host "    -RefreshInterval SEC   Refresh interval in seconds (default: 2)"
    Write-Host "    -Help                  Show this help message"
    Write-Host ""
    Write-Host "Controls:" -ForegroundColor Yellow
    Write-Host "    Ctrl+C                 Exit monitor"
    Write-Host ""
}

function Clear-Screen {
    [Console]::Clear()
    [Console]::SetCursorPosition(0, 0)
}

function Get-TerminalWidth {
    try {
        return [Console]::WindowWidth
    }
    catch {
        return 80
    }
}

function Write-Header {
    $width = Get-TerminalWidth
    $title = " RALPH TASK MODE MONITOR "
    $padding = [Math]::Max(0, [Math]::Floor(($width - $title.Length) / 2))
    
    Write-Host ("=" * $width) -ForegroundColor Cyan
    Write-Host (" " * $padding + $title) -ForegroundColor Cyan
    Write-Host ("=" * $width) -ForegroundColor Cyan
    Write-Host ""
}

function Get-LoopStatus {
    $default = @{
        status = "Not Running"
        loop_count = 0
        calls_made_this_hour = 0
        max_calls_per_hour = 100
        last_action = "N/A"
        exit_reason = ""
        timestamp = ""
    }
    
    if (-not (Test-Path $script:StatusFile)) {
        return $default
    }
    
    try {
        return Get-Content $script:StatusFile -Raw | ConvertFrom-Json
    }
    catch {
        return $default
    }
}

function Get-ProgressStatus {
    $default = @{
        status = "idle"
        indicator = "-"
        elapsed_seconds = 0
    }
    
    if (-not (Test-Path $script:ProgressFile)) {
        return $default
    }
    
    try {
        return Get-Content $script:ProgressFile -Raw | ConvertFrom-Json
    }
    catch {
        return $default
    }
}

function Get-CircuitBreakerStatus {
    $default = @{
        state = "CLOSED"
        reason = "Normal operation"
        consecutive_no_progress = 0
    }
    
    if (-not (Test-Path $script:CircuitBreakerFile)) {
        return $default
    }
    
    try {
        return Get-Content $script:CircuitBreakerFile -Raw | ConvertFrom-Json
    }
    catch {
        return $default
    }
}

function Get-RunStateInfo {
    $default = @{
        CurrentTask = ""
        CurrentFeature = ""
        CurrentBranch = ""
        Status = "idle"
    }
    
    if (-not (Test-Path $script:RunStateFile)) {
        return $default
    }
    
    try {
        $content = Get-Content $script:RunStateFile -Raw
        
        if ($content -match "\*\*Current Task:\*\*\s*(\S+)") {
            $default.CurrentTask = $Matches[1]
        }
        if ($content -match "\*\*Current Feature:\*\*\s*(\S+)") {
            $default.CurrentFeature = $Matches[1]
        }
        if ($content -match "\*\*Branch:\*\*\s*(\S+)") {
            $default.CurrentBranch = $Matches[1]
        }
        if ($content -match "\*\*Status:\*\*\s*(\S+)") {
            $default.Status = $Matches[1]
        }
        
        return $default
    }
    catch {
        return $default
    }
}

function Get-TaskProgressInfo {
    if (-not (Get-Command Get-TaskProgress -ErrorAction SilentlyContinue)) {
        # Fallback if TaskReader not loaded
        return @{
            Total = 0
            Completed = 0
            InProgress = 0
            NotStarted = 0
            Blocked = 0
            Percentage = 0
        }
    }
    
    try {
        return Get-TaskProgress -BasePath "."
    }
    catch {
        return @{
            Total = 0
            Completed = 0
            InProgress = 0
            NotStarted = 0
            Blocked = 0
            Percentage = 0
        }
    }
}

function Get-FeatureProgressInfo {
    if (-not (Get-Command Get-AllFeatures -ErrorAction SilentlyContinue)) {
        return @{
            Total = 0
            Completed = 0
            InProgress = 0
        }
    }
    
    try {
        $features = Get-AllFeatures -BasePath "."
        $total = $features.Count
        $completed = @($features | Where-Object { $_.Status -eq "COMPLETED" }).Count
        $inProgress = @($features | Where-Object { $_.Status -eq "IN_PROGRESS" }).Count
        
        return @{
            Total = $total
            Completed = $completed
            InProgress = $inProgress
        }
    }
    catch {
        return @{
            Total = 0
            Completed = 0
            InProgress = 0
        }
    }
}

function Get-CurrentTaskInfo {
    $runState = Get-RunStateInfo
    
    if ([string]::IsNullOrEmpty($runState.CurrentTask)) {
        return $null
    }
    
    if (-not (Get-Command Get-TaskById -ErrorAction SilentlyContinue)) {
        return @{
            TaskId = $runState.CurrentTask
            Name = "Unknown"
            FeatureId = $runState.CurrentFeature
        }
    }
    
    try {
        return Get-TaskById -TaskId $runState.CurrentTask -BasePath "."
    }
    catch {
        return @{
            TaskId = $runState.CurrentTask
            Name = "Unknown"
            FeatureId = $runState.CurrentFeature
        }
    }
}

function Write-ProgressBar {
    param(
        [int]$Current,
        [int]$Total,
        [int]$Width = 30,
        [string]$ForegroundColor = "Green"
    )
    
    if ($Total -eq 0) {
        $percentage = 0
        $filled = 0
    }
    else {
        $percentage = [Math]::Round(($Current / $Total) * 100)
        $filled = [Math]::Floor(($Current / $Total) * $Width)
    }
    
    $empty = $Width - $filled
    $bar = "[" + ("█" * $filled) + ("░" * $empty) + "]"
    
    Write-Host $bar -ForegroundColor $ForegroundColor -NoNewline
    Write-Host " $Current/$Total ($percentage%)"
}

function Write-RateLimitBar {
    param(
        [int]$Current,
        [int]$Max
    )
    
    $barWidth = 30
    $filled = [Math]::Min($barWidth, [Math]::Floor(($Current / [Math]::Max(1, $Max)) * $barWidth))
    $empty = $barWidth - $filled
    
    $bar = "[" + ("=" * $filled) + (" " * $empty) + "]"
    $percentage = [Math]::Round(($Current / [Math]::Max(1, $Max)) * 100)
    
    $color = if ($percentage -lt 50) { "Green" }
             elseif ($percentage -lt 80) { "Yellow" }
             else { "Red" }
    
    Write-Host "  API Calls: " -NoNewline
    Write-Host $bar -ForegroundColor $color -NoNewline
    Write-Host " $Current/$Max ($percentage%)"
}

function Get-RecentLogs {
    param([int]$Lines = 10)
    
    if (-not (Test-Path $script:LogFile)) {
        return @("No log file found")
    }
    
    try {
        return Get-Content $script:LogFile -Tail $Lines -ErrorAction SilentlyContinue
    }
    catch {
        return @("Error reading log file")
    }
}

function Write-LogSection {
    param([int]$Lines = 6)
    
    Write-Host "[Recent Logs]" -ForegroundColor Yellow
    
    $logs = Get-RecentLogs -Lines $Lines
    foreach ($line in $logs) {
        if ([string]::IsNullOrEmpty($line)) { continue }
        
        $color = "Gray"
        if ($line -match "\[ERROR\]") { $color = "Red" }
        elseif ($line -match "\[WARN\]") { $color = "Yellow" }
        elseif ($line -match "\[SUCCESS\]") { $color = "Green" }
        elseif ($line -match "\[LOOP\]") { $color = "Magenta" }
        elseif ($line -match "\[INFO\]") { $color = "Cyan" }
        
        # Truncate long lines
        $maxLen = (Get-TerminalWidth) - 4
        if ($line.Length -gt $maxLen) {
            $line = $line.Substring(0, $maxLen - 3) + "..."
        }
        
        Write-Host "  $line" -ForegroundColor $color
    }
    Write-Host ""
}

function Write-Dashboard {
    Clear-Screen
    Write-Header
    
    # Get all status info
    $loopStatus = Get-LoopStatus
    $progressStatus = Get-ProgressStatus
    $circuitStatus = Get-CircuitBreakerStatus
    $runState = Get-RunStateInfo
    $taskProgress = Get-TaskProgressInfo
    $featureProgress = Get-FeatureProgressInfo
    $currentTask = Get-CurrentTaskInfo
    
    # Task Progress Section
    Write-Host "[Task Progress]" -ForegroundColor Yellow
    Write-Host "  Tasks:    " -NoNewline
    Write-ProgressBar -Current $taskProgress.Completed -Total $taskProgress.Total -ForegroundColor "Green"
    Write-Host "  Features: " -NoNewline
    Write-ProgressBar -Current $featureProgress.Completed -Total $featureProgress.Total -Width 30 -ForegroundColor "Cyan"
    Write-Host ""
    
    # Current Task Section
    Write-Host "[Current Task]" -ForegroundColor Yellow
    if ($currentTask) {
        $statusColor = switch ($runState.Status) {
            "IN_PROGRESS" { "Yellow" }
            "COMPLETED" { "Green" }
            "BLOCKED" { "Red" }
            default { "White" }
        }
        
        Write-Host "  Task:      " -NoNewline
        Write-Host "$($currentTask.TaskId)" -ForegroundColor Cyan -NoNewline
        Write-Host " - $($currentTask.Name)"
        
        Write-Host "  Feature:   " -NoNewline
        Write-Host "$($runState.CurrentFeature)" -ForegroundColor Magenta
        
        if (-not [string]::IsNullOrEmpty($runState.CurrentBranch)) {
            Write-Host "  Branch:    " -NoNewline
            Write-Host "$($runState.CurrentBranch)" -ForegroundColor Blue
        }
        
        Write-Host "  Status:    " -NoNewline
        Write-Host "$($runState.Status)" -ForegroundColor $statusColor
    }
    else {
        Write-Host "  No active task" -ForegroundColor Gray
    }
    Write-Host ""
    
    # Execution Status Section
    Write-Host "[Execution]" -ForegroundColor Yellow
    
    $statusColor = switch ($loopStatus.status) {
        "running" { "Green" }
        "success" { "Green" }
        "paused"  { "Yellow" }
        "halted"  { "Red" }
        "error"   { "Red" }
        "completed" { "Green" }
        default   { "Gray" }
    }
    
    Write-Host "  Status:    " -NoNewline
    Write-Host $loopStatus.status -ForegroundColor $statusColor
    Write-Host "  Loops:     $($loopStatus.loop_count)"
    
    if ($progressStatus.status -eq "executing") {
        $elapsed = $progressStatus.elapsed_seconds
        $minutes = [Math]::Floor($elapsed / 60)
        $seconds = $elapsed % 60
        Write-Host "  Running:   ${minutes}m ${seconds}s" -ForegroundColor Cyan
    }
    
    if (-not [string]::IsNullOrEmpty($loopStatus.exit_reason)) {
        Write-Host "  Exit:      $($loopStatus.exit_reason)" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Task Statistics
    Write-Host "[Statistics]" -ForegroundColor Yellow
    Write-Host "  Completed:   " -NoNewline
    Write-Host "$($taskProgress.Completed)" -ForegroundColor Green
    Write-Host "  In Progress: " -NoNewline
    Write-Host "$($taskProgress.InProgress)" -ForegroundColor Yellow
    Write-Host "  Not Started: " -NoNewline
    Write-Host "$($taskProgress.NotStarted)" -ForegroundColor Gray
    Write-Host "  Blocked:     " -NoNewline
    Write-Host "$($taskProgress.Blocked)" -ForegroundColor Red
    Write-Host ""
    
    # Rate Limit Section
    Write-Host "[Rate Limiting]" -ForegroundColor Yellow
    
    $callsMade = 0
    if (Test-Path $script:CallCountFile) {
        try {
            $callsMade = [int](Get-Content $script:CallCountFile -Raw).Trim()
        }
        catch { }
    }
    
    Write-RateLimitBar -Current $callsMade -Max $loopStatus.max_calls_per_hour
    Write-Host ""
    
    # Circuit Breaker Section
    Write-Host "[Circuit Breaker]" -ForegroundColor Yellow
    
    $cbColor = switch ($circuitStatus.state) {
        "CLOSED"    { "Green" }
        "HALF_OPEN" { "Yellow" }
        "OPEN"      { "Red" }
        default     { "Gray" }
    }
    
    $cbIcon = switch ($circuitStatus.state) {
        "CLOSED"    { "[OK]" }
        "HALF_OPEN" { "[!!]" }
        "OPEN"      { "[XX]" }
        default     { "[??]" }
    }
    
    Write-Host "  State:     " -NoNewline
    Write-Host "$cbIcon $($circuitStatus.state)" -ForegroundColor $cbColor
    Write-Host "  No-Progress: $($circuitStatus.consecutive_no_progress) loops"
    Write-Host ""
    
    # Logs Section
    Write-LogSection -Lines 6
    
    # Footer
    $width = Get-TerminalWidth
    Write-Host ("=" * $width) -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to exit | Refresh: ${RefreshInterval}s | $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
}

function Start-Monitor {
    # Check if in Ralph project
    $isRalphProject = (Test-Path "PROMPT.md") -or (Test-Path $script:StatusFile) -or (Test-Path $script:TasksDir)
    
    if (-not $isRalphProject) {
        Write-Host ""
        Write-Host "Warning: This doesn't appear to be a Ralph project directory." -ForegroundColor Yellow
        Write-Host "Looking for: PROMPT.md, status.json, or tasks/" -ForegroundColor Gray
        Write-Host "Current directory: $(Get-Location)" -ForegroundColor Gray
        Write-Host ""
        Write-Host "The monitor will still run, but may not show useful information." -ForegroundColor Yellow
        Write-Host ""
        Start-Sleep -Seconds 2
    }
    
    Write-Host "Starting Ralph Task Mode Monitor..." -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
    Start-Sleep -Seconds 1
    
    try {
        while ($true) {
            Write-Dashboard
            Start-Sleep -Seconds $RefreshInterval
        }
    }
    catch {
        # Handle Ctrl+C gracefully
    }
    finally {
        Clear-Screen
        Write-Host ""
        Write-Host "Monitor stopped." -ForegroundColor Cyan
        Write-Host ""
    }
}

# Main entry point
if ($Help) {
    Show-Help
    exit 0
}

Start-Monitor
