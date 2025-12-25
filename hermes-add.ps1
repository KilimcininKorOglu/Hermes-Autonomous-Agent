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
. "$scriptDir\lib\Logger.ps1"
. "$scriptDir\lib\ConfigManager.ps1"
. "$scriptDir\lib\AIProvider.ps1"
. "$scriptDir\lib\FeatureAnalyzer.ps1"

# Load configuration
$hermesConfig = Get-HermesConfig

# Initialize logger
Initialize-Logger -Command "hermes-add" | Out-Null

function Show-Usage {
    Write-Host ""
    Write-Host "Hermes Autonomous Agent - Feature Addition" -ForegroundColor Cyan
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
Write-Host "Hermes Autonomous Agent - Feature Addition" -ForegroundColor Cyan
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

# Determine AI provider (CLI > config > auto-detect)
$configProvider = Get-ConfigValue -Key "ai.provider"
if ($AI -eq "auto" -and $configProvider -ne "auto") {
    $AI = $configProvider
} elseif ($AI -eq "auto") {
    $AI = Get-AutoProvider
    if (-not $AI) {
        Write-Error "No AI provider found. Install claude, droid, or aider."
        exit 1
    }
}

# Get timeout from config if not overridden
$configTimeout = Get-ConfigValue -Key "ai.timeout"
$configMaxRetries = Get-ConfigValue -Key "ai.maxRetries"
if ($Timeout -eq 300 -and $configTimeout) { $Timeout = $configTimeout }
if ($MaxRetries -eq 3 -and $configMaxRetries) { $MaxRetries = $configMaxRetries }

if (-not (Test-AIProvider -Provider $AI)) {
    Write-Log -Level "ERROR" -Message "AI provider '$AI' is not installed or not in PATH"
    Close-Logger -Success $false
    exit 1
}

Write-Log -Level "INFO" -Message "Using AI: $AI (timeout: ${Timeout}s)"

# Build prompt
Write-Log -Level "INFO" -Message "Building analysis prompt..." -NoConsole
try {
    $prompt = Build-FeaturePrompt -FeatureDescription $featureInput.Content `
        -NextFeatureId $ids.NextFeatureId `
        -NextTaskId $ids.NextTaskId `
        -PriorityOverride $Priority
}
catch {
    Write-Log -Level "ERROR" -Message "Failed to build prompt: $_"
    Close-Logger -Success $false
    exit 1
}

# Call AI
Write-Log -Level "INFO" -Message "Analyzing feature with $AI..."
Write-Host ""

$result = Invoke-AIWithRetry -Provider $AI `
    -PromptText $prompt `
    -Content $featureInput.Content `
    -MaxRetries $MaxRetries `
    -TimeoutSeconds $Timeout

if (-not $result.Success) {
    Write-Log -Level "ERROR" -Message "Failed to analyze feature: $($result.Error)"
    Close-Logger -Success $false
    exit 1
}

# Parse output
Write-Log -Level "INFO" -Message "Parsing AI output..." -NoConsole
$feature = Parse-FeatureOutput -Output $result.Raw

if (-not $feature) {
    Write-Log -Level "ERROR" -Message "Failed to parse AI output. No valid feature file found."
    Close-Logger -Success $false
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
    Write-Log -Level "INFO" -Message "DryRun completed" -NoConsole
    Close-Logger -Success $true
    exit 0
}

# Write feature file
Write-Log -Level "INFO" -Message "Creating feature file..." -NoConsole
$filePath = Write-FeatureFile -Feature $feature -OutputDir $OutputDir

Write-Log -Level "SUCCESS" -Message "Created: $filePath"

# Update tasks-status.md
Write-Log -Level "INFO" -Message "Updating tasks-status.md..." -NoConsole
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
Write-Host "Next: Run 'hermes -TaskMode -AutoBranch -AutoCommit' to implement" -ForegroundColor Cyan
Write-Host ""

Write-Log -Level "SUCCESS" -Message "Feature $($feature.FeatureId) added with $($feature.TaskCount) tasks" -NoConsole
Close-Logger -Success $true
