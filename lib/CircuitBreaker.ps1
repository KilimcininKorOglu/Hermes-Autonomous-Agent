#Requires -Version 7.0

<#
.SYNOPSIS
    Circuit Breaker Module for Hermes
.DESCRIPTION
    Prevents runaway token consumption by detecting stagnation.
    Based on Michael Nygard's "Release It!" pattern.
.NOTES
    States:
    - CLOSED: Normal operation, progress detected
    - HALF_OPEN: Monitoring mode, checking for recovery
    - OPEN: Failure detected, execution halted
#>

# Circuit Breaker States
$script:CB_STATE_CLOSED = "CLOSED"
$script:CB_STATE_HALF_OPEN = "HALF_OPEN"
$script:CB_STATE_OPEN = "OPEN"

# Configuration
$script:CB_STATE_FILE = ".hermes\.circuit_breaker_state"
$script:CB_HISTORY_FILE = ".hermes\.circuit_breaker_history"
$script:CB_NO_PROGRESS_THRESHOLD = 3
$script:CB_SAME_ERROR_THRESHOLD = 5

function Initialize-CircuitBreaker {
    <#
    .SYNOPSIS
        Initializes the circuit breaker state files
    .DESCRIPTION
        Creates or validates the circuit breaker state and history files.
        If files are corrupted, they are recreated with default values.
    #>
    
    # Validate and create state file
    if (Test-Path $script:CB_STATE_FILE) {
        try {
            $null = Get-Content $script:CB_STATE_FILE -Raw | ConvertFrom-Json
        }
        catch {
            # Corrupted JSON, remove and recreate
            Remove-Item $script:CB_STATE_FILE -Force -ErrorAction SilentlyContinue
        }
    }
    
    if (-not (Test-Path $script:CB_STATE_FILE)) {
        $initialState = @{
            state = $script:CB_STATE_CLOSED
            last_change = (Get-Date -Format "o")
            consecutive_no_progress = 0
            consecutive_same_error = 0
            last_progress_loop = 0
            total_opens = 0
            reason = ""
            current_loop = 0
        }
        $initialState | ConvertTo-Json -Depth 10 | Set-Content $script:CB_STATE_FILE -Encoding UTF8
    }
    
    # Validate and create history file
    if (Test-Path $script:CB_HISTORY_FILE) {
        try {
            $null = Get-Content $script:CB_HISTORY_FILE -Raw | ConvertFrom-Json
        }
        catch {
            Remove-Item $script:CB_HISTORY_FILE -Force -ErrorAction SilentlyContinue
        }
    }
    
    if (-not (Test-Path $script:CB_HISTORY_FILE)) {
        "[]" | Set-Content $script:CB_HISTORY_FILE -Encoding UTF8
    }
}

function Get-CircuitState {
    <#
    .SYNOPSIS
        Gets the current circuit breaker state
    .OUTPUTS
        String - Current state (CLOSED, HALF_OPEN, or OPEN)
    #>
    
    if (-not (Test-Path $script:CB_STATE_FILE)) {
        return $script:CB_STATE_CLOSED
    }
    
    try {
        $stateData = Get-Content $script:CB_STATE_FILE -Raw | ConvertFrom-Json
        return $stateData.state
    }
    catch {
        return $script:CB_STATE_CLOSED
    }
}

function Get-CircuitBreakerData {
    <#
    .SYNOPSIS
        Gets the full circuit breaker state data
    .OUTPUTS
        PSObject - Full state data object
    #>
    
    if (-not (Test-Path $script:CB_STATE_FILE)) {
        return $null
    }
    
    try {
        return Get-Content $script:CB_STATE_FILE -Raw | ConvertFrom-Json
    }
    catch {
        return $null
    }
}

function Test-CanExecute {
    <#
    .SYNOPSIS
        Checks if circuit breaker allows execution
    .OUTPUTS
        Boolean - True if execution is allowed
    #>
    
    $state = Get-CircuitState
    return $state -ne $script:CB_STATE_OPEN
}

function Add-LoopResult {
    <#
    .SYNOPSIS
        Records loop execution result and updates circuit breaker state
    .PARAMETER LoopNumber
        Current loop iteration number
    .PARAMETER FilesChanged
        Number of files modified in this loop
    .PARAMETER HasErrors
        Whether errors were detected in the output
    .PARAMETER OutputLength
        Length of the output in bytes
    .OUTPUTS
        Boolean - True if execution can continue, False if circuit opened
    #>
    param(
        [Parameter(Mandatory)]
        [int]$LoopNumber,
        
        [int]$FilesChanged = 0,
        
        [bool]$HasErrors = $false,
        
        [int]$OutputLength = 0,
        
        [bool]$HasProgress = $false
    )
    
    Initialize-CircuitBreaker
    
    $stateData = Get-Content $script:CB_STATE_FILE -Raw | ConvertFrom-Json
    
    $currentState = $stateData.state
    $consecutiveNoProgress = [int]$stateData.consecutive_no_progress
    $consecutiveSameError = [int]$stateData.consecutive_same_error
    $lastProgressLoop = [int]$stateData.last_progress_loop
    $totalOpens = [int]$stateData.total_opens
    
    # Detect progress (explicit HasProgress flag OR files changed)
    $hasProgress = $HasProgress -or ($FilesChanged -gt 0)
    if ($hasProgress) {
        $consecutiveNoProgress = 0
        $lastProgressLoop = $LoopNumber
    }
    else {
        $consecutiveNoProgress++
    }
    
    # Detect same error repetition
    if ($HasErrors) {
        $consecutiveSameError++
    }
    else {
        $consecutiveSameError = 0
    }
    
    # Determine new state and reason
    $newState = $currentState
    $reason = $stateData.reason
    
    switch ($currentState) {
        $script:CB_STATE_CLOSED {
            # Normal operation - check for failure conditions
            if ($consecutiveNoProgress -ge $script:CB_NO_PROGRESS_THRESHOLD) {
                $newState = $script:CB_STATE_OPEN
                $reason = "No progress detected in $consecutiveNoProgress consecutive loops"
            }
            elseif ($consecutiveSameError -ge $script:CB_SAME_ERROR_THRESHOLD) {
                $newState = $script:CB_STATE_OPEN
                $reason = "Same error repeated in $consecutiveSameError consecutive loops"
            }
            elseif ($consecutiveNoProgress -ge 2) {
                $newState = $script:CB_STATE_HALF_OPEN
                $reason = "Monitoring: $consecutiveNoProgress loops without progress"
            }
        }
        
        $script:CB_STATE_HALF_OPEN {
            # Monitoring mode - either recover or fail
            if ($hasProgress) {
                $newState = $script:CB_STATE_CLOSED
                $reason = "Progress detected, circuit recovered"
            }
            elseif ($consecutiveNoProgress -ge $script:CB_NO_PROGRESS_THRESHOLD) {
                $newState = $script:CB_STATE_OPEN
                $reason = "No recovery, opening circuit after $consecutiveNoProgress loops"
            }
        }
        
        $script:CB_STATE_OPEN {
            # Circuit is open - stays open until manual reset
            $reason = "Circuit breaker is open, execution halted"
        }
    }
    
    # Update opens counter
    if ($newState -eq $script:CB_STATE_OPEN -and $currentState -ne $script:CB_STATE_OPEN) {
        $totalOpens++
    }
    
    # Update state file
    $newStateData = @{
        state = $newState
        last_change = (Get-Date -Format "o")
        consecutive_no_progress = $consecutiveNoProgress
        consecutive_same_error = $consecutiveSameError
        last_progress_loop = $lastProgressLoop
        total_opens = $totalOpens
        reason = $reason
        current_loop = $LoopNumber
    }
    $newStateData | ConvertTo-Json -Depth 10 | Set-Content $script:CB_STATE_FILE -Encoding UTF8
    
    # Log state transition
    if ($newState -ne $currentState) {
        Write-CircuitTransition -FromState $currentState -ToState $newState -Reason $reason -LoopNumber $LoopNumber
    }
    
    # Return whether execution can continue
    return $newState -ne $script:CB_STATE_OPEN
}

function Write-CircuitTransition {
    <#
    .SYNOPSIS
        Logs circuit breaker state transition to history and console
    #>
    param(
        [string]$FromState,
        [string]$ToState,
        [string]$Reason,
        [int]$LoopNumber
    )
    
    # Add to history file
    $history = @()
    if (Test-Path $script:CB_HISTORY_FILE) {
        try {
            $existingHistory = Get-Content $script:CB_HISTORY_FILE -Raw | ConvertFrom-Json
            if ($existingHistory) {
                $history = @($existingHistory)
            }
        }
        catch {
            $history = @()
        }
    }
    
    $transition = @{
        timestamp = (Get-Date -Format "o")
        loop = $LoopNumber
        from_state = $FromState
        to_state = $ToState
        reason = $Reason
    }
    
    $history += $transition
    
    # Keep only last 100 transitions
    if ($history.Count -gt 100) {
        $history = $history[-100..-1]
    }
    
    $history | ConvertTo-Json -Depth 10 -AsArray | Set-Content $script:CB_HISTORY_FILE -Encoding UTF8
    
    # Console output with colors
    Write-Host ""
    switch ($ToState) {
        $script:CB_STATE_OPEN {
            Write-Host "[XX] CIRCUIT BREAKER OPENED" -ForegroundColor Red
            Write-Host "     Reason: $Reason" -ForegroundColor Red
        }
        $script:CB_STATE_HALF_OPEN {
            Write-Host "[!!] CIRCUIT BREAKER: Monitoring Mode" -ForegroundColor Yellow
            Write-Host "     Reason: $Reason" -ForegroundColor Yellow
        }
        $script:CB_STATE_CLOSED {
            Write-Host "[OK] CIRCUIT BREAKER: Normal Operation" -ForegroundColor Green
            Write-Host "     Reason: $Reason" -ForegroundColor Green
        }
    }
    Write-Host ""
}

function Show-CircuitStatus {
    <#
    .SYNOPSIS
        Displays current circuit breaker status in a formatted view
    #>
    
    Initialize-CircuitBreaker
    
    $stateData = Get-Content $script:CB_STATE_FILE -Raw | ConvertFrom-Json
    
    $color = switch ($stateData.state) {
        $script:CB_STATE_CLOSED { "Green" }
        $script:CB_STATE_HALF_OPEN { "Yellow" }
        $script:CB_STATE_OPEN { "Red" }
        default { "Gray" }
    }
    
    $icon = switch ($stateData.state) {
        $script:CB_STATE_CLOSED { "[OK]" }
        $script:CB_STATE_HALF_OPEN { "[!!]" }
        $script:CB_STATE_OPEN { "[XX]" }
        default { "[??]" }
    }
    
    Write-Host ""
    Write-Host ("=" * 60) -ForegroundColor $color
    Write-Host "           Circuit Breaker Status" -ForegroundColor $color
    Write-Host ("=" * 60) -ForegroundColor $color
    Write-Host "State:                 $icon $($stateData.state)" -ForegroundColor $color
    Write-Host "Reason:                $($stateData.reason)" -ForegroundColor White
    Write-Host "Loops since progress:  $($stateData.consecutive_no_progress)" -ForegroundColor White
    Write-Host "Last progress:         Loop #$($stateData.last_progress_loop)" -ForegroundColor White
    Write-Host "Current loop:          #$($stateData.current_loop)" -ForegroundColor White
    Write-Host "Total opens:           $($stateData.total_opens)" -ForegroundColor White
    Write-Host ("=" * 60) -ForegroundColor $color
    Write-Host ""
}

function Reset-CircuitBreaker {
    <#
    .SYNOPSIS
        Resets circuit breaker to CLOSED state
    .PARAMETER Reason
        Reason for the reset (logged in state)
    #>
    param(
        [string]$Reason = "Manual reset"
    )
    
    $resetState = @{
        state = $script:CB_STATE_CLOSED
        last_change = (Get-Date -Format "o")
        consecutive_no_progress = 0
        consecutive_same_error = 0
        last_progress_loop = 0
        total_opens = 0
        reason = $Reason
        current_loop = 0
    }
    $resetState | ConvertTo-Json -Depth 10 | Set-Content $script:CB_STATE_FILE -Encoding UTF8
    
    Write-Host ""
    Write-Host "[OK] Circuit breaker reset to CLOSED state" -ForegroundColor Green
    Write-Host "     Reason: $Reason" -ForegroundColor Gray
    Write-Host ""
}

function Test-ShouldHalt {
    <#
    .SYNOPSIS
        Checks if loop should halt due to circuit breaker being open
    .OUTPUTS
        Boolean - True if loop should halt
    #>
    
    $state = Get-CircuitState
    
    if ($state -eq $script:CB_STATE_OPEN) {
        Show-CircuitStatus
        
        Write-Host ""
        Write-Host ("=" * 60) -ForegroundColor Red
        Write-Host "  EXECUTION HALTED: Circuit Breaker Opened" -ForegroundColor Red
        Write-Host ("=" * 60) -ForegroundColor Red
        Write-Host ""
        Write-Host "Hermes has detected that no progress is being made." -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Possible reasons:" -ForegroundColor Yellow
        Write-Host "  - Project may be complete (check @fix_plan.md)" -ForegroundColor White
        Write-Host "  - Claude may be stuck on an error" -ForegroundColor White
        Write-Host "  - PROMPT.md may need clarification" -ForegroundColor White
        Write-Host "  - Manual intervention may be required" -ForegroundColor White
        Write-Host ""
        Write-Host "To continue:" -ForegroundColor Yellow
        Write-Host "  1. Review recent logs:" -ForegroundColor White
        Write-Host "     Get-Content logs\Hermes.log -Tail 20" -ForegroundColor Cyan
        Write-Host "  2. Check Claude output:" -ForegroundColor White
        Write-Host "     Get-ChildItem logs\claude_output_*.log | Sort-Object LastWriteTime -Descending | Select-Object -First 1" -ForegroundColor Cyan
        Write-Host "  3. Update @fix_plan.md if needed" -ForegroundColor White
        Write-Host "  4. Reset circuit breaker:" -ForegroundColor White
        Write-Host "     hermes -ResetCircuit" -ForegroundColor Cyan
        Write-Host ""
        
        return $true
    }
    
    return $false
}

# Export functions for dot-sourcing
# When used as module (.psm1), use Export-ModuleMember instead
