#Requires -Module Pester

<#
.SYNOPSIS
    Integration tests for Hermes loop execution
#>

BeforeAll {
    $script:ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    
    # Store original location
    $script:OriginalLocation = Get-Location
    
    # Create temp directory for tests
    $script:TestDir = Join-Path $env:TEMP "HermesIntegration_$(Get-Random)"
    New-Item -ItemType Directory -Path $script:TestDir -Force | Out-Null
    Set-Location $script:TestDir
    
    # Import modules
    . "$script:ProjectRoot\lib\CircuitBreaker.ps1"
    . "$script:ProjectRoot\lib\ResponseAnalyzer.ps1"
}

AfterAll {
    Set-Location $script:OriginalLocation
    if (Test-Path $script:TestDir) {
        Remove-Item -Path $script:TestDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Describe "Loop Execution Integration" {
    BeforeEach {
        # Clean state for each test
        Get-ChildItem -Filter ".*" -Force -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
        Remove-Item "*.json" -Force -ErrorAction SilentlyContinue
        Remove-Item "*.log" -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path "logs" -Force | Out-Null
    }
    
    Context "Full Loop Cycle Simulation" {
        It "should complete a successful loop cycle" {
            # Initialize
            Initialize-CircuitBreaker
            
            # Simulate Claude output with progress
            $outputFile = "logs\claude_output_test.log"
            @"
Implementing new feature...
Created src/component.js
Added tests
---Hermes_STATUS---
STATUS: IN_PROGRESS
TASKS_COMPLETED_THIS_LOOP: 2
FILES_MODIFIED: 3
TESTS_STATUS: PASSING
WORK_TYPE: IMPLEMENTATION
EXIT_SIGNAL: false
RECOMMENDATION: Continue with next task
---END_Hermes_STATUS---
"@ | Set-Content $outputFile
            
            # Analyze response
            $analysisResult = Invoke-ResponseAnalysis -OutputFile $outputFile -LoopNumber 1
            $analysisResult | Should -Be $true
            
            # Update exit signals
            Update-ExitSignals
            
            # Record loop result (simulating file changes)
            $loopResult = Add-LoopResult -LoopNumber 1 -FilesChanged 3 -HasErrors $false -OutputLength 500
            $loopResult | Should -Be $true
            
            # Verify state
            Get-CircuitState | Should -Be "CLOSED"
            
            $analysis = Get-AnalysisResult
            $analysis.analysis.exit_signal | Should -Be $false
        }
        
        It "should detect project completion" {
            Initialize-CircuitBreaker
            
            $outputFile = "logs\claude_output_complete.log"
            @"
All tasks have been completed.
---Hermes_STATUS---
STATUS: COMPLETE
TASKS_COMPLETED_THIS_LOOP: 1
FILES_MODIFIED: 1
TESTS_STATUS: PASSING
WORK_TYPE: DOCUMENTATION
EXIT_SIGNAL: true
RECOMMENDATION: All requirements met, project ready for review
---END_Hermes_STATUS---
"@ | Set-Content $outputFile
            
            Invoke-ResponseAnalysis -OutputFile $outputFile -LoopNumber 5
            Update-ExitSignals
            
            $analysis = Get-AnalysisResult
            $analysis.analysis.exit_signal | Should -Be $true
            $analysis.analysis.confidence_score | Should -Be 100
        }
        
        It "should track multiple loops correctly" {
            Initialize-CircuitBreaker
            
            # Loop 1 - Implementation
            $output1 = "logs\claude_output_1.log"
            "Implementing feature A`nCreating files" | Set-Content $output1
            Invoke-ResponseAnalysis -OutputFile $output1 -LoopNumber 1
            Update-ExitSignals
            Add-LoopResult -LoopNumber 1 -FilesChanged 5
            
            # Loop 2 - More implementation
            $output2 = "logs\claude_output_2.log"
            "Implementing feature B`nAdding more code" | Set-Content $output2
            Invoke-ResponseAnalysis -OutputFile $output2 -LoopNumber 2
            Update-ExitSignals
            Add-LoopResult -LoopNumber 2 -FilesChanged 3
            
            # Loop 3 - Tests
            $output3 = "logs\claude_output_3.log"
            "Running npm test`nAll tests passed" | Set-Content $output3
            Invoke-ResponseAnalysis -OutputFile $output3 -LoopNumber 3
            Update-ExitSignals
            Add-LoopResult -LoopNumber 3 -FilesChanged 1
            
            # Verify state after 3 loops
            Get-CircuitState | Should -Be "CLOSED"
            
            $cbData = Get-CircuitBreakerData
            $cbData.current_loop | Should -Be 3
        }
    }
    
    Context "Exit Signal Accumulation" {
        It "should accumulate test-only loops" {
            Initialize-CircuitBreaker
            
            # 3 test-only loops
            for ($i = 1; $i -le 3; $i++) {
                $output = "logs\claude_output_$i.log"
                "Running npm test`nAll tests passed`npytest completed" | Set-Content $output
                Invoke-ResponseAnalysis -OutputFile $output -LoopNumber $i
                Update-ExitSignals
            }
            
            $signals = Get-ExitSignals
            @($signals.test_only_loops).Count | Should -Be 3
        }
        
        It "should accumulate done signals" {
            Initialize-CircuitBreaker
            
            # 2 completion signals
            for ($i = 1; $i -le 2; $i++) {
                $output = "logs\claude_output_$i.log"
                "Task is complete and done`nAll finished" | Set-Content $output
                Invoke-ResponseAnalysis -OutputFile $output -LoopNumber $i
                Update-ExitSignals
            }
            
            $signals = Get-ExitSignals
            @($signals.done_signals).Count | Should -Be 2
        }
        
        It "should clear test-only loops on implementation" {
            Initialize-CircuitBreaker
            
            # First, accumulate test-only loops
            for ($i = 1; $i -le 2; $i++) {
                $output = "logs\claude_output_$i.log"
                "Running npm test`nAll tests passed" | Set-Content $output
                Invoke-ResponseAnalysis -OutputFile $output -LoopNumber $i
                Update-ExitSignals
            }
            
            $signals = Get-ExitSignals
            @($signals.test_only_loops).Count | Should -Be 2
            
            # Now do implementation with progress
            $output = "logs\claude_output_3.log"
            @"
Implementing new feature
Creating component class
function getData() { }
"@ | Set-Content $output
            Invoke-ResponseAnalysis -OutputFile $output -LoopNumber 3
            
            # Manually set has_progress since we don't have git changes
            $analysisFile = ".response_analysis"
            $analysis = Get-Content $analysisFile -Raw | ConvertFrom-Json
            $analysis.analysis.has_progress = $true
            $analysis | ConvertTo-Json -Depth 10 | Set-Content $analysisFile
            
            Update-ExitSignals
            
            $signals = Get-ExitSignals
            # Should be cleared because has_progress was true
            @($signals.test_only_loops).Count | Should -Be 0
        }
    }
    
    Context "Circuit Breaker Integration" {
        It "should open circuit after consecutive no-progress loops" {
            Initialize-CircuitBreaker
            
            for ($i = 1; $i -le 3; $i++) {
                $output = "logs\claude_output_$i.log"
                "Some output without real progress" | Set-Content $output
                Invoke-ResponseAnalysis -OutputFile $output -LoopNumber $i
                Update-ExitSignals
                Add-LoopResult -LoopNumber $i -FilesChanged 0
            }
            
            Get-CircuitState | Should -Be "OPEN"
            Test-CanExecute | Should -Be $false
        }
        
        It "should recover circuit on progress" {
            Initialize-CircuitBreaker
            
            # Get to HALF_OPEN
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            Get-CircuitState | Should -Be "HALF_OPEN"
            
            # Make progress
            $output = "logs\claude_output_3.log"
            "Implementing feature`nCreating new files" | Set-Content $output
            Invoke-ResponseAnalysis -OutputFile $output -LoopNumber 3
            Add-LoopResult -LoopNumber 3 -FilesChanged 5
            
            Get-CircuitState | Should -Be "CLOSED"
        }
        
        It "should reset circuit breaker manually" {
            Initialize-CircuitBreaker
            
            # Open circuit
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            Add-LoopResult -LoopNumber 3 -FilesChanged 0
            Get-CircuitState | Should -Be "OPEN"
            
            # Reset
            Reset-CircuitBreaker -Reason "Manual reset for testing"
            
            Get-CircuitState | Should -Be "CLOSED"
            Test-CanExecute | Should -Be $true
        }
    }
    
    Context "Error Handling Integration" {
        It "should track errors across loops" {
            Initialize-CircuitBreaker
            
            for ($i = 1; $i -le 3; $i++) {
                $output = "logs\claude_output_$i.log"
                @"
Error: Something went wrong
Error: Failed to compile
Error: Cannot find module
Trying to fix...
"@ | Set-Content $output
                Invoke-ResponseAnalysis -OutputFile $output -LoopNumber $i
                Add-LoopResult -LoopNumber $i -FilesChanged 1 -HasErrors $true
            }
            
            $cbData = Get-CircuitBreakerData
            $cbData.consecutive_same_error | Should -Be 3
        }
        
        It "should detect stuck condition from analysis" {
            Initialize-CircuitBreaker
            
            $output = "logs\claude_output_stuck.log"
            @"
Error: Build failed
Error: Cannot resolve dependency
Error: Module not found
Error: Syntax error
Error: Type mismatch
Error: Runtime exception
Error: Stack overflow
"@ | Set-Content $output
            
            Invoke-ResponseAnalysis -OutputFile $output -LoopNumber 1
            
            $analysis = Get-AnalysisResult
            $analysis.analysis.is_stuck | Should -Be $true
            $analysis.analysis.error_count | Should -BeGreaterThan 5
        }
    }
    
    Context "Status File Integration" {
        It "should maintain consistent state across components" {
            Initialize-CircuitBreaker
            
            # Create a complete workflow
            $output = "logs\claude_output_1.log"
            @"
Working on the project...
Implementing features
---Hermes_STATUS---
STATUS: IN_PROGRESS
EXIT_SIGNAL: false
---END_Hermes_STATUS---
"@ | Set-Content $output
            
            Invoke-ResponseAnalysis -OutputFile $output -LoopNumber 1
            Update-ExitSignals
            Add-LoopResult -LoopNumber 1 -FilesChanged 3
            
            # Verify all state files exist and are valid JSON
            ".circuit_breaker_state" | Should -Exist
            ".response_analysis" | Should -Exist
            ".exit_signals" | Should -Exist
            
            # Verify JSON is valid
            { Get-Content ".circuit_breaker_state" -Raw | ConvertFrom-Json } | Should -Not -Throw
            { Get-Content ".response_analysis" -Raw | ConvertFrom-Json } | Should -Not -Throw
            { Get-Content ".exit_signals" -Raw | ConvertFrom-Json } | Should -Not -Throw
        }
    }
}

Describe "Fix Plan Completion Detection" {
    BeforeEach {
        Get-ChildItem -Filter ".*" -Force -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue
        Remove-Item "*.md" -Force -ErrorAction SilentlyContinue
    }
    
    Context "@fix_plan.md Integration" {
        It "should detect all items completed" {
            # Create completed fix plan
            @"
# Fix Plan

## Tasks
- [x] Task 1
- [x] Task 2
- [x] Task 3
"@ | Set-Content "@fix_plan.md"
            
            # Initialize signals
            @{
                test_only_loops = @()
                done_signals = @()
                completion_indicators = @()
            } | ConvertTo-Json | Set-Content ".exit_signals"
            
            # Read and check fix plan
            $content = Get-Content "@fix_plan.md" -Raw
            $totalItems = ([regex]::Matches($content, "(?m)^- \[")).Count
            $completedItems = ([regex]::Matches($content, "(?mi)^- \[x\]")).Count
            
            $totalItems | Should -Be 3
            $completedItems | Should -Be 3
            $completedItems | Should -Be $totalItems
        }
        
        It "should detect incomplete items" {
            @"
# Fix Plan

## Tasks
- [x] Task 1 (done)
- [ ] Task 2 (pending)
- [x] Task 3 (done)
"@ | Set-Content "@fix_plan.md"
            
            $content = Get-Content "@fix_plan.md" -Raw
            $totalItems = ([regex]::Matches($content, "(?m)^- \[")).Count
            $completedItems = ([regex]::Matches($content, "(?mi)^- \[x\]")).Count
            
            $totalItems | Should -Be 3
            $completedItems | Should -Be 2
            $completedItems | Should -Not -Be $totalItems
        }
    }
}
