# Plan 04: Incremental Update for ralph-prd

## Ozet

`ralph-prd` komutunu mevcut tasklari koruyacak sekilde gelistirme. Ayni PRD tekrar calistirildiginda sadece yeni/degisen feature'lari ekle, mevcut ilerlemeyi koru.

## Mevcut Durum

`ralph-prd` her seferinde:

- Tum task dosyalarini sifirdan olusturur
- Mevcut ilerleme (COMPLETED, IN_PROGRESS) kaybolur
- Mevcut ID'ler conflict olabilir

## Hedef

```powershell
# Ilk calistirma
ralph-prd docs/PRD.md
# Olusturur: F001-F003, T001-T015

# PRD guncellendi, yeni feature eklendi
ralph-prd docs/PRD.md
# Sadece yeni feature'i ekler: F004, T016-T020
# Mevcut F001-F003'u KORUR
```

## Gereksinimler

### 1. Mevcut State Okuma

```powershell
# Oku:
# - tasks/*.md dosyalari
# - Her task'in Status'u
# - En yuksek FXXX ve TXXX ID'leri
```

### 2. PRD Karsilastirma

| PRD'de | tasks/'de | Aksiyon |
|--------|-----------|---------|
| Feature A | Yok | EKLE (yeni FXXX) |
| Feature B | COMPLETED | ATLA (degistirme) |
| Feature B (degisik) | IN_PROGRESS | UYAR |
| Feature B (degisik) | NOT_STARTED | GUNCELLE |
| Yok | Feature C | UYAR (silme) |

### 3. Status Koruma

```
COMPLETED tasklari asla degistirme
IN_PROGRESS tasklari uyar, degistirme
NOT_STARTED tasklari guncellenebilir
```

### 4. Change Summary

```
[INFO] Analyzing PRD changes...

Changes detected:
  Added:
    - F004: Webhook Integration (T016-T020)
  
  Preserved (completed):
    - F001: User Registration (T001-T005) - 100%
  
  Preserved (in progress):
    - F002: Password Reset (T006-T008) - 50%
  
  Warnings:
    - F003 definition changed but has NOT_STARTED tasks
      Consider: ralph-prd --force to overwrite
  
  Removed from PRD (not deleted):
    - F999: Legacy Feature - still in tasks/
```

## Teknik Tasarim

### Yeni Parametreler

```powershell
param(
    # ... mevcut parametreler ...
    
    [switch]$Force,          # Mevcut tasklari yeniden yaz (status koru)
    [switch]$Clean,          # tasks/ klasorunu temizle, sifirdan baslat
    [switch]$DryRun          # Sadece degisiklikleri goster
)
```

### Yeni Fonksiyonlar

```powershell
function Get-ExistingTaskState {
    <#
    .SYNOPSIS
        Reads current task state from tasks/ directory
    .OUTPUTS
        Hashtable with features, tasks, statuses
    #>
}

function Compare-PrdWithTasks {
    <#
    .SYNOPSIS
        Compares PRD content with existing tasks
    .OUTPUTS
        Hashtable with added, modified, removed, preserved
    #>
    param(
        [array]$PrdFeatures,
        [hashtable]$ExistingState
    )
}

function Merge-PrdChanges {
    <#
    .SYNOPSIS
        Merges PRD changes into existing task structure
    #>
    param(
        [hashtable]$Changes,
        [switch]$Force
    )
}

function Write-ChangesSummary {
    <#
    .SYNOPSIS
        Displays summary of changes
    #>
    param(
        [hashtable]$Changes
    )
}
```

### Feature Matching

PRD'deki feature ile mevcut feature'i eslestirmek icin:

1. **Feature Name Match**: Isimleri normalize et, karsilastir
2. **Content Hash**: Feature icerigi hash'le, degisiklik tespit et
3. **Task Count**: Task sayisi degistiyse uyar

```powershell
function Get-FeatureHash {
    param([string]$FeatureContent)
    
    # Normalize: status, dates, vs. cikar
    $normalized = $FeatureContent -replace "\*\*Status:\*\*.*", ""
    $normalized = $normalized -replace "\*\*Last Updated:\*\*.*", ""
    
    # Hash
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($normalized)
    $hash = [System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
    return [BitConverter]::ToString($hash) -replace '-', ''
}
```

## Prompt Template Guncelleme

`lib/prompts/prd-parser.md`'ye ekle:

```markdown
## Existing Features (DO NOT RECREATE)

The following features already exist. Do NOT include them in your output:

{EXISTING_FEATURES}

## Starting IDs

- Next Feature ID: F{NEXT_FEATURE_ID}
- Next Task ID: T{NEXT_TASK_ID}

## Only output NEW features not listed above.
```

## Uygulama Adimlari

1. [ ] `Get-ExistingTaskState` fonksiyonu yaz
2. [ ] `Compare-PrdWithTasks` fonksiyonu yaz
3. [ ] `Merge-PrdChanges` fonksiyonu yaz
4. [ ] `ralph-prd.ps1`'e -Force, -Clean parametreleri ekle
5. [ ] Prompt template'i guncelle
6. [ ] Change summary output'u ekle
7. [ ] Unit testler
8. [ ] README.md guncelle

## Edge Cases

| Durum | Davranis |
|-------|----------|
| tasks/ yok | Normal: sifirdan olustur |
| Bos tasks/ | Normal: sifirdan olustur |
| Sadece status.md var | Sifirdan olustur |
| PRD degismedi | "No changes detected" mesaji |
| PRD feature silindi | Uyar, silme |
| Task ID conflict | Hata, dur |

## Bagimliliklari

- `lib/AIProvider.ps1` (mevcut)
- `lib/TaskReader.ps1` (mevcut)

## Tahmini Sure

3-4 saat
