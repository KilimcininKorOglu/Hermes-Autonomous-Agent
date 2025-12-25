<#
.SYNOPSIS
    Hermes Configuration Manager
.DESCRIPTION
    Manages configuration with priority: CLI params > Project > Global > Defaults
#>

$script:DefaultConfig = @{
    ai = @{
        provider = "auto"
        timeout = 300           # Default timeout for task mode (5 min)
        prdTimeout = 1200       # Timeout for PRD parsing (20 min)
        maxRetries = 10
    }
    taskMode = @{
        autoBranch = $false
        autoCommit = $false
        autonomous = $false
        maxConsecutiveErrors = 5
    }
    loop = @{
        maxCallsPerHour = 100
        timeoutMinutes = 15
    }
    paths = @{
        tasksDir = "tasks"
        logsDir = "logs"
    }
}

function Get-DefaultConfig {
    <#
    .SYNOPSIS
        Returns default configuration
    #>
    return $script:DefaultConfig.Clone()
}

function Get-GlobalConfigPath {
    <#
    .SYNOPSIS
        Returns global config file path
    #>
    return Join-Path $env:LOCALAPPDATA "Hermes\config.json"
}

function Get-ProjectConfigPath {
    <#
    .SYNOPSIS
        Returns project config file path
    .PARAMETER BasePath
        Base path for project (default: current directory)
    #>
    param([string]$BasePath = ".")
    return Join-Path $BasePath "hermes.config.json"
}

function Merge-ConfigHashtable {
    <#
    .SYNOPSIS
        Deep merge two hashtables, override wins
    #>
    param(
        [hashtable]$Base,
        [hashtable]$Override
    )
    
    $result = @{}
    
    foreach ($key in $Base.Keys) {
        $result[$key] = $Base[$key]
    }
    
    foreach ($key in $Override.Keys) {
        if ($result.ContainsKey($key) -and $result[$key] -is [hashtable] -and $Override[$key] -is [hashtable]) {
            $result[$key] = Merge-ConfigHashtable -Base $result[$key] -Override $Override[$key]
        } else {
            $result[$key] = $Override[$key]
        }
    }
    
    return $result
}

function ConvertTo-Hashtable {
    <#
    .SYNOPSIS
        Convert PSCustomObject to hashtable recursively
    #>
    param([Parameter(ValueFromPipeline)]$InputObject)
    
    if ($null -eq $InputObject) { return @{} }
    
    if ($InputObject -is [hashtable]) {
        $result = @{}
        foreach ($key in $InputObject.Keys) {
            $result[$key] = ConvertTo-Hashtable -InputObject $InputObject[$key]
        }
        return $result
    }
    
    if ($InputObject -is [PSCustomObject]) {
        $result = @{}
        foreach ($prop in $InputObject.PSObject.Properties) {
            $result[$prop.Name] = ConvertTo-Hashtable -InputObject $prop.Value
        }
        return $result
    }
    
    return $InputObject
}

function Read-ConfigFile {
    <#
    .SYNOPSIS
        Read and parse config file
    .PARAMETER Path
        Path to config file
    #>
    param([string]$Path)
    
    if (-not (Test-Path $Path)) {
        return $null
    }
    
    try {
        $content = Get-Content $Path -Raw -ErrorAction Stop
        $json = $content | ConvertFrom-Json -ErrorAction Stop
        return ConvertTo-Hashtable -InputObject $json
    } catch {
        Write-Warning "Failed to read config file: $Path - $_"
        return $null
    }
}

function Get-HermesConfig {
    <#
    .SYNOPSIS
        Get merged configuration from all sources
    .PARAMETER BasePath
        Base path for project config (default: current directory)
    #>
    param([string]$BasePath = ".")
    
    # Start with defaults
    $config = Get-DefaultConfig
    
    # Merge global config
    $globalPath = Get-GlobalConfigPath
    $globalConfig = Read-ConfigFile -Path $globalPath
    if ($null -ne $globalConfig) {
        $config = Merge-ConfigHashtable -Base $config -Override $globalConfig
    }
    
    # Merge project config
    $projectPath = Get-ProjectConfigPath -BasePath $BasePath
    $projectConfig = Read-ConfigFile -Path $projectPath
    if ($null -ne $projectConfig) {
        $config = Merge-ConfigHashtable -Base $config -Override $projectConfig
    }
    
    return $config
}

function Get-ConfigValue {
    <#
    .SYNOPSIS
        Get a specific config value using dot notation
    .PARAMETER Key
        Config key in dot notation (e.g., "ai.provider")
    .PARAMETER Override
        Override value from CLI parameter
    .PARAMETER BasePath
        Base path for project config
    #>
    param(
        [Parameter(Mandatory)][string]$Key,
        $Override = $null,
        [string]$BasePath = "."
    )
    
    # CLI override takes priority
    if ($null -ne $Override -and $Override -ne "") {
        return $Override
    }
    
    $config = Get-HermesConfig -BasePath $BasePath
    $parts = $Key.Split(".")
    $value = $config
    
    foreach ($part in $parts) {
        if ($value -is [hashtable] -and $value.ContainsKey($part)) {
            $value = $value[$part]
        } else {
            return $null
        }
    }
    
    return $value
}

function Initialize-DefaultConfig {
    <#
    .SYNOPSIS
        Create default global config file
    .PARAMETER Force
        Overwrite existing config
    #>
    param([switch]$Force)
    
    $configPath = Get-GlobalConfigPath
    $configDir = Split-Path $configPath -Parent
    
    if (-not (Test-Path $configDir)) {
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }
    
    if ((Test-Path $configPath) -and -not $Force) {
        return $false
    }
    
    $defaultConfig = Get-DefaultConfig
    $json = $defaultConfig | ConvertTo-Json -Depth 10
    $json | Set-Content $configPath -Encoding UTF8
    
    return $true
}

function Initialize-ProjectConfig {
    <#
    .SYNOPSIS
        Create project config file
    .PARAMETER BasePath
        Base path for project
    .PARAMETER Force
        Overwrite existing config
    #>
    param(
        [string]$BasePath = ".",
        [switch]$Force
    )
    
    $configPath = Get-ProjectConfigPath -BasePath $BasePath
    
    if ((Test-Path $configPath) -and -not $Force) {
        return $false
    }
    
    # Project config has same structure as global config
    $projectConfig = Get-DefaultConfig
    
    $json = $projectConfig | ConvertTo-Json -Depth 10
    $json | Set-Content $configPath -Encoding UTF8
    
    return $true
}

function Set-ConfigValue {
    <#
    .SYNOPSIS
        Set a config value
    .PARAMETER Key
        Config key in dot notation
    .PARAMETER Value
        Value to set
    .PARAMETER Scope
        global or project
    .PARAMETER BasePath
        Base path for project config
    #>
    param(
        [Parameter(Mandatory)][string]$Key,
        [Parameter(Mandatory)]$Value,
        [ValidateSet("global", "project")][string]$Scope = "global",
        [string]$BasePath = "."
    )
    
    if ($Scope -eq "global") {
        $configPath = Get-GlobalConfigPath
    } else {
        $configPath = Get-ProjectConfigPath -BasePath $BasePath
    }
    
    # Read existing or create new
    $config = Read-ConfigFile -Path $configPath
    if ($null -eq $config) {
        $config = @{}
    }
    
    # Set value using dot notation
    $parts = $Key.Split(".")
    $current = $config
    
    for ($i = 0; $i -lt $parts.Count - 1; $i++) {
        $part = $parts[$i]
        if (-not $current.ContainsKey($part) -or $current[$part] -isnot [hashtable]) {
            $current[$part] = @{}
        }
        $current = $current[$part]
    }
    
    $lastKey = $parts[-1]
    
    # Convert string to appropriate type
    if ($Value -eq "true") { $Value = $true }
    elseif ($Value -eq "false") { $Value = $false }
    elseif ($Value -match '^\d+$') { $Value = [int]$Value }
    
    $current[$lastKey] = $Value
    
    # Ensure directory exists
    $configDir = Split-Path $configPath -Parent
    if (-not (Test-Path $configDir)) {
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }
    
    # Write config
    $json = $config | ConvertTo-Json -Depth 10
    $json | Set-Content $configPath -Encoding UTF8
    
    return $true
}

function Test-ConfigExists {
    <#
    .SYNOPSIS
        Check if config file exists
    .PARAMETER Scope
        global or project
    .PARAMETER BasePath
        Base path for project config
    #>
    param(
        [ValidateSet("global", "project")][string]$Scope = "global",
        [string]$BasePath = "."
    )
    
    if ($Scope -eq "global") {
        return Test-Path (Get-GlobalConfigPath)
    } else {
        return Test-Path (Get-ProjectConfigPath -BasePath $BasePath)
    }
}

function Show-Config {
    <#
    .SYNOPSIS
        Display current configuration
    .PARAMETER BasePath
        Base path for project config
    #>
    param([string]$BasePath = ".")
    
    $config = Get-HermesConfig -BasePath $BasePath
    
    Write-Host "`nHermes Configuration" -ForegroundColor Cyan
    Write-Host "====================" -ForegroundColor Cyan
    
    Write-Host "`n[AI Settings]" -ForegroundColor Yellow
    Write-Host "  Provider:    $($config.ai.provider)"
    Write-Host "  Timeout:     $($config.ai.timeout) seconds"
    Write-Host "  Max Retries: $($config.ai.maxRetries)"
    
    Write-Host "`n[Task Mode]" -ForegroundColor Yellow
    Write-Host "  Auto Branch: $($config.taskMode.autoBranch)"
    Write-Host "  Auto Commit: $($config.taskMode.autoCommit)"
    Write-Host "  Autonomous:  $($config.taskMode.autonomous)"
    Write-Host "  Max Errors:  $($config.taskMode.maxConsecutiveErrors)"
    
    Write-Host "`n[Loop]" -ForegroundColor Yellow
    Write-Host "  Max Calls/Hour:  $($config.loop.maxCallsPerHour)"
    Write-Host "  Timeout Minutes: $($config.loop.timeoutMinutes)"
    
    Write-Host "`n[Paths]" -ForegroundColor Yellow
    Write-Host "  Tasks Dir: $($config.paths.tasksDir)"
    Write-Host "  Logs Dir:  $($config.paths.logsDir)"
    
    Write-Host "`n[Config Files]" -ForegroundColor Yellow
    $globalPath = Get-GlobalConfigPath
    $projectPath = Get-ProjectConfigPath -BasePath $BasePath
    Write-Host "  Global:  $globalPath $(if (Test-Path $globalPath) { '(exists)' } else { '(not found)' })"
    Write-Host "  Project: $projectPath $(if (Test-Path $projectPath) { '(exists)' } else { '(not found)' })"
    Write-Host ""
}
