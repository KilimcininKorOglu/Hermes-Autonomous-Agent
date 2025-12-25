<#
.SYNOPSIS
    Hermes Add - Add a single feature to task plan
.DESCRIPTION
    Analyzes a feature description and creates a task file with breakdown.
    Works with existing tasks, continuing from highest Feature/Task IDs.
.EXAMPLE
    hermes-add "kullanici kayit sistemi"
    hermes-add @docs/webhook-spec.md
    hermes-add "sifre sifirlama" -Priority P1
#>

param(
    [Parameter(Position = 0)]
    [string]$Feature,
    
    [ValidateSet("claude", "droid", "aider", "auto")]
    [string]$AI = "auto",
    
    [switch]$DryRun,
    
    [string]$OutputDir = "tasks",
    
    [int]$Timeout = 300,
    
    [int]$MaxRetries = 3,
    
    [ValidateSet("P1", "P2", "P3", "P4", "")]
    [string]$Priority = "",
    
    [switch]$Help
)

$ErrorActionPreference = "Stop"

# Get script directory
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Import modules
. "$scriptDir\lib\AIProvider.ps1"
. "$scriptDir\lib\FeatureAnalyzer.ps1"

function Show-Usage {
    Write-Host ""
    Write-Host "Hermes Add - Single Feature Addition" -ForegroundColor Cyan
    Write-Host "====================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage:" -ForegroundColor Yellow
    Write-Host "  hermes-add <feature-description>"
    Write-Host "  hermes-add @<file-path>"
    Write-Host "  hermes-add <description> -Priority P1"
    Write-Host ""
    Write-Host "Parameters:" -ForegroundColor Yellow
    Write-Host "  <feature>      Feature description or @filepath"
    Write-Host "  -AI            AI provider: claude, droid, aider, auto (default: auto)"
    Write-Host "  -DryRun        Show what would be created without writing"
    Write-Host "  -OutputDir     Output directory (default: tasks)"
    Write-Host "  -Timeout       AI timeout in seconds (default: 300)"
    Write-Host "  -Priority      Override priority: P1, P2, P3, P4"
    Write-Host "  -Help          Show this help"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  hermes-add `"kullanici kayit sistemi`""
    Write-Host "  hermes-add @docs/webhook-spec.md"
    Write-Host "  hermes-add `"sifre sifirlama`" -Priority P1"
    Write-Host "  hermes-add `"email dogrulama`" -DryRun"
    Write-Host ""
}

# Show help
if ($Help -or -not $Feature) {
    Show-Usage
    exit 0
}

Write-Host ""
Write-Host "Hermes Add - Single Feature Addition" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host ""

# Read feature input
try {
    Write-Host "[INFO] Reading feature input..." -ForegroundColor Cyan
    $featureInput = Read-FeatureInput -InputText $Feature
    
    if ($featureInput.Type -eq "file") {
        Write-Host "[INFO] Source: $($featureInput.Path)" -ForegroundColor Gray
        Write-Host "[INFO] Size: $($featureInput.Content.Length) characters" -ForegroundColor Gray
    }
    else {
        Write-Host "[INFO] Source: inline description" -ForegroundColor Gray
    }
}
catch {
    Write-Error "Failed to read input: $_"
    exit 1
}

# Get next IDs
Write-Host "[INFO] Checking existing tasks..." -ForegroundColor Cyan
$ids = Get-NextIds -BasePath "."

Write-Host "[INFO] Next Feature ID: F$($ids.NextFeatureIdPadded)" -ForegroundColor Gray
Write-Host "[INFO] Next Task ID: T$($ids.NextTaskIdPadded)" -ForegroundColor Gray

# Determine AI provider
if ($AI -eq "auto") {
    $AI = Get-AutoProvider
    if (-not $AI) {
        Write-Error "No AI provider found. Install claude, droid, or aider."
        exit 1
    }
}

if (-not (Test-AIProvider -Provider $AI)) {
    Write-Error "AI provider '$AI' is not installed or not in PATH"
    exit 1
}

Write-Host "[INFO] Using AI: $AI" -ForegroundColor Cyan

# Build prompt
Write-Host "[INFO] Building analysis prompt..." -ForegroundColor Cyan
try {
    $prompt = Build-FeaturePrompt -FeatureDescription $featureInput.Content `
        -NextFeatureId $ids.NextFeatureId `
        -NextTaskId $ids.NextTaskId `
        -PriorityOverride $Priority
}
catch {
    Write-Error "Failed to build prompt: $_"
    exit 1
}

# Call AI
Write-Host "[INFO] Analyzing feature with $AI..." -ForegroundColor Cyan
Write-Host ""

$result = Invoke-AIWithRetry -Provider $AI `
    -PromptText $prompt `
    -Content $featureInput.Content `
    -MaxRetries $MaxRetries `
    -TimeoutSeconds $Timeout

if (-not $result.Success) {
    Write-Error "Failed to analyze feature: $($result.Error)"
    exit 1
}

# Parse output
Write-Host "[INFO] Parsing AI output..." -ForegroundColor Cyan
$feature = Parse-FeatureOutput -Output $result.Raw

if (-not $feature) {
    Write-Error "Failed to parse AI output. No valid feature file found."
    exit 1
}

# DryRun mode
if ($DryRun) {
    Write-Host ""
    Write-Host "DryRun Mode - Would create:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  File: $($feature.FileName)" -ForegroundColor White
    Write-Host "  Feature ID: $($feature.FeatureId)" -ForegroundColor Gray
    Write-Host "  Feature Name: $($feature.FeatureName)" -ForegroundColor Gray
    Write-Host "  Priority: $($feature.Priority)" -ForegroundColor Gray
    Write-Host "  Tasks: $($feature.TaskCount) ($($feature.TaskRange))" -ForegroundColor Gray
    Write-Host "  Effort: $($feature.TotalEffort) days" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Run without -DryRun to create the file." -ForegroundColor Cyan
    exit 0
}

# Write feature file
Write-Host "[INFO] Creating feature file..." -ForegroundColor Cyan
$filePath = Write-FeatureFile -Feature $feature -OutputDir $OutputDir

Write-Host "[OK] Created: $filePath" -ForegroundColor Green

# Update tasks-status.md
Write-Host "[INFO] Updating tasks-status.md..." -ForegroundColor Cyan
Update-TasksStatus -Feature $feature -OutputDir $OutputDir

# Success output
Write-Host ""
Write-Host ("=" * 50) -ForegroundColor Green
Write-Host "  Feature added!" -ForegroundColor Green
Write-Host ("=" * 50) -ForegroundColor Green
Write-Host ""
Write-Host "  Feature ID: $($feature.FeatureId)" -ForegroundColor White
Write-Host "  File:       $filePath" -ForegroundColor White
Write-Host "  Name:       $($feature.FeatureName)" -ForegroundColor White
Write-Host "  Priority:   $($feature.Priority)" -ForegroundColor White
Write-Host "  Tasks:      $($feature.TaskCount) ($($feature.TaskRange))" -ForegroundColor White
Write-Host "  Effort:     $($feature.TotalEffort) days (total)" -ForegroundColor White
Write-Host ""
Write-Host ("=" * 50) -ForegroundColor Green
Write-Host ""
Write-Host "Next: Run 'Hermes -TaskMode -AutoBranch -AutoCommit' to implement" -ForegroundColor Cyan
Write-Host ""
