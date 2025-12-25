# Plan 02: Feature Add Command (hermes-add)

## Ozet

Tek bir feature eklemek icin `hermes-add` komutu. PRD olmadan, inline veya dosyadan feature tanimlamasi.

## Mevcut Durum

- `hermes-prd`: PRD dosyasindan task olusturur (tum PRD)
- Tek feature eklemek icin komut **YOK**

## Hedef

```powershell
# Inline aciklama ile
hermes-add "kullanici kayit sistemi"

# Dosyadan
hermes-add @docs/webhook-spec.md

# Mevcut tasklara ekleme
hermes-add "sifre sifirlama"  # F002, T006-T008 olarak devam eder
```

## Gereksinimler

### 1. ID Continuation

Mevcut en yuksek Feature ID ve Task ID'yi bul, devam et:

```powershell
# Mevcut: F001 (T001-T005)
hermes-add "sifre sifirlama"
# Olusturur: F002 (T006-T008)
```

### 2. Input Modlari

| Mod | Ornek | Aciklama |
|-----|-------|----------|
| Inline | `hermes-add "aciklama"` | Kisa aciklama, AI tarafindan analiz |
| File | `hermes-add @file.md` | Dosya icerigi analiz edilir |

### 3. AI Entegrasyonu

`lib/AIProvider.ps1` kullanarak feature'i analiz et ve task'lara bol.

### 4. Cikti Formati

```
[INFO] Analyzing feature...
[INFO] Using AI: claude
[OK] Feature created!

  Feature ID: F002
  File:       tasks/002-sifre-sifirlama.md
  Name:       Sifre Sifirlama
  Priority:   P2
  Tasks:      3 (T006-T008)
  Effort:     2 days (total)

Next: Run 'hermes -TaskMode' to implement
```

## Teknik Tasarim

### Dosya Yapisi

```
hermes-add.ps1           # Ana script
lib/FeatureAnalyzer.ps1 # Feature analiz modulu (yeni)
lib/prompts/
  feature-analyzer.md   # Feature analiz prompt'u (yeni)
```

### hermes-add.ps1 Parametreleri

```powershell
param(
    [Parameter(Position = 0)]
    [string]$Feature,        # Inline veya @file
    
    [ValidateSet("claude", "droid", "aider", "auto")]
    [string]$AI = "auto",
    
    [switch]$DryRun,
    
    [string]$OutputDir = "tasks",
    
    [int]$Timeout = 300,     # 5 dakika (tek feature icin daha kisa)
    
    [string]$Priority        # Override priority (P1-P4)
)
```

### lib/FeatureAnalyzer.ps1 Fonksiyonlari

```powershell
function Get-HighestIds {
    # Mevcut en yuksek FXXX ve TXXX ID'leri bul
}

function Read-FeatureInput {
    # Inline veya @file oku
}

function Invoke-FeatureAnalysis {
    # AI'ya gonder, analiz ettir
}

function New-FeatureFile {
    # Task dosyasi olustur
}
```

### lib/prompts/feature-analyzer.md

```markdown
# Feature Analyzer

Analyze the feature description and create a task breakdown.

## Input
{FEATURE_DESCRIPTION}

## Starting IDs
- Feature ID: F{NEXT_FEATURE_ID}
- Task ID starts at: T{NEXT_TASK_ID}

## Output Format
### FILE: tasks/{FILE_NUMBER}-{feature-name}.md
...
```

## Uygulama Adimlari

1. [ ] `lib/FeatureAnalyzer.ps1` olustur
2. [ ] `lib/prompts/feature-analyzer.md` olustur
3. [ ] `hermes-add.ps1` olustur
4. [ ] `install.ps1` guncelle
5. [ ] Unit testler
6. [ ] README.md guncelle

## Bagimliliklari

- `lib/AIProvider.ps1` (mevcut)
- `lib/TaskReader.ps1` (mevcut - ID okuma icin)

## Tahmini Sure

2-3 saat
