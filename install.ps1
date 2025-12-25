#Requires -Version 7.0

<#
.SYNOPSIS
    Hermes for Claude Code - Windows Installation Script
.DESCRIPTION
    Installs Hermes globally on Windows systems.
    Creates commands: Hermes, Hermes-monitor, Hermes-setup, Hermes-import
.PARAMETER Uninstall
    Remove Hermes installation
.PARAMETER Help
    Show help message
.EXAMPLE
    .\install.ps1
.EXAMPLE
    .\install.ps1 -Uninstall
#>

[CmdletBinding()]
param(
    [switch]$Uninstall,
    
    [Alias('h')]
    [switch]$Help
)

# Configuration
$script:HermesHome = Join-Path $env:LOCALAPPDATA "Hermes"
$script:BinDir = Join-Path $script:HermesHome "bin"
$script:TemplatesDir = Join-Path $script:HermesHome "templates"
$script:LibDir = Join-Path $script:HermesHome "lib"
$script:ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Show-Help {
    Write-Host ""
    Write-Host "Hermes for Claude Code - Windows Installation" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\install.ps1 [OPTIONS]" -ForegroundColor White
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Yellow
    Write-Host "    -Uninstall    Remove Hermes installation"
    Write-Host "    -Help         Show this help message"
    Write-Host ""
    Write-Host "Installation paths:" -ForegroundColor Yellow
    Write-Host "    Commands:     $script:BinDir"
    Write-Host "    Scripts:      $script:HermesHome"
    Write-Host "    Templates:    $script:TemplatesDir"
    Write-Host ""
}

function Write-Log {
    <#
    .SYNOPSIS
        Logs a message with color
    #>
    param(
        [ValidateSet("INFO", "WARN", "ERROR", "SUCCESS")]
        [string]$Level,
        [string]$Message
    )
    
    $color = switch ($Level) {
        "INFO"    { "Cyan" }
        "WARN"    { "Yellow" }
        "ERROR"   { "Red" }
        "SUCCESS" { "Green" }
    }
    
    $timestamp = Get-Date -Format "HH:mm:ss"
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Test-Dependencies {
    <#
    .SYNOPSIS
        Checks for required dependencies
    #>
    
    Write-Log -Level "INFO" -Message "Checking dependencies..."
    
    $missingDeps = @()
    
    # Check PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 7) {
        $missingDeps += "PowerShell 7+ (current: $($PSVersionTable.PSVersion))"
    }
    
    # Check Node.js/npm
    $nodeCmd = Get-Command node -ErrorAction SilentlyContinue
    $npxCmd = Get-Command npx -ErrorAction SilentlyContinue
    if (-not $nodeCmd -and -not $npxCmd) {
        $missingDeps += "Node.js/npm"
    }
    
    # Check Git
    $gitCmd = Get-Command git -ErrorAction SilentlyContinue
    if (-not $gitCmd) {
        $missingDeps += "Git"
    }
    
    if ($missingDeps.Count -gt 0) {
        Write-Log -Level "ERROR" -Message "Missing required dependencies:"
        foreach ($dep in $missingDeps) {
            Write-Host "    - $dep" -ForegroundColor Red
        }
        Write-Host ""
        Write-Host "Please install the missing dependencies:" -ForegroundColor Yellow
        Write-Host "  - Node.js: https://nodejs.org/"
        Write-Host "  - Git: https://git-scm.com/download/win"
        Write-Host "  - PowerShell 7: https://github.com/PowerShell/PowerShell/releases"
        Write-Host ""
        Write-Host "Or install via winget:" -ForegroundColor Cyan
        Write-Host "  winget install Microsoft.PowerShell OpenJS.NodeJS.LTS Git.Git"
        Write-Host ""
        return $false
    }
    
    Write-Log -Level "INFO" -Message "Claude Code CLI will be downloaded when first used."
    Write-Log -Level "SUCCESS" -Message "Dependencies check completed"
    return $true
}

function New-InstallDirs {
    <#
    .SYNOPSIS
        Creates installation directories
    #>
    
    Write-Log -Level "INFO" -Message "Creating installation directories..."
    
    $dirs = @($script:HermesHome, $script:BinDir, $script:TemplatesDir, $script:LibDir)
    
    foreach ($dir in $dirs) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }
    
    Write-Log -Level "SUCCESS" -Message "Directories created: $script:HermesHome"
}

function Install-Scripts {
    <#
    .SYNOPSIS
        Installs Hermes scripts to Hermes home directory
    #>
    
    Write-Log -Level "INFO" -Message "Installing Hermes scripts..."
    
    # Copy main scripts
    $mainScripts = @(
        "hermes_loop.ps1",
        "hermes_monitor.ps1",
        "setup.ps1",
        "hermes_import.ps1",
        "hermes-prd.ps1",
        "hermes-add.ps1"
    )
    
    foreach ($scriptName in $mainScripts) {
        $sourcePath = Join-Path $script:ScriptDir $scriptName
        if (Test-Path $sourcePath) {
            Copy-Item -Path $sourcePath -Destination $script:HermesHome -Force
            Write-Host "    Installed: $scriptName" -ForegroundColor Gray
        }
        else {
            Write-Log -Level "WARN" -Message "Script not found: $scriptName"
        }
    }
    
    # Copy lib folder
    $libSource = Join-Path $script:ScriptDir "lib"
    if (Test-Path $libSource) {
        Copy-Item -Path "$libSource\*" -Destination $script:LibDir -Force -Recurse
        Write-Host "    Installed: lib\*.ps1" -ForegroundColor Gray
    }
    
    # Copy templates
    $templatesSource = Join-Path $script:ScriptDir "templates"
    if (Test-Path $templatesSource) {
        Copy-Item -Path "$templatesSource\*" -Destination $script:TemplatesDir -Force -Recurse
        Write-Host "    Installed: templates\*" -ForegroundColor Gray
    }
    
    Write-Log -Level "SUCCESS" -Message "Hermes scripts installed to $script:HermesHome"
}

function Install-Commands {
    <#
    .SYNOPSIS
        Creates command wrappers in bin directory
    #>
    
    Write-Log -Level "INFO" -Message "Creating command wrappers..."
    
    # hermes.cmd - Main command (for CMD)
    $hermesCmd = @"
@echo off
pwsh -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Hermes\hermes_loop.ps1" %*
"@
    $hermesCmd | Set-Content (Join-Path $script:BinDir "hermes.cmd") -Encoding ASCII
    
    # hermes-monitor.cmd
    $monitorCmd = @"
@echo off
pwsh -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Hermes\hermes_monitor.ps1" %*
"@
    $monitorCmd | Set-Content (Join-Path $script:BinDir "hermes-monitor.cmd") -Encoding ASCII
    
    # hermes-setup.cmd
    $setupCmd = @"
@echo off
pwsh -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Hermes\setup.ps1" %*
"@
    $setupCmd | Set-Content (Join-Path $script:BinDir "hermes-setup.cmd") -Encoding ASCII
    
    # hermes-import.cmd
    $importCmd = @"
@echo off
pwsh -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Hermes\hermes_import.ps1" %*
"@
    $importCmd | Set-Content (Join-Path $script:BinDir "hermes-import.cmd") -Encoding ASCII
    
    # PowerShell wrappers (.ps1) for PowerShell users
    
    # hermes.ps1
    $hermesPs1 = @"
#Requires -Version 7.0
# Hermes for Claude Code - PowerShell Wrapper
`$HermesHome = Join-Path `$env:LOCALAPPDATA "Hermes"
& (Join-Path `$HermesHome "hermes_loop.ps1") @args
"@
    $hermesPs1 | Set-Content (Join-Path $script:BinDir "hermes.ps1") -Encoding UTF8
    
    # hermes-monitor.ps1
    $monitorPs1 = @"
#Requires -Version 7.0
# Hermes Monitor - PowerShell Wrapper
`$HermesHome = Join-Path `$env:LOCALAPPDATA "Hermes"
& (Join-Path `$HermesHome "hermes_monitor.ps1") @args
"@
    $monitorPs1 | Set-Content (Join-Path $script:BinDir "hermes-monitor.ps1") -Encoding UTF8
    
    # hermes-setup.ps1
    $setupPs1 = @"
#Requires -Version 7.0
# Hermes Setup - PowerShell Wrapper
`$HermesHome = Join-Path `$env:LOCALAPPDATA "Hermes"
& (Join-Path `$HermesHome "setup.ps1") @args
"@
    $setupPs1 | Set-Content (Join-Path $script:BinDir "hermes-setup.ps1") -Encoding UTF8
    
    # hermes-import.ps1
    $importPs1 = @"
#Requires -Version 7.0
# Hermes Import - PowerShell Wrapper
`$HermesHome = Join-Path `$env:LOCALAPPDATA "Hermes"
& (Join-Path `$HermesHome "hermes_import.ps1") @args
"@
    $importPs1 | Set-Content (Join-Path $script:BinDir "hermes-import.ps1") -Encoding UTF8
    
    # hermes-prd.cmd
    $prdCmd = @"
@echo off
pwsh -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Hermes\hermes-prd.ps1" %*
"@
    $prdCmd | Set-Content (Join-Path $script:BinDir "hermes-prd.cmd") -Encoding ASCII
    
    # hermes-prd.ps1
    $prdPs1 = @"
#Requires -Version 7.0
# Hermes PRD Parser - PowerShell Wrapper
`$HermesHome = Join-Path `$env:LOCALAPPDATA "Hermes"
& (Join-Path `$HermesHome "hermes-prd.ps1") @args
"@
    $prdPs1 | Set-Content (Join-Path $script:BinDir "hermes-prd.ps1") -Encoding UTF8
    
    # hermes-add.cmd
    $addCmd = @"
@echo off
pwsh -NoProfile -ExecutionPolicy Bypass -File "%LOCALAPPDATA%\Hermes\hermes-add.ps1" %*
"@
    $addCmd | Set-Content (Join-Path $script:BinDir "hermes-add.cmd") -Encoding ASCII
    
    # hermes-add.ps1
    $addPs1 = @"
#Requires -Version 7.0
# Hermes Add - PowerShell Wrapper
`$HermesHome = Join-Path `$env:LOCALAPPDATA "Hermes"
& (Join-Path `$HermesHome "hermes-add.ps1") @args
"@
    $addPs1 | Set-Content (Join-Path $script:BinDir "hermes-add.ps1") -Encoding UTF8
    
    Write-Log -Level "SUCCESS" -Message "Command wrappers created in $script:BinDir"
}

function Add-ToPath {
    <#
    .SYNOPSIS
        Adds Hermes bin directory to user PATH
    #>
    
    Write-Log -Level "INFO" -Message "Checking PATH configuration..."
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    
    if ($currentPath -notlike "*$($script:BinDir)*") {
        Write-Log -Level "INFO" -Message "Adding $($script:BinDir) to user PATH..."
        
        # Add to user PATH (permanent)
        $newPath = "$($script:BinDir);$currentPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        
        # Also update current session
        $env:PATH = "$($script:BinDir);$env:PATH"
        
        Write-Log -Level "SUCCESS" -Message "PATH updated successfully"
        Write-Host ""
        Write-Host "NOTE: You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
    }
    else {
        Write-Log -Level "SUCCESS" -Message "$($script:BinDir) is already in PATH"
    }
}

function Remove-FromPath {
    <#
    .SYNOPSIS
        Removes Hermes bin directory from user PATH
    #>
    
    Write-Log -Level "INFO" -Message "Removing from PATH..."
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    
    if ($currentPath -like "*$($script:BinDir)*") {
        $pathParts = $currentPath -split ';' | Where-Object { $_ -ne $script:BinDir -and $_ -ne "" }
        $newPath = $pathParts -join ';'
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Log -Level "SUCCESS" -Message "Removed from PATH"
    }
    else {
        Write-Log -Level "INFO" -Message "Hermes bin directory was not in PATH"
    }
}

function Uninstall-Hermes {
    <#
    .SYNOPSIS
        Removes Hermes installation
    #>
    
    Write-Log -Level "INFO" -Message "Uninstalling Hermes for Claude Code..."
    
    # Remove from PATH
    Remove-FromPath
    
    # Remove installation directory
    if (Test-Path $script:HermesHome) {
        Remove-Item -Path $script:HermesHome -Recurse -Force
        Write-Log -Level "SUCCESS" -Message "Removed $($script:HermesHome)"
    }
    else {
        Write-Log -Level "INFO" -Message "Installation directory not found"
    }
    
    Write-Host ""
    Write-Log -Level "SUCCESS" -Message "Hermes for Claude Code uninstalled successfully"
    Write-Host ""
}

function Install-Hermes {
    <#
    .SYNOPSIS
        Main installation function
    #>
    
    Write-Host ""
    Write-Host "Installing Hermes for Claude Code globally..." -ForegroundColor Cyan
    Write-Host ""
    
    # Check dependencies
    if (-not (Test-Dependencies)) {
        exit 1
    }
    
    # Create directories
    New-InstallDirs
    
    # Install scripts
    Install-Scripts
    
    # Create command wrappers
    Install-Commands
    
    # Update PATH
    Add-ToPath
    
    # Success message
    Write-Host ""
    Write-Log -Level "SUCCESS" -Message "Hermes for Claude Code installed successfully!"
    Write-Host ""
    Write-Host "Global commands available:" -ForegroundColor Cyan
    Write-Host "  hermes -TaskMode          Start Hermes Task Mode"
    Write-Host "  hermes -Help              Show Hermes options"
    Write-Host "  hermes-setup my-project   Create new Hermes project"
    Write-Host "  hermes-prd prd.md         Parse PRD to task files"
    Write-Host "  hermes-add `"feature`"      Add single feature to task plan"
    Write-Host "  hermes-monitor            Live monitoring dashboard"
    Write-Host ""
    Write-Host "Quick start:" -ForegroundColor Cyan
    Write-Host "  1. hermes-setup my-project"
    Write-Host "  2. cd my-project"
    Write-Host "  3. hermes-prd docs/PRD.md"
    Write-Host "  4. hermes -TaskMode -AutoBranch -AutoCommit"
    Write-Host ""
    Write-Host "Installation paths:" -ForegroundColor Gray
    Write-Host "  Commands:  $($script:BinDir)"
    Write-Host "  Scripts:   $($script:HermesHome)"
    Write-Host "  Templates: $($script:TemplatesDir)"
    Write-Host ""
}

# Main entry point
if ($Help) {
    Show-Help
    exit 0
}

if ($Uninstall) {
    Uninstall-Hermes
    exit 0
}

Install-Hermes
