<#
.SYNOPSIS
    AI CLI Provider abstraction for hermes-prd
.DESCRIPTION
    Provides unified interface for calling different AI CLI tools
    Supports: claude, droid, aider
#>

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
    
    Write-Host "[INFO] PRD size: $size characters, $lineCount lines" -ForegroundColor Cyan
    
    if ($size -gt $script:SizeThresholds.MaxSize) {
        Write-Warning "PRD is very large ($size chars). This may take 15-20 minutes."
        Write-Warning "Consider breaking PRD into smaller feature documents."
        Write-Host ""
        Write-Host "Recommendations:" -ForegroundColor Yellow
        Write-Host "  - Split by feature/module into separate files"
        Write-Host "  - Run hermes-prd on each file separately"
        Write-Host "  - Use -Timeout 1800 for extra time"
        Write-Host ""
    }
    elseif ($size -gt $script:SizeThresholds.LargeSize) {
        Write-Warning "PRD is large ($size chars). This may take 10-15 minutes."
    }
    elseif ($size -gt $script:SizeThresholds.WarningSize) {
        Write-Host "[INFO] PRD is medium size. Processing may take 5-10 minutes." -ForegroundColor Yellow
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
    $startTime = Get-Date
    
    Write-Host "[DEBUG] Starting $Provider execution at $($startTime.ToString('HH:mm:ss'))..." -ForegroundColor DarkGray
    Write-Host "[DEBUG] Timeout: $TimeoutSeconds seconds" -ForegroundColor DarkGray
    Write-Host "[DEBUG] Prompt length: $($PromptText.Length) chars" -ForegroundColor DarkGray
    
    try {
        switch ($Provider) {
            "claude" {
                Write-Host "[DEBUG] Creating claude job..." -ForegroundColor DarkGray
                # Use job for claude (may need stdin)
                $job = Start-Job -ScriptBlock {
                    param($content, $prompt)
                    $content | claude -p --dangerously-skip-permissions $prompt
                } -ArgumentList $Content, $PromptText
                
                Write-Host "[DEBUG] Waiting for claude job (timeout: $TimeoutSeconds s)..." -ForegroundColor DarkGray
                $completed = Wait-Job $job -Timeout $TimeoutSeconds
                if (-not $completed) {
                    Write-Host "[DEBUG] Job timed out after $TimeoutSeconds seconds!" -ForegroundColor Red
                    Stop-Job $job -ErrorAction SilentlyContinue
                    Remove-Job $job -Force -ErrorAction SilentlyContinue
                    throw "AI timeout after $TimeoutSeconds seconds"
                }
                Write-Host "[DEBUG] Job completed, receiving output..." -ForegroundColor DarkGray
                $result = Receive-Job $job
                Remove-Job $job -Force
            }
            "droid" {
                # Write prompt to temp file and call droid directly
                $tempPromptFile = Join-Path $env:TEMP "hermes-prompt-$(Get-Random).md"
                $PromptText | Set-Content -Path $tempPromptFile -Encoding UTF8
                Write-Host "[DEBUG] Prompt written to: $tempPromptFile" -ForegroundColor DarkGray
                
                # Use Start-Process with timeout for droid
                $pinfo = New-Object System.Diagnostics.ProcessStartInfo
                $pinfo.FileName = "droid"
                $pinfo.Arguments = "exec --auto medium --file `"$tempPromptFile`""
                $pinfo.RedirectStandardOutput = $true
                $pinfo.RedirectStandardError = $true
                $pinfo.UseShellExecute = $false
                $pinfo.CreateNoWindow = $true
                
                Write-Host "[DEBUG] Starting droid process..." -ForegroundColor DarkGray
                $process = New-Object System.Diagnostics.Process
                $process.StartInfo = $pinfo
                $process.Start() | Out-Null
                
                Write-Host "[DEBUG] Waiting for droid process (timeout: $TimeoutSeconds s)..." -ForegroundColor DarkGray
                $exited = $process.WaitForExit($TimeoutSeconds * 1000)
                if (-not $exited) {
                    Write-Host "[DEBUG] Process timed out!" -ForegroundColor Red
                    $process.Kill()
                    throw "AI timeout after $TimeoutSeconds seconds"
                }
                
                $result = $process.StandardOutput.ReadToEnd()
                $stderr = $process.StandardError.ReadToEnd()
                Write-Host "[DEBUG] Process exited with code: $($process.ExitCode)" -ForegroundColor DarkGray
                if ($stderr) {
                    Write-Warning "Droid stderr: $stderr"
                }
            }
            "aider" {
                Write-Host "[DEBUG] Creating aider job..." -ForegroundColor DarkGray
                $job = Start-Job -ScriptBlock {
                    param($prompt, $inputFile)
                    aider --yes --no-auto-commits --message $prompt $inputFile
                } -ArgumentList $PromptText, $InputFile
                
                Write-Host "[DEBUG] Waiting for aider job (timeout: $TimeoutSeconds s)..." -ForegroundColor DarkGray
                $completed = Wait-Job $job -Timeout $TimeoutSeconds
                if (-not $completed) {
                    Write-Host "[DEBUG] Job timed out!" -ForegroundColor Red
                    Stop-Job $job -ErrorAction SilentlyContinue
                    Remove-Job $job -Force -ErrorAction SilentlyContinue
                    throw "AI timeout after $TimeoutSeconds seconds"
                }
                $result = Receive-Job $job
                Remove-Job $job -Force
            }
        }
        
        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds
        Write-Host "[DEBUG] $Provider completed in $([Math]::Round($duration, 1)) seconds" -ForegroundColor DarkGray
    }
    finally {
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
        Write-Warning "No files parsed from output"
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
            Write-Warning "File $($file.FileName) missing Feature ID"
        }
    }
    
    if (-not $hasFeatureFile) {
        Write-Warning "No feature files found (expected 001-xxx.md format)"
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
            Write-Host "[INFO] Attempt $attempt/$MaxRetries..." -ForegroundColor Cyan
            
            $result = Invoke-AIWithTimeout -Provider $Provider `
                -PromptText $PromptText -Content $Content `
                -InputFile $InputFile -TimeoutSeconds $TimeoutSeconds
            
            # Debug: log result length and preview
            $resultLen = if ($result) { $result.Length } else { 0 }
            Write-Host "[DEBUG] AI output length: $resultLen chars" -ForegroundColor Gray
            if ($resultLen -gt 0 -and $resultLen -lt 2000) {
                # Log short outputs for debugging
                Write-Host "[DEBUG] Output preview:" -ForegroundColor Gray
                Write-Host $result.Substring(0, [Math]::Min(500, $resultLen)) -ForegroundColor DarkGray
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
            
            Write-Host "[OK] AI completed successfully" -ForegroundColor Green
            return @{
                Success  = $true
                Files    = $files
                Attempts = $attempt
                Raw      = $result
            }
        }
        catch {
            Write-Warning "Attempt $attempt failed: $_"
            
            if ($attempt -lt $MaxRetries) {
                Write-Host "[INFO] Retrying in $retryDelay seconds..." -ForegroundColor Yellow
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
    
    $job = Start-Job -ScriptBlock {
        param($content, $provider)
        
        switch ($provider) {
            "claude" {
                $content | claude --dangerously-skip-permissions 2>&1
            }
            "droid" {
                $content | droid exec --skip-permissions-unsafe 2>&1
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
    } -ArgumentList $PromptContent, $Provider
    
    $completed = Wait-Job $job -Timeout $TimeoutSeconds
    
    if (-not $completed) {
        Stop-Job $job -ErrorAction SilentlyContinue
        Remove-Job $job -Force -ErrorAction SilentlyContinue
        return @{
            Success = $false
            Error   = "Timeout after $TimeoutSeconds seconds"
            Output  = $null
        }
    }
    
    $output = Receive-Job $job
    Remove-Job $job -Force
    
    return @{
        Success = $true
        Output  = $output
        Error   = $null
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
