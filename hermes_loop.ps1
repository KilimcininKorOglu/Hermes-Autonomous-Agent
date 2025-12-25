#Requires -Version 7.0

<#
.SYNOPSIS
    Hermes Loop for Claude Code - Windows PowerShell Version
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
    .\hermes_loop.ps1 -Monitor
.EXAMPLE
    .\hermes_loop.ps1 -Calls 50 -Timeout 30
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
    
    [switch]$TaskStatus,
    
    # Task status filters
    [ValidateSet("", "COMPLETED", "IN_PROGRESS", "NOT_STARTED", "BLOCKED")]
    [string]$StatusFilter = "",
    
    [string]$FeatureFilter = "",
    
    [ValidateSet("", "P1", "P2", "P3", "P4")]
    [string]$PriorityFilter = "",
    
    # Autonomous execution (no confirmation, no pause)
    [switch]$Autonomous,
    
    [int]$MaxConsecutiveErrors = 5,
    
    # AI Provider selection
    [ValidateSet("claude", "droid", "aider", "auto")]
    [string]$AI = "auto"
)

# Get script directory for module imports
$script:ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Import library modules
. "$script:ScriptDir\lib\Logger.ps1"
. "$script:ScriptDir\lib\ConfigManager.ps1"
. "$script:ScriptDir\lib\AIProvider.ps1"
. "$script:ScriptDir\lib\CircuitBreaker.ps1"
. "$script:ScriptDir\lib\ResponseAnalyzer.ps1"
. "$script:ScriptDir\lib\TaskReader.ps1"
. "$script:ScriptDir\lib\TaskStatusUpdater.ps1"
. "$script:ScriptDir\lib\GitBranchManager.ps1"
. "$script:ScriptDir\lib\PromptInjector.ps1"
. "$script:ScriptDir\lib\TableFormatter.ps1"

# Load configuration (merges global, project, and CLI params)
$script:HermesConfig = Get-HermesConfig

# Resolve AI Provider from config or CLI
$configProvider = Get-ConfigValue -Key "ai.provider"
$script:ResolvedAIProvider = if ($AI -ne "auto") {
    $AI
} elseif ($configProvider -ne "auto") {
    $configProvider
} else {
    Get-AutoProvider
}

# Validate AI provider
if (-not $script:ResolvedAIProvider) {
    Write-Host "[ERROR] No AI provider found. Install claude, droid, or aider." -ForegroundColor Red
    exit 1
}

if (-not (Test-AIProvider -Provider $script:ResolvedAIProvider)) {
    Write-Host "[ERROR] AI provider '$AI' is not installed." -ForegroundColor Red
    exit 1
}

# Configuration (merge CLI params with config file values)
$script:Config = @{
    PromptFile = $Prompt
    LogDir = Get-ConfigValue -Key "paths.logsDir" -Override $(if ($PSBoundParameters.ContainsKey('LogDir')) { $LogDir } else { $null })
    DocsDir = "docs\generated"
    StatusFile = "status.json"
    ProgressFile = "progress.json"
    AIProvider = $script:ResolvedAIProvider
    AITimeoutMinutes = Get-ConfigValue -Key "loop.timeoutMinutes" -Override $(if ($Timeout -ne 15) { $Timeout } else { $null })
    MaxCallsPerHour = Get-ConfigValue -Key "loop.maxCallsPerHour" -Override $(if ($Calls -ne 100) { $Calls } else { $null })
    VerboseProgress = $VerboseProgress
    CallCountFile = ".call_count"
    TimestampFile = ".last_reset"
    ExitSignalsFile = ".exit_signals"
    MaxConsecutiveTestLoops = 3
    MaxConsecutiveDoneSignals = 2
    # Task Mode Config
    TaskMode = $TaskMode
    TasksDir = Get-ConfigValue -Key "paths.tasksDir" -Override $(if ($TasksDir -ne "tasks") { $TasksDir } else { $null })
    AutoBranch = if ($AutoBranch) { $true } else { Get-ConfigValue -Key "taskMode.autoBranch" }
    AutoCommit = if ($AutoCommit) { $true } else { Get-ConfigValue -Key "taskMode.autoCommit" }
    StartFromTask = $StartFrom
    # Task State
    CurrentTask = $null
    CurrentFeature = $null
    CurrentBranch = ""
    # Autonomous Mode
    Autonomous = if ($Autonomous) { $true } else { Get-ConfigValue -Key "taskMode.autonomous" }
    MaxConsecutiveErrors = Get-ConfigValue -Key "taskMode.maxConsecutiveErrors" -Override $(if ($MaxConsecutiveErrors -ne 5) { $MaxConsecutiveErrors } else { $null })
    ConsecutiveErrors = 0
    StartTime = $null
    ErrorsRecovered = 0
}

# Global loop counter for cleanup
$script:LoopCount = 0

function Show-Help {
    <#
    .SYNOPSIS
        Displays help information
    #>
    
    Write-Host ""
    Write-Host "Hermes Autonomous Agent" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "IMPORTANT: This command must be run from a Hermes project directory." -ForegroundColor Yellow
    Write-Host "           Use 'hermes-setup project-name' to create a new project first."
    Write-Host ""
    Write-Host "Usage: Hermes [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Yellow
    Write-Host "    -h, -Help              Show this help message"
    Write-Host "    -AI PROVIDER           AI provider: claude, droid, aider, auto (default: auto)"
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
    Write-Host "    -StatusFilter STATUS   Filter by status (COMPLETED, IN_PROGRESS, NOT_STARTED, BLOCKED)"
    Write-Host "    -FeatureFilter FXXX    Filter by feature ID"
    Write-Host "    -PriorityFilter PX     Filter by priority (P1, P2, P3, P4)"
    Write-Host "    -Autonomous            Run without pausing between tasks/features"
    Write-Host "    -MaxConsecutiveErrors  Max errors before stopping (default: 5)"
    Write-Host ""
    Write-Host "Files created:" -ForegroundColor Yellow
    Write-Host "    - logs\              All execution logs"
    Write-Host "    - docs\generated\    Generated documentation"
    Write-Host "    - status.json        Current status (JSON)"
    Write-Host ""
    Write-Host "Example workflow:" -ForegroundColor Yellow
    Write-Host "    hermes-setup my-project     # Create project"
    Write-Host "    cd my-project              # Enter project directory"
    Write-Host "    hermes -Monitor             # Start Hermes with monitoring"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "    hermes -Calls 50 -Prompt my_prompt.md"
    Write-Host "    hermes -Monitor -Timeout 30"
    Write-Host "    hermes -VerboseProgress"
    Write-Host ""
    Write-Host "Task Mode Examples:" -ForegroundColor Yellow
    Write-Host "    hermes -TaskMode -AutoBranch -AutoCommit"
    Write-Host "    hermes -TaskMode -AutoBranch -AutoCommit -Autonomous"
    Write-Host "    hermes -TaskMode -AI droid -AutoBranch -AutoCommit"
    Write-Host "    hermes -TaskMode -StartFrom T005"
    Write-Host "    hermes -TaskStatus"
    Write-Host "    hermes -TaskStatus -StatusFilter BLOCKED"
    Write-Host "    hermes -TaskStatus -FeatureFilter F001"
    Write-Host "    hermes -TaskStatus -PriorityFilter P1"
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
    
    # Append to dated log file
    $dateStr = Get-Date -Format "yyyy-MM-dd"
    $logFile = Join-Path $script:Config.LogDir "hermes-loop-$dateStr.log"
    "[$timestamp] [$Level] $Message" | Add-Content -Path $logFile -Encoding UTF8
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
    
    return ""
}

function Invoke-AIExecution {
    <#
    .SYNOPSIS
        Executes AI provider with the prompt and handles the response
    .PARAMETER LoopCount
        Current loop iteration number
    .OUTPUTS
        Int - Exit code (0=success, 1=error, 2=API limit, 3=circuit breaker)
    #>
    param(
        [int]$LoopCount
    )
    
    $provider = $script:Config.AIProvider
    $timestamp = Get-Date -Format "yyyy-MM-dd_HH-mm-ss"
    $outputFile = Join-Path $script:Config.LogDir "${provider}_output_$timestamp.log"
    $callsMade = Add-CallCount
    
    Write-Status -Level "LOOP" -Message "Executing $provider (Call $callsMade/$($script:Config.MaxCallsPerHour))"
    
    $timeoutSeconds = $script:Config.AITimeoutMinutes * 60
    Write-Status -Level "INFO" -Message "Starting $provider execution... (timeout: $($script:Config.AITimeoutMinutes)m)"
    
    # Check if prompt file exists
    if (-not (Test-Path $script:Config.PromptFile)) {
        Write-Status -Level "ERROR" -Message "Prompt file not found: $($script:Config.PromptFile)"
        return 1
    }
    
    try {
        # Read prompt content
        $promptContent = Get-Content $script:Config.PromptFile -Raw
        
        # Start AI execution as a background job
        $job = Start-Job -ScriptBlock {
            param($content, $provider)
            
            switch ($provider) {
                "claude" {
                    $content | claude 2>&1
                }
                "droid" {
                    $content | droid exec --auto low 2>&1
                }
                "aider" {
                    $tempFile = [System.IO.Path]::GetTempFileName()
                    $tempFile = $tempFile -replace '\.tmp$', '.md'
                    $content | Set-Content $tempFile -Encoding UTF8
                    try {
                        aider --yes --no-auto-commits --message "Execute the task described in this file" $tempFile 2>&1
                    }
                    finally {
                        Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
                    }
                }
            }
        } -ArgumentList $promptContent, $provider
        
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
                provider = $provider
                indicator = $indicator
                elapsed_seconds = $elapsedSeconds
                timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
            } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
            
            if ($script:Config.VerboseProgress) {
                Write-Host "`r[$indicator] $provider working... ($elapsedSeconds`s elapsed)" -ForegroundColor Cyan -NoNewline
            }
            
            Start-Sleep -Seconds 5
            
            # Check timeout
            if ($elapsedSeconds -ge $timeoutSeconds) {
                Stop-Job $job -ErrorAction SilentlyContinue
                Remove-Job $job -Force -ErrorAction SilentlyContinue
                
                Write-Host ""
                Write-Status -Level "ERROR" -Message "Execution timed out after $($script:Config.AITimeoutMinutes) minutes"
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
        
        # Write output to file
        if ($output) {
            $output | Out-File $outputFile -Encoding UTF8
        }
        else {
            "No output received from $provider" | Out-File $outputFile -Encoding UTF8
        }
        
        if ($jobState -eq 'Completed') {
            # Update progress file
            @{
                status = "completed"
                provider = $provider
                timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
            } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
            
            Write-Status -Level "SUCCESS" -Message "$provider execution completed successfully"
            
            # Analyze the response
            Write-Status -Level "INFO" -Message "Analyzing $provider response..."
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
                provider = $provider
                timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
            } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
            
            # Check if failure is due to API limit
            $outputContent = if ($output) { $output -join "`n" } else { "" }
            if ($outputContent -match "(?i)(5.*hour.*limit|limit.*reached.*try.*back|usage.*limit.*reached)") {
                Write-Status -Level "ERROR" -Message "API usage limit reached"
                return 2
            }
            
            Write-Status -Level "ERROR" -Message "$provider execution failed, check: $outputFile"
            return 1
        }
    }
    catch {
        Write-Status -Level "ERROR" -Message "Exception during $provider execution: $($_.Exception.Message)"
        
        @{
            status = "failed"
            provider = $provider
            error = $_.Exception.Message
            timestamp = (Get-Date -Format "yyyy-MM-dd HH:mm:ss")
        } | ConvertTo-Json | Set-Content $script:Config.ProgressFile -Encoding UTF8
        
        return 1
    }
}

function Start-HermesLoop {
    <#
    .SYNOPSIS
        Main Hermes loop execution
    #>
    
    Write-Status -Level "SUCCESS" -Message "Hermes loop starting with $($script:Config.AIProvider)"
    Write-Status -Level "INFO" -Message "AI Provider: $($script:Config.AIProvider)"
    Write-Status -Level "INFO" -Message "Max calls per hour: $($script:Config.MaxCallsPerHour)"
    Write-Status -Level "INFO" -Message "Logs: $($script:Config.LogDir)\ | Status: $($script:Config.StatusFile)"
    
    # Initialize directories
    if (-not (Test-Path $script:Config.LogDir)) {
        New-Item -ItemType Directory -Path $script:Config.LogDir -Force | Out-Null
    }
    if (-not (Test-Path $script:Config.DocsDir)) {
        New-Item -ItemType Directory -Path $script:Config.DocsDir -Force | Out-Null
    }
    
    # Check if this is a Hermes project
    if (-not (Test-Path $script:Config.PromptFile)) {
        Write-Status -Level "ERROR" -Message "Prompt file '$($script:Config.PromptFile)' not found!"
        Write-Host ""
        
        if ((Test-Path "tasks") -or (Test-Path "specs")) {
            Write-Host "This appears to be a Hermes project but is missing PROMPT.md." -ForegroundColor Yellow
            Write-Host "You may need to create or restore the PROMPT.md file."
        }
        else {
            Write-Host "This directory is not a Hermes project." -ForegroundColor Yellow
        }
        
        Write-Host ""
        Write-Host "To fix this:" -ForegroundColor Cyan
        Write-Host "  1. Create a new project: hermes-setup my-project"
        Write-Host "  2. Import existing requirements: hermes-import requirements.md"
        Write-Host "  3. Navigate to an existing Hermes project directory"
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
            Write-Status -Level "SUCCESS" -Message "Hermes has completed the project! Final stats:"
            Write-Status -Level "INFO" -Message "  - Total loops: $($script:LoopCount)"
            Write-Status -Level "INFO" -Message "  - API calls used: $(Get-CallCount)"
            Write-Status -Level "INFO" -Message "  - Exit reason: $exitReason"
            Write-Host ""
            break
        }
        
        # Update status
        Update-LoopStatus -LoopCount $script:LoopCount -CallsMade (Get-CallCount) `
            -LastAction "executing" -Status "running"
        
        # Execute AI
        $execResult = Invoke-AIExecution -LoopCount $script:LoopCount
        
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
                Write-Status -Level "INFO" -Message "Run 'hermes -ResetCircuit' to reset the circuit breaker after addressing issues"
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
        Shows current Hermes status
    #>
    
    if (Test-Path $script:Config.StatusFile) {
        Write-Host ""
        Write-Host "Current Hermes Status:" -ForegroundColor Cyan
        Write-Host ""
        $status = Get-Content $script:Config.StatusFile -Raw | ConvertFrom-Json
        $status | Format-List
    }
    else {
        Write-Host ""
        Write-Host "No status file found. Hermes may not be running." -ForegroundColor Yellow
        Write-Host "Status file expected at: $($script:Config.StatusFile)" -ForegroundColor Gray
        Write-Host ""
    }
}

function Start-WithMonitor {
    <#
    .SYNOPSIS
        Starts Hermes with a separate monitor window
    #>
    
    Write-Status -Level "INFO" -Message "Starting with monitoring..."
    
    $monitorScript = Join-Path $script:ScriptDir "hermes_monitor.ps1"
    
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
    Start-HermesLoop
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

function Get-ProgressBar {
    <#
    .SYNOPSIS
        Creates a visual progress bar
    #>
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

function Show-TaskCompletionSummary {
    <#
    .SYNOPSIS
        Shows summary after task completion (autonomous mode)
    #>
    param(
        [hashtable]$Task,
        [string]$Duration = ""
    )
    
    $progress = Get-TaskProgress -BasePath "."
    $bar = Get-ProgressBar -Percentage $progress.Percentage -Width 20
    
    Write-Host ""
    Write-Host ([char]0x2501 * 50) -ForegroundColor Cyan
    Write-Host "$([char]0x2713) $($Task.TaskId): $($Task.Name) completed" -ForegroundColor Green -NoNewline
    if ($Duration) {
        Write-Host " ($Duration)" -ForegroundColor Gray
    }
    else {
        Write-Host ""
    }
    Write-Host ""
    Write-Host "  Progress: $bar $($progress.Percentage)% ($($progress.Completed)/$($progress.Total) tasks)" -ForegroundColor White
    
    $nextTask = Get-NextTask -BasePath "."
    if ($nextTask -and $nextTask.TaskId -ne $Task.TaskId) {
        Write-Host "  Next: $($nextTask.TaskId) - $($nextTask.Name)" -ForegroundColor Gray
    }
    Write-Host ([char]0x2501 * 50) -ForegroundColor Cyan
    Write-Host ""
}

function Show-FeatureCompletionSummary {
    <#
    .SYNOPSIS
        Shows summary after feature completion (autonomous mode)
    #>
    param(
        [string]$FeatureId
    )
    
    $feature = Get-FeatureById -FeatureId $FeatureId -BasePath "."
    $fp = Get-FeatureProgress -FeatureId $FeatureId -BasePath "."
    
    $allFeatures = @(Get-AllFeatures -BasePath ".")
    $completedFeatures = @($allFeatures | Where-Object { $_.Status -eq "COMPLETED" }).Count
    $totalFeatures = $allFeatures.Count
    
    $percentage = if ($totalFeatures -gt 0) { [int](($completedFeatures / $totalFeatures) * 100) } else { 0 }
    $bar = Get-ProgressBar -Percentage $percentage -Width 20
    
    Write-Host ""
    Write-Host ([char]0x2501 * 50) -ForegroundColor Green
    Write-Host "$([char]0x2713) $FeatureId`: $($feature.FeatureName) - COMPLETED & MERGED" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Tasks: $($fp.Completed)/$($fp.Total) completed" -ForegroundColor White
    Write-Host ""
    Write-Host "  Feature Progress: $bar $completedFeatures/$totalFeatures features" -ForegroundColor White
    
    # Get next feature
    $nextFeature = $allFeatures | Where-Object { $_.Status -eq "NOT_STARTED" -or $_.Status -eq "IN_PROGRESS" } | Select-Object -First 1
    if ($nextFeature) {
        Write-Host "  Next Feature: $($nextFeature.FeatureId) - $($nextFeature.FeatureName)" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  Continuing automatically..." -ForegroundColor Cyan
    }
    Write-Host ([char]0x2501 * 50) -ForegroundColor Green
    Write-Host ""
}

function Show-FinalCompletionSummary {
    <#
    .SYNOPSIS
        Shows final summary when all tasks are done
    #>
    
    $duration = if ($script:Config.StartTime) {
        ((Get-Date) - $script:Config.StartTime).ToString('hh\:mm\:ss')
    }
    else {
        "N/A"
    }
    
    $progress = Get-TaskProgress -BasePath "."
    $allFeatures = @(Get-AllFeatures -BasePath ".")
    $completedFeatures = @($allFeatures | Where-Object { $_.Status -eq "COMPLETED" }).Count
    
    Write-Host ""
    Write-Host ([char]0x2501 * 50) -ForegroundColor Green
    Write-Host "$([char]0x2713) ALL TASKS COMPLETED" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Duration: $duration" -ForegroundColor White
    Write-Host "  Tasks: $($progress.Completed)/$($progress.Total) completed" -ForegroundColor White
    Write-Host "  Features: $completedFeatures/$($allFeatures.Count) completed" -ForegroundColor White
    Write-Host ""
    Write-Host "  Blocked: $($progress.Blocked)" -ForegroundColor $(if ($progress.Blocked -gt 0) { "Yellow" } else { "White" })
    Write-Host "  Errors: $($script:Config.ErrorsRecovered) (recovered)" -ForegroundColor $(if ($script:Config.ErrorsRecovered -gt 0) { "Yellow" } else { "White" })
    Write-Host ""
    Write-Host "  Git: All branches merged to main" -ForegroundColor White
    Write-Host ([char]0x2501 * 50) -ForegroundColor Green
    Write-Host ""
}

function Test-FeatureCompleted {
    <#
    .SYNOPSIS
        Checks if all tasks in a feature are completed
    #>
    param(
        [string]$FeatureId
    )
    
    $fp = Get-FeatureProgress -FeatureId $FeatureId -BasePath "."
    return ($fp.Completed -eq $fp.Total -and $fp.Total -gt 0)
}

function Start-TaskModeLoop {
    <#
    .SYNOPSIS
        Main loop for Task Mode execution
    #>
    
    # Initialize start time for autonomous mode
    $script:Config.StartTime = Get-Date
    $script:Config.ConsecutiveErrors = 0
    $script:Config.ErrorsRecovered = 0
    
    Write-Status -Level "SUCCESS" -Message "Hermes Task Mode starting..."
    Write-Status -Level "INFO" -Message "AI Provider: $($script:Config.AIProvider)"
    Write-Status -Level "INFO" -Message "Tasks directory: $($script:Config.TasksDir)"
    Write-Status -Level "INFO" -Message "Auto-branch: $($script:Config.AutoBranch)"
    Write-Status -Level "INFO" -Message "Auto-commit: $($script:Config.AutoCommit)"
    
    if ($script:Config.Autonomous) {
        Write-Status -Level "INFO" -Message "Autonomous mode: ENABLED (no pauses)"
    }
    
    # Check for resume if no StartFrom specified
    if (-not $script:Config.StartFromTask -and (Test-ShouldResume -BasePath ".")) {
        $resumeInfo = Get-ResumeInfo -BasePath "."
        if ($resumeInfo -and $resumeInfo.ResumeTaskId) {
            Write-Host ""
            Write-Host ("=" * 50) -ForegroundColor Yellow
            Write-Host "  Previous run detected - Resuming" -ForegroundColor Yellow
            Write-Host ("=" * 50) -ForegroundColor Yellow
            Write-Host "  Resume Task: $($resumeInfo.ResumeTaskId)" -ForegroundColor White
            if ($resumeInfo.CurrentBranch) {
                Write-Host "  Branch: $($resumeInfo.CurrentBranch)" -ForegroundColor White
            }
            Write-Host ("=" * 50) -ForegroundColor Yellow
            Write-Host ""
            
            $script:Config.StartFromTask = $resumeInfo.ResumeTaskId
            
            # Switch to branch if needed
            if ($resumeInfo.CurrentBranch -and $script:Config.AutoBranch) {
                $currentBranch = git rev-parse --abbrev-ref HEAD 2>$null
                if ($currentBranch -ne $resumeInfo.CurrentBranch) {
                    Write-Status -Level "INFO" -Message "Switching to branch: $($resumeInfo.CurrentBranch)"
                    git checkout $resumeInfo.CurrentBranch 2>$null
                }
            }
        }
    }
    
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
            
            if ($script:Config.Autonomous) {
                Show-FinalCompletionSummary
            }
            else {
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
            }
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
        
        # Execute AI
        $execResult = Invoke-AIExecution -LoopCount $script:LoopCount
        
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
                
                # Reset consecutive errors on success
                $script:Config.ConsecutiveErrors = 0
                
                # Show task completion summary in autonomous mode
                if ($script:Config.Autonomous) {
                    Show-TaskCompletionSummary -Task $task
                }
                
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
                    
                    # Show feature completion summary in autonomous mode
                    if ($script:Config.Autonomous) {
                        Show-FeatureCompletionSummary -FeatureId $task.FeatureId
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
            # Error - increment counter and possibly stop
            $script:Config.ConsecutiveErrors++
            $script:Config.ErrorsRecovered++
            
            if ($script:Config.Autonomous -and $script:Config.ConsecutiveErrors -ge $script:Config.MaxConsecutiveErrors) {
                Write-Status -Level "ERROR" -Message "Too many consecutive errors ($($script:Config.ConsecutiveErrors)). Stopping."
                break
            }
            
            Write-Status -Level "WARN" -Message "Execution failed (error $($script:Config.ConsecutiveErrors)), retrying in 30s..."
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
    Show-EnhancedTaskStatus -StatusFilter $StatusFilter -FeatureFilter $FeatureFilter `
        -PriorityFilter $PriorityFilter -BasePath "."
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
        $monitorScript = Join-Path $script:ScriptDir "hermes_monitor.ps1"
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
    Start-HermesLoop
}
