<#
.SYNOPSIS
    AI CLI Provider abstraction for hermes-prd
.DESCRIPTION
    Provides unified interface for calling different AI CLI tools
    Supports: claude, droid, aider
#>

# Helper function to write to both console and log
function Write-AIStatus {
    param(
        [ValidateSet("INFO", "WARN", "ERROR", "SUCCESS", "DEBUG")]
        [string]$Level = "INFO",
        [string]$Message
    )
    
    # Try Write-Log first (if Logger.ps1 is loaded)
    if (Get-Command Write-Log -ErrorAction SilentlyContinue) {
        Write-Log -Level $Level -Message $Message
    }
    else {
        # Fallback to Write-Host
        $color = switch ($Level) {
            "INFO"    { "Cyan" }
            "WARN"    { "Yellow" }
            "ERROR"   { "Red" }
            "SUCCESS" { "Green" }
            "DEBUG"   { "DarkGray" }
            default   { "White" }
        }
        Write-Host "[$Level] $Message" -ForegroundColor $color
    }
}

# Configuration
$script:Config = @{
    TimeoutSeconds     = 1200      # 20 minutes
    MaxRetries         = 10
    RetryDelaySeconds  = 10
}

# Stream output display colors
$script:StreamColors = @{
    Init     = "DarkGray"
    Text     = "White"
    Tool     = "Yellow"
    ToolDone = "DarkYellow"
    Result   = "Green"
    Error    = "Red"
    Cost     = "Cyan"
}

function Read-AIStreamOutput {
    <#
    .SYNOPSIS
        Read and display AI output in real-time using stream-json format
    .DESCRIPTION
        Parses stream-json output from Claude or Droid CLI and displays it with colors.
        Supports both Claude and Droid stream-json formats.
        Returns the final result text for further processing.
    .PARAMETER Provider
        AI provider: claude or droid (affects JSON parsing)
    #>
    param(
        [Parameter(Mandatory)]
        [System.Diagnostics.Process]$Process,
        
        [ValidateSet("claude", "droid")]
        [string]$Provider = "claude",
        
        [int]$TimeoutSeconds = 1200,
        
        [int]$CheckIntervalMs = 100
    )
    
    $fullOutput = ""
    $resultText = ""
    $reader = $Process.StandardOutput
    $elapsed = 0
    $timeoutMs = $TimeoutSeconds * 1000
    $currentToolName = ""
    
    Write-Host ""  # New line before output
    
    while (-not $Process.HasExited -or $reader.Peek() -ge 0) {
        # Check timeout
        if ($elapsed -gt $timeoutMs) {
            Write-Host "`n[TIMEOUT] Process exceeded $TimeoutSeconds seconds" -ForegroundColor $script:StreamColors.Error
            return @{ Success = $false; Output = $fullOutput; Result = $resultText; Error = "Timeout" }
        }
        
        # Check for Ctrl+C
        if ([Console]::KeyAvailable) {
            $key = [Console]::ReadKey($true)
            if ($key.Key -eq 'C' -and $key.Modifiers -eq 'Control') {
                Write-Host "`n[CANCELLED] Stopped by user" -ForegroundColor $script:StreamColors.Error
                return @{ Success = $false; Output = $fullOutput; Result = $resultText; Error = "Cancelled" }
            }
        }
        
        # Try to read a line
        if ($reader.Peek() -ge 0) {
            $line = $reader.ReadLine()
            if ($line) {
                $fullOutput += $line + "`n"
                
                try {
                    $json = $line | ConvertFrom-Json -ErrorAction Stop
                    
                    switch ($json.type) {
                        # Common: system init (both Claude and Droid)
                        "system" {
                            if ($json.subtype -eq "init") {
                                Write-Host "[Session] " -ForegroundColor $script:StreamColors.Init -NoNewline
                                Write-Host "Model: $($json.model)" -ForegroundColor $script:StreamColors.Init
                            }
                        }
                        
                        # Claude: assistant message with content array
                        "assistant" {
                            if ($json.message -and $json.message.content) {
                                foreach ($content in $json.message.content) {
                                    if ($content.type -eq "text" -and $content.text) {
                                        Write-Host $content.text -ForegroundColor $script:StreamColors.Text -NoNewline
                                        $resultText += $content.text
                                    }
                                    elseif ($content.type -eq "tool_use") {
                                        $currentToolName = $content.name
                                        $inputPreview = Get-ToolInputPreview -Input $content.input
                                        Write-Host "`n[Tool: $currentToolName]$inputPreview" -ForegroundColor $script:StreamColors.Tool
                                    }
                                }
                            }
                        }
                        
                        # Droid: message with role and text
                        "message" {
                            if ($json.role -eq "assistant" -and $json.text) {
                                Write-Host $json.text -ForegroundColor $script:StreamColors.Text -NoNewline
                                $resultText += $json.text
                            }
                        }
                        
                        # Droid: tool_call
                        "tool_call" {
                            $currentToolName = $json.toolName
                            $inputPreview = Get-ToolInputPreview -Input $json.parameters
                            Write-Host "`n[Tool: $currentToolName]$inputPreview" -ForegroundColor $script:StreamColors.Tool
                        }
                        
                        # Droid: tool_result
                        "tool_result" {
                            if ($currentToolName) {
                                Write-Host "[Done: $currentToolName]" -ForegroundColor $script:StreamColors.ToolDone
                                $currentToolName = ""
                            }
                        }
                        
                        # Claude: user message (tool result)
                        "user" {
                            if ($currentToolName) {
                                Write-Host "[Done: $currentToolName]" -ForegroundColor $script:StreamColors.ToolDone
                                $currentToolName = ""
                            }
                        }
                        
                        # Claude: result
                        "result" {
                            Write-Host "`n" -NoNewline
                            if ($json.subtype -eq "success") {
                                $duration = [math]::Round($json.duration_ms / 1000, 1)
                                $cost = if ($json.total_cost_usd) { [math]::Round($json.total_cost_usd, 4) } else { 0 }
                                Write-Host "[Complete] " -ForegroundColor $script:StreamColors.Result -NoNewline
                                Write-Host "Duration: ${duration}s | Cost: `$$cost" -ForegroundColor $script:StreamColors.Cost
                                
                                if ($json.result) {
                                    $resultText = $json.result
                                }
                            }
                            else {
                                Write-Host "[Error] $($json.subtype)" -ForegroundColor $script:StreamColors.Error
                            }
                        }
                        
                        # Droid: completion
                        "completion" {
                            Write-Host "`n" -NoNewline
                            $duration = [math]::Round($json.durationMs / 1000, 1)
                            Write-Host "[Complete] " -ForegroundColor $script:StreamColors.Result -NoNewline
                            Write-Host "Duration: ${duration}s | Turns: $($json.numTurns)" -ForegroundColor $script:StreamColors.Cost
                            
                            if ($json.finalText) {
                                $resultText = $json.finalText
                            }
                        }
                    }
                }
                catch {
                    # Not valid JSON, show as raw output
                    Write-Host $line -ForegroundColor $script:StreamColors.Text
                    $resultText += $line + "`n"
                }
            }
        }
        else {
            Start-Sleep -Milliseconds $CheckIntervalMs
            $elapsed += $CheckIntervalMs
        }
    }
    
    Write-Host ""  # Final newline
    
    return @{
        Success = $true
        Output = $fullOutput
        Result = $resultText
        Error = $null
    }
}

function Get-ToolInputPreview {
    <#
    .SYNOPSIS
        Extract a preview string from tool input parameters
    #>
    param($Input)
    
    if (-not $Input) { return "" }
    
    if ($Input.file_path) {
        return " $($Input.file_path)"
    }
    elseif ($Input.command) {
        $cmd = $Input.command
        if ($cmd.Length -gt 50) { $cmd = $cmd.Substring(0, 47) + "..." }
        return " $cmd"
    }
    elseif ($Input.pattern) {
        return " '$($Input.pattern)'"
    }
    elseif ($Input.path) {
        return " $($Input.path)"
    }
    
    return ""
}

$script:SizeThresholds = @{
    WarningSize = 100000    # 100K chars - warning
    LargeSize   = 200000    # 200K chars - serious warning
    MaxSize     = 500000    # 500K chars - very large warning
}

# Supported AI providers
$script:Providers = @{
    claude = @{
        Command     = "claude"
        CheckArgs   = "--version"
        Description = "Claude Code CLI"
    }
    droid  = @{
        Command     = "droid"
        CheckArgs   = "--version"
        Description = "Factory Droid CLI"
    }
    aider  = @{
        Command     = "aider"
        CheckArgs   = "--version"
        Description = "Aider AI CLI"
    }
}

function Test-AIProvider {
    <#
    .SYNOPSIS
        Check if an AI provider is available
    #>
    param(
        [Parameter(Mandatory)]
        [ValidateSet("claude", "droid", "aider")]
        [string]$Provider
    )
    
    $providerInfo = $script:Providers[$Provider]
    $command = $providerInfo.Command
    
    try {
        $null = Get-Command $command -ErrorAction Stop
        return $true
    }
    catch {
        return $false
    }
}

function Get-AvailableProviders {
    <#
    .SYNOPSIS
        Get list of available AI providers
    #>
    $available = @()
    
    foreach ($provider in $script:Providers.Keys) {
        if (Test-AIProvider -Provider $provider) {
            $available += @{
                Name        = $provider
                Description = $script:Providers[$provider].Description
                Command     = $script:Providers[$provider].Command
            }
        }
    }
    
    return $available
}

function Get-AutoProvider {
    <#
    .SYNOPSIS
        Get first available provider (priority: claude > droid > aider)
    #>
    $priority = @("claude", "droid", "aider")
    
    foreach ($provider in $priority) {
        if (Test-AIProvider -Provider $provider) {
            return $provider
        }
    }
    
    return $null
}

function Test-PrdSize {
    <#
    .SYNOPSIS
        Check PRD size and warn if large
    #>
    param(
        [Parameter(Mandatory)]
        [string]$PrdFile
    )
    
    $content = Get-Content $PrdFile -Raw
    $size = $content.Length
    $lineCount = ($content -split "`n").Count
    
    Write-AIStatus -Level "INFO" -Message "PRD size: $size characters, $lineCount lines"
    
    if ($size -gt $script:SizeThresholds.MaxSize) {
        Write-AIStatus -Level "WARN" -Message "PRD is very large ($size chars). This may take 15-20 minutes."
        Write-AIStatus -Level "WARN" -Message "Consider breaking PRD into smaller feature documents."
        Write-AIStatus -Level "INFO" -Message "Recommendations: Split by feature/module, run hermes-prd separately, use -Timeout 1800"
    }
    elseif ($size -gt $script:SizeThresholds.LargeSize) {
        Write-AIStatus -Level "WARN" -Message "PRD is large ($size chars). This may take 10-15 minutes."
    }
    elseif ($size -gt $script:SizeThresholds.WarningSize) {
        Write-AIStatus -Level "INFO" -Message "PRD is medium size. Processing may take 5-10 minutes."
    }
    
    return @{
        Size    = $size
        Lines   = $lineCount
        Content = $content
        IsLarge = $size -gt $script:SizeThresholds.LargeSize
    }
}

function Invoke-AICommand {
    <#
    .SYNOPSIS
        Execute AI CLI command with content
    #>
    param(
        [Parameter(Mandatory)]
        [ValidateSet("claude", "droid", "aider")]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptText,
        
        [Parameter(Mandatory)]
        [string]$Content,
        
        [string]$InputFile
    )
    
    switch ($Provider) {
        "claude" {
            $result = $Content | claude -p --dangerously-skip-permissions $PromptText 2>&1
        }
        "droid" {
            $result = $Content | droid exec --skip-permissions-unsafe $PromptText 2>&1
        }
        "aider" {
            if (-not $InputFile) {
                throw "Aider requires InputFile parameter"
            }
            $result = aider --yes --no-auto-commits --message $PromptText $InputFile 2>&1
        }
    }
    
    return $result
}

function Wait-ProcessWithCtrlC {
    <#
    .SYNOPSIS
        Wait for process with Ctrl+C support
    #>
    param(
        [Parameter(Mandatory)]
        [System.Diagnostics.Process]$Process,
        
        [int]$TimeoutSeconds = 1200,
        
        [int]$CheckIntervalMs = 500
    )
    
    $elapsed = 0
    $timeoutMs = $TimeoutSeconds * 1000
    
    while (-not $Process.HasExited -and $elapsed -lt $timeoutMs) {
        Start-Sleep -Milliseconds $CheckIntervalMs
        $elapsed += $CheckIntervalMs
        
        # Check for Ctrl+C (KeyAvailable doesn't work in all scenarios, but process check does)
        if ([Console]::KeyAvailable) {
            $key = [Console]::ReadKey($true)
            if ($key.Key -eq 'C' -and $key.Modifiers -eq 'Control') {
                Write-Host "`n[WARN] Ctrl+C detected, stopping AI process..." -ForegroundColor Yellow
                $Process.Kill()
                throw "Cancelled by user (Ctrl+C)"
            }
        }
    }
    
    if (-not $Process.HasExited) {
        return $false  # Timeout
    }
    
    return $true  # Completed
}

function Invoke-AIWithTimeout {
    <#
    .SYNOPSIS
        Execute AI command with timeout
    .PARAMETER StreamOutput
        Show real-time output (claude only, requires --verbose)
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptText,
        
        [AllowEmptyString()]
        [string]$Content = "",
        
        [string]$InputFile,
        
        [int]$TimeoutSeconds = 1200,
        
        [switch]$StreamOutput
    )
    
    $result = $null
    $tempPromptFile = $null
    $process = $null
    $startTime = Get-Date
    
    Write-AIStatus -Level "DEBUG" -Message "Starting $Provider execution at $($startTime.ToString('HH:mm:ss'))..."
    Write-AIStatus -Level "DEBUG" -Message "Timeout: $TimeoutSeconds seconds"
    Write-AIStatus -Level "DEBUG" -Message "Prompt length: $($PromptText.Length) chars"
    Write-AIStatus -Level "INFO" -Message "Press Ctrl+C to cancel..."
    
    try {
        switch ($Provider) {
            "claude" {
                # Claude CLI usage (from https://code.claude.com/docs/en/quickstart):
                # - Interactive: claude (starts REPL)
                # - One-shot: claude "task" 
                # - Headless: claude -p "prompt" --output-format text
                # - Streaming: claude -p "prompt" --output-format stream-json --verbose
                # - Piped: cat content | claude -p "analyze this"
                
                # Strategy: Use stdin for content, -p for prompt instruction
                # If no content, just use -p with prompt
                # If content exists, pipe it via stdin with a processing instruction
                
                $stdinContent = $null
                $promptArg = $PromptText
                
                if ($Content) {
                    # When we have content, pipe it via stdin and adjust prompt
                    $stdinContent = $Content
                    # Prompt tells Claude what to do with the piped content
                    $promptArg = "Process the following input according to these instructions:`n`n$PromptText"
                }
                
                Write-AIStatus -Level "DEBUG" -Message "Prompt length: $($promptArg.Length) chars"
                if ($stdinContent) {
                    Write-AIStatus -Level "DEBUG" -Message "Content length: $($stdinContent.Length) chars (via stdin)"
                }
                if ($StreamOutput) {
                    Write-AIStatus -Level "DEBUG" -Message "Stream output: enabled"
                }
                
                # Escape prompt for command line - use temp file for long/complex prompts
                $tempPromptFile = Join-Path $env:TEMP "hermes-claude-prompt-$(Get-Random).txt"
                $promptArg | Set-Content -Path $tempPromptFile -Encoding UTF8
                $escapedPrompt = (Get-Content $tempPromptFile -Raw) -replace '"', '\"'
                
                # Build arguments based on stream mode
                $outputFormat = if ($StreamOutput) { "stream-json" } else { "text" }
                $verboseFlag = if ($StreamOutput) { " --verbose" } else { "" }
                
                # Use Start-Process with timeout for claude
                $pinfo = New-Object System.Diagnostics.ProcessStartInfo
                $pinfo.FileName = "claude"
                $pinfo.Arguments = "-p `"$escapedPrompt`" --dangerously-skip-permissions --output-format $outputFormat$verboseFlag"
                $pinfo.RedirectStandardOutput = $true
                $pinfo.RedirectStandardError = $true
                $pinfo.RedirectStandardInput = $true
                $pinfo.UseShellExecute = $false
                $pinfo.CreateNoWindow = $true
                
                Write-AIStatus -Level "DEBUG" -Message "Starting claude process..."
                $process = New-Object System.Diagnostics.Process
                $process.StartInfo = $pinfo
                $process.Start() | Out-Null
                
                # Write content to stdin if we have it (like: cat content | claude -p "prompt")
                if ($stdinContent) {
                    $process.StandardInput.Write($stdinContent)
                }
                $process.StandardInput.Close()
                
                if ($StreamOutput) {
                    # Use streaming reader for real-time output
                    Write-AIStatus -Level "INFO" -Message "Streaming output..."
                    $streamResult = Read-AIStreamOutput -Process $process -Provider "claude" -TimeoutSeconds $TimeoutSeconds
                    
                    if (-not $streamResult.Success) {
                        Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
                        throw "AI execution failed: $($streamResult.Error)"
                    }
                    
                    $result = $streamResult.Result
                    $stderr = $process.StandardError.ReadToEnd()
                }
                else {
                    # Traditional wait and read
                    Write-AIStatus -Level "DEBUG" -Message "Waiting for claude process (timeout: $TimeoutSeconds s)..."
                    $exited = Wait-ProcessWithCtrlC -Process $process -TimeoutSeconds $TimeoutSeconds
                    if (-not $exited) {
                        Write-AIStatus -Level "ERROR" -Message "Process timed out!"
                        $process.Kill()
                        Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
                        throw "AI timeout after $TimeoutSeconds seconds"
                    }
                    
                    $result = $process.StandardOutput.ReadToEnd()
                    $stderr = $process.StandardError.ReadToEnd()
                }
                
                Write-AIStatus -Level "DEBUG" -Message "Process exited with code: $($process.ExitCode)"
                if ($stderr) {
                    Write-AIStatus -Level "WARN" -Message "Claude stderr: $stderr"
                }
                
                # Cleanup temp file
                Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
            }
            "droid" {
                # Write prompt to temp file and call droid directly
                $tempPromptFile = Join-Path $env:TEMP "hermes-prompt-$(Get-Random).md"
                $PromptText | Set-Content -Path $tempPromptFile -Encoding UTF8
                Write-AIStatus -Level "DEBUG" -Message "Prompt written to: $tempPromptFile"
                if ($StreamOutput) {
                    Write-AIStatus -Level "DEBUG" -Message "Stream output: enabled"
                }
                
                # Build arguments based on stream mode
                $outputFormat = if ($StreamOutput) { " --output-format stream-json" } else { "" }
                
                # Use Start-Process with timeout for droid
                $pinfo = New-Object System.Diagnostics.ProcessStartInfo
                $pinfo.FileName = "droid"
                $pinfo.Arguments = "exec --auto medium --file `"$tempPromptFile`"$outputFormat"
                $pinfo.RedirectStandardOutput = $true
                $pinfo.RedirectStandardError = $true
                $pinfo.UseShellExecute = $false
                $pinfo.CreateNoWindow = $true
                
                Write-AIStatus -Level "DEBUG" -Message "Starting droid process..."
                $process = New-Object System.Diagnostics.Process
                $process.StartInfo = $pinfo
                $process.Start() | Out-Null
                
                if ($StreamOutput) {
                    # Use streaming reader for real-time output
                    Write-AIStatus -Level "INFO" -Message "Streaming output..."
                    $streamResult = Read-AIStreamOutput -Process $process -Provider "droid" -TimeoutSeconds $TimeoutSeconds
                    
                    if (-not $streamResult.Success) {
                        Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
                        throw "AI execution failed: $($streamResult.Error)"
                    }
                    
                    $result = $streamResult.Result
                    $stderr = $process.StandardError.ReadToEnd()
                }
                else {
                    # Traditional wait and read
                    Write-AIStatus -Level "DEBUG" -Message "Waiting for droid process (timeout: $TimeoutSeconds s)..."
                    $exited = Wait-ProcessWithCtrlC -Process $process -TimeoutSeconds $TimeoutSeconds
                    if (-not $exited) {
                        Write-AIStatus -Level "ERROR" -Message "Process timed out!"
                        $process.Kill()
                        Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
                        throw "AI timeout after $TimeoutSeconds seconds"
                    }
                    
                    $result = $process.StandardOutput.ReadToEnd()
                    $stderr = $process.StandardError.ReadToEnd()
                }
                
                Write-AIStatus -Level "DEBUG" -Message "Process exited with code: $($process.ExitCode)"
                if ($stderr) {
                    Write-AIStatus -Level "WARN" -Message "Droid stderr: $stderr"
                }
                
                # Cleanup temp file
                Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
            }
            "aider" {
                if (-not $InputFile) {
                    throw "Aider requires InputFile parameter"
                }
                
                # Use Start-Process with timeout for aider
                $pinfo = New-Object System.Diagnostics.ProcessStartInfo
                $pinfo.FileName = "aider"
                $pinfo.Arguments = "--yes --no-auto-commits --message `"$PromptText`" `"$InputFile`""
                $pinfo.RedirectStandardOutput = $true
                $pinfo.RedirectStandardError = $true
                $pinfo.UseShellExecute = $false
                $pinfo.CreateNoWindow = $true
                
                Write-AIStatus -Level "DEBUG" -Message "Starting aider process..."
                $process = New-Object System.Diagnostics.Process
                $process.StartInfo = $pinfo
                $process.Start() | Out-Null
                
                Write-AIStatus -Level "DEBUG" -Message "Waiting for aider process (timeout: $TimeoutSeconds s)..."
                $exited = Wait-ProcessWithCtrlC -Process $process -TimeoutSeconds $TimeoutSeconds
                if (-not $exited) {
                    Write-AIStatus -Level "ERROR" -Message "Process timed out!"
                    $process.Kill()
                    throw "AI timeout after $TimeoutSeconds seconds"
                }
                
                $result = $process.StandardOutput.ReadToEnd()
                $stderr = $process.StandardError.ReadToEnd()
                Write-AIStatus -Level "DEBUG" -Message "Process exited with code: $($process.ExitCode)"
                if ($stderr) {
                    Write-AIStatus -Level "WARN" -Message "Aider stderr: $stderr"
                }
            }
        }
        
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        Write-AIStatus -Level "DEBUG" -Message "$Provider completed in $([Math]::Round($duration, 1)) seconds"
    }
    finally {
        # Kill process if still running (e.g., on Ctrl+C)
        if ($process -and -not $process.HasExited) {
            try {
                $process.Kill()
                Write-AIStatus -Level "DEBUG" -Message "Process killed during cleanup"
            } catch { }
        }
        
        # Cleanup temp prompt file
        if ($tempPromptFile -and (Test-Path $tempPromptFile)) {
            Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
        }
    }
    
    # Ensure result is a string (can return array)
    if ($result -is [array]) {
        $result = $result -join "`n"
    }
    if (-not $result) {
        $result = ""
    }
    
    return $result
}

function Split-AIOutput {
    <#
    .SYNOPSIS
        Parse AI output into separate files using FILE markers
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Output
    )
    
    # Debug: log output length
    Write-Verbose "Split-AIOutput: Input length = $($Output.Length)"
    
    $files = @()
    $pattern = '### FILE:\s*(.+\.md)'
    
    # Check if pattern exists
    if ($Output -notmatch $pattern) {
        Write-Verbose "Split-AIOutput: No FILE markers found in output"
        # Log first 500 chars for debugging
        $preview = if ($Output.Length -gt 500) { $Output.Substring(0, 500) + "..." } else { $Output }
        Write-Verbose "Output preview: $preview"
        return @()
    }
    
    $segments = $Output -split $pattern
    
    for ($i = 1; $i -lt $segments.Count; $i += 2) {
        $fileName = $segments[$i].Trim()
        $content = if ($i + 1 -lt $segments.Count) { $segments[$i + 1].Trim() } else { "" }
        
        if ($fileName -and $content) {
            $files += @{
                FileName = $fileName
                Content  = $content
            }
        }
    }
    
    return $files
}

function Test-ParsedOutput {
    <#
    .SYNOPSIS
        Validate parsed output has required structure
    #>
    param(
        [Parameter(Mandatory, ValueFromPipeline)]
        [AllowEmptyCollection()]
        [array]$Files
    )
    
    if ($null -eq $Files -or $Files.Count -eq 0) {
        Write-AIStatus -Level "WARN" -Message "No files parsed from output"
        return $false
    }
    
    $hasStatusFile = $false
    $hasFeatureFile = $false
    
    foreach ($file in $Files) {
        if ($file.FileName -match "tasks-status\.md") {
            $hasStatusFile = $true
        }
        if ($file.FileName -match "\d{3}-.*\.md$") {
            $hasFeatureFile = $true
        }
        
        # Check for required fields
        if ($file.Content -notmatch "Feature ID:" -and $file.FileName -notmatch "status") {
            Write-AIStatus -Level "WARN" -Message "File $($file.FileName) missing Feature ID"
        }
    }
    
    if (-not $hasFeatureFile) {
        Write-AIStatus -Level "WARN" -Message "No feature files found (expected 001-xxx.md format)"
        return $false
    }
    
    return $true
}

function Invoke-AIWithRetry {
    <#
    .SYNOPSIS
        Execute AI command with retry logic
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptText,
        
        [AllowEmptyString()]
        [string]$Content = "",
        
        [string]$InputFile,
        
        [int]$MaxRetries = 10,
        
        [int]$TimeoutSeconds = 1200
    )
    
    $retryDelay = $script:Config.RetryDelaySeconds
    
    for ($attempt = 1; $attempt -le $MaxRetries; $attempt++) {
        try {
            Write-AIStatus -Level "INFO" -Message "Attempt $attempt/$MaxRetries..."
            
            $result = Invoke-AIWithTimeout -Provider $Provider `
                -PromptText $PromptText -Content $Content `
                -InputFile $InputFile -TimeoutSeconds $TimeoutSeconds
            
            # Debug: log result length and preview
            $resultLen = if ($result) { $result.Length } else { 0 }
            Write-AIStatus -Level "DEBUG" -Message "AI output length: $resultLen chars"
            if ($resultLen -gt 0 -and $resultLen -lt 2000) {
                # Log short outputs for debugging
                Write-AIStatus -Level "DEBUG" -Message "Output preview: $($result.Substring(0, [Math]::Min(500, $resultLen)))"
            }
            
            # Validate output
            $files = Split-AIOutput -Output $result
            
            if ($files.Count -eq 0) {
                throw "No files parsed from AI output"
            }
            
            $isValid = Test-ParsedOutput -Files $files
            
            if (-not $isValid) {
                throw "Invalid output format"
            }
            
            Write-AIStatus -Level "SUCCESS" -Message "AI completed successfully"
            return @{
                Success  = $true
                Files    = $files
                Attempts = $attempt
                Raw      = $result
            }
        }
        catch {
            Write-AIStatus -Level "WARN" -Message "Attempt $attempt failed: $_"
            
            if ($attempt -lt $MaxRetries) {
                Write-AIStatus -Level "INFO" -Message "Retrying in $retryDelay seconds..."
                Start-Sleep -Seconds $retryDelay
            }
            else {
                Write-Error "All $MaxRetries attempts failed"
                return @{
                    Success  = $false
                    Error    = $_.Exception.Message
                    Attempts = $attempt
                }
            }
        }
    }
}

function Invoke-TaskExecution {
    <#
    .SYNOPSIS
        Execute AI for task mode (simpler than PRD parsing)
    .DESCRIPTION
        Executes the specified AI provider with prompt content for task execution.
        Returns output without parsing/validation (task mode handles its own analysis).
    .PARAMETER StreamOutput
        Show real-time output (claude only)
    #>
    param(
        [Parameter(Mandatory)]
        [ValidateSet("claude", "droid", "aider")]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptContent,
        
        [int]$TimeoutSeconds = 900,
        
        [switch]$StreamOutput
    )
    
    $result = $null
    $tempPromptFile = $null
    
    try {
        # Write prompt content to temp file
        $tempPromptFile = Join-Path $env:TEMP "hermes-task-$(Get-Random).md"
        $PromptContent | Set-Content -Path $tempPromptFile -Encoding UTF8
        
        $pinfo = New-Object System.Diagnostics.ProcessStartInfo
        $pinfo.RedirectStandardOutput = $true
        $pinfo.RedirectStandardError = $true
        $pinfo.RedirectStandardInput = $true
        $pinfo.UseShellExecute = $false
        $pinfo.CreateNoWindow = $true
        
        switch ($Provider) {
            "claude" {
                $pinfo.FileName = "claude"
                # Read prompt from file and escape for command line
                $promptContent = Get-Content $tempPromptFile -Raw
                $escapedPrompt = $promptContent -replace '"', '\"'
                
                if ($StreamOutput) {
                    $pinfo.Arguments = "-p `"$escapedPrompt`" --dangerously-skip-permissions --output-format stream-json --verbose"
                }
                else {
                    $pinfo.Arguments = "-p `"$escapedPrompt`" --dangerously-skip-permissions --output-format text"
                }
            }
            "droid" {
                $pinfo.FileName = "droid"
                $outputFormat = if ($StreamOutput) { " --output-format stream-json" } else { "" }
                $pinfo.Arguments = "exec --skip-permissions-unsafe --file `"$tempPromptFile`"$outputFormat"
            }
            "aider" {
                $pinfo.FileName = "aider"
                $pinfo.Arguments = "--yes --no-auto-commits --message `"Execute the task described in this file`" `"$tempPromptFile`""
            }
        }
        
        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $pinfo
        $process.Start() | Out-Null
        $process.StandardInput.Close()
        
        if (($Provider -eq "claude" -or $Provider -eq "droid") -and $StreamOutput) {
            # Use streaming reader for real-time output
            $streamResult = Read-AIStreamOutput -Process $process -Provider $Provider -TimeoutSeconds $TimeoutSeconds
            
            if (-not $streamResult.Success) {
                return @{
                    Success = $false
                    Error   = $streamResult.Error
                    Output  = $streamResult.Output
                }
            }
            
            $result = $streamResult.Result
            $stderr = $process.StandardError.ReadToEnd()
        }
        else {
            # Traditional wait and read
            $exited = $process.WaitForExit($TimeoutSeconds * 1000)
            
            if (-not $exited) {
                $process.Kill()
                return @{
                    Success = $false
                    Error   = "Timeout after $TimeoutSeconds seconds"
                    Output  = $null
                }
            }
            
            $result = $process.StandardOutput.ReadToEnd()
            $stderr = $process.StandardError.ReadToEnd()
        }
        
        if ($stderr) {
            Write-AIStatus -Level "WARN" -Message "$Provider stderr: $stderr"
        }
        
        return @{
            Success = $true
            Output  = $result
            Error   = $null
        }
    }
    finally {
        if ($tempPromptFile -and (Test-Path $tempPromptFile)) {
            Remove-Item $tempPromptFile -Force -ErrorAction SilentlyContinue
        }
    }
}

function Write-AIProviderList {
    <#
    .SYNOPSIS
        Display available AI providers
    #>
    $providers = Get-AvailableProviders
    
    Write-Host ""
    Write-Host "Available AI Providers:" -ForegroundColor Cyan
    Write-Host ""
    
    if ($providers.Count -eq 0) {
        Write-Warning "No AI providers found!"
        Write-Host ""
        Write-Host "Install one of the following:" -ForegroundColor Yellow
        Write-Host "  - claude  : npm install -g @anthropic-ai/claude-code"
        Write-Host "  - droid   : npm install -g @anthropic-ai/droid"
        Write-Host "  - aider   : pip install aider-chat"
        return
    }
    
    foreach ($p in $providers) {
        Write-Host "  [OK] $($p.Name)" -ForegroundColor Green -NoNewline
        Write-Host " - $($p.Description)" -ForegroundColor Gray
    }
    
    Write-Host ""
    Write-Host "Default (auto): " -NoNewline
    $auto = Get-AutoProvider
    if ($auto) {
        Write-Host $auto -ForegroundColor Green
    }
    else {
        Write-Host "none" -ForegroundColor Red
    }
    Write-Host ""
}

# Export functions when loaded as module
if ($MyInvocation.ScriptName -match '\.psm1$') {
    Export-ModuleMember -Function @(
        'Test-AIProvider',
        'Get-AvailableProviders',
        'Get-AutoProvider',
        'Test-PrdSize',
        'Invoke-AICommand',
        'Invoke-AIWithTimeout',
        'Invoke-TaskExecution',
        'Split-AIOutput',
        'Test-ParsedOutput',
        'Invoke-AIWithRetry',
        'Write-AIProviderList',
        'Read-AIStreamOutput',
        'Write-AIStatus'
    )
}
