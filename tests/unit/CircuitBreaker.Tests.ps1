#Requires -Module Pester

<#
.SYNOPSIS
    Unit tests for CircuitBreaker.ps1 module
#>

BeforeAll {
    # Import the module under test
    . "$PSScriptRoot\..\..\lib\CircuitBreaker.ps1"
    
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

Describe "CircuitBreaker Module" {
    BeforeEach {
        # Clean up state files before each test
        Remove-Item ".circuit_breaker_state" -ErrorAction SilentlyContinue
        Remove-Item ".circuit_breaker_history" -ErrorAction SilentlyContinue
    }
    
    Context "Initialize-CircuitBreaker" {
        It "should create state file on initialization" {
            Initialize-CircuitBreaker
            ".circuit_breaker_state" | Should -Exist
        }
        
        It "should create history file on initialization" {
            Initialize-CircuitBreaker
            ".circuit_breaker_history" | Should -Exist
        }
        
        It "should initialize state to CLOSED" {
            Initialize-CircuitBreaker
            $state = Get-CircuitState
            $state | Should -Be "CLOSED"
        }
        
        It "should initialize consecutive_no_progress to 0" {
            Initialize-CircuitBreaker
            $data = Get-Content ".circuit_breaker_state" -Raw | ConvertFrom-Json
            $data.consecutive_no_progress | Should -Be 0
        }
        
        It "should handle corrupted state file" {
            "invalid json{{{" | Set-Content ".circuit_breaker_state"
            Initialize-CircuitBreaker
            $state = Get-CircuitState
            $state | Should -Be "CLOSED"
        }
    }
    
    Context "Get-CircuitState" {
        It "should return CLOSED when no state file exists" {
            $state = Get-CircuitState
            $state | Should -Be "CLOSED"
        }
        
        It "should return stored state" {
            @{
                state = "HALF_OPEN"
                consecutive_no_progress = 2
            } | ConvertTo-Json | Set-Content ".circuit_breaker_state"
            
            $state = Get-CircuitState
            $state | Should -Be "HALF_OPEN"
        }
    }
    
    Context "Test-CanExecute" {
        It "should allow execution when CLOSED" {
            Initialize-CircuitBreaker
            Test-CanExecute | Should -Be $true
        }
        
        It "should allow execution when HALF_OPEN" {
            @{
                state = "HALF_OPEN"
                consecutive_no_progress = 2
            } | ConvertTo-Json | Set-Content ".circuit_breaker_state"
            
            Test-CanExecute | Should -Be $true
        }
        
        It "should block execution when OPEN" {
            @{
                state = "OPEN"
                consecutive_no_progress = 3
            } | ConvertTo-Json | Set-Content ".circuit_breaker_state"
            
            Test-CanExecute | Should -Be $false
        }
    }
    
    Context "Add-LoopResult - State Transitions" {
        It "should stay CLOSED with progress" {
            Initialize-CircuitBreaker
            
            $result = Add-LoopResult -LoopNumber 1 -FilesChanged 5
            
            $result | Should -Be $true
            Get-CircuitState | Should -Be "CLOSED"
        }
        
        It "should stay CLOSED after 1 no-progress loop" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            
            Get-CircuitState | Should -Be "CLOSED"
        }
        
        It "should transition to HALF_OPEN after 2 no-progress loops" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            
            Get-CircuitState | Should -Be "HALF_OPEN"
        }
        
        It "should transition to OPEN after 3 no-progress loops" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            Add-LoopResult -LoopNumber 3 -FilesChanged 0
            
            Get-CircuitState | Should -Be "OPEN"
        }
        
        It "should return false when circuit opens" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            $result = Add-LoopResult -LoopNumber 3 -FilesChanged 0
            
            $result | Should -Be $false
        }
        
        It "should recover from HALF_OPEN to CLOSED on progress" {
            Initialize-CircuitBreaker
            
            # Get to HALF_OPEN
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            Get-CircuitState | Should -Be "HALF_OPEN"
            
            # Make progress
            Add-LoopResult -LoopNumber 3 -FilesChanged 5
            Get-CircuitState | Should -Be "CLOSED"
        }
        
        It "should reset no-progress counter on progress" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 5  # Progress!
            Add-LoopResult -LoopNumber 3 -FilesChanged 0
            
            # Should still be CLOSED because counter was reset
            Get-CircuitState | Should -Be "CLOSED"
        }
    }
    
    Context "Add-LoopResult - Error Tracking" {
        It "should track consecutive errors" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 1 -HasErrors $true
            
            $data = Get-Content ".circuit_breaker_state" -Raw | ConvertFrom-Json
            $data.consecutive_same_error | Should -Be 1
        }
        
        It "should reset error counter when no errors" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 1 -HasErrors $true
            Add-LoopResult -LoopNumber 2 -FilesChanged 1 -HasErrors $false
            
            $data = Get-Content ".circuit_breaker_state" -Raw | ConvertFrom-Json
            $data.consecutive_same_error | Should -Be 0
        }
    }
    
    Context "Reset-CircuitBreaker" {
        It "should reset state to CLOSED" {
            Initialize-CircuitBreaker
            
            # Get to OPEN state
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            Add-LoopResult -LoopNumber 3 -FilesChanged 0
            Get-CircuitState | Should -Be "OPEN"
            
            # Reset
            Reset-CircuitBreaker -Reason "Test reset"
            
            Get-CircuitState | Should -Be "CLOSED"
        }
        
        It "should reset all counters" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            
            Reset-CircuitBreaker
            
            $data = Get-Content ".circuit_breaker_state" -Raw | ConvertFrom-Json
            $data.consecutive_no_progress | Should -Be 0
            $data.consecutive_same_error | Should -Be 0
        }
        
        It "should store reset reason" {
            Initialize-CircuitBreaker
            
            Reset-CircuitBreaker -Reason "Manual test reset"
            
            $data = Get-Content ".circuit_breaker_state" -Raw | ConvertFrom-Json
            $data.reason | Should -Be "Manual test reset"
        }
    }
    
    Context "History Logging" {
        It "should log state transitions" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0  # Transition to HALF_OPEN
            
            $history = Get-Content ".circuit_breaker_history" -Raw | ConvertFrom-Json
            $history.Count | Should -BeGreaterThan 0
            $history[-1].to_state | Should -Be "HALF_OPEN"
        }
        
        It "should include loop number in history" {
            Initialize-CircuitBreaker
            
            Add-LoopResult -LoopNumber 1 -FilesChanged 0
            Add-LoopResult -LoopNumber 2 -FilesChanged 0
            
            $history = Get-Content ".circuit_breaker_history" -Raw | ConvertFrom-Json
            $history[-1].loop | Should -Be 2
        }
    }
    
    Context "Test-ShouldHalt" {
        It "should return false when CLOSED" {
            Initialize-CircuitBreaker
            Test-ShouldHalt | Should -Be $false
        }
        
        It "should return false when HALF_OPEN" {
            @{
                state = "HALF_OPEN"
                consecutive_no_progress = 2
                reason = "Monitoring"
            } | ConvertTo-Json | Set-Content ".circuit_breaker_state"
            
            Test-ShouldHalt | Should -Be $false
        }
        
        It "should return true when OPEN" {
            @{
                state = "OPEN"
                consecutive_no_progress = 3
                reason = "No progress"
                current_loop = 3
                last_progress_loop = 0
                total_opens = 1
            } | ConvertTo-Json | Set-Content ".circuit_breaker_state"
            
            Test-ShouldHalt | Should -Be $true
        }
    }
}
