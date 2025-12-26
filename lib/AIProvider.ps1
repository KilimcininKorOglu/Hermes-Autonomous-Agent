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
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptText,
        
        [AllowEmptyString()]
        [string]$Content = "",
        
        [string]$InputFile,
        
        [int]$TimeoutSeconds = 1200
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
                # Write combined prompt+content to temp file for claude
                $tempInputFile = Join-Path $env:TEMP "hermes-claude-input-$(Get-Random).md"
                
                # Combine prompt and content into single input
                $fullInput = $PromptText
                if ($Content) {
                    $fullInput = $Content + "`n`n" + $PromptText
                }
                $fullInput | Set-Content -Path $tempInputFile -Encoding UTF8
                Write-AIStatus -Level "DEBUG" -Message "Input written to: $tempInputFile ($(($fullInput).Length) chars)"
                
                # Use Start-Process with timeout for claude
                # Claude CLI: cat file | claude -p "prompt" OR claude -p "prompt" with stdin
                $pinfo = New-Object System.Diagnostics.ProcessStartInfo
                $pinfo.FileName = "claude"
                $pinfo.Arguments = "-p --dangerously-skip-permissions"
                $pinfo.RedirectStandardOutput = $true
                $pinfo.RedirectStandardError = $true
                $pinfo.RedirectStandardInput = $true
                $pinfo.UseShellExecute = $false
                $pinfo.CreateNoWindow = $true
                
                Write-AIStatus -Level "DEBUG" -Message "Starting claude process..."
                $process = New-Object System.Diagnostics.Process
                $process.StartInfo = $pinfo
                $process.Start() | Out-Null
                
                # Write full input to stdin
                $process.StandardInput.Write($fullInput)
                $process.StandardInput.Close()
                
                Write-AIStatus -Level "DEBUG" -Message "Waiting for claude process (timeout: $TimeoutSeconds s)..."
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
                    Write-AIStatus -Level "WARN" -Message "Claude stderr: $stderr"
                }
                
                # Cleanup temp file
                Remove-Item $tempInputFile -Force -ErrorAction SilentlyContinue
            }
            "droid" {
                # Write prompt to temp file and call droid directly
                $tempPromptFile = Join-Path $env:TEMP "hermes-prompt-$(Get-Random).md"
                $PromptText | Set-Content -Path $tempPromptFile -Encoding UTF8
                Write-AIStatus -Level "DEBUG" -Message "Prompt written to: $tempPromptFile"
                
                # Use Start-Process with timeout for droid
                $pinfo = New-Object System.Diagnostics.ProcessStartInfo
                $pinfo.FileName = "droid"
                $pinfo.Arguments = "exec --auto medium --file `"$tempPromptFile`""
                $pinfo.RedirectStandardOutput = $true
                $pinfo.RedirectStandardError = $true
                $pinfo.UseShellExecute = $false
                $pinfo.CreateNoWindow = $true
                
                Write-AIStatus -Level "DEBUG" -Message "Starting droid process..."
                $process = New-Object System.Diagnostics.Process
                $process.StartInfo = $pinfo
                $process.Start() | Out-Null
                
                Write-AIStatus -Level "DEBUG" -Message "Waiting for droid process (timeout: $TimeoutSeconds s)..."
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
                    Write-AIStatus -Level "WARN" -Message "Droid stderr: $stderr"
                }
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
    #>
    param(
        [Parameter(Mandatory)]
        [ValidateSet("claude", "droid", "aider")]
        [string]$Provider,
        
        [Parameter(Mandatory)]
        [string]$PromptContent,
        
        [int]$TimeoutSeconds = 900
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
        $pinfo.UseShellExecute = $false
        $pinfo.CreateNoWindow = $true
        
        switch ($Provider) {
            "claude" {
                $pinfo.FileName = "claude"
                $pinfo.Arguments = "--dangerously-skip-permissions `"$tempPromptFile`""
            }
            "droid" {
                $pinfo.FileName = "droid"
                $pinfo.Arguments = "exec --skip-permissions-unsafe --file `"$tempPromptFile`""
            }
            "aider" {
                $pinfo.FileName = "aider"
                $pinfo.Arguments = "--yes --no-auto-commits --message `"Execute the task described in this file`" `"$tempPromptFile`""
            }
        }
        
        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $pinfo
        $process.Start() | Out-Null
        
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
        'Write-AIProviderList'
    )
}
