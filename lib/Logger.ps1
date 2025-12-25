<#
.SYNOPSIS
    Hermes Logger Module
.DESCRIPTION
    Centralized logging for all Hermes commands.
    Each command has its own log file with timestamped entries.
#>

$script:LogConfig = @{
    LogDir = "logs"
    DateFormat = "yyyy-MM-dd HH:mm:ss"
    FileNameFormat = "yyyy-MM-dd"
}

$script:CurrentLogFile = $null

function Initialize-Logger {
    <#
    .SYNOPSIS
        Initialize logger for a specific command
    .PARAMETER Command
        Command name (e.g., "hermes-prd", "hermes-loop", "hermes-add")
    .PARAMETER BasePath
        Base path for logs directory
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Command,
        
        [string]$BasePath = "."
    )
    
    # Get log directory from config if available
    $logDir = $script:LogConfig.LogDir
    if (Get-Command Get-ConfigValue -ErrorAction SilentlyContinue) {
        $configLogDir = Get-ConfigValue -Key "paths.logsDir"
        if ($configLogDir) { $logDir = $configLogDir }
    }
    
    $fullLogDir = Join-Path $BasePath $logDir
    
    # Create logs directory if not exists
    if (-not (Test-Path $fullLogDir)) {
        New-Item -ItemType Directory -Path $fullLogDir -Force | Out-Null
    }
    
    # Set current log file (command-date.log)
    $dateStr = Get-Date -Format $script:LogConfig.FileNameFormat
    $script:CurrentLogFile = Join-Path $fullLogDir "$Command-$dateStr.log"
    
    # Write session start
    Write-Log -Level "INFO" -Message "=== Session started ==="
    Write-Log -Level "INFO" -Message "Command: $Command"
    Write-Log -Level "INFO" -Message "Working directory: $(Get-Location)"
    
    return $script:CurrentLogFile
}

function Write-Log {
    <#
    .SYNOPSIS
        Write a log entry
    .PARAMETER Level
        Log level: INFO, WARN, ERROR, SUCCESS, DEBUG
    .PARAMETER Message
        Log message
    .PARAMETER NoConsole
        Don't write to console, only to file
    #>
    param(
        [ValidateSet("INFO", "WARN", "ERROR", "SUCCESS", "DEBUG")]
        [string]$Level = "INFO",
        
        [Parameter(Mandatory)]
        [string]$Message,
        
        [switch]$NoConsole
    )
    
    $timestamp = Get-Date -Format $script:LogConfig.DateFormat
    $logEntry = "[$timestamp] [$Level] $Message"
    
    # Write to file if initialized
    if ($script:CurrentLogFile) {
        $logEntry | Add-Content -Path $script:CurrentLogFile -Encoding UTF8
    }
    
    # Write to console unless NoConsole
    if (-not $NoConsole) {
        $color = switch ($Level) {
            "INFO"    { "Cyan" }
            "WARN"    { "Yellow" }
            "ERROR"   { "Red" }
            "SUCCESS" { "Green" }
            "DEBUG"   { "Gray" }
            default   { "White" }
        }
        Write-Host "[$Level] $Message" -ForegroundColor $color
    }
}

function Write-LogOnly {
    <#
    .SYNOPSIS
        Write only to log file, not to console
    #>
    param(
        [string]$Level = "INFO",
        [string]$Message
    )
    
    Write-Log -Level $Level -Message $Message -NoConsole
}

function Write-LogSection {
    <#
    .SYNOPSIS
        Write a section header to log
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Title
    )
    
    $separator = "=" * 50
    Write-Log -Level "INFO" -Message $separator -NoConsole
    Write-Log -Level "INFO" -Message $Title -NoConsole
    Write-Log -Level "INFO" -Message $separator -NoConsole
}

function Write-LogError {
    <#
    .SYNOPSIS
        Write error with optional exception details
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Message,
        
        [System.Exception]$Exception
    )
    
    Write-Log -Level "ERROR" -Message $Message
    
    if ($Exception) {
        Write-Log -Level "DEBUG" -Message "Exception: $($Exception.GetType().Name)" -NoConsole
        Write-Log -Level "DEBUG" -Message "Details: $($Exception.Message)" -NoConsole
        if ($Exception.StackTrace) {
            Write-Log -Level "DEBUG" -Message "Stack: $($Exception.StackTrace)" -NoConsole
        }
    }
}

function Write-LogResult {
    <#
    .SYNOPSIS
        Write operation result summary
    #>
    param(
        [Parameter(Mandatory)]
        [bool]$Success,
        
        [string]$Operation,
        
        [string]$Details
    )
    
    if ($Success) {
        Write-Log -Level "SUCCESS" -Message "$Operation completed successfully"
    } else {
        Write-Log -Level "ERROR" -Message "$Operation failed"
    }
    
    if ($Details) {
        Write-Log -Level "INFO" -Message $Details -NoConsole
    }
}

function Close-Logger {
    <#
    .SYNOPSIS
        Close logger and write session end
    #>
    param(
        [bool]$Success = $true
    )
    
    if ($Success) {
        Write-Log -Level "INFO" -Message "=== Session completed successfully ===" -NoConsole
    } else {
        Write-Log -Level "WARN" -Message "=== Session ended with errors ===" -NoConsole
    }
    
    $script:CurrentLogFile = $null
}

function Get-LogFile {
    <#
    .SYNOPSIS
        Get current log file path
    #>
    return $script:CurrentLogFile
}

function Get-RecentLogs {
    <#
    .SYNOPSIS
        Get recent log entries
    .PARAMETER Command
        Filter by command name
    .PARAMETER Lines
        Number of lines to return
    .PARAMETER BasePath
        Base path for logs
    #>
    param(
        [string]$Command = "*",
        [int]$Lines = 50,
        [string]$BasePath = "."
    )
    
    $logDir = Join-Path $BasePath $script:LogConfig.LogDir
    
    if (-not (Test-Path $logDir)) {
        return @()
    }
    
    $logFiles = Get-ChildItem -Path $logDir -Filter "$Command-*.log" | 
                Sort-Object LastWriteTime -Descending |
                Select-Object -First 1
    
    if ($logFiles) {
        return Get-Content $logFiles.FullName -Tail $Lines
    }
    
    return @()
}
