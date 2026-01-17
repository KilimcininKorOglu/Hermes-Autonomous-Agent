# Hermes Autonomous Agent - Kullanım Kılavuzu

Yapay zeka destekli otonom uygulama geliştirme sistemi Hermes'in eksiksiz kullanım kılavuzu.

## İçerik

1. [Kurulum](#kurulum)
2. [Hızlı Başlangıç](#hızlı-başlangıç)
3. [Proje Başlatma](#proje-başlatma)
4. [Fikirden PRD Üretme](#fikirden-prd-üretme)
5. [PRD Ayrıştırma](#prd-ayrıştırma)
6. [PRD Dönüştürme](#prd-dönüştürme)
7. [Özellik Ekleme](#özellik-ekleme)
8. [Görev Yürütme](#görev-yürütme)
9. [Durum ve İzleme](#durum-ve-izleme)
10. [İnteraktif TUI](#interaktif-tui)
11. [Yapılandırma](#yapılandırma)
12. [Devre Kesici](#devre-kesici)
13. [Kurulum ve Güncelleme](#kurulum-ve-güncelleme)
14. [Otomatik Git Etiketleme](#otomatik-git-etiketleme)
15. [Sorun Giderme](#sorun-giderme)

---

## Kurulum

### Gereksinimler

- Go 1.24 veya üstü
- Git
- Aşağıdaki AI CLI'lardan biri:
  - Claude CLI: `npm install -g @anthropic-ai/claude-code`
  - Droid CLI: `curl -fsSL https://app.factory.ai/cli | sh`
  - Gemini CLI: `npm install -g @google/gemini-cli`

### Kaynaktan Derleme

```bash
# Depoyu klonla
git clone https://github.com/YourUsername/hermes.git
cd hermes

# Platformunuz için derleyin
build.bat              # Windows (varsayılan olarak tüm platformlar)
make                   # Linux/macOS (varsayılan olarak tüm platformlar)
make build             # Sadece mevcut platform

# Binary konumu
bin/hermes-windows-amd64.exe    # Windows
bin/hermes-linux-amd64          # Linux
bin/hermes-darwin-arm64         # macOS Apple Silicon
```

### Kurulumu Doğrulama

```bash
hermes --version
hermes --help
```

---

## Hızlı Başlangıç

```bash
# 1. Yeni proje başlat
hermes init projem
cd projem

# 2. Fikirden PRD üret (v1.1.0'da yeni)
hermes idea "kullanıcı doğrulama ve ödeme ile e-ticaret platformu"

# Veya PRD'nizi manuel olarak .hermes/docs/PRD.md konumuna yerleştirin

# 3. PRD'yi görevlere ayrıştır
hermes prd .hermes/docs/PRD.md

# 4. Oluşturulan görevleri görüntüle
hermes status

# 5. Otonom yürütmeyi başlat
hermes run --auto-branch --auto-commit
```

---

## Proje Başlatma

`hermes init` komutu proje yapısını oluşturur.

### Kullanım

```bash
# Mevcut dizinde başlat
hermes init

# Yeni dizinde başlat
hermes init projem
```

### Oluşturulan Dosyalar

```
projem/
├── .git/                    # Git deposu (yoksa oluşturulur)
├── .gitignore               # Kapsamlı gitignore
└── .hermes/                 # Hermes dizini (gitignore'da)
    ├── config.json          # Proje yapılandırması
    ├── PROMPT.md            # AI prompt şablonu
    ├── tasks/               # Görev dosyaları dizini
    ├── logs/                # Yürütme günlükleri
    └── docs/                # Dokümantasyon (PRD buraya)
```

### Oluşturulan .gitignore

Init komutu kapsamlı bir `.gitignore` oluşturur:

- `.hermes/` - Tüm Hermes verileri
- `node_modules/`, `vendor/`, `venv/` - Bağımlılıklar
- `dist/`, `build/`, `bin/` - Derleme çıktıları
- `.env`, `.env.local` - Ortam dosyaları
- `.idea/`, `.vscode/` - IDE dosyaları
- `*.log`, `logs/` - Günlük dosyaları

### İlk Commit

Başlatma sonrası Hermes ilk commit'i oluşturur:

```
chore: Initialize project with Hermes
```

---

## Fikirden PRD Üretme

Basit bir fikir veya açıklamadan detaylı PRD üretir. Bu özellik v1.1.0'da eklendi.

### Kullanım

```bash
hermes idea <açıklama> [bayraklar]
```

### Bayraklar

| Bayrak          | Kısa | Varsayılan            | Açıklama                            |
|-----------------|------|-----------------------|-------------------------------------|
| `--output`      | `-o` | `.hermes/docs/PRD.md` | Çıktı dosyası yolu                  |
| `--dry-run`     |      | false                 | Dosya yazmadan önizleme             |
| `--interactive` | `-i` | false                 | İnteraktif mod (ek sorular)         |
| `--language`    | `-l` | `en`                  | PRD dili (en/tr)                    |
| `--timeout`     |      | 600                   | AI zaman aşımı (saniye)             |
| `--debug`       |      | false                 | Hata ayıklama çıktısını etkinleştir |

### Örnekler

```bash
# İngilizce PRD üret
hermes idea "alışveriş sepetli e-ticaret sitesi"

# Türkçe PRD üret
hermes idea "blog platformu" --language tr

# Ek sorularla interaktif mod
hermes idea "CRM sistemi" --interactive

# Kaydetmeden önizle
hermes idea "görev yöneticisi" --dry-run

# Özel çıktı yolu
hermes idea "sohbet uygulaması" -o docs/chat-prd.md
```

### İnteraktif Mod

`--interactive` kullanıldığında, Hermes ek sorular sorar:

- Hedef kitle
- Tercih edilen teknoloji stack'i
- Beklenen ölçek (small/medium/large/enterprise)
- Beklenen zaman çizelgesi
- Olmazsa olmaz özellikler

Bu cevaplar daha özelleştirilmiş bir PRD oluşturmaya yardımcı olur.

### Çıktı

Üretilen PRD şunları içerir:

1. Proje Genel Bakışı (ad, açıklama, hedef kitle, hedefler)
2. Özellikler (kullanıcı hikayeleri ve kabul kriterleri ile)
3. Teknik Gereksinimler (stack, mimari, entegrasyonlar)
4. Fonksiyonel Olmayan Gereksinimler (güvenlik, ölçeklenebilirlik, erişilebilirlik)
5. Başarı Metrikleri (KPI'lar, başarı kriterleri)
6. Zaman Çizelgesi ve Kilometre Taşları

---

## PRD Ayrıştırma

Ürün Gereksinimleri Belgesini yapılandırılmış görev dosyalarına dönüştürür.

### Kullanım

```bash
hermes prd <prd-dosyası> [bayraklar]
```

### Bayraklar

| Bayrak          | Varsayılan | Açıklama                            |
|-----------------|------------|-------------------------------------|
| `--dry-run`     | false      | Yazmadan önizleme                   |
| `--timeout`     | 1200       | Saniye cinsinden zaman aşımı        |
| `--max-retries` | 10         | Maksimum yeniden deneme sayısı      |
| `--debug`       | false      | Hata ayıklama çıktısını etkinleştir |

### Örnekler

```bash
# PRD'yi ayrıştır
hermes prd .hermes/docs/PRD.md

# Dosya oluşturmadan önizle
hermes prd gereksinimler.md --dry-run

# Büyük PRD'ler için uzun zaman aşımı
hermes prd buyuk-prd.md --timeout 1800
```

### PRD Format Önerileri

PRD'niz şunları içermelidir:

- Proje genel bakışı
- Özellik açıklamaları
- Kullanıcı hikayeleri
- Teknik gereksinimler
- Kabul kriterleri

### Oluşturulan Görev Dosyaları

Görev dosyaları `.hermes/tasks/` dizininde oluşturulur:

```
.hermes/tasks/
├── 001-kullanici-dogrulama.md
├── 002-urun-katalogu.md
├── 003-alisveris-sepeti.md
└── 004-odeme.md
```

---

## PRD Dönüştürme

PRD dosyalarını farklı formatlar arasında dönüştürür.

### Kullanım

```bash
hermes convertprd [bayraklar]
```

### Açıklama

`convertprd` komutu PRD belgelerini çeşitli formatlar arasında dönüştürmeye yardımcı olur, farklı dokümantasyon stilleriyle çalışmayı ve harici araçlarla entegrasyonu kolaylaştırır.

### Örnekler

```bash
# PRD formatını dönüştür
hermes convertprd

# Belirli seçeneklerle
hermes convertprd --debug
```

---

## Özellik Ekleme

Tüm PRD'yi yeniden ayrıştırmadan bireysel özellikler ekleyin.

### Kullanım

```bash
hermes add <özellik-açıklaması> [bayraklar]
```

### Bayraklar

| Bayrak      | Varsayılan | Açıklama                            |
|-------------|------------|-------------------------------------|
| `--dry-run` | false      | Yazmadan önizleme                   |
| `--timeout` | 300        | Saniye cinsinden zaman aşımı        |
| `--debug`   | false      | Hata ayıklama çıktısını etkinleştir |

### Örnekler

```bash
# Yeni özellik ekle
hermes add "JWT ile kullanıcı doğrulama"

# Önizleme ile ekle
hermes add "karanlık mod" --dry-run

# Karmaşık özellik ekle
hermes add "WebSocket ile gerçek zamanlı bildirimler"
```

### ID Sürekliliği

Hermes otomatik olarak sıradaki uygun ID'leri atar:

- Özellik ID'leri: F001, F002, F003...
- Görev ID'leri: T001, T002... (tüm özellikler boyunca devam eder)

---

## Görev Yürütme

Otomatik ilerleme takibi ile AI kullanarak görevleri yürütür.

### Kullanım

```bash
hermes run [bayraklar]
```

### Bayraklar

| Bayrak          | Varsayılan | Açıklama                               |
|-----------------|------------|----------------------------------------|
| `--ai`          | auto       | AI sağlayıcı (claude/droid/gemini)     |
| `--auto-branch` | config'den | Özellik dalları oluştur                |
| `--auto-commit` | config'den | Tamamlandığında commit at              |
| `--autonomous`  | true       | Duraklamadan çalıştır                  |
| `--timeout`     | config'den | AI zaman aşımı (saniye)                |
| `--debug`       | false      | Hata ayıklama çıktısını etkinleştir    |
| `--parallel`    | false      | Paralel yürütmeyi etkinleştir (v2.0.0) |
| `--workers`     | 3          | Paralel çalışan sayısı                 |
| `--dry-run`     | false      | Sadece yürütme planını önizle          |

### Örnekler

```bash
# Otomatik AI algıla ile çalıştır
hermes run

# Tam otomasyon
hermes run --auto-branch --auto-commit

# Belirli AI sağlayıcı kullan
hermes run --ai gemini

# İnteraktif mod (görevler arası duraklama)
hermes run --autonomous=false

# Özel zaman aşımı ile
hermes run --timeout 600

# Paralel yürütme (v2.0.0)
hermes run --parallel --workers 3

# Yürütme planını çalıştırmadan önizle
hermes run --dry-run
```

### AI Sağlayıcı Önceliği

`--ai auto` (varsayılan) kullanıldığında, sağlayıcılar sırayla denenir:

1. Claude (`claude` komutu)
2. Droid (`droid` komutu)
3. Gemini (`gemini` komutu)

### Paralel Yürütme (v2.0.0)

Ayrı AI ajanları ile birden fazla bağımsız görevi eşzamanlı olarak yürütün:

```bash
# 3 worker ile paralel çalıştır
hermes run --parallel --workers 3 --auto-commit

# Yürütme planını önizle
hermes run --dry-run
```

**Temel Özellikler:**

- **Bağımlılık Grafiği**: Görev bağımlılıklarını otomatik olarak hesaplar
- **Worker Havuzu**: Paralel çalışan birden fazla AI ajanı
- **İzole Çalışma Alanları**: Her worker ayrı git worktree kullanır
- **Çakışma Algılama**: Görevler arası dosya çakışmalarını algılar
- **AI Destekli Birleştirme**: LLM karmaşık çakışmaları çözer
- **Geri Alma Desteği**: Başarısızlıklarda otomatik kurtarma

### Yürütme Akışı

1. Sıradaki tamamlanmamış görevi yükle
2. Görev durumunu `IN_PROGRESS` olarak ayarla
3. Özellik dalı oluştur (`--auto-branch` ise)
4. Görevi AI prompt'una enjekte et
5. Görev talimatlarıyla AI'yı çalıştır
6. Yanıtı tamamlanma için analiz et
7. Görev durumunu `COMPLETED` olarak ayarla
8. Değişiklikleri commit et (`--auto-commit` ise)
9. Tüm görevler tamamlanana kadar tekrarla

### Dal Adlandırması

Özellik dalları şu formatı takip eder:

```
feature/F001-dogrulama
feature/F002-urun-katalogu
```

### Commit Formatı

Commit'ler konvansiyonel format kullanır:

```
feat(T001): Veritabanı Şeması tamamlandı
feat(T002): Kullanıcı Kayıt API tamamlandı
```

### Yürütmeyi Durdurma

Yürütme sırasında `Ctrl+C` tuşuna basın. İlerleme otomatik kaydedilir.

---

## Durum ve İzleme

### Görev Durumu

Tüm görevleri ve durumlarını görüntüle:

```bash
hermes status
```

#### Filtreleme

```bash
# Duruma göre filtrele
hermes status --filter IN_PROGRESS
hermes status --filter COMPLETED
hermes status --filter NOT_STARTED
hermes status --filter BLOCKED

# Önceliğe göre filtrele
hermes status --priority P1
hermes status --priority P2
```

#### Çıktı

```
+--------+---------------------------+--------------+----------+---------+
| ID     | Ad                        | Durum        | Öncelik  | Özellik |
+--------+---------------------------+--------------+----------+---------+
| T001   | Veritabanı Şeması         | COMPLETED    | P1       | F001    |
| T002   | Kullanıcı Kayıt API       | IN_PROGRESS  | P1       | F001    |
| T003   | E-posta Doğrulama         | NOT_STARTED  | P1       | F001    |
+--------+---------------------------+--------------+----------+---------+

Görev İlerlemesi
----------------------------------------
[##########--------------------] 33.3%

Toplam:      3
Tamamlanan:  1
Devam Eden:  1
Başlamadı:   1
Engellenen:  0
----------------------------------------
```

### Görev Detayları

Belirli bir görev hakkında detaylı bilgi görüntüle:

```bash
# Tam ID ile
hermes task T001

# Kısa ID ile
hermes task 1
hermes task 001
```

#### Çıktı

```
Görev: T001
--------------------------------------------------
Ad:       Veritabanı Şeması
Durum:    COMPLETED
Öncelik:  P1
Özellik:  F001

Dokunulacak Dosyalar:
  - db/migrations/001_users.sql
  - db/schema.go

Bağımlılıklar:
  - Yok

Başarı Kriterleri:
  - Migrasyon başarıyla çalışır
  - Geri alma doğru çalışır
  - Şema tasarım belgesiyle eşleşir
```

### Günlükleri Görüntüleme

Yürütme günlüklerini görüntüle:

```bash
# Son 50 satırı göster
hermes log

# Son N satırı göster
hermes log -n 100

# Gerçek zamanlı takip et
hermes log -f

# Seviyeye göre filtrele
hermes log --level ERROR
hermes log --level WARN
```

#### Günlük Seviyeleri

| Seviye  | Renk    | Açıklama              |
|---------|---------|-----------------------|
| ERROR   | Kırmızı | Hata mesajları        |
| WARN    | Sarı    | Uyarı mesajları       |
| SUCCESS | Yeşil   | Başarı mesajları      |
| INFO    | Beyaz   | Bilgilendirme         |
| DEBUG   | Gri     | Hata ayıklama bilgisi |

---

## İnteraktif TUI

İnteraktif terminal kullanıcı arayüzünü başlat:

```bash
hermes tui
```

### Ekranlar

| Tuş | Ekran     | Açıklama                              |
|-----|-----------|---------------------------------------|
| 1   | Dashboard | İlerleme genel bakışı ve devre kesici |
| 2   | Görevler  | Filtreleme ile görev listesi          |
| 3   | Günlükler | Gerçek zamanlı günlük görüntüleyici   |
| ?   | Yardım    | Klavye kısayolları referansı          |

### Dashboard Ekranı

Gösterir:

- Genel ilerleme çubuğu
- Devre kesici durumu
- Mevcut/sıradaki görev
- Görev istatistikleri

### Görevler Ekranı

Özellikler:

- Kaydırılabilir görev listesi
- Durum filtreleme
- Görev detay görünümü

#### Görev Filtreleri

| Tuş | Filtre       |
|-----|--------------|
| a   | Tüm görevler |
| c   | Tamamlanan   |
| p   | Devam Eden   |
| n   | Başlamadı    |
| b   | Engellenen   |

### Günlükler Ekranı

Özellikler:

- Kaydırılabilir günlük görüntüleyici
- Renk kodlu günlük seviyeleri
- Otomatik kaydırma açma/kapama

### Klavye Kısayolları

| Tuş     | Eylem                         |
|---------|-------------------------------|
| 1/2/3/? | Ekran değiştir                |
| r       | Görev yürütmeyi başlat        |
| s       | Yürütmeyi durdur              |
| Shift+R | Manuel yenile                 |
| Enter   | Görev detayını aç             |
| Esc     | Önceki ekrana dön             |
| j/k     | Aşağı/yukarı kaydır           |
| g       | Başa git                      |
| Shift+G | Sona git                      |
| f       | Otomatik kaydırma (günlükler) |
| q       | Çıkış                         |

---

## Yapılandırma

### Yapılandırma Dosyaları

Hermes katmanlı yapılandırma kullanır:

1. CLI bayrakları (en yüksek öncelik)
2. Proje yapılandırması: `.hermes/config.json`
3. Global yapılandırma: `~/.hermes/config.json`
4. Varsayılan değerler (en düşük öncelik)

### Yapılandırma Seçenekleri

```json
{
  "ai": {
    "planning": "claude",
    "coding": "claude",
    "timeout": 300,
    "prdTimeout": 1200,
    "maxRetries": 10,
    "streamOutput": true
  },
  "taskMode": {
    "autoBranch": true,
    "autoCommit": true,
    "autonomous": true,
    "maxConsecutiveErrors": 5
  },
  "loop": {
    "maxCallsPerHour": 100,
    "timeoutMinutes": 15,
    "errorDelay": 10
  },
  "paths": {
    "hermesDir": ".hermes",
    "tasksDir": ".hermes/tasks",
    "logsDir": ".hermes/logs",
    "docsDir": ".hermes/docs"
  }
}
```

### AI Yapılandırması

| Seçenek        | Tip    | Varsayılan | Açıklama                        |
|----------------|--------|------------|---------------------------------|
| `planning`     | string | "claude"   | PRD ayrıştırma için AI          |
| `coding`       | string | "claude"   | Görev yürütme için AI           |
| `timeout`      | int    | 300        | Görev yürütme zaman aşımı (sn)  |
| `prdTimeout`   | int    | 1200       | PRD ayrıştırma zaman aşımı (sn) |
| `maxRetries`   | int    | 10         | Maksimum yeniden deneme         |
| `streamOutput` | bool   | true       | AI çıktısını aktar              |

### Görev Modu Yapılandırması

| Seçenek                | Tip  | Varsayılan | Açıklama                    |
|------------------------|------|------------|-----------------------------|
| `autoBranch`           | bool | true       | Özellik dalları oluştur     |
| `autoCommit`           | bool | true       | Tamamlandığında commit      |
| `autonomous`           | bool | true       | Duraklamadan çalıştır       |
| `maxConsecutiveErrors` | int  | 5          | N ardışık hatadan sonra dur |

### Döngü Yapılandırması

| Seçenek           | Tip | Varsayılan | Açıklama                   |
|-------------------|-----|------------|----------------------------|
| `maxCallsPerHour` | int | 100        | Hız sınırı                 |
| `timeoutMinutes`  | int | 15         | Döngü zaman aşımı          |
| `errorDelay`      | int | 10         | Hatadan sonra gecikme (sn) |

### Paralel Yapılandırma (v2.0.0)

| Seçenek              | Tip    | Varsayılan        | Açıklama                        |
|----------------------|--------|-------------------|---------------------------------|
| `enabled`            | bool   | false             | Varsayılan olarak paralel aç    |
| `maxWorkers`         | int    | 3                 | Maksimum paralel çalışan        |
| `strategy`           | string | "branch-per-task" | Dallanma stratejisi             |
| `conflictResolution` | string | "ai-assisted"     | Çakışma çözüm yöntemi           |
| `isolatedWorkspaces` | bool   | true              | Git worktree kullan             |
| `mergeStrategy`      | string | "sequential"      | Sonuçları birleştirme yöntemi   |
| `maxCostPerHour`     | float  | 0                 | Maliyet sınırı (0 = sınırsız)   |
| `failureStrategy`    | string | "continue"        | fail-fast veya continue         |
| `maxRetries`         | int    | 2                 | Başarısız görevleri tekrar dene |

---

## Devre Kesici

Devre kesici, ilerleme algılanmadığında kontrolsüz yürütmeyi önler.

### Durumlar

| Durum     | Açıklama                                     |
|-----------|----------------------------------------------|
| CLOSED    | Normal çalışma, yürütme izinli               |
| HALF_OPEN | İzleme modu, 2 döngü ilerleme yok            |
| OPEN      | Yürütme durduruldu, manuel sıfırlama gerekli |

### Eşikler

- **HALF_OPEN**: 2 ardışık ilerleme olmayan döngüden sonra tetiklenir
- **OPEN**: 3 ardışık ilerleme olmayan döngüden sonra tetiklenir

### İlerleme Algılama

Hermes AI yanıtlarını şunlar için analiz eder:

- Dosya değişiklikleri
- Kod değişiklikleri
- Tamamlanma sinyalleri
- Hata kalıpları

### Durumu Görüntüleme

Devre kesici durumu şunlarda görünür:

```bash
hermes status    # CLOSED değilse gösterir
hermes tui       # Dashboard ekranı
```

### Sıfırlama

Devre kesici açıldığında, sıfırlayın:

```bash
hermes reset
```

Çıktı:

```
Mevcut durum: OPEN
Sebep: 3 döngü ilerleme yok, devre açılıyor

Devre kesici başarıyla sıfırlandı.
Şimdi 'hermes run' çalıştırabilirsiniz.
```

### Otomatik Kurtarma

İlerleme algılandığında devre kesici otomatik kurtarılır:

- Durum CLOSED'a döner
- Manuel müdahale gerekmez

---

## Kurulum ve Güncelleme

### Sistem Genelinde Kurulum

Hermes'i kolay erişim için sistem PATH'ine kurun:

```bash
# Sistem PATH'ine kur
hermes install

# Sistem PATH'inden kaldır
hermes install --uninstall
```

### Güncelleme Kontrolü

GitHub sürümlerinden güncelleme kontrol edin ve yükleyin:

```bash
# Güncelleme kontrol et ve yükle
hermes update
```

---

## Otomatik Git Etiketleme

Bir özellikteki tüm görevler tamamlandığında ve özellik `Target Version` alanına sahipse, Hermes otomatik olarak bir git etiketi oluşturur.

```bash
# Örnek: Target Version v1.0.0 olan F001 özelliği tamamlandığında
# Hermes şunu oluşturur:
git tag -a v1.0.0 -m "Release v1.0.0: F001 - User Authentication"
```

Etiketler yalnızca şu durumlarda oluşturulur:

- Özellikteki tüm görevler `COMPLETED` durumunda
- Özellik `**Target Version:**` alanına sahip
- Etiket zaten mevcut değil

---

## Sorun Giderme

### Yaygın Sorunlar

#### AI Sağlayıcı Bulunamadı

```
Hata: AI sağlayıcı bulunamadı (claude veya droid yükleyin)
```

**Çözüm**: En az bir AI CLI yükleyin:

```bash
npm install -g @anthropic-ai/claude-code
# veya
curl -fsSL https://app.factory.ai/cli | sh
# veya
npm install -g @google/gemini-cli
```

#### Görev Bulunamadı

```
Hata: görev bulunamadı, önce 'hermes prd <dosya>' çalıştırın
```

**Çözüm**: Önce bir PRD dosyası ayrıştırın:

```bash
hermes prd .hermes/docs/PRD.md
```

#### Devre Kesici Açık

```
Devre kesici AÇIK - yürütme durduruldu
```

**Çözüm**: Devre kesiciyi sıfırlayın:

```bash
hermes reset
```

#### Görev Bulunamadı

```
Hata: görev T001 bulunamadı
```

**Çözüm**: Mevcut görevleri kontrol edin:

```bash
hermes status
```

### Günlük Analizi

Detaylı hata bilgisi için günlükleri kontrol edin:

```bash
# Son hataları görüntüle
hermes log --level ERROR

# Gerçek zamanlı günlükleri takip et
hermes log -f
```

### Hata Ayıklama Modu

Daha fazla bilgi için hata ayıklama çıktısını etkinleştirin:

```bash
hermes run --debug
hermes prd dosya.md --debug
```

### Yardım Alma

```bash
hermes --help
hermes run --help
hermes prd --help
```

---

## Görev Dosyası Format Referansı

### Özellik Başlığı

```markdown
# Özellik N: Özellik Adı
**Özellik ID:** FXXX
**Durum:** NOT_STARTED
```

### Görev Tanımı

```markdown
### TXXX: Görev Adı
**Durum:** NOT_STARTED
**Öncelik:** P1
**Dokunulacak Dosyalar:** dosya1.go, dosya2.go
**Bağımlılıklar:** T001, T002
**Başarı Kriterleri:**
- Kriter 1
- Kriter 2
- Kriter 3
```

### Durum Değerleri

| Durum       | Açıklama                              |
|-------------|---------------------------------------|
| NOT_STARTED | Görev başlamadı                       |
| IN_PROGRESS | Görev üzerinde çalışılıyor            |
| COMPLETED   | Görev tamamlandı                      |
| BLOCKED     | Görev bağımlılık nedeniyle engellendi |
| AT_RISK     | Görev son tarihi kaçırabilir          |
| PAUSED      | Görev geçici olarak askıya alındı     |

### Öncelik Değerleri

| Öncelik | Açıklama |
|---------|----------|
| P1      | Kritik   |
| P2      | Yüksek   |
| P3      | Orta     |
| P4      | Düşük    |

---

## En İyi Uygulamalar

### PRD Yazımı

1. Gereksinimler hakkında spesifik olun
2. Kabul kriterlerini ekleyin
3. Bağımlılıkları açıkça tanımlayın
4. Büyük özellikleri parçalayın

### Görev Yönetimi

1. P1 görevlerle başlayın
2. Görevleri küçük ve odaklı tutun
3. Açık başarı kriterleri tanımlayın
4. Değiştirilecek dosyaları belirtin

### Yürütme

1. Temiz geçmiş için `--auto-branch` kullanın
2. Artımlı kayıtlar için `--auto-commit` kullanın
3. `hermes tui` ile izleyin
4. Günlükleri düzenli kontrol edin

### Kurtarma

1. İlerleme için `hermes status` kontrol edin
2. İzlemek için `hermes log -f` kullanın
3. Takılırsa devre kesiciyi sıfırlayın
4. Görev bağımlılıklarını gözden geçirin
