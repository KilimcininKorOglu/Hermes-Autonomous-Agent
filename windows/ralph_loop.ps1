#Requires -Version 7.0

<#
.SYNOPSIS
    Ralph Loop for Claude Code - Windows PowerShell Version
.DESCRIPTION
    Autonomous AI development loop with intelligent exit detection and rate limiting.
    Continuously executes Claude Code against your project until completion.
.PARAMETER Help
    Show help message
.PARAMETER Calls
    Maximum API calls per hour (default: 100)
.PARAMETER Prompt
    Path to prompt file (default: PROMPT.md)
.PARAMETER Status
    Show current status and exit
.PARAMETER Monitor
    Start with Windows Terminal split pane monitoring
.PARAMETER Verbose
    Enable detailed progress updates
.PARAMETER Timeout
    Claude Code execution timeout in minutes (1-120, default: 15)
.PARAMETER ResetCircuit
    Reset the circuit breaker to CLOSED state
.PARAMETER CircuitStatus
    Show circuit breaker status and exit
.EXAMPLE
    .\ralph_loop.ps1 -Monitor
.EXAMPLE
    .\ralph_loop.ps1 -Calls 50 -Timeout 30
#>

[CmdletBinding()]
param(
    [Alias('h')]
    [switch]$Help,
    
    [Alias('c')]
    [int]$Calls = 100,
    
    [Alias('p')]
    [string]$Prompt = "PROMPT.md",
    
    [Alias('s')]
    [switch]$Status,
    
    [Alias('m')]
    [switch]$Monitor,
    
    [Alias('v')]
    [switch]$VerboseProgress,
    
    [Alias('t')]
    [ValidateRange(1, 120)]
    [int]$Timeout = 15,
    
    [switch]$ResetCircuit,
    
    [switch]$CircuitStatus,
    
    # Task Mode Parameters
    [switch]$TaskMode,
    
    [string]$TasksDir = "tasks",
    
    [switch]$AutoBranch,
    
    [switch]$AutoCommit,
    
    [string]$StartFrom = "",
    
    [switch]$TaskStatus
)

# Get script directory for module imports
$script:ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Import library modules
. "$script:ScriptDir\lib\CircuitBreaker.ps1"
. "$script:ScriptDir\lib\ResponseAnalyzer.ps1"
. "$script:ScriptDir\lib\TaskReader.ps1"
. "$script:ScriptDir\lib\TaskStatusUpdater.ps1"
. "$script:ScriptDir\lib\GitBranchManager.ps1"
. "$script:ScriptDir\lib\PromptInjector.ps1"

# Configuration
$script:Config = @{
    PromptFile = $Prompt
    LogDir = "logs"
    DocsDir = "docs\generated"
    StatusFile = "status.json"
    ProgressFile = "progress.json"
    ClaudeCommand = "claude"
    MaxCallsPerHour = $Calls
    VerboseProgress = $VerboseProgress
    ClaudeTimeoutMinutes = $Timeout
    CallCountFile = ".call_count"
    TimestampFile = ".last_reset"
    ExitSignalsFile = ".exit_signals"
    MaxConsecutiveTestLoops = 3
    MaxConsecutiveDoneSignals = 2
    # Task Mode Config
    TaskMode = $TaskMode
    TasksDir = $TasksDir
    AutoBranch = $AutoBranch
    AutoCommit = $AutoCommit
    StartFromTask = $StartFrom
    # Task State
    CurrentTask = $null
    CurrentFeature = $null
    CurrentBranch = ""
}

# Global loop counter for cleanup
$script:LoopCount = 0

function Show-Help {
    <#
    .SYNOPSIS
        Displays help information
    #>
    
    Write-Host ""
    Write-Host "Ralph Loop for Claude Code - Windows PowerShell Version" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "IMPORTANT: This command must be run from a Ralph project directory." -ForegroundColor Yellow
    Write-Host "           Use 'ralph-setup project-name' to create a new project first."
    Write-Host ""
    Write-Host "Usage: ralph [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Yellow
    Write-Host "    -h, -Help              Show this help message"
    Write-Host "    -c, -Calls NUM         Set max calls per hour (default: 100)"
    Write-Host "    -p, -Prompt FILE       Set prompt file (default: PROMPT.md)"
    Write-Host "    -s, -Status            Show current status and exit"
    Write-Host "    -m, -Monitor           Start with monitoring (new terminal window)"
    Write-Host "    -v, -VerboseProgress   Show detailed progress updates"
    Write-Host "    -t, -Timeout MIN       Set timeout in minutes (1-120, default: 15)"
    Write-Host "    -ResetCircuit          Reset circuit breaker to CLOSED state"
    Write-Host "    -CircuitStatus         Show circuit breaker status"
    Write-Host ""
    Write-Host "Task Mode Options:" -ForegroundColor Yellow
    Write-Host "    -TaskMode              Enable task-plan integration mode"
    Write-Host "    -TasksDir DIR          Tasks directory (default: tasks)"
    Write-Host "    -AutoBranch            Auto-create/switch feature branches"
    Write-Host "    -AutoCommit            Auto-commit on task completion"
    Write-Host "    -StartFrom TXXX        Start from specific task ID"
    Write-Host "    -TaskStatus            Show task progress and exit"
    Write-Host ""
    Write-Host "Files created:" -ForegroundColor Yellow
    Write-Host "    - logs\              All execution logs"
    Write-Host "    - docs\generated\    Generated documentation"
    Write-Host "    - status.json        Current status (JSON)"
    Write-Host ""
    Write-Host "Example workflow:" -ForegroundColor Yellow
    Write-Host "    ralph-setup my-project     # Create project"
    Write-Host "    cd my-project              # Enter project directory"
    Write-Host "    ralph -Monitor             # Start Ralph with monitoring"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "    ralph -Calls 50 -Prompt my_prompt.md"
    Write-Host "    ralph -Monitor -Timeout 30"
    Write-Host "    ralph -VerboseProgress"
    Write-Host ""
    Write-Host "Task Mode Examples:" -ForegroundColor Yellow
    Write-Host "    ralph -TaskMode -AutoBranch -AutoCommit"
    Write-Host "    ralph -TaskMode -StartFrom T005"
    Write-Host "    ralph -TaskStatus"
    Write-Host ""
}

function Write-Status {
    <#
    .SYNOPSIS
        Logs a message with timestamp and level
    #>
    param(
        [ValidateSet("INFO", "WARN", "ERROR", "SUCCESS", "LOOP")]
        [string]$Level,
        [string]$Message
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "INFO"    { "Cyan" }
        "WARN"    { "Yellow" }
        "ERROR"   { "Red" }
        "SUCCESS" { "Green" }
        "LOOP"    { "Magenta" }
    }
    
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
    
    # Ensure log directory exists
    if (-not (Test-Path $script:Config.LogDir)) {
        New-Item -ItemType Directory -Path $script:Config.LogDir -Force | Out-Null
    }
    
    # Append to log file
    "[$timestamp] [$Level] $Message" | Add-Content -Path "$($script:Config.LogDir)\ralph.log" -Encoding UTF8
}

function Initialize-CallTracking {
    <#
    .SYNOPSIS
        Initializes or resets the call tracking for rate limiting
    #>
    
    Write-Status -Level "INFO" -Message "Initializing call tracking..."
    
    $currentHour = Get-Date -Format "yyyyMMddHH"
    $lastResetHour = ""
    
    if (Test-Path $script:Config.TimestampFile) {
        $lastResetHour = (Get-Content $script:Config.TimestampFile -Raw).Trim()
    }
    
    # Reset counter if it's a new hour
    if ($currentHour -ne $lastResetHour) {
        "0" | Set-Content $script:Config.CallCountFile -Encoding UTF8
        $currentHour | Set-Content $script:Config.TimestampFile -Encoding UTF8
        Write-Status -Level "INFO" -Message "Call counter reset for new hour: $currentHour"
    }
    
    # Initialize exit signals tracking
    if (-not (Test-Path $script:Config.ExitSignalsFile)) {
        @{
            test_only_loops = @()
            done_signals = @()
            completion_indicators = @()
        } | ConvertTo-Json | Set-Content $script:Config.ExitSignalsFile -Encoding UTF8
    }
    
    # Initialize circuit breaker
    Initialize-CircuitBreaker
}

function Update-LoopStatus {
    <#
    .SYNOPSIS
        Updates the status JSON file for external monitoring
    #>
    param(
        [int]$LoopCount,
        [int]$CallsMade,
        [string]$LastAction,
        [string]$Status,
        [string]$ExitReason = ""
    )
    
    $nextReset = (Get-Date).AddHours(1)
    $nextResetStr = $nextReset.ToString("HH:mm:ss")
    
    $statusData = @{
        timestamp = (Get-Date -Format "o")
        loop_count = $LoopCount
        calls_made_this_hour = $CallsMade
        max_calls_per_hour = $script:Config.MaxCallsPerHour
        last_action = $LastAction
        status = $Status
        exit_reason = $ExitReason
        next_reset = $nextResetStr
    }
    
    $statusData | ConvertTo-Json -Depth 10 | Set-Content $script:Config.StatusFile -Encoding UTF8
}

function Test-CanMakeCall {
    <#
    .SYNOPSIS
        Checks if we can make another API call within rate limits
    #>
    
    $callsMade = 0
    if (Test-Path $script:Config.CallCountFile) {
        try {
            $callsMade = [int](Get-Content $script:Config.CallCountFile -Raw).Trim()
        }
        catch {
            $callsMade = 0
        }
    }
    
    return $callsMade -lt $script:Config.MaxCallsPerHour
}

function Add-CallCount {
    <#
    .SYNOPSIS
        Increments and returns the call counter
    #>
    
    $callsMade = 0
    if (Test-Path $script:Config.CallCountFile) {
        try {
            $callsMade = [int](Get-Content $script:Config.CallCountFile -Raw).Trim()
        }
        catch {
            $callsMade = 0
        }
    }
    
    $callsMade++
    $callsMade.ToString() | Set-Content $script:Config.CallCountFile -Encoding UTF8
    return $callsMade
}

function Get-CallCount {
    <#
    .SYNOPSIS
        Gets the current call count
    #>
    
    if (Test-Path $script:Config.CallCountFile) {
        try {
            return [int](Get-Content $script:Config.CallCountFile -Raw).Trim()
        }
        catch {
            return 0
        }
    }
    return 0
}

function Wait-ForReset {
    <#
    .SYNOPSIS
        Waits for the rate limit to reset with countdown display
    #>
    
    $callsMade = Get-CallCount
    Write-Status -Level "WARN" -Message "Rate limit reached ($callsMade/$($script:Config.MaxCallsPerHour)). Waiting for reset..."
    
    # Calculate time until next hour
    $now = Get-Date
    $nextHour = $now.Date.AddHours($now.Hour + 1)
    $waitTime = ($nextHour - $now).TotalSeconds
    
    Write-Status -Level "INFO" -Message "Sleeping for $([int]$waitTime) seconds until next hour..."
    
    # Countdown display
    while ($waitTime -gt 0) {
        $ts = [TimeSpan]::FromSeconds([int]$waitTime)
        $countdown = $ts.ToString("hh\:mm\:ss")
        Write-Host "`rTime until reset: $countdown" -ForegroundColor Yellow -NoNewline
        Start-Sleep -Seconds 1
        $waitTime--
    }
    Write-Host ""
    
    # Reset counter
    "0" | Set-Content $script:Config.CallCountFile -Encoding UTF8
    (Get-Date -Format "yyyyMMddHH") | Set-Content $script:Config.TimestampFile -Encoding UTF8
    Write-Status -Level "SUCCESS" -Message "Rate limit reset! Ready for new calls."
}

function Get-ExitReason {
    <#
    .SYNOPSIS
        Checks for graceful exit conditions
    .OUTPUTS
        String - Exit reason or empty string if should continue
    #>
    
    Write-Status -Level "INFO" -Message "Checking exit conditions..."
    
    if (-not (Test-Path $script:Config.ExitSignalsFile)) {
        return ""
    }
    
    try {
        $signals = Get-Content $script:Config.ExitSignalsFile -Raw | ConvertFrom-Json
    }
    catch {
        return ""
    }
    
    $recentTestLoops = @($signals.test_only_loops).Count
    $recentDoneSignals = @($signals.done_signals).Count
    $recentCompletionIndicators = @($signals.completion_indicators).Count
    
    Write-Status -Level "INFO" -Message "Exit counts - test_loops:$recentTestLoops, done_signals:$recentDoneSignals, completion:$recentCompletionIndicators"
    
    # Check exit conditions
    
    # 1. Too many consecutive test-only loops
    if ($recentTestLoops -ge $script:Config.MaxConsecutiveTestLoops) {
        Write-Status -Level "WARN" -Message "Exit condition: Too many test-focused loops ($recentTestLoops >= $($script:Config.MaxConsecutiveTestLoops))"
        return "test_saturation"
    }
    
    # 2. Multiple "done" signals
    if ($recentDoneSignals -ge $script:Config.MaxConsecutiveDoneSignals) {
        Write-Status -Level "WARN" -Message "Exit condition: Multiple completion signals ($recentDoneSignals >= $($script:Config.MaxConsecutiveDoneSignals))"
        return "completion_signals"
    }
    
    # 3. Strong completion indicators
    if ($recentCompletionIndicators -ge 2) {
        Write-Status -Level "WARN" -Message "Exit condition: Strong completion indicators ($recentCompletionIndicators)"
        return "project_complete"
    }
    
    # 4. Check @fix_plan.md for completion
    if (Test-Path "@fix_plan.md") {
        $content = Get-Content "@fix_plan.md" -Raw
        
        # Count total checkbox items and completed items
        $totalMatches = [regex]::Matches($content, "(?m)^- \[")
        $completedMatches = [regex]::Matches($content, "(?mi)^- \[x\]")
        
        $totalItems = $totalMatches.Count
        $completedItems = $completedMatches.Count
        
        Write-Status -Level "INFO" -Message "@fix_plan.md check - total:$totalItems, completed:$completedItems"
        
        if ($totalItems -gt 0 -and $completedItems -eq $totalItems) {
            Write-Status -Level "WARN" -Message "Exit condition: All fix_plan.md items completed ($completedItems/$totalItems)"
            return "plan_complete"
        }
    }
    
    return ""
}

function Invoke-ClaudeCode {
    <#
    .SYNOPSIS
        Executes Claude Code with the prompt and handles the response
    .PARAMETER LoopCount
        Current loop iteration number
    .OUTPUTS
        Int - Exit code (0=success, 1=error, 2=API limit, 3=circuit breaker)
    #>
    param(
        [int]$LoopCount
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
    $outputFile = Join-Path $script:Config.LogDir "claude_output_$timestamp.log"
    $callsMade = Add-CallCount
    
    Write-Status -Level "LOOP" -Message "Executing Claude Code (Call $callsMade/$($script:Config.MaxCallsPerHour))"
    
    $timeoutSeconds = $script:Config.ClaudeTimeoutMinutes * 60
    Write-Status -Level "INFO" -Message "Starting Claude Code execution... (timeout: $($script:Config.ClaudeTimeoutMinutes)m)"
    
    # Check if prompt file exists
    if (-not (Test-Path $script:Config.PromptFile)) {
        Write-Status -Level "ERROR" -Message "Prompt file not found: $($script:Config.PromptFile)"
        return 1
    }
    
    try {
        # Read prompt content
        $promptContent = Get-Content $script:Config.PromptFile -Raw
        
        # Create a temporary file for the prompt
        $tempPromptFile = [System.IO.Path]::GetTempFileName()
        $promptContent | Set-Content $tempPromptFile -Encoding UTF8
        
        # Start Claude Code as a background job
        $job = Start-Job -ScriptBlock {
            param($promptFile, $claudeCmd)
            
            $content = Get-Content $promptFile -Raw
            $result = $content | & $claudeCmd 2>&1
            return $result
        } -ArgumentList $tempPromptFile, $script:Config.ClaudeCommand
        
        $progressCounter = 0
        $indicators = @("|", "/", "-", "\")
        
        # Monitor progress
        while ($job.State -eq 'Running') {
            $progressCounter++
            $indicator = $indicators[$progressCounter % 4]
            $elapsedSeconds = $progressCounter * 5
            
            # Update progress file for monitor
            @{
                status = "executing"
                indicator = $indicator
                elapsed_seconds = $elapsedSeconds
                timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
            } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
            
            if ($script:Config.VerboseProgress) {
                Write-Host "`r[$indicator] Claude Code working... ($elapsedSeconds`s elapsed)" -ForegroundColor Cyan -NoNewline
            }
            
            Start-Sleep -Seconds 5
            
            # Check timeout
            if ($elapsedSeconds -ge $timeoutSeconds) {
                Stop-Job $job -ErrorAction SilentlyContinue
                Remove-Job $job -Force -ErrorAction SilentlyContinue
                Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
                
                Write-Host ""
                Write-Status -Level "ERROR" -Message "Execution timed out after $($script:Config.ClaudeTimeoutMinutes) minutes"
                return 1
            }
        }
        
        if ($script:Config.VerboseProgress) {
            Write-Host ""
        }
        
        # Get results
        $output = Receive-Job $job
        $jobState = $job.State
        Remove-Job $job -Force -ErrorAction SilentlyContinue
        Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
        
        # Write output to file
        if ($output) {
            $output | Out-File $outputFile -Encoding UTF8
        }
        else {
            "No output received from Claude Code" | Out-File $outputFile -Encoding UTF8
        }
        
        if ($jobState -eq 'Completed') {
            # Update progress file
            @{
                status = "completed"
                timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
            } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
            
            Write-Status -Level "SUCCESS" -Message "Claude Code execution completed successfully"
            
            # Analyze the response
            Write-Status -Level "INFO" -Message "Analyzing Claude Code response..."
            $analysisResult = Invoke-ResponseAnalysis -OutputFile $outputFile -LoopNumber $LoopCount
            
            if ($analysisResult) {
                Update-ExitSignals
                Write-AnalysisSummary
            }
            
            # Get file change count for circuit breaker
            $filesChanged = 0
            try {
                $gitDiff = git diff --name-only 2>$null
                if ($gitDiff) {
                    $filesChanged = ($gitDiff | Measure-Object).Count
                }
            }
            catch {
                # Ignore git errors
            }
            
            # Check for errors in output
            $hasErrors = $false
            $outputContent = Get-Content $outputFile -Raw -ErrorAction SilentlyContinue
            if ($outputContent -match "(?i)(error|exception|failed)") {
                $hasErrors = $true
                Write-Status -Level "WARN" -Message "Errors detected in output, check: $outputFile"
            }
            
            $outputLength = if (Test-Path $outputFile) { (Get-Item $outputFile).Length } else { 0 }
            
            # Record result in circuit breaker
            $circuitResult = Add-LoopResult -LoopNumber $LoopCount -FilesChanged $filesChanged -HasErrors $hasErrors -OutputLength $outputLength
            
            if (-not $circuitResult) {
                Write-Status -Level "WARN" -Message "Circuit breaker opened - halting execution"
                return 3
            }
            
            return 0
        }
        else {
            # Job failed
            @{
                status = "failed"
                timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
            } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
            
            # Check if failure is due to API limit
            $outputContent = if ($output) { $output -join "`n" } else { "" }
            if ($outputContent -match "(?i)(5.*hour.*limit|limit.*reached.*try.*back|usage.*limit.*reached)") {
                Write-Status -Level "ERROR" -Message "Claude API 5-hour usage limit reached"
                return 2
            }
            
            Write-Status -Level "ERROR" -Message "Claude Code execution failed, check: $outputFile"
            return 1
        }
    }
    catch {
        Write-Status -Level "ERROR" -Message "Exception during Claude Code execution: $($_.Exception.Message)"
        
        @{
            status = "failed"
            error = $_.Exception.Message
            timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
        } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
        
        return 1
    }
}

function Start-RalphLoop {
    <#
    .SYNOPSIS
        Main Ralph loop execution
    #>
    
    Write-Status -Level "SUCCESS" -Message "Ralph loop starting with Claude Code"
    Write-Status -Level "INFO" -Message "Max calls per hour: $($script:Config.MaxCallsPerHour)"
    Write-Status -Level "INFO" -Message "Logs: $($script:Config.LogDir)\ | Status: $($script:Config.StatusFile)"
    
    # Initialize directories
    if (-not (Test-Path $script:Config.LogDir)) {
        New-Item -ItemType Directory -Path $script:Config.LogDir -Force | Out-Null
    }
    if (-not (Test-Path $script:Config.DocsDir)) {
        New-Item -ItemType Directory -Path $script:Config.DocsDir -Force | Out-Null
    }
    
    # Check if this is a Ralph project
    if (-not (Test-Path $script:Config.PromptFile)) {
        Write-Status -Level "ERROR" -Message "Prompt file '$($script:Config.PromptFile)' not found!"
        Write-Host ""
        
        if ((Test-Path "@fix_plan.md") -or (Test-Path "specs") -or (Test-Path "@AGENT.md")) {
            Write-Host "This appears to be a Ralph project but is missing PROMPT.md." -ForegroundColor Yellow
            Write-Host "You may need to create or restore the PROMPT.md file."
        }
        else {
            Write-Host "This directory is not a Ralph project." -ForegroundColor Yellow
        }
        
        Write-Host ""
        Write-Host "To fix this:" -ForegroundColor Cyan
        Write-Host "  1. Create a new project: ralph-setup my-project"
        Write-Host "  2. Import existing requirements: ralph-import requirements.md"
        Write-Host "  3. Navigate to an existing Ralph project directory"
        Write-Host "  4. Or create PROMPT.md manually in this directory"
        Write-Host ""
        return
    }
    
    $script:LoopCount = 0
    
    while ($true) {
        $script:LoopCount++
        
        Write-Status -Level "INFO" -Message "Loop #$($script:LoopCount) - initializing..."
        Initialize-CallTracking
        
        Write-Status -Level "LOOP" -Message "=== Starting Loop #$($script:LoopCount) ==="
        
        # Check circuit breaker
        if (Test-ShouldHalt) {
            Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                -LastAction "circuit_breaker_open" -Status "halted" -ExitReason "stagnation_detected"
            Write-Status -Level "ERROR" -Message "Circuit breaker has opened - execution halted"
            break
        }
        
        # Check rate limits
        if (-not (Test-CanMakeCall)) {
            Wait-ForReset
            continue
        }
        
        # Check for graceful exit conditions
        $exitReason = Get-ExitReason
        if (-not [string]::IsNullOrEmpty($exitReason)) {
            Write-Status -Level "SUCCESS" -Message "Graceful exit triggered: $exitReason"
            Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                -LastAction "graceful_exit" -Status "completed" -ExitReason $exitReason
            
            Write-Host ""
            Write-Status -Level "SUCCESS" -Message "Ralph has completed the project! Final stats:"
            Write-Status -Level "INFO" -Message "  - Total loops: $($script:LoopCount)"
            Write-Status -Level "INFO" -Message "  - API calls used: $(Get-CallCount)"
            Write-Status -Level "INFO" -Message "  - Exit reason: $exitReason"
            Write-Host ""
            break
        }
        
        # Update status
        Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
            -LastAction "executing" -Status "running"
        
        # Execute Claude Code
        $execResult = Invoke-ClaudeCode -LoopCount $script:LoopCount
        
        switch ($execResult) {
            0 {
                # Success
                Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                    -LastAction "completed" -Status "success"
                Start-Sleep -Seconds 5
            }
            3 {
                # Circuit breaker opened
                Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                    -LastAction "circuit_breaker_open" -Status "halted" -ExitReason "stagnation_detected"
                Write-Status -Level "ERROR" -Message "Circuit breaker has opened - halting loop"
                Write-Status -Level "INFO" -Message "Run 'ralph -ResetCircuit' to reset the circuit breaker after addressing issues"
                break
            }
            2 {
                # API 5-hour limit reached
                Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                    -LastAction "api_limit" -Status "paused"
                Write-Status -Level "WARN" -Message "Claude API 5-hour limit reached!"
                
                Write-Host ""
                Write-Host "The Claude API 5-hour usage limit has been reached." -ForegroundColor Yellow
                Write-Host "You can either:" -ForegroundColor Yellow
                Write-Host "  1) Wait for the limit to reset (usually within an hour)" -ForegroundColor Green
                Write-Host "  2) Exit the loop and try again later" -ForegroundColor Green
                Write-Host ""
                Write-Host "Choose an option (1 or 2): " -ForegroundColor Cyan -NoNewline
                
                $choice = Read-Host
                if ($choice -eq "2" -or [string]::IsNullOrEmpty($choice)) {
                    Write-Status -Level "INFO" -Message "User chose to exit. Exiting loop..."
                    Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                        -LastAction "api_limit_exit" -Status "stopped" -ExitReason "api_5hour_limit"
                    break
                }
                else {
                    Write-Status -Level "INFO" -Message "User chose to wait. Waiting 60 minutes..."
                    $waitSeconds = 3600
                    while ($waitSeconds -gt 0) {
                        $ts = [TimeSpan]::FromSeconds($waitSeconds)
                        Write-Host "`rTime until retry: $($ts.ToString('mm\:ss'))" -ForegroundColor Yellow -NoNewline
                        Start-Sleep -Seconds 1
                        $waitSeconds--
                    }
                    Write-Host ""
                }
            }
            default {
                # Error
                Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
                    -LastAction "failed" -Status "error"
                Write-Status -Level "WARN" -Message "Execution failed, waiting 30 seconds before retry..."
                Start-Sleep -Seconds 30
            }
        }
        
        Write-Status -Level "LOOP" -Message "=== Completed Loop #$($script:LoopCount) ==="
    }
}

function Show-CurrentStatus {
    <#
    .SYNOPSIS
        Shows current Ralph status
    #>
    
    if (Test-Path $script:Config.StatusFile) {
        Write-Host ""
        Write-Host "Current Ralph Status:" -ForegroundColor Cyan
        Write-Host ""
        $status = Get-Content $script:Config.StatusFile -Raw | ConvertFrom-Json
        $status | Format-List
    }
    else {
        Write-Host ""
        Write-Host "No status file found. Ralph may not be running." -ForegroundColor Yellow
        Write-Host "Status file expected at: $($script:Config.StatusFile)" -ForegroundColor Gray
        Write-Host ""
    }
}

function Start-WithMonitor {
    <#
    .SYNOPSIS
        Starts Ralph with a separate monitor window
    #>
    
    Write-Status -Level "INFO" -Message "Starting with monitoring..."
    
    $monitorScript = Join-Path $script:ScriptDir "ralph_monitor.ps1"
    
    if (Test-Path $monitorScript) {
        # Start monitor in a new PowerShell window
        Start-Process pwsh -ArgumentList "-NoExit", "-File", $monitorScript -WindowStyle Normal
        Write-Status -Level "SUCCESS" -Message "Monitor window started"
    }
    else {
        Write-Status -Level "WARN" -Message "Monitor script not found: $monitorScript"
        Write-Status -Level "INFO" -Message "Continuing without monitor..."
    }
    
    # Start main loop
    Start-RalphLoop
}

function Show-TaskStatus {
    <#
    .SYNOPSIS
        Shows current task progress
    #>
    
    if (-not (Test-TasksDirectoryExists -BasePath ".")) {
        Write-Host ""
        Write-Host "No tasks directory found." -ForegroundColor Yellow
        Write-Host "Expected: $($script:Config.TasksDir)/" -ForegroundColor Gray
        Write-Host ""
        return
    }
    
    $progress = Get-TaskProgress -BasePath "."
    $features = Get-AllFeatures -BasePath "."
    
    Write-Host ""
    Write-Host ("=" * 60) -ForegroundColor Cyan
    Write-Host "              TASK PROGRESS SUMMARY" -ForegroundColor Cyan
    Write-Host ("=" * 60) -ForegroundColor Cyan
    Write-Host ""
    
    # Progress bar
    $barWidth = 30
    $filled = [Math]::Floor(($progress.Percentage / 100) * $barWidth)
    $empty = $barWidth - $filled
    $bar = "[" + ([char]0x2588).ToString() * $filled + ([char]0x2591).ToString() * $empty + "]"
    
    Write-Host "Overall: $bar $($progress.Percentage)%" -ForegroundColor White
    Write-Host ""
    Write-Host "Total Tasks:   $($progress.Total)" -ForegroundColor White
    Write-Host "Completed:     $($progress.Completed)" -ForegroundColor Green
    Write-Host "In Progress:   $($progress.InProgress)" -ForegroundColor Yellow
    Write-Host "Not Started:   $($progress.NotStarted)" -ForegroundColor Gray
    Write-Host "Blocked:       $($progress.Blocked)" -ForegroundColor Red
    Write-Host ""
    
    # Feature breakdown
    Write-Host "By Feature:" -ForegroundColor Cyan
    Write-Host ""
    
    foreach ($feature in $features) {
        $fp = Get-FeatureProgress -FeatureId $feature.FeatureId -BasePath "."
        $statusColor = switch ($feature.Status) {
            "COMPLETED" { "Green" }
            "IN_PROGRESS" { "Yellow" }
            "BLOCKED" { "Red" }
            default { "Gray" }
        }
        
        $featureLine = "  $($feature.FeatureId): $($feature.FeatureName)"
        if ($featureLine.Length -gt 45) {
            $featureLine = $featureLine.Substring(0, 42) + "..."
        }
        $featureLine = $featureLine.PadRight(48)
        
        Write-Host "$featureLine $($fp.Completed)/$($fp.Total) ($($fp.Percentage)%)" -ForegroundColor $statusColor
    }
    
    Write-Host ""
    
    # Next task
    $nextTask = Get-NextTask -BasePath "."
    if ($nextTask) {
        Write-Host "Next Task: $($nextTask.TaskId) - $($nextTask.Name)" -ForegroundColor Cyan
        Write-Host "Feature:   $($nextTask.FeatureId)" -ForegroundColor Gray
        Write-Host "Priority:  $($nextTask.Priority)" -ForegroundColor Gray
    }
    else {
        Write-Host "All tasks completed!" -ForegroundColor Green
    }
    
    Write-Host ""
    Write-Host ("=" * 60) -ForegroundColor Cyan
    Write-Host ""
}

function Start-TaskModeLoop {
    <#
    .SYNOPSIS
        Main loop for Task Mode execution
    #>
    
    Write-Status -Level "SUCCESS" -Message "Ralph Task Mode starting..."
    Write-Status -Level "INFO" -Message "Tasks directory: $($script:Config.TasksDir)"
    Write-Status -Level "INFO" -Message "Auto-branch: $($script:Config.AutoBranch)"
    Write-Status -Level "INFO" -Message "Auto-commit: $($script:Config.AutoCommit)"
    
    # Check if tasks directory exists
    if (-not (Test-TasksDirectoryExists -BasePath ".")) {
        Write-Status -Level "ERROR" -Message "Tasks directory not found: $($script:Config.TasksDir)"
        Write-Host ""
        Write-Host "Create tasks using task-plan or manually create $($script:Config.TasksDir)/ directory." -ForegroundColor Yellow
        Write-Host ""
        return
    }
    
    # Initialize directories
    if (-not (Test-Path $script:Config.LogDir)) {
        New-Item -ItemType Directory -Path $script:Config.LogDir -Force | Out-Null
    }
    if (-not (Test-Path $script:Config.DocsDir)) {
        New-Item -ItemType Directory -Path $script:Config.DocsDir -Force | Out-Null
    }
    
    # Check if PROMPT.md exists
    if (-not (Test-Path $script:Config.PromptFile)) {
        Write-Status -Level "ERROR" -Message "Prompt file not found: $($script:Config.PromptFile)"
        return
    }
    
    # Backup original PROMPT.md
    $promptBackup = Backup-Prompt -BasePath "."
    if ($promptBackup) {
        Write-Status -Level "INFO" -Message "PROMPT.md backed up to: $promptBackup"
    }
    
    $script:LoopCount = 0
    $taskLoopCount = 0
    $maxLoopsPerTask = 10
    
    # If StartFrom specified, find that task
    if ($script:Config.StartFromTask) {
        $startTask = Get-TaskById -TaskId $script:Config.StartFromTask -BasePath "."
        if ($startTask) {
            Write-Status -Level "INFO" -Message "Starting from task: $($script:Config.StartFromTask)"
        }
        else {
            Write-Status -Level "ERROR" -Message "Task not found: $($script:Config.StartFromTask)"
            return
        }
    }
    
    while ($true) {
        $script:LoopCount++
        
        # Get next task
        $task = if ($script:Config.StartFromTask -and $script:LoopCount -eq 1) {
            Get-TaskById -TaskId $script:Config.StartFromTask -BasePath "."
        }
        else {
            Get-NextTask -BasePath "."
        }
        
        if (-not $task) {
            Write-Status -Level "SUCCESS" -Message "All tasks completed!"
            Remove-TaskFromPrompt -BasePath "."
            
            $progress = Get-TaskProgress -BasePath "."
            Write-Host ""
            Write-Host ("=" * 50) -ForegroundColor Green
            Write-Host "  PROJECT COMPLETE!" -ForegroundColor Green
            Write-Host ("=" * 50) -ForegroundColor Green
            Write-Host "  Total Tasks: $($progress.Total)" -ForegroundColor White
            Write-Host "  Completed:   $($progress.Completed)" -ForegroundColor White
            Write-Host "  Total Loops: $($script:LoopCount)" -ForegroundColor White
            Write-Host ("=" * 50) -ForegroundColor Green
            Write-Host ""
            break
        }
        
        $currentTaskId = $task.TaskId
        $script:Config.CurrentTask = $task
        
        # Check if this is a new task or continuing
        if ($task.Status -eq "NOT_STARTED") {
            $taskLoopCount = 0
            
            Write-Status -Level "INFO" -Message "Starting new task: $currentTaskId - $($task.Name)"
            
            # Get feature info
            $feature = Get-FeatureById -FeatureId $task.FeatureId -BasePath "."
            $script:Config.CurrentFeature = $feature
            
            # Auto-branch management
            if ($script:Config.AutoBranch -and $feature) {
                $branchName = Get-FeatureBranchName -FeatureId $feature.FeatureId -FeatureName $feature.FeatureName
                
                if (-not (Test-BranchExists -Name $branchName)) {
                    New-FeatureBranch -FeatureId $feature.FeatureId -FeatureName $feature.FeatureName | Out-Null
                }
                else {
                    Switch-ToFeatureBranch -BranchName $branchName | Out-Null
                }
                
                $script:Config.CurrentBranch = $branchName
            }
            
            # Update task status to IN_PROGRESS
            Set-TaskStatus -TaskId $currentTaskId -Status "IN_PROGRESS" -BasePath "."
            
            # Inject task into PROMPT.md
            $featureName = if ($feature) { $feature.FeatureName } else { "" }
            Add-TaskToPrompt -Task $task -FeatureName $featureName -BranchName $script:Config.CurrentBranch -BasePath "."
            
            # Update run state
            $nextTask = Get-NextTask -BasePath "."
            $nextTaskId = if ($nextTask -and $nextTask.TaskId -ne $currentTaskId) { $nextTask.TaskId } else { "" }
            Update-RunState -CurrentTaskId $currentTaskId -CurrentFeatureId $task.FeatureId `
                -CurrentBranch $script:Config.CurrentBranch -NextTaskId $nextTaskId -BasePath "."
        }
        else {
            $taskLoopCount++
            Write-Status -Level "INFO" -Message "Continuing task: $currentTaskId (loop $taskLoopCount)"
        }
        
        # Check task timeout
        if ($taskLoopCount -ge $maxLoopsPerTask) {
            Write-Status -Level "WARN" -Message "Task $currentTaskId exceeded max loops ($maxLoopsPerTask)"
            Set-TaskStatus -TaskId $currentTaskId -Status "BLOCKED" -BasePath "."
            continue
        }
        
        # Initialize call tracking
        Initialize-CallTracking
        
        Write-Status -Level "LOOP" -Message "=== Task $currentTaskId - Loop #$($script:LoopCount) ==="
        
        # Check circuit breaker
        if (Test-ShouldHalt) {
            Write-Status -Level "ERROR" -Message "Circuit breaker opened - halting"
            break
        }
        
        # Check rate limits
        if (-not (Test-CanMakeCall)) {
            Wait-ForReset
            continue
        }
        
        # Execute Claude Code
        $execResult = Invoke-ClaudeCode -LoopCount $script:LoopCount
        
        if ($execResult -eq 0) {
            # Check if task completed
            $analysis = Get-AnalysisResult
            
            if ($analysis -and $analysis.analysis.task_completed) {
                Write-Status -Level "SUCCESS" -Message "Task $currentTaskId completed!"
                
                # Mark success criteria
                Complete-AllSuccessCriteria -TaskId $currentTaskId -BasePath "."
                
                # Auto-commit
                if ($script:Config.AutoCommit) {
                    Add-AllChanges | Out-Null
                    $commitResult = New-TaskCommit -TaskId $currentTaskId -TaskName $task.Name `
                        -SuccessCriteria $task.SuccessCriteria
                    
                    if ($commitResult) {
                        $commitHash = Get-LastCommitHash
                        Add-TaskCompletionLog -TaskId $currentTaskId -FeatureId $task.FeatureId `
                            -CommitHash $commitHash -BasePath "."
                    }
                }
                
                # Update task status
                Set-TaskStatus -TaskId $currentTaskId -Status "COMPLETED" -BasePath "."
                
                # Check if feature is complete
                if (Test-FeatureComplete -FeatureId $task.FeatureId -BasePath ".") {
                    Write-Status -Level "SUCCESS" -Message "Feature $($task.FeatureId) completed!"
                    Set-FeatureStatus -FeatureId $task.FeatureId -Status "COMPLETED" -BasePath "."
                    
                    # Merge to main if auto-branch
                    if ($script:Config.AutoBranch) {
                        $feature = Get-FeatureById -FeatureId $task.FeatureId -BasePath "."
                        $taskCount = (Get-TasksByFeature -FeatureId $task.FeatureId -BasePath ".").Count
                        
                        # Commit any remaining changes
                        if ($script:Config.AutoCommit) {
                            Add-AllChanges | Out-Null
                            New-FeatureCommit -FeatureId $task.FeatureId -FeatureName $feature.FeatureName `
                                -TaskCount $taskCount | Out-Null
                        }
                        
                        # Merge to main
                        $mergeResult = Merge-FeatureToMain -BranchName $script:Config.CurrentBranch `
                            -FeatureId $task.FeatureId -FeatureName $feature.FeatureName -DeleteBranch
                        
                        if ($mergeResult) {
                            Write-Status -Level "SUCCESS" -Message "Feature merged to main"
                        }
                        else {
                            Write-Status -Level "WARN" -Message "Merge failed - manual intervention required"
                        }
                    }
                }
                
                # Remove task from PROMPT.md for next task
                Remove-TaskFromPrompt -BasePath "."
                
                # Reset StartFrom so we get next task naturally
                $script:Config.StartFromTask = ""
            }
            elseif ($analysis -and $analysis.analysis.task_blocked) {
                Write-Status -Level "WARN" -Message "Task $currentTaskId is blocked: $($analysis.analysis.blocked_reason)"
                Set-TaskStatus -TaskId $currentTaskId -Status "BLOCKED" -BasePath "."
            }
            
            Start-Sleep -Seconds 5
        }
        elseif ($execResult -eq 3) {
            # Circuit breaker
            Write-Status -Level "ERROR" -Message "Circuit breaker opened"
            break
        }
        elseif ($execResult -eq 2) {
            # API limit
            Write-Status -Level "WARN" -Message "API limit reached"
            Wait-ForReset
        }
        else {
            # Error - retry after delay
            Write-Status -Level "WARN" -Message "Execution failed, retrying in 30s..."
            Start-Sleep -Seconds 30
        }
        
        Write-Status -Level "LOOP" -Message "=== Completed Loop #$($script:LoopCount) ==="
    }
    
    # Cleanup - restore PROMPT.md
    Remove-TaskFromPrompt -BasePath "."
    Set-RunStateCompleted -BasePath "."
}

# Main entry point
if ($Help) {
    Show-Help
    exit 0
}

if ($Status) {
    Show-CurrentStatus
    exit 0
}

if ($TaskStatus) {
    Show-TaskStatus
    exit 0
}

if ($ResetCircuit) {
    Reset-CircuitBreaker -Reason "Manual reset via command line"
    exit 0
}

if ($CircuitStatus) {
    Show-CircuitStatus
    exit 0
}

# Determine which loop to run
if ($TaskMode) {
    if ($Monitor) {
        Write-Status -Level "INFO" -Message "Starting with monitoring..."
        $monitorScript = Join-Path $script:ScriptDir "ralph_monitor.ps1"
        if (Test-Path $monitorScript) {
            Start-Process pwsh -ArgumentList "-NoExit", "-File", $monitorScript -WindowStyle Normal
        }
    }
    Start-TaskModeLoop
}
elseif ($Monitor) {
    Start-WithMonitor
}
else {
    Start-RalphLoop
}
