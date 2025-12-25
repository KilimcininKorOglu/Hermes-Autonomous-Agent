# Plan 03: Status Table Enhancement

## Ozet

`hermes -TaskStatus` komutunu gelistirerek ASCII table formati, filtreleme ve daha detayli gorunum ekleme.

## Mevcut Durum

`Show-TaskStatus` fonksiyonu mevcut:

- Basit progress bar
- Feature listesi
- Sonraki task

## Eksikler

1. ASCII table formati yok
2. Filtreleme yok (--filter, --feature, --priority)
3. Progress bar daha iyi olabilir

## Hedef

```powershell
# Tam tablo
hermes -TaskStatus

# Filtreleme
hermes -TaskStatus -Filter IN_PROGRESS
hermes -TaskStatus -Feature F001
hermes -TaskStatus -Priority P1
```

## Cikti Ornegi

### Tam Tablo

```
┌──────────┬─────────────────────────────────────┬──────────────┬──────────┬──────────┐
│ Task ID  │ Task Name                           │ Status       │ Priority │ Feature  │
├──────────┼─────────────────────────────────────┼──────────────┼──────────┼──────────┤
│ T001     │ Kayit formu UI                      │ COMPLETED    │ P2       │ F001     │
│ T002     │ Input validation                    │ COMPLETED    │ P2       │ F001     │
│ T003     │ API endpoint                        │ IN_PROGRESS  │ P1       │ F001     │
│ T004     │ Database migration                  │ NOT_STARTED  │ P1       │ F001     │
│ T005     │ Unit tests                          │ NOT_STARTED  │ P2       │ F001     │
└──────────┴─────────────────────────────────────┴──────────────┴──────────┴──────────┘

Summary:
  Total: 5 tasks
  COMPLETED:    2 (40%)
  IN_PROGRESS:  1 (20%)
  NOT_STARTED:  2 (40%)
  BLOCKED:      0 (0%)

Progress: [████████░░░░░░░░░░░░░░░░░] 40%
```

### Filtrelenmis

```powershell
hermes -TaskStatus -Filter BLOCKED
```

```
Blocked Tasks:
┌──────────┬─────────────────────────────────────┬──────────┬──────────────────────────┐
│ Task ID  │ Task Name                           │ Priority │ Blocked By               │
├──────────┼─────────────────────────────────────┼──────────┼──────────────────────────┤
│ T005     │ Unit test coverage                  │ P2       │ T003 (IN_PROGRESS)       │
└──────────┴─────────────────────────────────────┴──────────┴──────────────────────────┘

1 blocked task found.
```

## Teknik Tasarim

### Yeni Parametreler

```powershell
# hermes_loop.ps1'e eklenecek
[string]$Filter = "",      # COMPLETED, IN_PROGRESS, NOT_STARTED, BLOCKED
[string]$Feature = "",     # F001, F002, etc.
[string]$Priority = ""     # P1, P2, P3, P4
```

### Yeni Fonksiyonlar

```powershell
function Format-TaskTable {
    <#
    .SYNOPSIS
        Formats tasks as ASCII table
    #>
    param(
        [array]$Tasks,
        [string[]]$Columns = @("TaskId", "Name", "Status", "Priority", "Feature")
    )
}

function Get-FilteredTasks {
    <#
    .SYNOPSIS
        Filters tasks by status, feature, or priority
    #>
    param(
        [string]$Filter,
        [string]$Feature,
        [string]$Priority,
        [string]$BasePath = "."
    )
}

function Show-EnhancedTaskStatus {
    <#
    .SYNOPSIS
        Shows enhanced task status with table format
    #>
    param(
        [string]$Filter,
        [string]$Feature,
        [string]$Priority
    )
}
```

### ASCII Table Karakterleri

```powershell
$script:TableChars = @{
    TopLeft     = [char]0x250C  # ┌
    TopRight    = [char]0x2510  # ┐
    BottomLeft  = [char]0x2514  # └
    BottomRight = [char]0x2518  # ┘
    Horizontal  = [char]0x2500  # ─
    Vertical    = [char]0x2502  # │
    Cross       = [char]0x253C  # ┼
    TLeft       = [char]0x251C  # ├
    TRight      = [char]0x2524  # ┤
    TTop        = [char]0x252C  # ┬
    TBottom     = [char]0x2534  # ┴
}
```

## Uygulama Adimlari

1. [ ] `lib/TableFormatter.ps1` olustur (ASCII table modulu)
2. [ ] `hermes_loop.ps1`'e yeni parametreler ekle
3. [ ] `Show-EnhancedTaskStatus` fonksiyonu yaz
4. [ ] Mevcut `Show-TaskStatus`'u guncelle veya degistir
5. [ ] Unit testler
6. [ ] README.md guncelle

## Bagimliliklari

- `lib/TaskReader.ps1` (mevcut)

## Tahmini Sure

1-2 saat
