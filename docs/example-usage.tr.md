# Hermes Autonomous Agent - Örnek Kullanım (v2.4)

PRD'den tamamlanmaya kadar Hermes kullanarak proje oluşturmanın adım adım rehberi.

---

## Senaryo: E-Ticaret API Oluşturma

Kullanıcı doğrulama, ürün kataloğu ve alışveriş sepeti özellikleri ile basit bir e-ticaret REST API oluşturacağız.

---

## Adım 1: Projeyi Başlat

```bash
# Yeni proje oluştur ve başlat
hermes init eticaret-api
cd eticaret-api
```

**Çıktı:**

```
Hermes başlatılıyor: C:\Projeler\eticaret-api

  Başlatıldı: git deposu
  Oluşturuldu: .hermes/
  Oluşturuldu: .hermes/tasks/
  Oluşturuldu: .hermes/logs/
  Oluşturuldu: .hermes/docs/
  Oluşturuldu: .hermes/config.json
  Oluşturuldu: .hermes/PROMPT.md
  Oluşturuldu: .gitignore
  Oluşturuldu: main branch'ında ilk commit

Hermes başarıyla başlatıldı!

Sonraki adımlar:
  1. PRD'nizi .hermes/docs/PRD.md konumuna ekleyin
  2. Çalıştırın: hermes prd .hermes/docs/PRD.md
  3. Çalıştırın: hermes run --auto-branch --auto-commit
```

---

## Adım 2: Fikirden PRD Üret (v1.1.0)

PRD'yi manuel yazmak yerine, basit bir fikirden üretebilirsiniz:

```bash
hermes idea "kullanıcı doğrulama, ürün kataloğu ve alışveriş sepeti ile e-ticaret REST API"
```

**Çıktı:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      Yapay Zeka Destekli Uygulama Geliştirme

Fikirden PRD Üretici
====================

Fikir: kullanıcı doğrulama, ürün kataloğu ve alışveriş sepeti ile e-ticaret REST API
AI: claude
Dil: en

PRD üretiliyor...

[BAŞARILI] PRD üretildi: .hermes/docs/PRD.md

Sonraki adımlar:
  1. İncele: cat .hermes/docs/PRD.md
  2. Ayrıştır: hermes prd .hermes/docs/PRD.md
```

### İnteraktif Mod

Daha detaylı PRD için interaktif mod kullanın:

```bash
hermes idea "e-ticaret API" --interactive
```

Bu, hedef kitle, teknoloji stack'i, ölçek ve zaman çizelgesi hakkında ek sorular soracaktır.

---

## Adım 3: PRD'yi İncele veya Manuel Oluştur

Üretilen PRD'yi inceleyebilir veya `.hermes/docs/PRD.md` dosyasını manuel olarak gereksinimlerinizle oluşturabilirsiniz:

```markdown
# E-Ticaret API - Ürün Gereksinimleri Belgesi

## Genel Bakış
Go ve PostgreSQL kullanarak bir e-ticaret platformu için REST API oluşturun.

## Özellikler

### Özellik 1: Kullanıcı Doğrulama
- E-posta/şifre ile kullanıcı kaydı
- E-posta doğrulama
- JWT token'ları ile giriş
- Şifre sıfırlama işlevi

### Özellik 2: Ürün Kataloğu
- Ürünler için CRUD işlemleri
- Kategori yönetimi
- Ürün arama ve filtreleme
- Sayfalama desteği

### Özellik 3: Alışveriş Sepeti
- Ürün ekle/kaldır
- Miktarları güncelle
- Toplam hesapla
- Giriş yapmış kullanıcılar için sepeti sakla

## Teknik Gereksinimler
- Go 1.21+
- PostgreSQL 15+
- Doğrulama için JWT
- RESTful API tasarımı
- Girdi doğrulama
- Hata yönetimi
```

---

## Adım 4: PRD'yi Görevlere Ayrıştır

```bash
hermes prd .hermes/docs/PRD.md
```

**Çıktı:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      Yapay Zeka Destekli Uygulama Geliştirme

PRD Ayrıştırıcı
===============

PRD dosyası: .hermes/docs/PRD.md (1247 karakter)
Kullanılan AI: claude

Oluşturuldu: .hermes/tasks/001-kullanici-dogrulama.md
Oluşturuldu: .hermes/tasks/002-urun-katalogu.md
Oluşturuldu: .hermes/tasks/003-alisveris-sepeti.md

.hermes/tasks dizininde 3 görev dosyası oluşturuldu
```

---

## Adım 5: Oluşturulan Görevleri İncele

```bash
hermes status
```

**Çıktı:**

```
+-------+--------------------------------+--------------+----------+---------+
| ID    | Ad                             | Durum        | Öncelik  | Özellik |
+-------+--------------------------------+--------------+----------+---------+
| T001  | Kullanıcılar için DB Şeması    | NOT_STARTED  | P1       | F001    |
| T002  | Kullanıcı Kayıt Endpoint       | NOT_STARTED  | P1       | F001    |
| T003  | E-posta Doğrulama Sistemi      | NOT_STARTED  | P1       | F001    |
| T004  | JWT ile Giriş Endpoint         | NOT_STARTED  | P1       | F001    |
| T005  | Şifre Sıfırlama Akışı          | NOT_STARTED  | P2       | F001    |
| T006  | Ürünler için DB Şeması         | NOT_STARTED  | P1       | F002    |
| T007  | Ürün CRUD Endpoint'leri        | NOT_STARTED  | P1       | F002    |
| T008  | Kategori Yönetimi              | NOT_STARTED  | P2       | F002    |
| T009  | Ürün Arama ve Filtreleme       | NOT_STARTED  | P2       | F002    |
| T010  | Sayfalama Uygulaması           | NOT_STARTED  | P2       | F002    |
| T011  | Sepet DB Şeması                | NOT_STARTED  | P1       | F003    |
| T012  | Sepete Ürün Ekle/Kaldır        | NOT_STARTED  | P1       | F003    |
| T013  | Sepet Miktarlarını Güncelle    | NOT_STARTED  | P1       | F003    |
| T014  | Sepet Toplam Hesaplama         | NOT_STARTED  | P1       | F003    |
+-------+--------------------------------+--------------+----------+---------+

Görev İlerlemesi
----------------------------------------
[------------------------------] 0.0%

Toplam:      14
Tamamlanan:  0
Devam Eden:  0
Başlamadı:   14
Engellenen:  0
----------------------------------------
```

---

## Adım 6: Görev Detaylarını Görüntüle

```bash
hermes task T001
```

**Çıktı:**

```
Görev: T001
--------------------------------------------------
Ad:       Kullanıcılar için Veritabanı Şeması
Durum:    NOT_STARTED
Öncelik:  P1
Özellik:  F001

Dokunulacak Dosyalar:
  - db/migrations/001_create_users.sql
  - internal/models/user.go

Bağımlılıklar:
  - Yok

Başarı Kriterleri:
  - id, email, password_hash, created_at ile users tablosu oluşturuldu
  - Email sütununda unique constraint var
  - Migrasyon hatasız çalışıyor
  - Geri alma doğru çalışıyor
```

---

## Adım 7: Görev Yürütmeyi Başlat

### Seçenek A: Tam Otomasyon

```bash
hermes run --auto-branch --auto-commit
```

### Seçenek B: İnteraktif Mod

```bash
hermes run --auto-branch --auto-commit --autonomous=false
```

### Seçenek C: Belirli AI Sağlayıcı Kullan

```bash
hermes run --ai claude --auto-branch --auto-commit
hermes run --ai droid --auto-branch --auto-commit
hermes run --ai opencode --auto-branch --auto-commit
hermes run --ai gemini --auto-branch --auto-commit
```

### Seçenek D: Paralel Yürütme (v2.0.0)

```bash
# Önce yürütme planını önizle
hermes run --dry-run

# 3 paralel worker ile çalıştır
hermes run --parallel --workers 3 --auto-commit
```

**Çıktı:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      Yapay Zeka Destekli Uygulama Geliştirme

Görev Yürütme Döngüsü
=====================

[INFO] Kullanılan AI sağlayıcı: claude

========================================
Döngü #1
========================================

Görev: T001 - Kullanıcılar için Veritabanı Şeması
Özellik: F001 - Kullanıcı Doğrulama
Öncelik: P1
Durum: NOT_STARTED

[INFO] Görev üzerinde çalışılıyor: T001 - Kullanıcılar için Veritabanı Şeması
[INFO] Dal: feature/F001-kullanici-dogrulama

... AI yürütme çıktısı ...

[SUCCESS] Görev T001 tamamlandı
[SUCCESS] Görev T001 commit edildi

İlerleme: [###---------------------------] 7.1%

========================================
Döngü #2
========================================

Görev: T002 - Kullanıcı Kayıt Endpoint
...
```

---

## Adım 8: İlerlemeyi İzle

### Durumu Kontrol Et

```bash
hermes status
```

**Bazı görevler tamamlandıktan sonra çıktı:**

```
+-------+--------------------------------+--------------+----------+---------+
| ID    | Ad                             | Durum        | Öncelik  | Özellik |
+-------+--------------------------------+--------------+----------+---------+
| T001  | Kullanıcılar için DB Şeması    | COMPLETED    | P1       | F001    |
| T002  | Kullanıcı Kayıt Endpoint       | COMPLETED    | P1       | F001    |
| T003  | E-posta Doğrulama Sistemi      | IN_PROGRESS  | P1       | F001    |
| T004  | JWT ile Giriş Endpoint         | NOT_STARTED  | P1       | F001    |
...
+-------+--------------------------------+--------------+----------+---------+

Görev İlerlemesi
----------------------------------------
[######------------------------] 21.4%

Toplam:      14
Tamamlanan:  3
Devam Eden:  1
Başlamadı:   10
Engellenen:  0
----------------------------------------
```

### Günlükleri Görüntüle

```bash
# Son 50 satır
hermes log

# Gerçek zamanlı takip
hermes log -f

# Sadece hatalar
hermes log --level ERROR
```

### İnteraktif TUI Kullan

```bash
hermes tui
```

Gezinme:

- `1` - Dashboard
- `2` - Görev listesi
- `3` - Günlükler
- `4` - Fikir üretici
- `5` - PRD ayrıştırıcı
- `6` - Özellik ekle
- `7` - Ayarlar (27 yapılandırma seçeneği)
- `8` - Devre kesici
- `9` - Güncelleme
- `0` - Proje başlat
- `r` - Görev çalıştır (ilerleme çubuğu ile)
- `?` - Yardım

---

## Adım 9: Sorunları Ele Al

### Devre Kesici Açılırsa

```bash
# Durumu kontrol et
hermes status

# Devre kesiciyi sıfırla
hermes reset

# Yürütmeye devam et
hermes run --auto-branch --auto-commit
```

### Durmanız Gerekirse

Yürütme sırasında `Ctrl+C` tuşuna basın. İlerleme otomatik kaydedilir.

### Kesintiden Sonra Devam

```bash
# Tekrar çalıştırın - Hermes son tamamlanmamış görevden devam eder
hermes run --auto-branch --auto-commit
```

---

## Adım 10: Yeni Özellik Ekle

İlk geliştirmeden sonra yeni özellik ekleyin:

```bash
hermes add "ödeme akışı ile sipariş yönetimi"
```

**Çıktı:**

```
 _   _
| | | | ___ _ __ _ __ ___   ___  ___
| |_| |/ _ \ '__| '_ ` _ \ / _ \/ __|
|  _  |  __/ |  | | | | | |  __/\__ \
|_| |_|\___|_|  |_| |_| |_|\___||___/

      Yapay Zeka Destekli Uygulama Geliştirme

Özellik Ekleme
==============

Eklenen özellik: ödeme akışı ile sipariş yönetimi

Sonraki Özellik ID: F004
Sonraki Görev ID: T015

Kullanılan AI: claude

Oluşturuldu: .hermes/tasks/004-odeme-akisi-ile-sipar.md
```

---

## Adım 11: Projeyi Tamamla

Tüm görevler tamamlanana kadar çalıştırmaya devam edin:

```bash
hermes run --auto-branch --auto-commit
```

**Son Çıktı:**

```
[SUCCESS] Tüm görevler tamamlandı!

Görev İlerlemesi
----------------------------------------
[##############################] 100.0%

Toplam:      18
Tamamlanan:  18
Devam Eden:  0
Başlamadı:   0
Engellenen:  0
----------------------------------------
```

---

## Git Geçmişi

Tamamlandıktan sonra git geçmişiniz şöyle görünür:

```
* feat(T018): Sipariş onay e-postası
* feat(T017): Ödeme işleme entegrasyonu
* feat(T016): Ödeme endpoint
* feat(T015): Sipariş veritabanı şeması
* feat(T014): Sepet toplam hesaplama
* feat(T013): Sepet miktarlarını güncelle
* feat(T012): Sepete ürün ekle/kaldır
* feat(T011): Sepet veritabanı şeması
* feat(T010): Sayfalama uygulaması
* feat(T009): Ürün arama ve filtreleme
* feat(T008): Kategori yönetimi
* feat(T007): Ürün CRUD endpoint'leri
* feat(T006): Ürünler için veritabanı şeması
* feat(T005): Şifre sıfırlama akışı
* feat(T004): JWT ile giriş endpoint
* feat(T003): E-posta doğrulama sistemi
* feat(T002): Kullanıcı kayıt endpoint
* feat(T001): Kullanıcılar için veritabanı şeması
* chore: Initialize project with Hermes
```

---

## Komut Referansı

| Adım | Komut                                    | Açıklama                         |
|------|------------------------------------------|----------------------------------|
| 1    | `hermes init <ad>`                       | Projeyi başlat                   |
| 2    | `hermes idea "<açıklama>"`               | Fikirden PRD üret                |
| 3    | `.hermes/docs/PRD.md` incele/düzenle     | Gereksinimleri incele            |
| 4    | `hermes prd .hermes/docs/PRD.md`         | PRD'yi görevlere ayrıştır        |
| 5    | `hermes convertprd`                      | PRD formatları arası dönüştür    |
| 6    | `hermes status`                          | Tüm görevleri görüntüle          |
| 7    | `hermes task <id>`                       | Görev detaylarını görüntüle      |
| 8    | `hermes run --auto-branch --auto-commit` | Görevleri yürüt                  |
| 9    | `hermes run --parallel --workers 3`      | Paralel yürütme                  |
| 10   | `hermes run --dry-run`                   | Yürütme planını önizle           |
| 11   | `hermes log -f`                          | Günlükleri izle                  |
| 12   | `hermes reset`                           | Devre kesiciyi sıfırla           |
| 13   | `hermes add "<özellik>"`                 | Yeni özellik ekle                |
| 14   | `hermes tui`                             | İnteraktif arayüz (11 ekran)     |
| 15   | `hermes install`                         | Sistem PATH'ine kur              |
| 16   | `hermes update`                          | Güncellemeleri kontrol et ve kur |

---

## İpuçları

1. **Küçük Başlayın**: Daha iyi sonuçlar için odaklı bir PRD ile başlayın
2. **Görevleri İnceleyin**: Çalıştırmadan önce oluşturulan görevleri kontrol edin
3. **Dallar Kullanın**: Temiz geçmiş için her zaman `--auto-branch` kullanın
4. **İlerlemeyi İzleyin**: `hermes tui` veya `hermes log -f` kullanın
5. **Yinelemeli Çalışın**: `hermes add` ile özellikleri kademeli olarak ekleyin
6. **Önce Önizle**: Çalıştırmadan önce `--dry-run` ile yürütme planını görün
7. **Paralel Mod**: Bağımsız görevler için `--parallel` kullanarak hızlandırın
8. **Sistem Genelinde Kur**: Global erişim için `hermes install` çalıştırın
9. **TUI Çalıştır Ekranını Kullan**: TUI'da `r` tuşuna basarak ilerleme çubuğu ile görsel görev yürütme
