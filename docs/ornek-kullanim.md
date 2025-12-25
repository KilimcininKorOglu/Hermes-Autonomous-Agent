# Ralph Ornek Kullanim Rehberi

Bu dokuman, `sample-prd.md` dosyasini kullanarak Ralph ile nasil calisilacagini adim adim gostermektedir.

---

## Senaryo

E-ticaret platformu PRD'sini (`sample-prd.md`) kullanarak otonom gelistirme yapacagiz.

---

## Adim 1: Proje Olusturma

```powershell
# Yeni proje olustur
ralph-setup ecommerce-platform
cd ecommerce-platform
```

**Olusturulan Yapi:**
```
ecommerce-platform/
├── PROMPT.md
├── @fix_plan.md
├── @AGENT.md
├── specs/
├── src/
├── logs/
└── README.md
```

---

## Adim 2: PRD Dosyasini Kopyalama

```powershell
# PRD dosyasini proje icerisine kopyala
mkdir docs
copy "C:\path\to\ralph-claude-code\docs\sample-prd.md" "docs\PRD.md"
```

---

## Adim 3: PRD'yi Task'lara Donusturme

### Onizleme (DryRun)

```powershell
ralph-prd docs/PRD.md -DryRun
```

**Beklenen Cikti:**
```
Ralph PRD Parser
================

[INFO] Reading PRD: docs/PRD.md
[INFO] PRD size: 5200 characters, 180 lines
[INFO] Using AI: claude
[INFO] Attempt 1/10...
[OK] AI completed successfully

Files to create:

  [+] tasks/001-user-authentication.md (45 lines)
  [+] tasks/002-product-catalog.md (52 lines)
  [+] tasks/003-shopping-cart.md (48 lines)
  [+] tasks/004-checkout-orders.md (55 lines)
  [+] tasks/005-admin-panel.md (42 lines)
  [+] tasks/tasks-status.md (35 lines)

Summary (DryRun):
  New Features: 5
  New Tasks: 18
  Estimated: 25 days

Run without -DryRun to create files.
```

### Gercek Olusturma

```powershell
ralph-prd docs/PRD.md
```

**Olusturulan Dosyalar:**
```
tasks/
├── 001-user-authentication.md    # F001, T001-T004
├── 002-product-catalog.md        # F002, T005-T008
├── 003-shopping-cart.md          # F003, T009-T011
├── 004-checkout-orders.md        # F004, T012-T015
├── 005-admin-panel.md            # F005, T016-T018
└── tasks-status.md               # Durum takibi
```

---

## Adim 4: Task Durumunu Kontrol Etme

```powershell
ralph -TaskStatus
```

**Beklenen Cikti:**
```
============================================================
  TASK STATUS
============================================================

┌──────────┬─────────────────────────────────────┬──────────────┬──────────┬──────────┐
│ Task ID  │ Task Name                           │ Status       │ Priority │ Feature  │
├──────────┼─────────────────────────────────────┼──────────────┼──────────┼──────────┤
│ T001     │ Database Schema                     │ NOT_STARTED  │ P1       │ F001     │
│ T002     │ Registration API                    │ NOT_STARTED  │ P1       │ F001     │
│ T003     │ Login API                           │ NOT_STARTED  │ P1       │ F001     │
│ T004     │ Email Verification                  │ NOT_STARTED  │ P2       │ F001     │
│ T005     │ Product List API                    │ NOT_STARTED  │ P1       │ F002     │
│ ...      │ ...                                 │ ...          │ ...      │ ...      │
└──────────┴─────────────────────────────────────┴──────────────┴──────────┴──────────┘

Summary:
  Total: 18 tasks
  COMPLETED:    0 (0%)
  IN_PROGRESS:  0 (0%)
  NOT_STARTED:  18 (100%)
  BLOCKED:      0 (0%)

Progress: [░░░░░░░░░░░░░░░░░░░░░░░░░] 0%

Next Task: T001 - Database Schema

============================================================
```

### Filtreleme Ornekleri

```powershell
# Sadece P1 oncelikli task'lar
ralph -TaskStatus -PriorityFilter P1

# Belirli feature'in task'lari
ralph -TaskStatus -FeatureFilter F001

# Tamamlanan task'lar
ralph -TaskStatus -StatusFilter COMPLETED
```

---

## Adim 5: Task Mode'u Baslatma

### Temel Kullanim

```powershell
ralph -TaskMode
```

Bu komut:
1. Siradaki task'i bulur (T001)
2. PROMPT.md'ye task detaylarini enjekte eder
3. AI'i calistirir
4. Tamamlaninca durumu gunceller
5. Kullanicidan onay bekler

### Tam Otomasyon

```powershell
ralph -TaskMode -AutoBranch -AutoCommit
```

Bu komut ek olarak:
- `feature/F001-user-authentication` branch'i olusturur
- Her task sonrasi otomatik commit atar
- Feature tamamlaninca main'e merge eder

### Otonom Mod

```powershell
ralph -TaskMode -AutoBranch -AutoCommit -Autonomous
```

Bu komut:
- Kullanici mudahalesi olmadan tum task'lari yapar
- Feature'lar arasi otomatik gecer
- Hata durumunda devam etmeye calisir

---

## Adim 6: Ilerlemeyi Izleme

### Ayri Terminal Penceresinde

```powershell
# Terminal 1: Ralph calistir
ralph -TaskMode -AutoBranch -AutoCommit -Monitor

# Terminal 2: Izleme paneli otomatik acilir
```

### Manuel Izleme

```powershell
# Baska bir terminalde
ralph-monitor
```

### Log Dosyalarini Inceleme

```powershell
# Son log
Get-Content logs/ralph.log -Tail 50

# Son AI ciktisi
Get-ChildItem logs/*_output_*.log | 
    Sort-Object LastWriteTime -Descending | 
    Select-Object -First 1 | 
    Get-Content
```

---

## Adim 7: Belirli Task'tan Baslama

Eger belirli bir task'tan baslamak isterseniz:

```powershell
# T005'ten basla
ralph -TaskMode -StartFrom T005 -AutoBranch -AutoCommit
```

---

## Adim 8: Kesinti Sonrasi Devam Etme

Ralph otomatik olarak kaldigi yerden devam eder:

```powershell
# Ilk calisma - T003'te kesiliyor
ralph -TaskMode -AutoBranch -AutoCommit
# Ctrl+C veya context limit

# Ikinci calisma - otomatik T004'ten devam
ralph -TaskMode -AutoBranch -AutoCommit
```

**Cikti:**
```
============================================================
  Previous run detected - Resuming
============================================================
  Resume Task: T004
  Branch: feature/F001-user-authentication
============================================================
```

---

## Adim 9: Yeni Feature Ekleme

PRD'ye yeni ozellik eklendiyse:

### Yontem 1: PRD'yi Tekrar Calistirma (Incremental)

```powershell
# PRD'yi guncelle
notepad docs/PRD.md

# Tekrar calistir - sadece yeni feature'lar eklenir
ralph-prd docs/PRD.md
```

### Yontem 2: Tekil Feature Ekleme

```powershell
# Satir ici aciklama
ralph-add "Kullanici profil sayfasi ve avatar yukleme"

# Dosyadan
ralph-add @docs/new-feature-spec.md
```

---

## Adim 10: Farkli AI Provider Kullanma

```powershell
# Droid ile PRD parse
ralph-prd docs/PRD.md -AI droid

# Aider ile Task Mode
ralph -TaskMode -AI aider -AutoBranch -AutoCommit

# Mevcut provider'lari listele
ralph-prd -List
```

---

## Ornek Is Akisi Ozeti

```
1. ralph-setup ecommerce-platform
2. cd ecommerce-platform
3. # PRD dosyasini docs/PRD.md olarak kopyala
4. ralph-prd docs/PRD.md -DryRun          # Onizle
5. ralph-prd docs/PRD.md                   # Task olustur
6. ralph -TaskStatus                       # Durumu gor
7. ralph -TaskMode -AutoBranch -AutoCommit -Autonomous  # Baslat
8. # ... Ralph calisiyor ...
9. ralph -TaskStatus                       # Ilerlemeyi kontrol et
```

---

## Beklenen Git Gecmisi

Task Mode tamamlandiktan sonra:

```
* abc1234 (HEAD -> main) Merge feature F005 - Admin Panel
|\
| * def5678 feat(T018): Admin Reports completed
| * ghi9012 feat(T017): Order Management completed
| * jkl3456 feat(T016): Product CRUD completed
|/
* mno7890 Merge feature F004 - Checkout Orders
|\
| * ...
|/
* pqr1234 Merge feature F003 - Shopping Cart
* stu5678 Merge feature F002 - Product Catalog
* vwx9012 Merge feature F001 - User Authentication
* yza3456 Initial commit
```

---

## Sorun Giderme

### PRD Parse Basarisiz

```powershell
# Timeout artir
ralph-prd docs/PRD.md -Timeout 1800

# Farkli AI dene
ralph-prd docs/PRD.md -AI droid
```

### Task Blocked

```powershell
# Blocked task'lari gor
ralph -TaskStatus -StatusFilter BLOCKED

# Manuel olarak sonraki task'a gec
ralph -TaskMode -StartFrom T006
```

### Circuit Breaker Acildi

```powershell
# Durumu kontrol et
ralph -CircuitStatus

# Sifirla
ralph -ResetCircuit

# Tekrar baslat
ralph -TaskMode -AutoBranch -AutoCommit
```

---

## Ipuclari

1. **Buyuk PRD'ler icin:** PRD'yi feature bazli ayri dosyalara bolun
2. **Hata takibi:** `logs/` klasorunu duzenli kontrol edin
3. **Branch temizligi:** Merge edilen branch'ler otomatik silinir
4. **Incremental:** PRD'yi guncelleyip tekrar calistirdiginizda sadece yeni feature'lar eklenir
5. **DryRun:** Her zaman once `-DryRun` ile kontrol edin

---

**Hazir!** Artik Ralph ile otonom gelistirme yapabilirsiniz.
