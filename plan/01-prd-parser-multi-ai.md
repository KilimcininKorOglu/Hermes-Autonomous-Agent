# PRD Parser - Multi-AI CLI Destegi Plani

## Genel Bakis

Kullanicinin PRD dosyasini alip, sectigi AI CLI araciyla `tasks/` klasorune task-plan formatinda dosyalar olusturan bir sistem.

## Desteklenecek AI CLI Araclari

| Arac | Komut | Kurulum | Non-Interactive |
|------|-------|---------|-----------------|
| Claude Code | `claude` | `npm install -g @anthropic-ai/claude-code` | `-p` flag |
| Droid CLI | `droid` | `curl -fsSL https://app.factory.ai/cli \| sh` | `exec` mode |
| Aider | `aider` | `pip install aider-chat` | `--message` flag |

## Kullanim

```powershell
# Varsayilan (claude)
hermes-prd docs/PRD.md

# AI secerek
hermes-prd docs/PRD.md -AI claude
hermes-prd docs/PRD.md -AI droid
hermes-prd docs/PRD.md -AI aider

# Hangi AI'lar mevcut?
hermes-prd -List
```

## Nasil Calisir?

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   PRD.md    │────▶│   hermes-prd.ps1  │────▶│  tasks/*.md     │
└─────────────┘     └──────────────────┘     └─────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  AI CLI Sec   │
                    ├───────────────┤
                    │ • claude      │
                    │ • droid       │
                    │ • aider       │
                    └───────────────┘
```

## Akis

```
1. PRD dosyasini oku
2. Prompt template'i yukle (lib/prompts/prd-parser.md)
3. Secilen AI CLI'yi cagir
4. AI ciktisini parse et
5. tasks/ klasorune dosyalari yaz
6. tasks-status.md olustur
```

## Dosya Yapisi

```
hermes-claude-code/
├── hermes-prd.ps1              # Ana script (YENI)
├── lib/
│   ├── AIProvider.ps1         # AI CLI abstraction (YENI)
│   └── prompts/
│       └── prd-parser.md      # PRD parse prompt (YENI)
```

## lib/AIProvider.ps1

Her AI CLI icin ortak interface:

```powershell
function Invoke-AICommand {
    param(
        [ValidateSet("claude", "droid", "aider")]
        [string]$Provider,
        [string]$PromptText,
        [string]$InputFile
    )
    
    $content = Get-Content $InputFile -Raw
    
    switch ($Provider) {
        "claude" {
            # -p flag ile non-interactive mode
            # Pipe ile icerik gonderilir
            $content | claude -p $PromptText
        }
        "droid" {
            # exec mode ile non-interactive
            # --auto low ile dosya olusturma izni
            $content | droid exec --auto low $PromptText
        }
        "aider" {
            # --message ile tek seferlik calistirma
            # --yes otomatik onay
            # --no-auto-commits git commit yapmasin
            aider --yes --no-auto-commits --message $PromptText $InputFile
        }
    }
}

function Test-AIAvailable {
    param([string]$Provider)
    
    switch ($Provider) {
        "claude" { 
            $null = Get-Command claude -ErrorAction SilentlyContinue
            return $?
        }
        "droid" {
            $null = Get-Command droid -ErrorAction SilentlyContinue
            return $?
        }
        "aider" {
            $null = Get-Command aider -ErrorAction SilentlyContinue
            return $?
        }
    }
    return $false
}

function Get-AvailableAIs {
    $available = @()
    @("claude", "droid", "aider") | ForEach-Object {
        if (Test-AIAvailable -Provider $_) {
            $available += $_
        }
    }
    return $available
}

function Get-FirstAvailableAI {
    # Oncelik sirasi: claude > droid > aider
    $priority = @("claude", "droid", "aider")
    foreach ($ai in $priority) {
        if (Test-AIAvailable -Provider $ai) {
            return $ai
        }
    }
    return $null
}
```

## AI CLI Komut Detaylari

### Claude Code

| Islem | Komut |
|-------|-------|
| Interactive mode | `claude` |
| Non-interactive | `claude -p "query"` |
| Pipe ile girdi | `cat file \| claude -p "query"` |
| Cikti formati | `claude -p "query" --output-format json` |

### Droid (Factory AI)

| Islem | Komut |
|-------|-------|
| Interactive mode | `droid` |
| Non-interactive | `droid exec "query"` |
| Pipe ile girdi | `cat file \| droid exec "query"` |
| Dosyadan prompt | `droid exec -f prompt.md` |
| Autonomy seviyesi | `droid exec --auto low/medium/high` |
| Cikti formati | `droid exec -o json "query"` |

### Aider

| Islem | Komut |
|-------|-------|
| Interactive mode | `aider file1 file2` |
| Non-interactive | `aider --message "query" file1` |
| Dosyadan mesaj | `aider --message-file prompt.md file1` |
| Otomatik onay | `aider --yes --message "query"` |
| Auto-commit kapat | `aider --no-auto-commits` |
| Dry run | `aider --dry-run --message "query"` |

## Cikti Parse Stratejisi

### Yaklasim

AI'ya dogru formati ogretelim, parse etmeye gerek kalmasin.

| Yaklasim | Sorun |
|----------|-------|
| AI serbest yazsin, biz parse edelim | Regex karmasik, hataya acik, her AI farkli yazar |
| AI bizim formatimizda yazsin | Temiz cikti, dogrudan dosyaya yazilir |

### Dosya Ayirici Format

AI ciktisinda her dosya `### FILE: path` marker'i ile baslar:

```markdown
### FILE: tasks/001-user-registration.md
# Feature 1: User Registration

**Feature ID:** F001
**Feature Name:** User Registration
**Priority:** P1 - Critical
**Status:** NOT_STARTED
...

### FILE: tasks/002-password-reset.md
# Feature 2: Password Reset
...

### FILE: tasks/tasks-status.md
# Task Status Tracker
...
```

### Parse Fonksiyonu

```powershell
function Split-AIOutput {
    param(
        [Parameter(Mandatory)]
        [string]$Output,
        
        [string]$OutputDir = "tasks"
    )
    
    # tasks/ klasorunu olustur
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    }
    
    # "### FILE: path" ile bolelim
    $pattern = "(?m)^### FILE:\s*(.+?)$"
    $matches = [regex]::Matches($Output, $pattern)
    
    $files = @()
    
    for ($i = 0; $i -lt $matches.Count; $i++) {
        $match = $matches[$i]
        $filePath = $match.Groups[1].Value.Trim()
        
        # Icerik: bu match'ten sonraki match'e kadar
        $startIndex = $match.Index + $match.Length
        $endIndex = if ($i -lt $matches.Count - 1) {
            $matches[$i + 1].Index
        } else {
            $Output.Length
        }
        
        $content = $Output.Substring($startIndex, $endIndex - $startIndex).Trim()
        
        # Dosyayi yaz
        $fullPath = Join-Path $OutputDir (Split-Path $filePath -Leaf)
        $content | Set-Content $fullPath -Encoding UTF8
        
        $files += @{
            Path = $fullPath
            Name = Split-Path $filePath -Leaf
        }
    }
    
    return $files
}
```

### Validasyon

Parse sonrasi kontrol:

```powershell
function Test-ParsedOutput {
    param([array]$Files)
    
    $valid = $true
    
    foreach ($file in $Files) {
        $content = Get-Content $file.Path -Raw
        
        # Feature dosyasi mi?
        if ($file.Name -match "^\d{3}-") {
            # Feature ID var mi?
            if ($content -notmatch "\*\*Feature ID:\*\*\s*F\d+") {
                Write-Warning "Missing Feature ID in $($file.Name)"
                $valid = $false
            }
            
            # En az bir task var mi?
            if ($content -notmatch "###\s+T\d+:") {
                Write-Warning "No tasks found in $($file.Name)"
                $valid = $false
            }
        }
    }
    
    return $valid
}
```

## lib/prompts/prd-parser.md

AI'ya gonderilecek zenginlestirilmis prompt:

````markdown
# PRD to Task-Plan Parser

You are a technical project planner. Analyze the PRD below and create task files.

## Output Rules

1. Output ONLY the file contents - no explanations, no commentary
2. Each file starts with `### FILE: tasks/XXX-filename.md` marker
3. Follow the exact format shown in the example below
4. Create tasks-status.md as the last file

## File Naming

- Feature files: `001-feature-name.md`, `002-feature-name.md`, etc.
- Status file: `tasks-status.md`
- Use kebab-case for filenames (lowercase, hyphens)

## ID System

- Feature IDs: F001, F002, F003... (3 digits, padded)
- Task IDs: T001, T002, T003... (continues across ALL features)
- Example: F001 has T001-T005, F002 starts with T006

## Priority Guidelines

| Priority | When to Use | Examples |
|----------|-------------|----------|
| P1 - Critical | Core functionality, blockers | Auth, Database, Core API |
| P2 - High | Important features | User registration, Main UI |
| P3 - Medium | Nice to have | Settings, Preferences |
| P4 - Low | Polish, optimization | Analytics, Minor UX |

## Effort Estimation

| Task Type | Typical Effort |
|-----------|---------------|
| Simple UI component | 0.5 days |
| Complex UI with state | 1-2 days |
| API endpoint (CRUD) | 1 day |
| API with business logic | 2-3 days |
| Database migration | 0.5-1 day |
| Authentication/Security | 2-3 days |
| Integration (3rd party) | 2-4 days |
| Unit tests | 0.5-1 day per feature |

## Dependency Rules

- UI depends on API (usually)
- API depends on Database schema
- Tests depend on implementation
- Integration depends on core features
- Use task IDs: `- T001 (must complete first)`

## Required Fields (Feature)

- Feature ID (FXXX)
- Feature Name
- Priority (P1-P4)
- Status (always NOT_STARTED)
- Estimated Duration

## Required Fields (Task)

- Task ID (TXXX)
- Task Name
- Status (always NOT_STARTED)
- Priority (P1-P4)
- Estimated Effort (X days)
- Description
- Files to Touch
- Dependencies (or "None")
- Success Criteria (minimum 3 checkboxes)

---

## EXAMPLE OUTPUT FORMAT

### FILE: tasks/001-user-authentication.md
# Feature 1: User Authentication

**Feature ID:** F001
**Feature Name:** User Authentication
**Priority:** P1 - Critical
**Status:** NOT_STARTED
**Estimated Duration:** 1-2 weeks

## Overview

User authentication system with email/password login, session management, and security features.

## Goals

- Secure user authentication
- Session persistence
- Password security

## Success Criteria

- [ ] All tasks completed (T001-T004)
- [ ] Security audit passed
- [ ] Tests passing with 80%+ coverage

## Tasks

### T001: Database Schema

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description

Create users table with required fields for authentication.

#### Technical Details

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Files to Touch

- `migrations/001_create_users.sql` (new)
- `src/models/user.ts` (new)

#### Dependencies

- None

#### Success Criteria

- [ ] Migration runs successfully
- [ ] Rollback works
- [ ] Indexes on email column

---

### T002: Registration API

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

POST /api/auth/register endpoint for new user registration.

#### Files to Touch

- `src/api/auth/register.ts` (new)
- `src/api/routes.ts` (update)

#### Dependencies

- T001 (must complete first)

#### Success Criteria

- [ ] Endpoint accepts email/password
- [ ] Password hashed with bcrypt
- [ ] Returns JWT token
- [ ] Validates email format

---

### T003: Login API

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description

POST /api/auth/login endpoint for user authentication.

#### Files to Touch

- `src/api/auth/login.ts` (new)
- `src/api/routes.ts` (update)

#### Dependencies

- T001 (must complete first)
- T002 (must complete first)

#### Success Criteria

- [ ] Validates credentials
- [ ] Returns JWT token
- [ ] Rate limiting (5 attempts/minute)
- [ ] Logs failed attempts

---

### T004: Auth Unit Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description

Unit tests for authentication flow.

#### Files to Touch

- `tests/auth/register.test.ts` (new)
- `tests/auth/login.test.ts` (new)

#### Dependencies

- T001, T002, T003 (must complete first)

#### Success Criteria

- [ ] Registration tests pass
- [ ] Login tests pass
- [ ] Edge cases covered
- [ ] 80%+ code coverage

---

### FILE: tasks/tasks-status.md
# Task Status Tracker

**Last Updated:** {CURRENT_DATE}
**Total Features:** 1
**Total Tasks:** 4

## Progress Overview

| Feature | ID | Tasks | Completed | Progress |
|---------|-----|-------|-----------|----------|
| User Authentication | F001 | 4 | 0 | 0% |

## By Priority

- **P1 (Critical):** 3 tasks
- **P2 (High):** 1 task
- **P3 (Medium):** 0 tasks
- **P4 (Low):** 0 tasks

## Task List

| Task | Name | Feature | Status | Priority |
|------|------|---------|--------|----------|
| T001 | Database Schema | F001 | NOT_STARTED | P1 |
| T002 | Registration API | F001 | NOT_STARTED | P1 |
| T003 | Login API | F001 | NOT_STARTED | P1 |
| T004 | Auth Unit Tests | F001 | NOT_STARTED | P2 |

---

## NOW PARSE THIS PRD:

{PRD_CONTENT}
````

## Timeout ve Retry Mekanizmasi

### Konfigürasyon

```powershell
$script:Config = @{
    TimeoutSeconds = 1200      # 20 dakika
    MaxRetries = 10
    RetryDelaySeconds = 10
    ExponentialBackoff = $true # Her retry'da delay 2x artar
}
```

### Timeout Fonksiyonu

```powershell
function Invoke-AIWithTimeout {
    param(
        [string]$Provider,
        [string]$PromptText,
        [string]$InputFile,
        [int]$TimeoutSeconds = 1200
    )
    
    $content = Get-Content $InputFile -Raw
    
    # Job olarak baslat
    $job = Start-Job -ScriptBlock {
        param($provider, $content, $prompt)
        
        switch ($provider) {
            "claude" { $content | claude -p $prompt }
            "droid"  { $content | droid exec --auto low $prompt }
            "aider"  { aider --yes --no-auto-commits --message $prompt $using:InputFile }
        }
    } -ArgumentList $Provider, $content, $PromptText
    
    # Timeout ile bekle
    $completed = Wait-Job $job -Timeout $TimeoutSeconds
    
    if (-not $completed) {
        Stop-Job $job -ErrorAction SilentlyContinue
        Remove-Job $job -Force -ErrorAction SilentlyContinue
        throw "AI timeout after $TimeoutSeconds seconds"
    }
    
    $result = Receive-Job $job
    Remove-Job $job -Force
    
    return $result
}
```

### Retry Fonksiyonu

```powershell
function Invoke-AIWithRetry {
    param(
        [string]$Provider,
        [string]$PromptText,
        [string]$InputFile
    )
    
    $maxRetries = $script:Config.MaxRetries
    $retryDelay = $script:Config.RetryDelaySeconds
    $timeout = $script:Config.TimeoutSeconds
    
    for ($attempt = 1; $attempt -le $maxRetries; $attempt++) {
        try {
            Write-Host "[INFO] Attempt $attempt/$maxRetries..." -ForegroundColor Cyan
            
            $result = Invoke-AIWithTimeout -Provider $Provider `
                -PromptText $PromptText -InputFile $InputFile `
                -TimeoutSeconds $timeout
            
            # Ciktiyi validate et
            $files = Split-AIOutput -Output $result
            
            if ($files.Count -eq 0) {
                throw "No files parsed from AI output"
            }
            
            $isValid = Test-ParsedOutput -Files $files
            
            if (-not $isValid) {
                throw "Invalid output format"
            }
            
            # Basarili
            Write-Host "[OK] AI completed successfully" -ForegroundColor Green
            return @{
                Success = $true
                Files = $files
                Attempts = $attempt
            }
        }
        catch {
            Write-Warning "Attempt $attempt failed: $_"
            
            if ($attempt -lt $maxRetries) {
                Write-Host "[INFO] Retrying in $retryDelay seconds..." -ForegroundColor Yellow
                Start-Sleep -Seconds $retryDelay
                
                # Exponential backoff
                if ($script:Config.ExponentialBackoff) {
                    $retryDelay = [Math]::Min($retryDelay * 2, 300)  # Max 5 dakika
                }
            }
            else {
                Write-Error "All $maxRetries attempts failed"
                return @{
                    Success = $false
                    Error = $_.Exception.Message
                    Attempts = $attempt
                }
            }
        }
    }
}
```

### Hata Turleri ve Handling

| Hata Turu | Davranis |
|-----------|----------|
| Timeout | Retry with same timeout |
| Network error | Retry with exponential backoff |
| API rate limit | Retry after longer delay (60s) |
| Invalid format | Retry with same prompt |
| Parse error | Retry up to max attempts |

### Ornek Cikti

```
[INFO] Reading PRD: docs/PRD.md
[INFO] Using AI: claude
[INFO] Attempt 1/10...
[WARN] Attempt 1 failed: AI timeout after 1200 seconds
[INFO] Retrying in 10 seconds...
[INFO] Attempt 2/10...
[OK] AI completed successfully

Created 3 files in 2 attempts.
```

## Buyuk PRD Handling

### Strateji: Warn Only

Buyuk PRD'ler icin kullaniciyi uyar, devam et. Chunking yapmiyoruz cunku:

- Feature bagimliliklari kopabilir
- ID sirasi bozulabilir
- Karmasiklik artar

### Esik Degerleri

```powershell
$script:SizeThresholds = @{
    WarningSize = 100000    # 100K karakter - uyari ver
    LargeSize = 200000      # 200K karakter - ciddi uyari
    MaxSize = 500000        # 500K karakter - cok buyuk uyari
}
```

### Kontrol Fonksiyonu

```powershell
function Test-PrdSize {
    param(
        [Parameter(Mandatory)]
        [string]$PrdFile
    )
    
    $content = Get-Content $PrdFile -Raw
    $size = $content.Length
    $lineCount = ($content -split "`n").Count
    
    Write-Host "[INFO] PRD size: $size characters, $lineCount lines" -ForegroundColor Cyan
    
    if ($size -gt $script:SizeThresholds.MaxSize) {
        Write-Warning "PRD is very large ($size chars). This may take 15-20 minutes."
        Write-Warning "Consider breaking PRD into smaller feature documents."
        Write-Host ""
        Write-Host "Recommendations:" -ForegroundColor Yellow
        Write-Host "  - Split by feature/module into separate files"
        Write-Host "  - Run hermes-prd on each file separately"
        Write-Host "  - Use -Timeout 1800 for extra time"
        Write-Host ""
    }
    elseif ($size -gt $script:SizeThresholds.LargeSize) {
        Write-Warning "PRD is large ($size chars). This may take 10-15 minutes."
    }
    elseif ($size -gt $script:SizeThresholds.WarningSize) {
        Write-Host "[INFO] PRD is medium size. Processing may take 5-10 minutes." -ForegroundColor Yellow
    }
    
    return @{
        Size = $size
        Lines = $lineCount
        IsLarge = $size -gt $script:SizeThresholds.LargeSize
    }
}
```

### Ornek Cikti

```
[INFO] Reading PRD: docs/large-prd.md
[INFO] PRD size: 250000 characters, 3500 lines
[WARN] PRD is large (250000 chars). This may take 10-15 minutes.
[INFO] Using AI: claude
[INFO] Attempt 1/10...
```

## hermes-prd.ps1 Parametreleri

```powershell
param(
    [string]$PrdFile,           # PRD dosya yolu
    [ValidateSet("claude", "droid", "aider", "auto")]
    [string]$AI = "auto",       # auto = ilk bulunani kullan
    [switch]$List,              # Mevcut AI'lari listele
    [switch]$DryRun,            # Sadece goster, dosya olusturma
    [string]$OutputDir = "tasks",
    [int]$Timeout = 1200,       # 20 dakika
    [int]$MaxRetries = 10
)
```

## Cikti Ornegi

```powershell
PS> hermes-prd docs/PRD.md -AI claude

[INFO] Reading PRD: docs/PRD.md
[INFO] Using AI: claude
[INFO] Parsing PRD with Claude Code...

[OK] Created: tasks/001-user-registration.md (F001, T001-T005)
[OK] Created: tasks/002-password-reset.md (F002, T006-T008)
[OK] Created: tasks/003-email-verification.md (F003, T009-T011)
[OK] Created: tasks/tasks-status.md

Summary:
  Features: 3
  Tasks: 11
  Estimated: 8 days

Next: Run 'hermes -TaskMode -AutoBranch -AutoCommit' to start
```

## Uygulama Sirasi

1. `lib/AIProvider.ps1` - AI CLI abstraction
2. `lib/prompts/prd-parser.md` - Parse prompt template
3. `hermes-prd.ps1` - Ana script
4. `install.ps1` guncelle - hermes-prd komutunu ekle
5. Test ve dokumantasyon

## Sonraki Adimlar

Onay sonrasi uygulama:

### Faz 1: Core

- [ ] lib/AIProvider.ps1
- [ ] lib/prompts/prd-parser.md

### Faz 2: Ana Script

- [ ] hermes-prd.ps1

### Faz 3: Entegrasyon

- [ ] install.ps1 guncelleme
- [ ] README.md guncelleme

### Faz 4: Test

- [ ] Unit testler
- [ ] Integration testler
