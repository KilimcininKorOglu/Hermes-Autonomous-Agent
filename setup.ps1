#Requires -Version 7.0

<#
.SYNOPSIS
    Hermes Project Setup - Windows PowerShell Version
.DESCRIPTION
    Creates a new Hermes project with standard structure and templates.
.PARAMETER ProjectName
    Name of the project to create (default: my-project)
.PARAMETER Help
    Show help message
.EXAMPLE
    .\setup.ps1 my-awesome-project
.EXAMPLE
    hermes-setup my-awesome-project
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$ProjectName = "my-project",
    
    [Alias('h')]
    [switch]$Help
)

# Get Hermes home directory
$script:HermesHome = if ($env:HERMES_HOME) { 
    $env:HERMES_HOME 
} 
else { 
    Join-Path $env:LOCALAPPDATA "Hermes" 
}

$script:TemplatesDir = Join-Path $script:HermesHome "templates"
$script:ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Show-Help {
    Write-Host ""
    Write-Host "Hermes Project Setup" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: hermes-setup [PROJECT_NAME]" -ForegroundColor White
    Write-Host ""
    Write-Host "Arguments:" -ForegroundColor Yellow
    Write-Host "    PROJECT_NAME    Name of the project (default: my-project)"
    Write-Host ""
    Write-Host "Example:" -ForegroundColor Yellow
    Write-Host "    hermes-setup my-awesome-project"
    Write-Host "    cd my-awesome-project"
    Write-Host "    Hermes -Monitor"
    Write-Host ""
    Write-Host "This creates a new directory with:" -ForegroundColor Gray
    Write-Host "    PROMPT.md        Main development instructions for Hermes"
    Write-Host "    tasks/           Task files (created by hermes-prd)"
    Write-Host "    specs/           Project specifications"
    Write-Host "    src/             Source code"
    Write-Host "    logs/            Execution logs"
    Write-Host ""
}

function Get-TemplatesPath {
    <#
    .SYNOPSIS
        Finds the templates directory
    #>
    
    # First check Hermes home
    if (Test-Path $script:TemplatesDir) {
        return $script:TemplatesDir
    }
    
    # Check relative to script
    $localTemplates = Join-Path $script:ScriptDir "templates"
    if (Test-Path $localTemplates) {
        return $localTemplates
    }
    
    # Check parent directory (for development)
    $parentTemplates = Join-Path (Split-Path $script:ScriptDir -Parent) "templates"
    if (Test-Path $parentTemplates) {
        return $parentTemplates
    }
    
    return $null
}

function New-HermesProject {
    <#
    .SYNOPSIS
        Creates a new Hermes project
    #>
    param([string]$Name)
    
    Write-Host ""
    Write-Host "Setting up Hermes project: $Name" -ForegroundColor Cyan
    Write-Host ""
    
    # Find templates
    $templatesPath = Get-TemplatesPath
    
    if (-not $templatesPath) {
        Write-Host "[ERROR] Templates not found!" -ForegroundColor Red
        Write-Host ""
        Write-Host "Expected locations:" -ForegroundColor Yellow
        Write-Host "  - $script:TemplatesDir"
        Write-Host "  - $(Join-Path $script:ScriptDir 'templates')"
        Write-Host ""
        Write-Host "Please run install.ps1 first to install Hermes globally."
        Write-Host ""
        exit 1
    }
    
    Write-Host "Using templates from: $templatesPath" -ForegroundColor Gray
    
    # Check if directory already exists
    if (Test-Path $Name) {
        Write-Host "[ERROR] Directory '$Name' already exists!" -ForegroundColor Red
        Write-Host ""
        Write-Host "Please choose a different name or remove the existing directory."
        Write-Host ""
        exit 1
    }
    
    # Create project directory
    New-Item -ItemType Directory -Path $Name -Force | Out-Null
    Write-Host "[OK] Created directory: $Name" -ForegroundColor Green
    
    # Change to project directory
    Push-Location $Name
    
    try {
        # Create directory structure
        $directories = @(
            "specs",
            "specs\stdlib",
            "src",
            "examples",
            "logs",
            "docs\generated"
        )
        
        foreach ($dir in $directories) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
        Write-Host "[OK] Created directory structure" -ForegroundColor Green
        
        # Copy PROMPT.md template
        $promptSource = Join-Path $templatesPath "PROMPT.md"
        if (Test-Path $promptSource) {
            Copy-Item -Path $promptSource -Destination "PROMPT.md" -Force
            Write-Host "[OK] Created: PROMPT.md" -ForegroundColor Green
        }
        else {
            $content = Get-DefaultPromptTemplate -ProjectName $Name
            $content | Set-Content "PROMPT.md" -Encoding UTF8
            Write-Host "[OK] Created: PROMPT.md (default template)" -ForegroundColor Green
        }
        
        # Create tasks directory
        New-Item -ItemType Directory -Path "tasks" -Force | Out-Null
        Write-Host "[OK] Created: tasks/" -ForegroundColor Green
        
        # Copy specs templates if they exist
        $specsSource = Join-Path $templatesPath "specs"
        if (Test-Path $specsSource) {
            Get-ChildItem $specsSource -File | ForEach-Object {
                Copy-Item -Path $_.FullName -Destination "specs\" -Force
            }
        }
        
        # Create .gitkeep files for empty directories
        $emptyDirs = @("src", "examples", "logs", "docs\generated", "specs\stdlib")
        foreach ($dir in $emptyDirs) {
            $gitkeep = Join-Path $dir ".gitkeep"
            if (-not (Test-Path $gitkeep)) {
                New-Item -ItemType File -Path $gitkeep -Force | Out-Null
            }
        }
        
        # Create README.md
        $readme = Get-ProjectReadme -ProjectName $Name
        $readme | Set-Content "README.md" -Encoding UTF8
        Write-Host "[OK] Created: README.md" -ForegroundColor Green
        
        # Create .gitignore
        $gitignore = Get-GitIgnoreContent
        $gitignore | Set-Content ".gitignore" -Encoding UTF8
        Write-Host "[OK] Created: .gitignore" -ForegroundColor Green
        
        # Initialize git repository
        try {
            $gitOutput = git init 2>&1
            git add . 2>&1 | Out-Null
            git commit -m "Initial Hermes project setup" 2>&1 | Out-Null
            Write-Host "[OK] Initialized git repository" -ForegroundColor Green
        }
        catch {
            Write-Host "[WARN] Could not initialize git repository" -ForegroundColor Yellow
        }
        
        # Success message
        Write-Host ""
        Write-Host "Project '$Name' created successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "  1. cd $Name"
        Write-Host "  2. Create a PRD document (e.g., docs/PRD.md)"
        Write-Host "  3. Run: hermes-prd docs/PRD.md"
        Write-Host "  4. Run: Hermes -TaskMode -AutoBranch -AutoCommit"
        Write-Host ""
    }
    finally {
        Pop-Location
    }
}

function Get-DefaultPromptTemplate {
    param([string]$ProjectName)
    
    return @"
# Hermes Development Instructions

## Context
You are Hermes, an autonomous AI development agent working on the $ProjectName project.

## Current Objectives
1. Complete the current task from tasks/*.md
2. Follow the success criteria for each task
3. Run tests after implementation
4. Commit working changes with descriptive messages

## Key Principles
- ONE task per loop - focus on the current task only
- Search the codebase before assuming something isn't implemented
- Write comprehensive tests with clear documentation
- Commit working changes with descriptive messages

## Testing Guidelines
- LIMIT testing to ~20% of your total effort per loop
- PRIORITIZE: Implementation > Documentation > Tests
- Only write tests for NEW functionality you implement
- Focus on CORE functionality first

## Status Reporting

At the end of your response, include this status block:

``````
---HERMES_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
RECOMMENDATION: <one line summary of what to do next>
---END_HERMES_STATUS---
``````

## File Structure
- tasks/: Task files with feature definitions
- src/: Source code implementation
- docs/: Project documentation

## Current Task
The current task will be injected below by Hermes Task Mode.
"@
}

function Get-ProjectReadme {
    param([string]$ProjectName)
    
    return @"
# $ProjectName

A Hermes-managed project for autonomous AI development.

## Getting Started

1. Create a PRD document in ``docs/PRD.md``
2. Run: ``hermes-prd docs/PRD.md``
3. Run: ``Hermes -TaskMode -AutoBranch -AutoCommit``

## Project Structure

- ``PROMPT.md`` - Main development instructions for Hermes
- ``tasks/`` - Task files (created by hermes-prd)
- ``src/`` - Source code
- ``docs/`` - Project documentation
- ``logs/`` - Hermes execution logs

## Commands

``````powershell
hermes-prd docs/PRD.md                          # Parse PRD to tasks
Hermes -TaskMode -AutoBranch -AutoCommit        # Run Task Mode
Hermes -TaskStatus                              # Show task progress
Hermes -TaskMode -Autonomous                    # Run without pausing
``````

## Created with Hermes for Claude Code

[Hermes](https://github.com/frankbria/hermes-claude-code) - Autonomous AI development loop
"@
}

function Get-GitIgnoreContent {
    return @"
# Hermes state files
.call_count
.last_reset
.exit_signals
.circuit_breaker_state
.circuit_breaker_history
.response_analysis
.last_output_length
progress.json
status.json

# Logs
logs/

# Generated docs
docs/generated/

# Node modules (if applicable)
node_modules/

# Python (if applicable)
__pycache__/
*.pyc
.venv/
venv/

# OS files
.DS_Store
Thumbs.db
Desktop.ini

# IDE
.idea/
.vscode/
*.swp
*.swo
*~
"@
}

# Main entry point
if ($Help) {
    Show-Help
    exit 0
}

# Validate project name
if ($ProjectName -match '[<>:"/\\|?*]') {
    Write-Host ""
    Write-Host "[ERROR] Project name contains invalid characters: $ProjectName" -ForegroundColor Red
    Write-Host "        Avoid: < > : `" / \ | ? *"
    Write-Host ""
    exit 1
}

New-HermesProject -Name $ProjectName
