# Ralph + Task-Plan Entegrasyon Plani

## Ozet

Ralph'i task-plan sistemiyle entegre ederek:
- Her gorev icin ayri commit
- Feature branch stratejisi
- Task ID sistemi (T001, T002...)

## Mevcut Durum

### Ralph (Simdi)
```
PROMPT.md → Claude Code → Analiz → Tekrar → Proje bitince dur
```
- Tek PROMPT.md
- Commit kullaniciya bagli
- Branch yonetimi yok
- Task ID yok

### Task-Plan Sistemi
```
PRD.md → tasks/*.md → Her task icin branch/commit
```
- Yapisal gorev listesi
- Feature branch per feature
- Task commit per task
- T001, T002... ID sistemi

## Hedef Durum

```
tasks/
├── 001-user-registration.md (F001: T001-T005)
├── 002-password-reset.md (F002: T006-T008)
└── tasks-status.md
        ↓
Ralph (Task Mode)
        ↓
┌─────────────────────────────────────────────────────┐
│ 1. Siradaki task'i bul (NOT_STARTED, dependency ok) │
│ 2. Feature branch olustur/gecis yap                 │
│ 3. PROMPT.md'ye task detaylarini ekle               │
│ 4. Claude Code calistir                             │
│ 5. Basarili → git commit (feat(TXXX): ...)         │
│ 6. Task durumunu COMPLETED yap                      │
│ 7. Feature tamamsa → main'e merge                   │
│ 8. Sonraki task'a gec                               │
└─────────────────────────────────────────────────────┘
```

---

## Yeni Moduller

### 1. lib/TaskReader.ps1

Task dosyalarini okur ve parse eder.

```powershell
# Fonksiyonlar
Get-AllTasks              # Tum tasklari listele
Get-NextTask              # Siradaki calistirilacak task
Get-TaskById              # Belirli task detaylari
Get-FeatureById           # Feature detaylari
Get-TaskDependencies      # Bagimliliklari kontrol et
Test-TaskDependenciesMet  # Bagimliliklar tamamlandi mi?
```

**Task Parse Formati:**
```markdown
### T001: Kayit formu UI

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 days

#### Description
...

#### Files to Touch
- `src/components/RegisterForm.tsx` (new)

#### Dependencies
- T000 (must complete first)

#### Success Criteria
- [ ] Form tasarimi tamamlandi
- [ ] Responsive calisiyor
```

**Parse Edilen Veri:**
```powershell
@{
    TaskId = "T001"
    Name = "Kayit formu UI"
    Status = "NOT_STARTED"
    Priority = "P1"
    Effort = "1 days"
    Description = "..."
    FilesToTouch = @("src/components/RegisterForm.tsx")
    Dependencies = @("T000")
    SuccessCriteria = @("Form tasarimi tamamlandi", "Responsive calisiyor")
    FeatureId = "F001"
    FeatureFile = "tasks/001-user-registration.md"
}
```

### 2. lib/GitBranchManager.ps1

Feature branch ve commit yonetimi.

```powershell
# Fonksiyonlar
New-FeatureBranch         # feature/FXXX-name olustur
Switch-ToFeatureBranch    # Branch'e gec
Switch-ToMain             # main'e gec
New-TaskCommit            # feat(TXXX): ... commit
New-FeatureCommit         # feat(FXXX): ... commit
Merge-FeatureToMain       # main'e merge (--no-ff)
Get-CurrentBranch         # Hangi branch'teyiz?
Test-BranchExists         # Branch var mi?
Get-FeatureBranchName     # FXXX → feature/F001-user-registration
```

**Branch Naming:**
```
feature/F001-user-registration
feature/F002-password-reset
feature/F003-email-verification
```

**Commit Format:**
```
feat(T001): Kayit formu UI completed

Completed:
- [x] Form tasarimi tamamlandi
- [x] Responsive calisiyor

Files:
- src/components/RegisterForm.tsx (new)
```

### 3. lib/TaskStatusUpdater.ps1

Task ve feature durumlarini gunceller.

```powershell
# Fonksiyonlar
Set-TaskStatus            # Task durumunu guncelle
Set-FeatureStatus         # Feature durumunu guncelle
Update-TasksStatusFile    # tasks/tasks-status.md guncelle
Add-TaskCompletionLog     # Tamamlama logla
Get-FeatureProgress       # Feature ilerleme yuzdesi
Test-FeatureComplete      # Tum tasklar bitti mi?
```

**Status Degerleri:**
- NOT_STARTED
- IN_PROGRESS
- COMPLETED
- BLOCKED
- AT_RISK
- PAUSED

### 4. lib/PromptInjector.ps1

Task detaylarini PROMPT.md'ye ekler.

```powershell
# Fonksiyonlar
Add-TaskToPrompt          # Task detaylarini PROMPT.md'ye ekle
Remove-TaskFromPrompt     # Eski task detaylarini cikar
Get-TaskPromptSection     # Task icin prompt bolumu olustur
```

**Eklenen Bolum:**
```markdown
## CURRENT TASK

**Task ID:** T001
**Task Name:** Kayit formu UI
**Feature:** F001 - User Registration
**Priority:** P1
**Branch:** feature/F001-user-registration

### Description
[Task description]

### Files to Touch
- `src/components/RegisterForm.tsx` (new)

### Success Criteria
- [ ] Form tasarimi tamamlandi
- [ ] Responsive calisiyor

### Dependencies
- None (or list of completed dependencies)

---
IMPORTANT: Focus ONLY on this task. When complete, output:
---RALPH_STATUS---
STATUS: COMPLETE
TASK_ID: T001
EXIT_SIGNAL: false
---END_RALPH_STATUS---
```

---

## ralph_loop.ps1 Degisiklikleri

### Yeni Parametreler

```powershell
param(
    # ... mevcut parametreler ...
    
    [switch]$TaskMode,           # Task-plan modunu etkinlestir
    [string]$TasksDir = "tasks", # Tasks klasoru yolu
    [switch]$AutoBranch,         # Otomatik branch yonetimi
    [switch]$AutoCommit          # Otomatik commit
)
```

### Yeni Config

```powershell
$script:Config = @{
    # ... mevcut config ...
    
    # Task Mode
    TaskMode = $TaskMode
    TasksDir = $TasksDir
    AutoBranch = $AutoBranch
    AutoCommit = $AutoCommit
    
    # Task State
    CurrentTaskId = $null
    CurrentFeatureId = $null
    CurrentBranch = $null
}
```

### Yeni Akis

```powershell
function Start-RalphLoop {
    # ... mevcut init ...
    
    if ($script:Config.TaskMode) {
        Start-TaskModeLoop
    } else {
        Start-StandardLoop  # Mevcut davranis
    }
}

function Start-TaskModeLoop {
    while ($true) {
        # 1. Siradaki task'i bul
        $task = Get-NextTask -TasksDir $script:Config.TasksDir
        
        if (-not $task) {
            Write-Status -Level "SUCCESS" -Message "Tum tasklar tamamlandi!"
            break
        }
        
        # 2. Feature branch yonetimi
        if ($script:Config.AutoBranch) {
            $branchName = Get-FeatureBranchName -FeatureId $task.FeatureId
            if (-not (Test-BranchExists -Name $branchName)) {
                New-FeatureBranch -FeatureId $task.FeatureId -Name $task.FeatureName
            }
            Switch-ToFeatureBranch -Name $branchName
        }
        
        # 3. Task durumunu IN_PROGRESS yap
        Set-TaskStatus -TaskId $task.TaskId -Status "IN_PROGRESS"
        
        # 4. PROMPT.md'ye task ekle
        Add-TaskToPrompt -Task $task
        
        # 5. Claude Code calistir
        $result = Invoke-ClaudeCode -LoopCount $loopCount
        
        # 6. Sonucu analiz et
        $analysis = Get-AnalysisResult
        
        if ($analysis.analysis.task_completed) {
            # 7. Commit at
            if ($script:Config.AutoCommit) {
                New-TaskCommit -TaskId $task.TaskId -TaskName $task.Name
            }
            
            # 8. Task durumunu COMPLETED yap
            Set-TaskStatus -TaskId $task.TaskId -Status "COMPLETED"
            
            # 9. Feature tamamlandi mi kontrol et
            if (Test-FeatureComplete -FeatureId $task.FeatureId) {
                New-FeatureCommit -FeatureId $task.FeatureId
                Merge-FeatureToMain -FeatureId $task.FeatureId
                Set-FeatureStatus -FeatureId $task.FeatureId -Status "COMPLETED"
            }
        }
        
        # 10. Sonraki task'a gec (dongu devam)
    }
}
```

---

## ResponseAnalyzer Degisiklikleri

### Yeni Tespit

```powershell
# Task completion detection
if ($outputContent -match "TASK_ID:\s*(T\d+)") {
    $analysis.completed_task_id = $Matches[1]
}

if ($outputContent -match "---RALPH_STATUS---.*?TASK_ID:\s*(T\d+).*?STATUS:\s*COMPLETE") {
    $analysis.task_completed = $true
}
```

### Yeni Status Block Formati

```
---RALPH_STATUS---
STATUS: COMPLETE
TASK_ID: T001
FILES_MODIFIED: 3
TESTS_STATUS: PASSING
EXIT_SIGNAL: false
RECOMMENDATION: Task T001 completed, continue to T002
---END_RALPH_STATUS---
```

---

## Yeni Komutlar

```powershell
# Task modunda calistir
ralph -TaskMode -AutoBranch -AutoCommit

# Task durumunu goster
ralph -TaskStatus

# Belirli task'tan basla
ralph -TaskMode -StartFrom T005
```

---

## Dosya Yapisi

```
windows/
├── lib/
│   ├── CircuitBreaker.ps1      # Mevcut
│   ├── ResponseAnalyzer.ps1    # Guncellendi
│   ├── TaskReader.ps1          # YENİ
│   ├── GitBranchManager.ps1    # YENİ
│   ├── TaskStatusUpdater.ps1   # YENİ
│   └── PromptInjector.ps1      # YENİ
├── ralph_loop.ps1              # Guncellendi
└── tests/
    └── unit/
        ├── TaskReader.Tests.ps1          # YENİ
        ├── GitBranchManager.Tests.ps1    # YENİ
        └── TaskStatusUpdater.Tests.ps1   # YENİ
```

---

## Uygulama Sirasi

### Faz 1: Core Moduller
1. [ ] lib/TaskReader.ps1 - Task dosyalarini oku
2. [ ] lib/TaskStatusUpdater.ps1 - Durum guncelle
3. [ ] Unit testler

### Faz 2: Git Entegrasyonu
4. [ ] lib/GitBranchManager.ps1 - Branch/commit yonetimi
5. [ ] Unit testler

### Faz 3: Prompt Entegrasyonu
6. [ ] lib/PromptInjector.ps1 - Task → PROMPT.md
7. [ ] ResponseAnalyzer guncellemesi

### Faz 4: Ana Dongu
8. [ ] ralph_loop.ps1 - TaskMode ekleme
9. [ ] Integration testler

### Faz 5: Dokumantasyon
10. [ ] README guncelleme
11. [ ] Ornek tasks/ klasoru

---

## Ornek Kullanim Senaryosu

```powershell
# 1. Proje olustur
ralph-setup my-app
cd my-app

# 2. PRD'den task olustur (task-plan ile)
# /task-plan docs/PRD.md
# Bu tasks/ klasorunu olusturur

# 3. Ralph'i task modunda calistir
ralph -TaskMode -AutoBranch -AutoCommit -Monitor

# Ralph otomatik olarak:
# - T001'i alir
# - feature/F001-user-registration branch'i olusturur
# - PROMPT.md'ye T001 detaylarini ekler
# - Claude Code calistirir
# - Basarili olursa commit atar: feat(T001): ...
# - T001'i COMPLETED yapar
# - T002'ye gecer
# - F001 bitince main'e merge eder
# - F002'ye gecer
# - ... tum tasklar bitene kadar devam
```

---

## Avantajlar

1. **Izlenebilirlik**: Her task icin ayri commit
2. **Paralel Calisma**: Feature branchler ile izolasyon
3. **Geri Alma**: Task bazinda rollback mumkun
4. **Ilerleme Takibi**: tasks-status.md ile net goruntu
5. **Entegrasyon**: task-plan sistemiyle uyumlu
6. **Esneklik**: TaskMode opsiyonel, eski mod hala calisiyor

---

## Risk ve Cozumler

| Risk | Cozum |
|------|-------|
| Merge conflict | --no-ff ile merge, conflict durumunda BLOCKED |
| Task parse hatasi | Strict format validation, fallback to standard mode |
| Branch karmasasi | Clear naming convention, auto-cleanup |
| Uzun task | Circuit breaker hala aktif, task bazinda da timeout |
