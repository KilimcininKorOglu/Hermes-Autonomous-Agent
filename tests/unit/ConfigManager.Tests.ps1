$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$lib = Join-Path (Split-Path -Parent (Split-Path -Parent $here)) "lib"
. "$lib\ConfigManager.ps1"

Describe "ConfigManager" {
    
    Context "Get-DefaultConfig" {
        It "Should return default configuration" {
            $config = Get-DefaultConfig
            
            $config | Should Not BeNullOrEmpty
            $config.ai.provider | Should Be "auto"
            $config.ai.timeout | Should Be 300
            $config.ai.maxRetries | Should Be 10
            $config.taskMode.autoBranch | Should Be $false
            $config.taskMode.autoCommit | Should Be $false
            $config.loop.maxCallsPerHour | Should Be 100
            $config.paths.tasksDir | Should Be "tasks"
        }
    }
    
    Context "Get-GlobalConfigPath" {
        It "Should return path in LOCALAPPDATA" {
            $path = Get-GlobalConfigPath
            
            $path | Should Match "Hermes\\config\.json$"
            $path | Should Match $env:LOCALAPPDATA.Replace("\", "\\")
        }
    }
    
    Context "Get-ProjectConfigPath" {
        It "Should return hermes.config.json in base path" {
            $path = Get-ProjectConfigPath -BasePath "C:\test"
            
            $path | Should Be "C:\test\hermes.config.json"
        }
        
        It "Should default to current directory" {
            $path = Get-ProjectConfigPath
            
            $path | Should Be ".\hermes.config.json"
        }
    }
    
    Context "Merge-ConfigHashtable" {
        It "Should merge simple values" {
            $base = @{ a = 1; b = 2 }
            $override = @{ b = 3; c = 4 }
            
            $result = Merge-ConfigHashtable -Base $base -Override $override
            
            $result.a | Should Be 1
            $result.b | Should Be 3
            $result.c | Should Be 4
        }
        
        It "Should deep merge nested hashtables" {
            $base = @{
                ai = @{ provider = "auto"; timeout = 300 }
                loop = @{ max = 100 }
            }
            $override = @{
                ai = @{ timeout = 600 }
            }
            
            $result = Merge-ConfigHashtable -Base $base -Override $override
            
            $result.ai.provider | Should Be "auto"
            $result.ai.timeout | Should Be 600
            $result.loop.max | Should Be 100
        }
    }
    
    Context "ConvertTo-Hashtable" {
        It "Should convert PSCustomObject to hashtable" {
            $json = '{"a": 1, "b": {"c": 2}}'
            $obj = $json | ConvertFrom-Json
            
            $result = ConvertTo-Hashtable -InputObject $obj
            
            $result.GetType().Name | Should Be "Hashtable"
            $result.a | Should Be 1
            $result.b.GetType().Name | Should Be "Hashtable"
            $result.b.c | Should Be 2
        }
        
        It "Should handle null input" {
            $result = ConvertTo-Hashtable -InputObject $null
            
            $result.GetType().Name | Should Be "Hashtable"
            $result.Count | Should Be 0
        }
    }
    
    Context "Read-ConfigFile" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path $script:testDir -Force | Out-Null
        }
        
        AfterEach {
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should read valid JSON config" {
            $configPath = Join-Path $script:testDir "config.json"
            '{"ai": {"provider": "claude"}}' | Set-Content $configPath
            
            $result = Read-ConfigFile -Path $configPath
            
            $result | Should Not BeNullOrEmpty
            $result.ai.provider | Should Be "claude"
        }
        
        It "Should return null for non-existent file" {
            $result = Read-ConfigFile -Path "nonexistent.json"
            
            $result | Should BeNullOrEmpty
        }
        
        It "Should return null for invalid JSON" {
            $configPath = Join-Path $script:testDir "invalid.json"
            "not valid json {" | Set-Content $configPath
            
            $result = Read-ConfigFile -Path $configPath
            
            $result | Should BeNullOrEmpty
        }
    }
    
    Context "Get-HermesConfig" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path $script:testDir -Force | Out-Null
        }
        
        AfterEach {
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return defaults when no config files exist" {
            $config = Get-HermesConfig -BasePath $script:testDir
            
            $config.ai.provider | Should Be "auto"
            $config.ai.timeout | Should Be 300
        }
        
        It "Should merge project config over defaults" {
            $projectConfig = Join-Path $script:testDir "hermes.config.json"
            '{"ai": {"provider": "droid", "timeout": 600}}' | Set-Content $projectConfig
            
            $config = Get-HermesConfig -BasePath $script:testDir
            
            $config.ai.provider | Should Be "droid"
            $config.ai.timeout | Should Be 600
            $config.ai.maxRetries | Should Be 10
        }
    }
    
    Context "Get-ConfigValue" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path $script:testDir -Force | Out-Null
        }
        
        AfterEach {
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should get value using dot notation" {
            $value = Get-ConfigValue -Key "ai.provider" -BasePath $script:testDir
            
            $value | Should Be "auto"
        }
        
        It "Should get nested value" {
            $value = Get-ConfigValue -Key "taskMode.maxConsecutiveErrors" -BasePath $script:testDir
            
            $value | Should Be 5
        }
        
        It "Should return override when provided" {
            $value = Get-ConfigValue -Key "ai.provider" -Override "claude" -BasePath $script:testDir
            
            $value | Should Be "claude"
        }
        
        It "Should return null for non-existent key" {
            $value = Get-ConfigValue -Key "nonexistent.key" -BasePath $script:testDir
            
            $value | Should BeNullOrEmpty
        }
        
        It "Should ignore empty string override" {
            $value = Get-ConfigValue -Key "ai.provider" -Override "" -BasePath $script:testDir
            
            $value | Should Be "auto"
        }
    }
    
    Context "Initialize-DefaultConfig" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$script:testDir\Hermes" -Force | Out-Null
            $script:originalLocalAppData = $env:LOCALAPPDATA
            $env:LOCALAPPDATA = $script:testDir
        }
        
        AfterEach {
            $env:LOCALAPPDATA = $script:originalLocalAppData
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should create default config file" {
            $result = Initialize-DefaultConfig
            
            $result | Should Be $true
            $configPath = Get-GlobalConfigPath
            Test-Path $configPath | Should Be $true
        }
        
        It "Should not overwrite existing config without Force" {
            $configPath = Get-GlobalConfigPath
            "existing" | Set-Content $configPath
            
            $result = Initialize-DefaultConfig
            
            $result | Should Be $false
            Get-Content $configPath | Should Be "existing"
        }
        
        It "Should overwrite with Force" {
            $configPath = Get-GlobalConfigPath
            "existing" | Set-Content $configPath
            
            $result = Initialize-DefaultConfig -Force
            
            $result | Should Be $true
            $content = Get-Content $configPath -Raw | ConvertFrom-Json
            $content.ai.provider | Should Be "auto"
        }
    }
    
    Context "Initialize-ProjectConfig" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path $script:testDir -Force | Out-Null
        }
        
        AfterEach {
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should create project config file" {
            $result = Initialize-ProjectConfig -BasePath $script:testDir
            
            $result | Should Be $true
            $configPath = Join-Path $script:testDir "hermes.config.json"
            Test-Path $configPath | Should Be $true
        }
        
        It "Should not overwrite existing without Force" {
            $configPath = Join-Path $script:testDir "hermes.config.json"
            "existing" | Set-Content $configPath
            
            $result = Initialize-ProjectConfig -BasePath $script:testDir
            
            $result | Should Be $false
        }
    }
    
    Context "Set-ConfigValue" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$script:testDir\Hermes" -Force | Out-Null
            $script:originalLocalAppData = $env:LOCALAPPDATA
            $env:LOCALAPPDATA = $script:testDir
        }
        
        AfterEach {
            $env:LOCALAPPDATA = $script:originalLocalAppData
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should set value in global config" {
            Set-ConfigValue -Key "ai.provider" -Value "claude" -Scope "global"
            
            $configPath = Get-GlobalConfigPath
            $config = Get-Content $configPath -Raw | ConvertFrom-Json
            $config.ai.provider | Should Be "claude"
        }
        
        It "Should set value in project config" {
            $projectDir = Join-Path $script:testDir "project"
            New-Item -ItemType Directory -Path $projectDir -Force | Out-Null
            
            Set-ConfigValue -Key "ai.timeout" -Value 600 -Scope "project" -BasePath $projectDir
            
            $configPath = Join-Path $projectDir "hermes.config.json"
            $config = Get-Content $configPath -Raw | ConvertFrom-Json
            $config.ai.timeout | Should Be 600
        }
        
        It "Should convert string true to boolean" {
            Set-ConfigValue -Key "taskMode.autoBranch" -Value "true" -Scope "global"
            
            $configPath = Get-GlobalConfigPath
            $config = Get-Content $configPath -Raw | ConvertFrom-Json
            $config.taskMode.autoBranch | Should Be $true
        }
        
        It "Should convert numeric string to integer" {
            Set-ConfigValue -Key "ai.timeout" -Value "500" -Scope "global"
            
            $configPath = Get-GlobalConfigPath
            $config = Get-Content $configPath -Raw | ConvertFrom-Json
            $config.ai.timeout | Should Be 500
        }
    }
    
    Context "Test-ConfigExists" {
        BeforeEach {
            $script:testDir = Join-Path $env:TEMP "hermes-config-test-$(Get-Random)"
            New-Item -ItemType Directory -Path "$script:testDir\Hermes" -Force | Out-Null
            $script:originalLocalAppData = $env:LOCALAPPDATA
            $env:LOCALAPPDATA = $script:testDir
        }
        
        AfterEach {
            $env:LOCALAPPDATA = $script:originalLocalAppData
            Remove-Item -Recurse -Force $script:testDir -ErrorAction SilentlyContinue
        }
        
        It "Should return false when global config does not exist" {
            $result = Test-ConfigExists -Scope "global"
            
            $result | Should Be $false
        }
        
        It "Should return true when global config exists" {
            $configPath = Get-GlobalConfigPath
            "{}" | Set-Content $configPath
            
            $result = Test-ConfigExists -Scope "global"
            
            $result | Should Be $true
        }
        
        It "Should check project config" {
            $projectDir = Join-Path $script:testDir "project"
            New-Item -ItemType Directory -Path $projectDir -Force | Out-Null
            
            $result = Test-ConfigExists -Scope "project" -BasePath $projectDir
            
            $result | Should Be $false
        }
    }
}
