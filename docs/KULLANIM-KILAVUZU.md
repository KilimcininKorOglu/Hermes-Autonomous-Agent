# Ralph for Claude Code - Kullanim Kilavuzu

Windows PowerShell icin otonom AI gelistirme dongu sistemi. Claude, Droid ve Aider CLI'larini destekler.

---

## Icindekiler

1. [Genel Bakis](#1-genel-bakis)
2. [Kurulum](#2-kurulum)
3. [Hizli Baslangic](#3-hizli-baslangic)
4. [Komutlar](#4-komutlar)
5. [Task Mode](#5-task-mode)
6. [PRD Parser](#6-prd-parser)
7. [Feature Add](#7-feature-add)
8. [AI Provider Sistemi](#8-ai-provider-sistemi)
9. [Modul Detaylari](#9-modul-detaylari)
10. [Konfigürasyon](#10-konfigurasyon)
11. [Sorun Giderme](#11-sorun-giderme)

---

## 1. Genel Bakis

### Ralph Nedir?

Ralph, AI CLI araclarini (Claude Code, Droid, Aider) otonom bir dongude calistirarak yazilim gelistirme surecini otomatiklestiren bir sistemdir.

### Temel Ozellikler

| Ozellik | Aciklama |
|---------|----------|
| Multi-AI Destegi | Claude, Droid ve Aider CLI'larini destekler |
| Task Mode | Yapilandirilmis gorev tabanli gelistirme |
| Otomatik Branch | Her feature icin otomatik Git branch olusturma |
| Otomatik Commit | Gorev tamamlandiginda otomatik commit |
| Autonomous Mode | Kullanici mudahalesi olmadan surekli calisma |
| Resume | Kesintiden sonra kaldigi yerden devam etme |
| Circuit Breaker | Stagnasyon tespiti ve koruma |
| ASCII Status | Renkli tablo formatinda ilerleme gosterimi |

### Mimari

```
ralph-claude-code/
├── ralph_loop.ps1          # Ana calisma dongusu
├── ralph-prd.ps1           # PRD'den task olusturma
├── ralph-add.ps1           # Tekil feature ekleme
├── ralph_monitor.ps1       # Canli izleme paneli
├── install.ps1             # Global kurulum
├── setup.ps1               # Proje olusturma
├── lib/                    # PowerShell modulleri
│   ├── AIProvider.ps1      # AI CLI soyutlamasi
│   ├── TaskReader.ps1      # Task dosyasi okuma
│   ├── TaskStatusUpdater.ps1 # Durum guncelleme
│   ├── GitBranchManager.ps1  # Git islemleri
│   ├── TableFormatter.ps1  # ASCII tablo formatlama
│   ├── CircuitBreaker.ps1  # Stagnasyon tespiti
│   ├── ResponseAnalyzer.ps1 # AI yanit analizi
│   ├── PromptInjector.ps1  # PROMPT.md yonetimi
│   └── FeatureAnalyzer.ps1 # Feature analizi
├── templates/              # Proje sablonlari
└── tests/unit/             # Pester testleri
```

---

## 2. Kurulum

### Gereksinimler

| Gereksinim | Aciklama |
|------------|----------|
| PowerShell 7+ | Windows PowerShell 5.1 DEGiL |
| Git | Versiyon kontrol |
| AI CLI | En az biri: claude, droid veya aider |

### Bagimliliklari Kurma

```powershell
# winget ile
winget install Microsoft.PowerShell Git.Git

# Chocolatey ile
choco install powershell-core git

# AI CLI kurulumu (en az birini kurun)
npm install -g @anthropic-ai/claude-code  # Claude
pip install aider-chat                     # Aider
```

### Ralph Kurulumu

```powershell
# Repoyu klonlayin
git clone https://github.com/frankbria/ralph-claude-code.git
cd ralph-claude-code

# Global olarak kurun
.\install.ps1

# Kurulumu dogrulayin
ralph -Help
```

### Kurulum Yollari

| Tur | Yol |
|-----|-----|
| Komutlar | `$env:LOCALAPPDATA\Ralph\bin\` |
| Scriptler | `$env:LOCALAPPDATA\Ralph\` |
| Sablonlar | `$env:LOCALAPPDATA\Ralph\templates\` |

### Kaldirma

```powershell
.\install.ps1 -Uninstall
```

---

## 3. Hizli Baslangic

### Yeni Proje Olusturma

```powershell
# Proje olustur
ralph-setup my-project
cd my-project

# PROMPT.md'yi duzenleyin
notepad PROMPT.md

# Ralph'i baslatin
ralph -Monitor
```

### Task Mode ile Calisma

```powershell
# PRD'den task olustur
ralph-prd docs/PRD.md

# Task modunu baslat
ralph -TaskMode -AutoBranch -AutoCommit

# Otonom modda calistir
ralph -TaskMode -Autonomous
```

### Proje Yapisi

```
my-project/
├── PROMPT.md           # Ana talimatlar
├── @fix_plan.md        # Gorev listesi
├── @AGENT.md           # Build/run talimatlari
├── specs/              # Proje spesifikasyonlari
├── src/                # Kaynak kod
├── logs/               # Log dosyalari
├── tasks/              # Task dosyalari (Task Mode)
│   ├── 001-feature.md  # Feature dosyalari
│   ├── tasks-status.md # Durum takibi
│   └── run-state.md    # Resume checkpoint
└── status.json         # Canli durum
```

---

## 4. Komutlar

### Ana Komutlar

| Komut | Aciklama |
|-------|----------|
| `ralph -Monitor` | Izleme penceresi ile baslat |
| `ralph -Status` | Mevcut durumu goster |
| `ralph -Help` | Yardim mesajini goster |
| `ralph -ResetCircuit` | Circuit breaker'i sifirla |
| `ralph-setup <name>` | Yeni proje olustur |
| `ralph-prd <file>` | PRD'yi task'lara donustur |
| `ralph-add "feature"` | Tekil feature ekle |
| `ralph-monitor` | Bagimsiz izleme paneli |

### Ralph Loop Parametreleri

```powershell
ralph [-Monitor] [-Calls <int>] [-Timeout <int>] [-VerboseProgress]
      [-Status] [-ResetCircuit] [-CircuitStatus] [-Help]
      [-AI <provider>] [-TaskMode] [-AutoBranch] [-AutoCommit]
      [-StartFrom <TaskId>] [-TaskStatus]
```

| Parametre | Varsayilan | Aciklama |
|-----------|------------|----------|
| `-AI` | auto | AI provider: claude, droid, aider, auto |
| `-Calls` | 100 | Saatlik maksimum API cagrisi |
| `-Timeout` | 15 | AI timeout (dakika) |
| `-Monitor` | - | Izleme penceresi ac |
| `-VerboseProgress` | - | Detayli ilerleme goster |

### Task Mode Parametreleri

| Parametre | Aciklama |
|-----------|----------|
| `-TaskMode` | Task-plan entegrasyonunu etkinlestir |
| `-AutoBranch` | Feature branch'lerini otomatik olustur |
| `-AutoCommit` | Gorev tamamlaninca otomatik commit |
| `-StartFrom T005` | Belirli task'tan basla |
| `-TaskStatus` | Task ilerleme tablosunu goster |
| `-Autonomous` | Duraklama olmadan surekli calis |
| `-MaxConsecutiveErrors` | Hata esigi (varsayilan: 5) |

### Filtreleme Parametreleri

| Parametre | Ornek | Aciklama |
|-----------|-------|----------|
| `-StatusFilter` | COMPLETED | Duruma gore filtrele |
| `-FeatureFilter` | F001 | Feature'a gore filtrele |
| `-PriorityFilter` | P1 | Oncelikle filtrele |

---

## 5. Task Mode

### Task Mode Nedir?

Task Mode, yapilandirilmis gorev dosyalariyla calisan bir gelistirme modudur. Her feature ve task ayri dosyalarda tanimlanir.

### Is Akisi

```
PRD.md -> ralph-prd -> tasks/*.md -> ralph -TaskMode -> Otomatik Uygulama
```

### Task Dosyasi Formati

```markdown
# Feature 1: Kullanici Dogrulama

**Feature ID:** F001
**Feature Name:** Kullanici Dogrulama
**Priority:** P1 - Critical
**Status:** NOT_STARTED
**Estimated Duration:** 1-2 hafta

## Overview

Kullanici dogrulama sistemi aciklamasi.

## Tasks

### T001: Veritabani Semasi

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 gun

#### Description
Kullanicilar tablosunu olustur.

#### Files to Touch
- `migrations/001_users.sql` (new)
- `src/models/user.ts` (new)

#### Dependencies
- None

#### Success Criteria
- [ ] Migration basarili
- [ ] Rollback calisiyor
- [ ] Index tanimli
```

### ID Sistemi

| Tur | Format | Ornek |
|-----|--------|-------|
| Feature ID | F + 3 basamak | F001, F002, F003 |
| Task ID | T + 3 basamak | T001, T002, T003 |

**Onemli:** Task ID'ler TUM feature'lar uzerinden devam eder:

- F001: T001, T002, T003
- F002: T004, T005, T006

### Branch Stratejisi

```
main
  └── feature/F001-kullanici-dogrulama
        ├── feat(T001): Veritabani Semasi completed
        ├── feat(T002): Kayit API completed
        └── feat(T003): Giris API completed
  └── feature/F002-dashboard
        └── ...
```

### Commit Formati

```
feat(T001): Task Adi completed

Completed:
- [x] Migration basarili
- [x] Rollback calisiyor

Files:
- migrations/001_users.sql
- src/models/user.ts
```

### Otomatik Resume

Ralph, `run-state.md` dosyasini kullanarak kesintiden sonra otomatik olarak devam eder:

```powershell
# Ilk calisma - T003'te kesiliyor
ralph -TaskMode -AutoBranch -AutoCommit
# ... kesinti veya context limit

# Sonraki calisma - T004'ten devam eder
ralph -TaskMode -AutoBranch -AutoCommit
# Cikti: "Previous run detected - Resuming from T004..."
```

### Durum Goruntuleme

```powershell
# Tam durum tablosu
ralph -TaskStatus

# Filtrelenmi durum
ralph -TaskStatus -StatusFilter BLOCKED
ralph -TaskStatus -FeatureFilter F001
ralph -TaskStatus -PriorityFilter P1

# Birlesik filtre
ralph -TaskStatus -StatusFilter NOT_STARTED -PriorityFilter P1
```

---

## 6. PRD Parser

### Kullanim

```powershell
ralph-prd <prd-dosyasi> [-AI <provider>] [-DryRun] [-Force] [-Clean]
```

### Parametreler

| Parametre | Aciklama |
|-----------|----------|
| `<prd-dosyasi>` | PRD markdown dosyasi |
| `-AI` | AI provider (varsayilan: auto) |
| `-DryRun` | Dosya olusturmadan onizle |
| `-OutputDir` | Cikti klasoru (varsayilan: tasks) |
| `-Timeout` | AI timeout saniye (varsayilan: 1200) |
| `-MaxRetries` | Tekrar deneme sayisi (varsayilan: 10) |
| `-Force` | NOT_STARTED feature'lari uzerine yaz |
| `-Clean` | Tum mevcut task'lari sil, sifirdan basla |

### Incremental Mode

Varsayilan olarak, `ralph-prd` artimsal modda calisir:

```powershell
# Ilk calisma - tum feature'lari olusturur
ralph-prd docs/PRD.md

# PRD guncellendi, yeni feature'lar eklendi
ralph-prd docs/PRD.md
# Sadece YENI feature'lari ekler, mevcut ilerlemeyi korur

# Temiz baslangic
ralph-prd docs/PRD.md -Clean
```

### Artimsal Mod Davranisi

| Mevcut Durum | Davranis |
|--------------|----------|
| COMPLETED | Asla uzerine yazilmaz |
| IN_PROGRESS | Korunur |
| NOT_STARTED | `-Force` ile uzerine yazilabilir |
| Yeni | Eklenir |

### Ornek Cikti

```
[INFO] Reading PRD: docs/PRD.md
[INFO] PRD size: 45000 characters, 800 lines
[INFO] Using AI: claude
[INFO] Attempt 1/10...
[OK] AI completed successfully

[OK] Created: tasks/001-user-authentication.md (F001, T001-T004)
[OK] Created: tasks/002-dashboard.md (F002, T005-T008)
[OK] Created: tasks/tasks-status.md

Summary:
  Features: 2
  Tasks: 8
  Estimated: 12 days

Next: Run 'ralph -TaskMode -AutoBranch -AutoCommit' to start
```

---

## 7. Feature Add

### Kullanim

```powershell
ralph-add <feature> [-AI <provider>] [-DryRun] [-Priority <P1-P4>]
```

### Giris Turleri

```powershell
# Satir ici aciklama
ralph-add "kullanici kayit sistemi"

# Dosyadan oku
ralph-add @docs/webhook-spec.md

# Oncelik belirt
ralph-add "sifre sifirlama" -Priority P1

# Onizleme
ralph-add "email dogrulama" -DryRun
```

### Parametreler

| Parametre | Aciklama |
|-----------|----------|
| `<feature>` | Feature aciklamasi veya @dosya-yolu |
| `-AI` | AI provider (varsayilan: auto) |
| `-DryRun` | Dosya olusturmadan onizle |
| `-Priority` | Oncelik: P1, P2, P3, P4 |
| `-OutputDir` | Cikti klasoru (varsayilan: tasks) |
| `-Timeout` | AI timeout saniye (varsayilan: 300) |

### Ornek Cikti

```
[INFO] Reading feature input...
[INFO] Source: inline description
[INFO] Next Feature ID: F003
[INFO] Next Task ID: T012
[INFO] Using AI: claude
[INFO] Analyzing feature with claude...

==================================================
  Feature added!
==================================================

  Feature ID: F003
  File:       tasks/003-email-dogrulama.md
  Name:       Email Dogrulama
  Priority:   P2 - High
  Tasks:      4 (T012-T015)
  Effort:     3 days (total)

==================================================

Next: Run 'ralph -TaskMode -AutoBranch -AutoCommit' to implement
```

---

## 8. AI Provider Sistemi

### Desteklenen Provider'lar

| Provider | Komut | Aciklama |
|----------|-------|----------|
| Claude | `claude` | Claude Code CLI |
| Droid | `droid` | Factory Droid CLI |
| Aider | `aider` | Aider AI CLI |

### Otomatik Tespit

Ralph, mevcut AI CLI'lari otomatik olarak tespit eder. Oncelik sirasi:

1. `claude` (en yuksek)
2. `droid`
3. `aider`

### Provider Secimi

```powershell
# Otomatik tespit (varsayilan)
ralph -TaskMode
ralph-prd docs/PRD.md
ralph-add "feature"

# Belirli provider
ralph -TaskMode -AI droid
ralph-prd docs/PRD.md -AI claude
ralph-add "feature" -AI aider
```

### Provider Kontrolu

```powershell
# Mevcut provider'lari listele
ralph-prd -List
```

### AIProvider.ps1 Fonksiyonlari

| Fonksiyon | Aciklama |
|-----------|----------|
| `Test-AIProvider` | Provider mevcut mu kontrol et |
| `Get-AutoProvider` | Ilk mevcut provider'i dondur |
| `Get-AvailableProviders` | Tum mevcut provider'lari listele |
| `Invoke-TaskExecution` | Task Mode icin AI calistir |
| `Invoke-AIWithRetry` | Retry logic ile AI calistir |
| `Write-AIProviderList` | Mevcut provider'lari goster |

---

## 9. Modul Detaylari

### TaskReader.ps1

Task dosyalarini okur ve parse eder.

| Fonksiyon | Aciklama |
|-----------|----------|
| `Get-AllTasks` | Tum task'lari dondur |
| `Get-AllFeatures` | Tum feature'lari dondur |
| `Get-TaskById` | ID ile task bul |
| `Get-FeatureById` | ID ile feature bul |
| `Get-NextTask` | Siradaki task'i bul |
| `Get-TaskProgress` | Genel ilerleme istatistikleri |
| `Test-TaskDependenciesMet` | Bagimliliklari kontrol et |

### TaskStatusUpdater.ps1

Task durumlarini gunceller ve resume mekanizmasini yonetir.

| Fonksiyon | Aciklama |
|-----------|----------|
| `Set-TaskStatus` | Task durumunu guncelle |
| `Set-FeatureStatus` | Feature durumunu guncelle |
| `Update-RunState` | Resume checkpoint guncelle |
| `Test-ShouldResume` | Resume gerekli mi kontrol et |
| `Get-ResumeInfo` | Resume detaylarini al |
| `Get-ExecutionQueue` | Oncelik sirali kuyruk |

### GitBranchManager.ps1

Git branch ve commit islemlerini yonetir.

| Fonksiyon | Aciklama |
|-----------|----------|
| `New-FeatureBranch` | Feature branch olustur |
| `Switch-ToFeatureBranch` | Branch'e gec |
| `New-TaskCommit` | Task commit'i olustur |
| `Merge-FeatureToMain` | main'e merge et |
| `Get-CurrentBranch` | Mevcut branch adini al |
| `Test-BranchExists` | Branch mevcut mu |

### CircuitBreaker.ps1

Stagnasyon tespiti ve koruma.

| Durum | Anlam |
|-------|-------|
| CLOSED | Normal islem |
| HALF_OPEN | Izleme modu (2 ilerleme olmayan dongu) |
| OPEN | Durduruldu (3+ ilerleme olmayan dongu) |

| Fonksiyon | Aciklama |
|-----------|----------|
| `Initialize-CircuitBreaker` | Baslat |
| `Add-LoopResult` | Dongu sonucunu kaydet |
| `Test-ShouldHalt` | Durdurulmali mi |
| `Reset-CircuitBreaker` | Sifirla |
| `Show-CircuitStatus` | Durumu goster |

### TableFormatter.ps1

ASCII tablo formatlama.

| Fonksiyon | Aciklama |
|-----------|----------|
| `Format-TaskTable` | Task tablosu olustur |
| `Write-TaskTable` | Renkli tablo yaz |
| `Get-FilteredTasks` | Filtrelenmi task'lar |
| `Show-EnhancedTaskStatus` | Gelismis durum goster |

### PromptInjector.ps1

PROMPT.md'ye task bilgisi enjekte eder.

| Fonksiyon | Aciklama |
|-----------|----------|
| `Add-TaskToPrompt` | Task bölumu ekle |
| `Remove-TaskFromPrompt` | Task bölumu kaldir |
| `Backup-Prompt` | PROMPT.md yedekle |
| `Get-CurrentTaskFromPrompt` | Mevcut task ID'yi al |

### FeatureAnalyzer.ps1

Feature analizi ve dosya olusturma.

| Fonksiyon | Aciklama |
|-----------|----------|
| `Get-NextIds` | Siradaki Feature/Task ID |
| `Read-FeatureInput` | Giris oku (inline/@dosya) |
| `Build-FeaturePrompt` | AI prompt olustur |
| `Parse-FeatureOutput` | AI ciktisini parse et |
| `Write-FeatureFile` | Feature dosyasi yaz |

---

## 10. Konfigurasyon

### ralph_loop.ps1 Konfigurasyonu

```powershell
$script:Config = @{
    AIProvider = "claude"        # Cozumlenen AI provider
    AITimeoutMinutes = 15        # Timeout suresi
    MaxCallsPerHour = 100        # Saatlik API limiti
    MaxConsecutiveErrors = 5     # Hata esigi
    TasksDir = "tasks"           # Task klasoru
    LogDir = "logs"              # Log klasoru
    StatusFile = "status.json"   # Durum dosyasi
}
```

### Proje Kontrol Dosyalari

| Dosya | Amac |
|-------|------|
| `PROMPT.md` | Her dongude AI'a verilen talimatlar |
| `tasks/*.md` | Feature ve task tanimlari |
| `tasks/run-state.md` | Resume checkpoint |
| `tasks/tasks-status.md` | Durum takibi |
| `status.json` | Canli dongu durumu |
| `.circuit_breaker_state` | Circuit breaker durumu |

### PROMPT.md Status Blogu

AI her yanitin sonunda bu blogu cikarmaldir:

```
---RALPH_STATUS---
STATUS: IN_PROGRESS | COMPLETE | BLOCKED
EXIT_SIGNAL: false | true
RECOMMENDATION: <sonraki aksiyon>
---END_RALPH_STATUS---
```

---

## 11. Sorun Giderme

### Script Calistirma Hatasi

```
File cannot be loaded because running scripts is disabled
```

**Cozum:**

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### AI CLI Bulunamadi

```
'claude' is not recognized
```

**Cozum:**

```powershell
# Claude
npm install -g @anthropic-ai/claude-code

# Aider
pip install aider-chat
```

### PATH Guncellenmedi

Kurulumdan sonra terminali yeniden baslatin veya:

```powershell
$env:PATH = "$env:LOCALAPPDATA\Ralph\bin;$env:PATH"
```

### Circuit Breaker Acildi

```
CIRCUIT BREAKER OPENED - Execution halted
```

**Cozum:**

1. Son loglari inceleyin:

   ```powershell
   Get-Content logs\ralph.log -Tail 20
   ```

2. AI ciktisini kontrol edin:

   ```powershell
   Get-ChildItem logs\*_output_*.log | Sort-Object LastWriteTime -Descending | Select-Object -First 1
   ```

3. Sorunu giderin ve sifirlayin:

   ```powershell
   ralph -ResetCircuit
   ```

### Task Bulunamadi

```
Task not found: T005
```

**Cozum:**

- Task ID'nin dogru yazildigindan emin olun
- `tasks/` klasorunun mevcut oldugunu kontrol edin
- `ralph -TaskStatus` ile mevcut task'lari listeleyin

### Resume Calismiyor

Resume mekanizmasi `run-state.md` dosyasina baglidir:

```powershell
# run-state.md'yi kontrol edin
Get-Content tasks/run-state.md

# Manuel olarak belirli task'tan baslayin
ralph -TaskMode -StartFrom T005
```

### Syntax Kontrolu

PowerShell dosyalarinda syntax hatasi kontrolu:

```powershell
$errors = $null
$null = [System.Management.Automation.Language.Parser]::ParseFile(
    'path/to/file.ps1', 
    [ref]$null, 
    [ref]$errors
)
if ($errors.Count -gt 0) { 
    $errors | ForEach-Object { Write-Host $_.Message } 
}
```

### Test Calistirma

```powershell
# Tum testleri calistir
Import-Module Pester -Force
Invoke-Pester -Path tests/unit/

# Tek test dosyasi
Invoke-Pester -Path tests/unit/AIProvider.Tests.ps1

# Detayli cikti
Invoke-Pester -Path tests/unit/ -PassThru
```

---

## Ek Kaynaklar

- [README.md](../README.md) - Proje genel bakisi
- [CLAUDE.md](../CLAUDE.md) - Claude Code icin rehber
- [AGENTS.md](../AGENTS.md) - AI ajanlar icin rehber
- [plan/](../plan/) - Gelistirme planlari

---

**Versiyon:** 1.0  
**Son Guncelleme:** 2025-12-25
