#Requires -Version 7.0

<#
.SYNOPSIS
    Ralph Monitor Dashboard - Windows PowerShell Version
.DESCRIPTION
    Live monitoring dashboard for Ralph loop status.
    Displays real-time information about loop progress, rate limiting,
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
$script:ExitSignalsFile = ".exit_signals"
$script:CircuitBreakerFile = ".circuit_breaker_state"

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
    <#
    .SYNOPSIS
        Clears the screen and resets cursor position
    #>
    [Console]::Clear()
    [Console]::SetCursorPosition(0, 0)
}

function Get-TerminalWidth {
    <#
    .SYNOPSIS
        Gets the terminal width, with fallback
    #>
    try {
        return [Console]::WindowWidth
    }
    catch {
        return 80
    }
}

function Write-Header {
    <#
    .SYNOPSIS
        Writes the dashboard header
    #>
    $width = Get-TerminalWidth
    $title = " RALPH MONITOR DASHBOARD "
    $padding = [Math]::Max(0, [Math]::Floor(($width - $title.Length) / 2))
    
    Write-Host ("=" * $width) -ForegroundColor Cyan
    Write-Host (" " * $padding + $title) -ForegroundColor Cyan
    Write-Host ("=" * $width) -ForegroundColor Cyan
    Write-Host ""
}

function Get-LoopStatus {
    <#
    .SYNOPSIS
        Gets the current loop status from status.json
    #>
    
    $default = @{
        status = "Not Running"
        loop_count = 0
        calls_made_this_hour = 0
        max_calls_per_hour = 100
        last_action = "N/A"
        exit_reason = ""
        next_reset = "N/A"
        timestamp = ""
    }
    
    if (-not (Test-Path $script:StatusFile)) {
        return $default
    }
    
    try {
        $status = Get-Content $script:StatusFile -Raw | ConvertFrom-Json
        return $status
    }
    catch {
        return $default
    }
}

function Get-ProgressStatus {
    <#
    .SYNOPSIS
        Gets the current execution progress
    #>
    
    $default = @{
        status = "idle"
        indicator = "-"
        elapsed_seconds = 0
        timestamp = ""
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
    <#
    .SYNOPSIS
        Gets the circuit breaker status
    #>
    
    $default = @{
        state = "CLOSED"
        reason = "Normal operation"
        consecutive_no_progress = 0
        current_loop = 0
        total_opens = 0
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

function Get-ExitSignalsStatus {
    <#
    .SYNOPSIS
        Gets the exit signals status
    #>
    
    $default = @{
        test_only_loops = @()
        done_signals = @()
        completion_indicators = @()
    }
    
    if (-not (Test-Path $script:ExitSignalsFile)) {
        return $default
    }
    
    try {
        return Get-Content $script:ExitSignalsFile -Raw | ConvertFrom-Json
    }
    catch {
        return $default
    }
}

function Get-RateLimitInfo {
    <#
    .SYNOPSIS
        Gets rate limit information
    #>
    
    $callsMade = 0
    if (Test-Path $script:CallCountFile) {
        try {
            $callsMade = [int](Get-Content $script:CallCountFile -Raw).Trim()
        }
        catch {
            $callsMade = 0
        }
    }
    
    $now = Get-Date
    $nextHour = $now.Date.AddHours($now.Hour + 1)
    $timeUntilReset = $nextHour - $now
    
    return @{
        CallsMade = $callsMade
        TimeUntilReset = $timeUntilReset
    }
}

function Write-RateLimitBar {
    <#
    .SYNOPSIS
        Displays a visual progress bar for rate limiting
    #>
    param(
        [int]$Current,
        [int]$Max
    )
    
    $barWidth = 40
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
    <#
    .SYNOPSIS
        Gets recent log entries
    #>
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
    <#
    .SYNOPSIS
        Displays the recent logs section
    #>
    param([int]$Lines = 8)
    
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
        if ($line.Length -gt (Get-TerminalWidth) - 4) {
            $line = $line.Substring(0, (Get-TerminalWidth) - 7) + "..."
        }
        
        Write-Host "  $line" -ForegroundColor $color
    }
    Write-Host ""
}

function Write-Dashboard {
    <#
    .SYNOPSIS
        Renders the complete dashboard
    #>
    
    Clear-Screen
    Write-Header
    
    # Get all status info
    $loopStatus = Get-LoopStatus
    $progressStatus = Get-ProgressStatus
    $circuitStatus = Get-CircuitBreakerStatus
    $exitSignals = Get-ExitSignalsStatus
    $rateLimitInfo = Get-RateLimitInfo
    
    # Loop Status Section
    Write-Host "[Loop Status]" -ForegroundColor Yellow
    
    $statusColor = switch ($loopStatus.status) {
        "running" { "Green" }
        "success" { "Green" }
        "paused"  { "Yellow" }
        "halted"  { "Red" }
        "error"   { "Red" }
        "completed" { "Green" }
        default   { "Gray" }
    }
    
    Write-Host "  Status:        " -NoNewline
    Write-Host $loopStatus.status -ForegroundColor $statusColor
    Write-Host "  Loop Count:    $($loopStatus.loop_count)"
    Write-Host "  Last Action:   $($loopStatus.last_action)"
    
    if (-not [string]::IsNullOrEmpty($loopStatus.exit_reason)) {
        Write-Host "  Exit Reason:   $($loopStatus.exit_reason)" -ForegroundColor Yellow
    }
    
    if (-not [string]::IsNullOrEmpty($loopStatus.timestamp)) {
        Write-Host "  Last Update:   $($loopStatus.timestamp)" -ForegroundColor Gray
    }
    Write-Host ""
    
    # Progress Section (if executing)
    if ($progressStatus.status -eq "executing") {
        Write-Host "[Execution Progress]" -ForegroundColor Yellow
        $elapsed = $progressStatus.elapsed_seconds
        $minutes = [Math]::Floor($elapsed / 60)
        $seconds = $elapsed % 60
        Write-Host "  $($progressStatus.indicator) Running for ${minutes}m ${seconds}s..." -ForegroundColor Cyan
        Write-Host ""
    }
    
    # Rate Limit Section
    Write-Host "[Rate Limiting]" -ForegroundColor Yellow
    Write-RateLimitBar -Current $rateLimitInfo.CallsMade -Max $loopStatus.max_calls_per_hour
    
    $resetTime = $rateLimitInfo.TimeUntilReset.ToString("mm\:ss")
    Write-Host "  Reset in:      $resetTime"
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
    
    Write-Host "  State:         " -NoNewline
    Write-Host "$cbIcon $($circuitStatus.state)" -ForegroundColor $cbColor
    
    if (-not [string]::IsNullOrEmpty($circuitStatus.reason)) {
        $reason = $circuitStatus.reason
        if ($reason.Length -gt 50) {
            $reason = $reason.Substring(0, 47) + "..."
        }
        Write-Host "  Reason:        $reason" -ForegroundColor White
    }
    
    Write-Host "  No-Progress:   $($circuitStatus.consecutive_no_progress) loops"
    Write-Host ""
    
    # Exit Signals Section
    Write-Host "[Exit Signals]" -ForegroundColor Yellow
    $testLoops = @($exitSignals.test_only_loops).Count
    $doneSignals = @($exitSignals.done_signals).Count
    $completionIndicators = @($exitSignals.completion_indicators).Count
    
    Write-Host "  Test-Only Loops:    $testLoops / 3" -ForegroundColor $(if ($testLoops -ge 3) { "Red" } else { "White" })
    Write-Host "  Done Signals:       $doneSignals / 2" -ForegroundColor $(if ($doneSignals -ge 2) { "Yellow" } else { "White" })
    Write-Host "  Completion Ind.:    $completionIndicators / 2" -ForegroundColor $(if ($completionIndicators -ge 2) { "Green" } else { "White" })
    Write-Host ""
    
    # Logs Section
    Write-LogSection -Lines 8
    
    # Footer
    $width = Get-TerminalWidth
    Write-Host ("=" * $width) -ForegroundColor Cyan
    Write-Host "Press Ctrl+C to exit | Refreshing every $RefreshInterval seconds | $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
}

function Start-Monitor {
    <#
    .SYNOPSIS
        Starts the monitoring loop
    #>
    
    # Check if in Ralph project
    $isRalphProject = (Test-Path "PROMPT.md") -or (Test-Path $script:StatusFile) -or (Test-Path "@fix_plan.md")
    
    if (-not $isRalphProject) {
        Write-Host ""
        Write-Host "Warning: This doesn't appear to be a Ralph project directory." -ForegroundColor Yellow
        Write-Host "Looking for: PROMPT.md, status.json, or @fix_plan.md" -ForegroundColor Gray
        Write-Host "Current directory: $(Get-Location)" -ForegroundColor Gray
        Write-Host ""
        Write-Host "The monitor will still run, but may not show useful information." -ForegroundColor Yellow
        Write-Host ""
        Start-Sleep -Seconds 2
    }
    
    Write-Host "Starting Ralph Monitor Dashboard..." -ForegroundColor Cyan
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
