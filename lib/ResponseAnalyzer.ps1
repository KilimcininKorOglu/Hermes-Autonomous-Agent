#Requires -Version 7.0

<#
.SYNOPSIS
    Response Analyzer Module for Hermes
.DESCRIPTION
    Analyzes Claude Code output to detect completion signals, 
    test-only loops, stuck conditions, and progress indicators.
.NOTES
    This module provides semantic analysis of Claude's responses
    to determine if the project is complete or stuck.
#>

# Configuration - Keywords and Patterns
$script:CompletionKeywords = @(
    "done"
    "complete"
    "finished"
    "all tasks complete"
    "project complete"
    "ready for review"
    "implementation complete"
    "all requirements met"
)

$script:TestOnlyPatterns = @(
    "npm test"
    "bats"
    "pytest"
    "jest"
    "cargo test"
    "go test"
    "running tests"
    "test passed"
    "tests passed"
    "all tests pass"
)

$script:ImplementationPatterns = @(
    "implementing"
    "creating"
    "writing"
    "adding"
    "function"
    "class"
    "module"
    "component"
    "feature"
    "building"
)

$script:StuckIndicators = @(
    "error"
    "failed"
    "cannot"
    "unable to"
    "blocked"
    "stuck"
    "issue"
    "problem"
)

$script:NoWorkPatterns = @(
    "nothing to do"
    "no changes"
    "already implemented"
    "up to date"
    "no remaining work"
    "all done"
)

# File paths
$script:AnalysisResultFile = ".response_analysis"
$script:LastOutputLengthFile = ".last_output_length"
$script:ExitSignalsFile = ".exit_signals"

function Invoke-ResponseAnalysis {
    <#
    .SYNOPSIS
        Analyzes Claude Code response and extracts signals
    .PARAMETER OutputFile
        Path to the Claude Code output file
    .PARAMETER LoopNumber
        Current loop iteration number
    .OUTPUTS
        Boolean - True if analysis succeeded
    #>
    param(
        [Parameter(Mandatory)]
        [string]$OutputFile,
        
        [Parameter(Mandatory)]
        [int]$LoopNumber
    )
    
    # Initialize analysis result
    $analysis = @{
        has_completion_signal = $false
        is_test_only = $false
        is_stuck = $false
        has_progress = $false
        confidence_score = 0
        exit_signal = $false
        work_summary = ""
        files_modified = 0
        output_length = 0
        error_count = 0
        # Task mode fields
        task_completed = $false
        completed_task_id = ""
        task_blocked = $false
        blocked_reason = ""
    }
    
    # Check if output file exists
    if (-not (Test-Path $OutputFile)) {
        Write-Host "[ERROR] Output file not found: $OutputFile" -ForegroundColor Red
        return $false
    }
    
    # Read output content
    $outputContent = Get-Content $OutputFile -Raw -ErrorAction SilentlyContinue
    if ([string]::IsNullOrEmpty($outputContent)) {
        $outputContent = ""
    }
    $analysis.output_length = $outputContent.Length
    
    # 1. Check for structured Hermes status output
    if ($outputContent -match "---Hermes_STATUS---") {
        # Extract STATUS
        if ($outputContent -match "STATUS:\s*(COMPLETE|DONE)") {
            $analysis.has_completion_signal = $true
            $analysis.exit_signal = $true
            $analysis.confidence_score = 100
        }
        
        # Extract EXIT_SIGNAL
        if ($outputContent -match "EXIT_SIGNAL:\s*true") {
            $analysis.exit_signal = $true
            $analysis.confidence_score = 100
        }
        
        # Extract WORK_TYPE
        if ($outputContent -match "WORK_TYPE:\s*TESTING") {
            $analysis.is_test_only = $true
        }
        
        # Extract RECOMMENDATION
        if ($outputContent -match "RECOMMENDATION:\s*(.+?)(?:\r?\n|---END)") {
            $analysis.work_summary = $Matches[1].Trim()
        }
        
        # Extract TASK_ID for task mode
        if ($outputContent -match "TASK_ID:\s*(T\d+)") {
            $analysis.completed_task_id = $Matches[1]
            
            # If STATUS is COMPLETE and we have TASK_ID, mark task as completed
            if ($outputContent -match "STATUS:\s*COMPLETE") {
                $analysis.task_completed = $true
            }
        }
        
        # Check for blocked task
        if ($outputContent -match "STATUS:\s*BLOCKED") {
            $analysis.task_blocked = $true
            if ($outputContent -match "BLOCKED_REASON:\s*(.+?)(?:\r?\n|---END)") {
                $analysis.blocked_reason = $Matches[1].Trim()
            }
        }
    }
    
    # 1b. Additional task completion detection (fallback patterns)
    if (-not $analysis.task_completed) {
        # Pattern: "Task TXXX completed" or "TXXX is done"
        if ($outputContent -match "(?i)Task\s+(T\d+)\s+(?:completed|done|finished)") {
            $analysis.completed_task_id = $Matches[1]
            $analysis.task_completed = $true
        }
        elseif ($outputContent -match "(?i)(T\d+)\s+is\s+(?:complete|done|finished)") {
            $analysis.completed_task_id = $Matches[1]
            $analysis.task_completed = $true
        }
    }
    
    # 2. Detect completion keywords in natural language
    foreach ($keyword in $script:CompletionKeywords) {
        $escapedKeyword = [regex]::Escape($keyword)
        if ($outputContent -match "(?i)$escapedKeyword") {
            $analysis.has_completion_signal = $true
            $analysis.confidence_score += 10
            break
        }
    }
    
    # 3. Detect test-only loops
    $testCommandCount = 0
    $implementationCount = 0
    
    foreach ($pattern in $script:TestOnlyPatterns) {
        $escapedPattern = [regex]::Escape($pattern)
        $matches = [regex]::Matches($outputContent, "(?i)$escapedPattern")
        $testCommandCount += $matches.Count
    }
    
    foreach ($pattern in $script:ImplementationPatterns) {
        $escapedPattern = [regex]::Escape($pattern)
        $matches = [regex]::Matches($outputContent, "(?i)$escapedPattern")
        $implementationCount += $matches.Count
    }
    
    if ($testCommandCount -gt 0 -and $implementationCount -eq 0) {
        $analysis.is_test_only = $true
        if ([string]::IsNullOrEmpty($analysis.work_summary)) {
            $analysis.work_summary = "Test execution only, no implementation"
        }
    }
    
    # 4. Detect stuck/error conditions
    $errorCount = 0
    foreach ($indicator in $script:StuckIndicators) {
        $escapedIndicator = [regex]::Escape($indicator)
        $matches = [regex]::Matches($outputContent, "(?i)$escapedIndicator")
        $errorCount += $matches.Count
    }
    $analysis.error_count = $errorCount
    
    if ($errorCount -gt 5) {
        $analysis.is_stuck = $true
    }
    
    # 5. Detect "nothing to do" patterns
    foreach ($pattern in $script:NoWorkPatterns) {
        $escapedPattern = [regex]::Escape($pattern)
        if ($outputContent -match "(?i)$escapedPattern") {
            $analysis.has_completion_signal = $true
            $analysis.confidence_score += 15
            if ([string]::IsNullOrEmpty($analysis.work_summary)) {
                $analysis.work_summary = "No work remaining"
            }
            break
        }
    }
    
    # 6. Check for file changes via git
    try {
        $gitOutput = git diff --name-only 2>$null
        if ($gitOutput) {
            $analysis.files_modified = ($gitOutput | Measure-Object).Count
        }
        if ($analysis.files_modified -gt 0) {
            $analysis.has_progress = $true
            $analysis.confidence_score += 20
        }
    }
    catch {
        # Not in git repo or git error - ignore
    }
    
    # 7. Analyze output length trends
    if (Test-Path $script:LastOutputLengthFile) {
        try {
            $lastLength = [int](Get-Content $script:LastOutputLengthFile -Raw)
            if ($lastLength -gt 0 -and $analysis.output_length -gt 0) {
                $lengthRatio = ($analysis.output_length * 100) / $lastLength
                if ($lengthRatio -lt 50) {
                    # Output is significantly shorter - possible completion
                    $analysis.confidence_score += 10
                }
            }
        }
        catch {
            # Ignore parse errors
        }
    }
    $analysis.output_length.ToString() | Set-Content $script:LastOutputLengthFile -Encoding UTF8
    
    # 8. Extract work summary if not already set
    if ([string]::IsNullOrEmpty($analysis.work_summary)) {
        # Try to find summary in output
        $summaryPatterns = @(
            "(?i)summary[:\s]+(.{10,100})"
            "(?i)completed[:\s]+(.{10,100})"
            "(?i)implemented[:\s]+(.{10,100})"
            "(?i)finished[:\s]+(.{10,100})"
        )
        
        foreach ($pattern in $summaryPatterns) {
            if ($outputContent -match $pattern) {
                $analysis.work_summary = $Matches[1].Trim()
                break
            }
        }
        
        if ([string]::IsNullOrEmpty($analysis.work_summary)) {
            $analysis.work_summary = "Output analyzed, no explicit summary found"
        }
    }
    
    # Truncate work summary if too long
    if ($analysis.work_summary.Length -gt 200) {
        $analysis.work_summary = $analysis.work_summary.Substring(0, 197) + "..."
    }
    
    # 9. Determine exit signal based on confidence
    if ($analysis.confidence_score -ge 40 -or $analysis.has_completion_signal) {
        $analysis.exit_signal = $true
    }
    
    # Write analysis results to file
    $result = @{
        loop_number = $LoopNumber
        timestamp = (Get-Date -Format "o")
        output_file = $OutputFile
        analysis = $analysis
    }
    
    $result | ConvertTo-Json -Depth 10 | Set-Content $script:AnalysisResultFile -Encoding UTF8
    
    return $true
}

function Update-ExitSignals {
    <#
    .SYNOPSIS
        Updates exit signals file based on latest analysis
    .PARAMETER AnalysisFile
        Path to analysis result file
    .PARAMETER ExitSignalsFile
        Path to exit signals file
    #>
    param(
        [string]$AnalysisFile = $script:AnalysisResultFile,
        [string]$ExitSignalsFile = $script:ExitSignalsFile
    )
    
    if (-not (Test-Path $AnalysisFile)) {
        Write-Host "[WARN] Analysis file not found: $AnalysisFile" -ForegroundColor Yellow
        return $false
    }
    
    # Read analysis results
    $analysisData = Get-Content $AnalysisFile -Raw | ConvertFrom-Json
    
    # Read or initialize exit signals
    $signals = @{
        test_only_loops = @()
        done_signals = @()
        completion_indicators = @()
    }
    
    if (Test-Path $ExitSignalsFile) {
        try {
            $existingSignals = Get-Content $ExitSignalsFile -Raw | ConvertFrom-Json
            $signals.test_only_loops = @($existingSignals.test_only_loops)
            $signals.done_signals = @($existingSignals.done_signals)
            $signals.completion_indicators = @($existingSignals.completion_indicators)
        }
        catch {
            # Use defaults on parse error
        }
    }
    
    $loopNumber = $analysisData.loop_number
    $analysis = $analysisData.analysis
    
    # Update test_only_loops
    if ($analysis.is_test_only) {
        $signals.test_only_loops += $loopNumber
    }
    elseif ($analysis.has_progress) {
        # Clear test_only_loops if we made progress
        $signals.test_only_loops = @()
    }
    
    # Update done_signals
    if ($analysis.has_completion_signal) {
        $signals.done_signals += $loopNumber
    }
    
    # Update completion_indicators (strong signals only)
    if ($analysis.confidence_score -ge 60) {
        $signals.completion_indicators += $loopNumber
    }
    
    # Keep only last 5 signals (rolling window)
    if ($signals.test_only_loops.Count -gt 5) {
        $signals.test_only_loops = @($signals.test_only_loops[-5..-1])
    }
    if ($signals.done_signals.Count -gt 5) {
        $signals.done_signals = @($signals.done_signals[-5..-1])
    }
    if ($signals.completion_indicators.Count -gt 5) {
        $signals.completion_indicators = @($signals.completion_indicators[-5..-1])
    }
    
    # Write updated signals
    $signals | ConvertTo-Json -Depth 10 | Set-Content $ExitSignalsFile -Encoding UTF8
    
    return $true
}

function Write-AnalysisSummary {
    <#
    .SYNOPSIS
        Displays analysis results in a formatted view
    .PARAMETER AnalysisFile
        Path to analysis result file
    #>
    param(
        [string]$AnalysisFile = $script:AnalysisResultFile
    )
    
    if (-not (Test-Path $AnalysisFile)) {
        return
    }
    
    try {
        $data = Get-Content $AnalysisFile -Raw | ConvertFrom-Json
    }
    catch {
        Write-Host "[WARN] Could not parse analysis file" -ForegroundColor Yellow
        return
    }
    
    $analysis = $data.analysis
    
    # Determine colors based on state
    $exitColor = if ($analysis.exit_signal) { "Green" } else { "Gray" }
    $stuckColor = if ($analysis.is_stuck) { "Red" } else { "Gray" }
    $testOnlyColor = if ($analysis.is_test_only) { "Yellow" } else { "Gray" }
    
    Write-Host ""
    Write-Host ("=" * 60) -ForegroundColor Cyan
    Write-Host "         Response Analysis - Loop #$($data.loop_number)" -ForegroundColor Cyan
    Write-Host ("=" * 60) -ForegroundColor Cyan
    Write-Host "Exit Signal:      $($analysis.exit_signal)" -ForegroundColor $exitColor
    Write-Host "Confidence:       $($analysis.confidence_score)%" -ForegroundColor White
    Write-Host "Test Only:        $($analysis.is_test_only)" -ForegroundColor $testOnlyColor
    Write-Host "Is Stuck:         $($analysis.is_stuck)" -ForegroundColor $stuckColor
    Write-Host "Files Changed:    $($analysis.files_modified)" -ForegroundColor White
    Write-Host "Error Count:      $($analysis.error_count)" -ForegroundColor White
    Write-Host "Output Length:    $($analysis.output_length) bytes" -ForegroundColor White
    Write-Host "Summary:          $($analysis.work_summary)" -ForegroundColor White
    
    # Task mode info
    if ($analysis.completed_task_id) {
        $taskColor = if ($analysis.task_completed) { "Green" } else { "Yellow" }
        Write-Host ("=" * 60) -ForegroundColor Cyan
        Write-Host "Task ID:          $($analysis.completed_task_id)" -ForegroundColor $taskColor
        Write-Host "Task Completed:   $($analysis.task_completed)" -ForegroundColor $taskColor
        if ($analysis.task_blocked) {
            Write-Host "Task Blocked:     $($analysis.task_blocked)" -ForegroundColor Red
            Write-Host "Blocked Reason:   $($analysis.blocked_reason)" -ForegroundColor Red
        }
    }
    
    Write-Host ("=" * 60) -ForegroundColor Cyan
    Write-Host ""
}

function Test-StuckLoop {
    <#
    .SYNOPSIS
        Detects if Claude is stuck on repeating errors
    .PARAMETER CurrentOutput
        Path to current output file
    .PARAMETER HistoryDir
        Directory containing historical output files
    .OUTPUTS
        Boolean - True if stuck on same error
    #>
    param(
        [Parameter(Mandatory)]
        [string]$CurrentOutput,
        
        [string]$HistoryDir = "logs"
    )
    
    # Get last 3 output files
    $recentOutputs = Get-ChildItem "$HistoryDir\claude_output_*.log" -ErrorAction SilentlyContinue | 
                     Sort-Object LastWriteTime -Descending | 
                     Select-Object -First 3
    
    if ($recentOutputs.Count -lt 3) {
        return $false
    }
    
    # Extract errors from current output
    $currentContent = Get-Content $CurrentOutput -Raw -ErrorAction SilentlyContinue
    if ([string]::IsNullOrEmpty($currentContent)) {
        return $false
    }
    
    $currentErrors = @()
    foreach ($line in ($currentContent -split "`n")) {
        if ($line -match "(?i)(error|failed|exception)") {
            $currentErrors += $line.Trim()
        }
    }
    
    if ($currentErrors.Count -eq 0) {
        return $false
    }
    
    # Check if same errors appear in recent outputs
    $stuckCount = 0
    foreach ($output in $recentOutputs) {
        $outputContent = Get-Content $output.FullName -Raw -ErrorAction SilentlyContinue
        if ([string]::IsNullOrEmpty($outputContent)) {
            continue
        }
        
        $hasMatchingError = $false
        foreach ($error in $currentErrors) {
            $escapedError = [regex]::Escape($error.Substring(0, [Math]::Min(50, $error.Length)))
            if ($outputContent -match $escapedError) {
                $hasMatchingError = $true
                break
            }
        }
        
        if ($hasMatchingError) {
            $stuckCount++
        }
    }
    
    return $stuckCount -ge 3
}

function Get-AnalysisResult {
    <#
    .SYNOPSIS
        Gets the latest analysis result
    .OUTPUTS
        PSObject - Analysis result or null
    #>
    
    if (-not (Test-Path $script:AnalysisResultFile)) {
        return $null
    }
    
    try {
        return Get-Content $script:AnalysisResultFile -Raw | ConvertFrom-Json
    }
    catch {
        return $null
    }
}

function Get-ExitSignals {
    <#
    .SYNOPSIS
        Gets the current exit signals
    .OUTPUTS
        PSObject - Exit signals or null
    #>
    
    if (-not (Test-Path $script:ExitSignalsFile)) {
        return $null
    }
    
    try {
        return Get-Content $script:ExitSignalsFile -Raw | ConvertFrom-Json
    }
    catch {
        return $null
    }
}

# Export functions for dot-sourcing
# When used as module (.psm1), use Export-ModuleMember instead
