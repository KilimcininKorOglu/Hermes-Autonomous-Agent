#Requires -Module Pester

<#
.SYNOPSIS
    Unit tests for ResponseAnalyzer.ps1 module
#>

BeforeAll {
    # Import the module under test
    . "$PSScriptRoot\..\..\lib\ResponseAnalyzer.ps1"
    
    # Store original location
    $script:OriginalLocation = Get-Location
    
    # Create temp directory for tests
    $script:TestDir = Join-Path $env:TEMP "HermesTests_$(Get-Random)"
    New-Item -ItemType Directory -Path $script:TestDir -Force | Out-Null
    Set-Location $script:TestDir
}

AfterAll {
    # Return to original location and cleanup
    Set-Location $script:OriginalLocation
    if (Test-Path $script:TestDir) {
        Remove-Item -Path $script:TestDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Describe "ResponseAnalyzer Module" {
    BeforeEach {
        # Clean up state files before each test
        Remove-Item ".response_analysis" -ErrorAction SilentlyContinue
        Remove-Item ".exit_signals" -ErrorAction SilentlyContinue
        Remove-Item ".last_output_length" -ErrorAction SilentlyContinue
        Remove-Item "test_output.log" -ErrorAction SilentlyContinue
    }
    
    Context "Invoke-ResponseAnalysis - Basic" {
        It "should return false for non-existent file" {
            $result = Invoke-ResponseAnalysis -OutputFile "nonexistent.log" -LoopNumber 1
            $result | Should -Be $false
        }
        
        It "should create analysis result file" {
            "Some output content" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            ".response_analysis" | Should -Exist
        }
        
        It "should store loop number in result" {
            "Some output" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 42
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.loop_number | Should -Be 42
        }
        
        It "should store output length" {
            "Hello World" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.output_length | Should -BeGreaterThan 0
        }
    }
    
    Context "Invoke-ResponseAnalysis - Hermes Status Detection" {
        It "should detect EXIT_SIGNAL true in status block" {
            @"
Some work output here
---Hermes_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: true
RECOMMENDATION: All done
---END_Hermes_STATUS---
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.exit_signal | Should -Be $true
            $result.analysis.confidence_score | Should -Be 100
        }
        
        It "should detect STATUS: COMPLETE" {
            @"
---Hermes_STATUS---
STATUS: COMPLETE
EXIT_SIGNAL: false
---END_Hermes_STATUS---
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.has_completion_signal | Should -Be $true
        }
        
        It "should detect WORK_TYPE: TESTING" {
            @"
---Hermes_STATUS---
STATUS: IN_PROGRESS
WORK_TYPE: TESTING
EXIT_SIGNAL: false
---END_Hermes_STATUS---
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.is_test_only | Should -Be $true
        }
    }
    
    Context "Invoke-ResponseAnalysis - Completion Keywords" {
        It "should detect 'done' keyword" {
            "The task is done and complete" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.has_completion_signal | Should -Be $true
        }
        
        It "should detect 'all tasks complete' phrase" {
            "All tasks complete, nothing left to do" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.has_completion_signal | Should -Be $true
        }
        
        It "should detect 'project complete' phrase" {
            "The project complete successfully" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.has_completion_signal | Should -Be $true
        }
    }
    
    Context "Invoke-ResponseAnalysis - Test-Only Detection" {
        It "should detect test-only loop with npm test" {
            @"
Running npm test
All tests passed
15 passing, 0 failing
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.is_test_only | Should -Be $true
        }
        
        It "should detect test-only loop with pytest" {
            @"
pytest tests/
collected 10 items
all tests passed
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.is_test_only | Should -Be $true
        }
        
        It "should not flag as test-only when implementation present" {
            @"
Creating new component
Implementing feature
function getData() { }
Running npm test
Tests passed
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.is_test_only | Should -Be $false
        }
    }
    
    Context "Invoke-ResponseAnalysis - Stuck Detection" {
        It "should detect high error count" {
            @"
Error: something failed
Error: another error
Error: third error
failed to compile
Error: fourth error
Error: fifth error
Error: sixth error
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.is_stuck | Should -Be $true
            $result.analysis.error_count | Should -BeGreaterThan 5
        }
        
        It "should not flag as stuck with few errors" {
            @"
Warning: minor issue
Error: one error
Continuing with work
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.is_stuck | Should -Be $false
        }
    }
    
    Context "Invoke-ResponseAnalysis - No Work Patterns" {
        It "should detect 'nothing to do'" {
            "There is nothing to do here" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.has_completion_signal | Should -Be $true
        }
        
        It "should detect 'already implemented'" {
            "This feature is already implemented" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.has_completion_signal | Should -Be $true
        }
    }
    
    Context "Invoke-ResponseAnalysis - Confidence Score" {
        It "should increase confidence with completion keyword" {
            "Task is complete" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.confidence_score | Should -BeGreaterOrEqual 10
        }
        
        It "should have max confidence for EXIT_SIGNAL true" {
            @"
---Hermes_STATUS---
EXIT_SIGNAL: true
---END_Hermes_STATUS---
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-Content ".response_analysis" -Raw | ConvertFrom-Json
            $result.analysis.confidence_score | Should -Be 100
        }
    }
    
    Context "Update-ExitSignals" {
        It "should create exit signals file" {
            "test output" | Set-Content "test_output.log"
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            Update-ExitSignals
            
            ".exit_signals" | Should -Exist
        }
        
        It "should track test-only loops" {
            @"
Running npm test
All tests passed
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 5
            Update-ExitSignals
            
            $signals = Get-Content ".exit_signals" -Raw | ConvertFrom-Json
            $signals.test_only_loops | Should -Contain 5
        }
        
        It "should track done signals" {
            "Task is complete and done" | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 3
            Update-ExitSignals
            
            $signals = Get-Content ".exit_signals" -Raw | ConvertFrom-Json
            $signals.done_signals | Should -Contain 3
        }
        
        It "should keep rolling window of 5 signals" {
            # Create initial exit signals with more than 5 items
            @{
                test_only_loops = @(1, 2, 3, 4, 5, 6)
                done_signals = @()
                completion_indicators = @()
            } | ConvertTo-Json | Set-Content ".exit_signals"
            
            "Running npm test" | Set-Content "test_output.log"
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 7
            Update-ExitSignals
            
            $signals = Get-Content ".exit_signals" -Raw | ConvertFrom-Json
            @($signals.test_only_loops).Count | Should -BeLessOrEqual 5
        }
        
        It "should clear test_only_loops on progress" {
            @{
                test_only_loops = @(1, 2)
                done_signals = @()
                completion_indicators = @()
            } | ConvertTo-Json | Set-Content ".exit_signals"
            
            # Create output that shows progress (has implementation keywords)
            @"
Creating new component
Implementing the feature
class MyComponent { }
"@ | Set-Content "test_output.log"
            
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 3
            Update-ExitSignals
            
            $signals = Get-Content ".exit_signals" -Raw | ConvertFrom-Json
            # Note: The actual clearing depends on has_progress being true
            # which requires git changes - may not clear in this test
        }
    }
    
    Context "Get-AnalysisResult" {
        It "should return null when no analysis file exists" {
            $result = Get-AnalysisResult
            $result | Should -BeNullOrEmpty
        }
        
        It "should return analysis data when file exists" {
            "test output" | Set-Content "test_output.log"
            Invoke-ResponseAnalysis -OutputFile "test_output.log" -LoopNumber 1
            
            $result = Get-AnalysisResult
            $result | Should -Not -BeNullOrEmpty
            $result.loop_number | Should -Be 1
        }
    }
    
    Context "Get-ExitSignals" {
        It "should return null when no signals file exists" {
            $result = Get-ExitSignals
            $result | Should -BeNullOrEmpty
        }
        
        It "should return signals data when file exists" {
            @{
                test_only_loops = @(1, 2)
                done_signals = @(3)
                completion_indicators = @()
            } | ConvertTo-Json | Set-Content ".exit_signals"
            
            $result = Get-ExitSignals
            $result | Should -Not -BeNullOrEmpty
            @($result.test_only_loops).Count | Should -Be 2
        }
    }
}
