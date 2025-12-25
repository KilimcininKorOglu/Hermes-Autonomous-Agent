#Requires -Version 7.0

<#
.SYNOPSIS
    Hermes PRD Import - Windows PowerShell Version
.DESCRIPTION
    Converts existing PRD/specification documents to Hermes project format.
    Uses Claude Code to intelligently parse and structure the requirements.
.PARAMETER InputFile
    Path to the PRD or specification file to import
.PARAMETER ProjectName
    Name of the project to create (optional, derived from filename if not specified)
.PARAMETER Help
    Show help message
.EXAMPLE
    .\Hermes_import.ps1 requirements.md my-project
.EXAMPLE
    Hermes-import product-spec.md
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$InputFile,
    
    [Parameter(Position = 1)]
    [string]$ProjectName,
    
    [Alias('h')]
    [switch]$Help
)

# Configuration
$script:HermesHome = if ($env:Hermes_HOME) { $env:Hermes_HOME } else { Join-Path $env:LOCALAPPDATA "Hermes" }
$script:TemplatesDir = Join-Path $script:HermesHome "templates"
$script:ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$script:ClaudeCommand = "claude"

function Show-Help {
    Write-Host ""
    Write-Host "Hermes PRD Import" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: Hermes-import <INPUT_FILE> [PROJECT_NAME]" -ForegroundColor White
    Write-Host ""
    Write-Host "Arguments:" -ForegroundColor Yellow
    Write-Host "    INPUT_FILE      Path to PRD, spec, or requirements file"
    Write-Host "    PROJECT_NAME    Name of project to create (optional)"
    Write-Host ""
    Write-Host "Supported formats:" -ForegroundColor Yellow
    Write-Host "    - Markdown (.md)"
    Write-Host "    - Text files (.txt)"
    Write-Host "    - JSON (.json)"
    Write-Host "    - Word documents (.docx) - requires pandoc"
    Write-Host "    - PDF (.pdf) - requires pdftotext"
    Write-Host ""
    Write-Host "Example:" -ForegroundColor Yellow
    Write-Host "    Hermes-import product-requirements.md my-app"
    Write-Host "    Hermes-import api-spec.json"
    Write-Host ""
    Write-Host "The import process:" -ForegroundColor Gray
    Write-Host "    1. Reads your PRD/specification file"
    Write-Host "    2. Uses Claude Code to analyze and convert content"
    Write-Host "    3. Creates a Hermes project with:"
    Write-Host "       - PROMPT.md (development instructions)"
    Write-Host "       - @fix_plan.md (prioritized tasks)"
    Write-Host "       - specs/requirements.md (technical specs)"
    Write-Host ""
}

function Get-ProjectNameFromFile {
    <#
    .SYNOPSIS
        Derives a project name from the input filename
    #>
    param([string]$FilePath)
    
    $baseName = [System.IO.Path]::GetFileNameWithoutExtension($FilePath)
    
    # Remove common suffixes
    $suffixes = @("prd", "spec", "specs", "requirements", "requirement", "design", "doc", "docs")
    foreach ($suffix in $suffixes) {
        $baseName = $baseName -replace "[-_]?$suffix`$", ""
    }
    
    # Clean up special characters
    $cleanName = $baseName -replace '[^\w-]', '-'
    $cleanName = $cleanName -replace '-+', '-'
    $cleanName = $cleanName.Trim('-').ToLower()
    
    if ([string]::IsNullOrEmpty($cleanName)) {
        $cleanName = "Hermes-project"
    }
    
    return $cleanName
}

function Read-InputFile {
    <#
    .SYNOPSIS
        Reads the input file content, handling various formats
    #>
    param([string]$FilePath)
    
    if (-not (Test-Path $FilePath)) {
        throw "File not found: $FilePath"
    }
    
    $extension = [System.IO.Path]::GetExtension($FilePath).ToLower()
    
    switch ($extension) {
        ".md" { return Get-Content $FilePath -Raw -Encoding UTF8 }
        ".txt" { return Get-Content $FilePath -Raw -Encoding UTF8 }
        ".json" { return Get-Content $FilePath -Raw -Encoding UTF8 }
        ".docx" {
            # Try to use pandoc for Word documents
            $pandoc = Get-Command pandoc -ErrorAction SilentlyContinue
            if ($pandoc) {
                return & pandoc -f docx -t markdown $FilePath
            }
            else {
                throw "pandoc not found. Install it for .docx support: https://pandoc.org/installing.html"
            }
        }
        ".pdf" {
            # Try to use pdftotext
            $pdftotext = Get-Command pdftotext -ErrorAction SilentlyContinue
            if ($pdftotext) {
                $tempFile = [System.IO.Path]::GetTempFileName()
                try {
                    & pdftotext $FilePath $tempFile
                    $content = Get-Content $tempFile -Raw -Encoding UTF8
                    return $content
                }
                finally {
                    Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
                }
            }
            else {
                throw "pdftotext not found. Install poppler for PDF support."
            }
        }
        default {
            # Try to read as text
            return Get-Content $FilePath -Raw -Encoding UTF8
        }
    }
}

function Invoke-ClaudeConversion {
    <#
    .SYNOPSIS
        Uses Claude Code to convert the PRD content to Hermes format
    #>
    param(
        [string]$Content,
        [string]$ProjectName
    )
    
    $conversionPrompt = @"
You are converting a Product Requirements Document (PRD) or specification into a Hermes project format.

The source document content is:
---
$Content
---

Please generate the following files. Output each file clearly separated with headers.

## 1. PROMPT.md
Create development instructions for an AI agent working on this project. Include:
- Project context and goals extracted from the PRD
- Key objectives and success criteria
- Implementation guidelines based on the requirements
- Testing requirements
- The standard Hermes status reporting block (Hermes_STATUS with STATUS, EXIT_SIGNAL, RECOMMENDATION)

## 2. @fix_plan.md
Create a prioritized task list in markdown checkbox format:
- [ ] High priority tasks first (core functionality)
- [ ] Break down into small, actionable items
- [ ] Include setup and configuration tasks
- [ ] Include testing tasks
- [ ] Group by feature area or component

## 3. specs/requirements.md
Extract and organize technical requirements:
- Functional requirements (what the system must do)
- Non-functional requirements (performance, security, etc.)
- API specifications (if mentioned)
- Data models or schemas (if mentioned)
- Integration points (if mentioned)
- Acceptance criteria

Output format - use these EXACT headers:
=== PROMPT.md ===
(PROMPT.md content here)

=== @fix_plan.md ===
(@fix_plan.md content here)

=== specs/requirements.md ===
(specs/requirements.md content here)
"@

    Write-Host "[INFO] Converting document with Claude Code..." -ForegroundColor Cyan
    Write-Host "       This may take a minute..." -ForegroundColor Gray
    
    try {
        # Create a temp file for the prompt
        $tempFile = [System.IO.Path]::GetTempFileName()
        $conversionPrompt | Set-Content $tempFile -Encoding UTF8
        
        # Execute Claude Code
        $result = Get-Content $tempFile -Raw | & $script:ClaudeCommand 2>&1
        
        Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
        
        if ($LASTEXITCODE -ne 0) {
            throw "Claude Code returned exit code: $LASTEXITCODE"
        }
        
        return $result -join "`n"
    }
    catch {
        throw "Failed to run Claude Code: $($_.Exception.Message)"
    }
}

function Split-ConversionOutput {
    <#
    .SYNOPSIS
        Parses the Claude output to extract individual files
    #>
    param([string]$Output)
    
    $files = @{}
    
    # Parse sections using regex
    $patterns = @{
        "PROMPT.md" = "(?s)=== PROMPT\.md ===\s*(.*?)(?==== |$)"
        "@fix_plan.md" = "(?s)=== @fix_plan\.md ===\s*(.*?)(?==== |$)"
        "specs/requirements.md" = "(?s)=== specs/requirements\.md ===\s*(.*?)(?==== |$)"
    }
    
    foreach ($pattern in $patterns.GetEnumerator()) {
        if ($Output -match $pattern.Value) {
            $content = $Matches[1].Trim()
            if (-not [string]::IsNullOrEmpty($content)) {
                $files[$pattern.Key] = $content
            }
        }
    }
    
    return $files
}

function New-HermesProjectFromPRD {
    <#
    .SYNOPSIS
        Creates a Hermes project from a PRD file
    #>
    param(
        [string]$InputFile,
        [string]$ProjectName
    )
    
    # Resolve input file path
    $InputFile = Resolve-Path $InputFile -ErrorAction Stop
    
    # Determine project name
    if ([string]::IsNullOrEmpty($ProjectName)) {
        $ProjectName = Get-ProjectNameFromFile -FilePath $InputFile
    }
    
    Write-Host ""
    Write-Host "Importing PRD to Hermes project: $ProjectName" -ForegroundColor Cyan
    Write-Host "Source file: $InputFile" -ForegroundColor Gray
    Write-Host ""
    
    # Check if project directory exists
    if (Test-Path $ProjectName) {
        Write-Host "[ERROR] Directory '$ProjectName' already exists!" -ForegroundColor Red
        Write-Host "        Please choose a different name or remove the existing directory."
        Write-Host ""
        exit 1
    }
    
    # Read input file
    Write-Host "[INFO] Reading input file..." -ForegroundColor Cyan
    try {
        $content = Read-InputFile -FilePath $InputFile
    }
    catch {
        Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
    
    if ([string]::IsNullOrEmpty($content)) {
        Write-Host "[ERROR] Input file is empty or could not be read" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "[OK] Read $($content.Length) characters from input file" -ForegroundColor Green
    
    # Convert with Claude
    try {
        $conversionOutput = Invoke-ClaudeConversion -Content $content -ProjectName $ProjectName
    }
    catch {
        Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
    
    # Parse output
    Write-Host "[INFO] Processing conversion output..." -ForegroundColor Cyan
    $files = Split-ConversionOutput -Output $conversionOutput
    
    if ($files.Count -eq 0) {
        Write-Host "[WARN] Could not parse Claude output. Creating project with raw output." -ForegroundColor Yellow
        $files = @{
            "PROMPT.md" = $conversionOutput
        }
    }
    
    Write-Host "[OK] Extracted $($files.Count) files from conversion" -ForegroundColor Green
    
    # Create project structure
    Write-Host "[INFO] Creating project structure..." -ForegroundColor Cyan
    New-Item -ItemType Directory -Path $ProjectName -Force | Out-Null
    Push-Location $ProjectName
    
    try {
        # Create directories
        $directories = @("specs", "specs\stdlib", "src", "examples", "logs", "docs\generated", "docs\original")
        foreach ($dir in $directories) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
        
        # Write generated files
        foreach ($file in $files.GetEnumerator()) {
            $filePath = $file.Key
            $fileContent = $file.Value
            
            # Ensure directory exists
            $fileDir = Split-Path $filePath -Parent
            if ($fileDir -and -not (Test-Path $fileDir)) {
                New-Item -ItemType Directory -Path $fileDir -Force | Out-Null
            }
            
            $fileContent | Set-Content $filePath -Encoding UTF8
            Write-Host "[OK] Created: $filePath" -ForegroundColor Green
        }
        
        # Copy @AGENT.md template if available
        $agentTemplate = Join-Path $script:TemplatesDir "AGENT.md"
        if (Test-Path $agentTemplate) {
            Copy-Item $agentTemplate "@AGENT.md"
            Write-Host "[OK] Created: @AGENT.md" -ForegroundColor Green
        }
        else {
            # Create minimal @AGENT.md
            $agentContent = @"
# Agent Instructions

## Build Commands
``````bash
# Add your build commands here
``````

## Run Commands
``````bash
# Add your run commands here
``````

## Test Commands
``````bash
# Add your test commands here
``````
"@
            $agentContent | Set-Content "@AGENT.md" -Encoding UTF8
            Write-Host "[OK] Created: @AGENT.md (default)" -ForegroundColor Green
        }
        
        # Create .gitignore
        $gitignore = @"
.call_count
.last_reset
.exit_signals
.circuit_breaker_state
.circuit_breaker_history
.response_analysis
.last_output_length
progress.json
status.json
logs/
docs/generated/
node_modules/
__pycache__/
*.pyc
.venv/
"@
        $gitignore | Set-Content ".gitignore" -Encoding UTF8
        Write-Host "[OK] Created: .gitignore" -ForegroundColor Green
        
        # Store original PRD
        $originalFileName = Split-Path $InputFile -Leaf
        Copy-Item $InputFile "docs\original\$originalFileName"
        Write-Host "[OK] Saved original: docs\original\$originalFileName" -ForegroundColor Green
        
        # Create .gitkeep files
        $emptyDirs = @("src", "examples", "logs", "docs\generated", "specs\stdlib")
        foreach ($dir in $emptyDirs) {
            $gitkeep = Join-Path $dir ".gitkeep"
            New-Item -ItemType File -Path $gitkeep -Force | Out-Null
        }
        
        # Initialize git
        try {
            git init 2>&1 | Out-Null
            git add . 2>&1 | Out-Null
            git commit -m "Initial Hermes project from PRD import" 2>&1 | Out-Null
            Write-Host "[OK] Initialized git repository" -ForegroundColor Green
        }
        catch {
            Write-Host "[WARN] Could not initialize git repository" -ForegroundColor Yellow
        }
        
        # Success
        Write-Host ""
        Write-Host "Project '$ProjectName' created successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Generated files:" -ForegroundColor Cyan
        Write-Host "  - PROMPT.md          (Hermes development instructions)"
        Write-Host "  - @fix_plan.md       (Prioritized task list)"
        Write-Host "  - specs/requirements.md (Technical requirements)"
        Write-Host "  - @AGENT.md          (Build/run instructions)"
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "  1. cd $ProjectName"
        Write-Host "  2. Review and adjust the generated files"
        Write-Host "  3. Run: Hermes -Monitor"
        Write-Host ""
    }
    finally {
        Pop-Location
    }
}

# Main entry point
if ($Help) {
    Show-Help
    exit 0
}

if ([string]::IsNullOrEmpty($InputFile)) {
    Write-Host ""
    Write-Host "[ERROR] Input file is required" -ForegroundColor Red
    Write-Host ""
    Write-Host "Usage: Hermes-import <INPUT_FILE> [PROJECT_NAME]"
    Write-Host ""
    Write-Host "Run 'Hermes-import -Help' for more information"
    Write-Host ""
    exit 1
}

if (-not (Test-Path $InputFile)) {
    Write-Host ""
    Write-Host "[ERROR] File not found: $InputFile" -ForegroundColor Red
    Write-Host ""
    exit 1
}

New-HermesProjectFromPRD -InputFile $InputFile -ProjectName $ProjectName
