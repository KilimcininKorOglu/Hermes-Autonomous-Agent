# Plan 08: Configuration System

## Overview

Merkezi ve proje bazlı config sistemi. Proje config'i öncelikli.

## Config Öncelik Sırası (Yüksekten Düşüğe)

1. **Komut satırı parametreleri** - `-AI claude`, `-Timeout 300`
2. **Proje config** - `./hermes.config.json`
3. **Merkezi config** - `$env:LOCALAPPDATA\Hermes\config.json`
4. **Varsayılan değerler** - Kod içinde hardcoded

## Config Dosya Konumları

| Tip | Konum |
|-----|-------|
| Merkezi | `$env:LOCALAPPDATA\Hermes\config.json` |
| Proje | `./hermes.config.json` (proje kök dizini) |

## Config Yapısı

```json
{
  "ai": {
    "provider": "auto",
    "timeout": 300,
    "maxRetries": 10
  },
  "taskMode": {
    "autoBranch": false,
    "autoCommit": false,
    "autonomous": false,
    "maxConsecutiveErrors": 5
  },
  "loop": {
    "maxCallsPerHour": 100,
    "timeoutMinutes": 15
  },
  "paths": {
    "tasksDir": "tasks",
    "logsDir": "logs"
  }
}
```

## Varsayılan Değerler

```json
{
  "ai": {
    "provider": "auto",
    "timeout": 300,
    "maxRetries": 10
  },
  "taskMode": {
    "autoBranch": false,
    "autoCommit": false,
    "autonomous": false,
    "maxConsecutiveErrors": 5
  },
  "loop": {
    "maxCallsPerHour": 100,
    "timeoutMinutes": 15
  },
  "paths": {
    "tasksDir": "tasks",
    "logsDir": "logs"
  }
}
```

## Yeni Modül: lib/ConfigManager.ps1

### Fonksiyonlar

| Fonksiyon | Açıklama |
|-----------|----------|
| `Get-HermesConfig` | Tüm config'i merge ederek döner |
| `Get-ConfigValue` | Tek bir değer alır (dot notation: `ai.provider`) |
| `Set-GlobalConfig` | Merkezi config'i günceller |
| `Set-ProjectConfig` | Proje config'ini günceller |
| `Initialize-DefaultConfig` | Varsayılan merkezi config oluşturur |
| `Test-ConfigExists` | Config dosyası var mı kontrol eder |

### Örnek Kullanım

```powershell
# Config'i yükle
$config = Get-HermesConfig

# Tek değer al
$provider = Get-ConfigValue -Key "ai.provider"
$timeout = Get-ConfigValue -Key "ai.timeout"

# Parametre override ile
$provider = Get-ConfigValue -Key "ai.provider" -Override $AI
```

## Değişiklikler

### install.ps1

```powershell
# Kurulum sırasında varsayılan config oluştur
$configPath = Join-Path $env:LOCALAPPDATA "Hermes\config.json"
if (-not (Test-Path $configPath)) {
    Initialize-DefaultConfig
}
```

### hermes_loop.ps1

```powershell
# Eski hardcoded config yerine
. "$PSScriptRoot\lib\ConfigManager.ps1"
$config = Get-HermesConfig

# Kullanım
$provider = Get-ConfigValue -Key "ai.provider" -Override $AI
$timeout = Get-ConfigValue -Key "loop.timeoutMinutes"
```

### hermes-prd.ps1

```powershell
# Timeout için
$timeout = Get-ConfigValue -Key "ai.timeout" -Override $Timeout
$maxRetries = Get-ConfigValue -Key "ai.maxRetries" -Override $MaxRetries
```

### setup.ps1

```powershell
# Proje oluştururken örnek config ekleme (opsiyonel)
# Kullanıcı isterse hermes.config.json oluşturabilir
```

## Config Merge Algoritması

```powershell
function Get-HermesConfig {
    # 1. Varsayılan değerler
    $config = Get-DefaultConfig
    
    # 2. Merkezi config (varsa üzerine yaz)
    $globalPath = Join-Path $env:LOCALAPPDATA "Hermes\config.json"
    if (Test-Path $globalPath) {
        $global = Get-Content $globalPath | ConvertFrom-Json
        $config = Merge-Config -Base $config -Override $global
    }
    
    # 3. Proje config (varsa üzerine yaz)
    $projectPath = "./hermes.config.json"
    if (Test-Path $projectPath) {
        $project = Get-Content $projectPath | ConvertFrom-Json
        $config = Merge-Config -Base $config -Override $project
    }
    
    return $config
}
```

## CLI Komutları (Gelecek)

```powershell
# Config görüntüle
hermes config show

# Değer ayarla (global)
hermes config set ai.provider claude

# Değer ayarla (proje)
hermes config set ai.provider droid --project

# Config dosyası oluştur
hermes config init
hermes config init --project
```

## Test Senaryoları

1. Merkezi config yokken varsayılan değerler kullanılmalı
2. Merkezi config varken değerler override edilmeli
3. Proje config varken merkezi config'i override etmeli
4. Komut satırı parametresi her şeyi override etmeli
5. Geçersiz JSON hata vermeli
6. Eksik alanlar varsayılandan tamamlanmalı

## Dosya Listesi

| Dosya | Değişiklik |
|-------|------------|
| `lib/ConfigManager.ps1` | Yeni |
| `install.ps1` | Güncelle - varsayılan config oluştur |
| `hermes_loop.ps1` | Güncelle - ConfigManager kullan |
| `hermes-prd.ps1` | Güncelle - ConfigManager kullan |
| `hermes-add.ps1` | Güncelle - ConfigManager kullan |
| `tests/unit/ConfigManager.Tests.ps1` | Yeni |

## Uygulama Sırası

1. [ ] `lib/ConfigManager.ps1` oluştur
2. [ ] `tests/unit/ConfigManager.Tests.ps1` oluştur
3. [ ] `install.ps1` güncelle
4. [ ] `hermes_loop.ps1` güncelle
5. [ ] `hermes-prd.ps1` güncelle
6. [ ] `hermes-add.ps1` güncelle
7. [ ] Tüm testleri çalıştır
8. [ ] Dokümantasyonu güncelle
