# Timezone Design — UTC Storage, WITA Experience

**Status:** Target desain untuk M1+ (belum diterapkan di konfigurasi M0)
**Target:** Server OS, PostgreSQL, backend, dan storage **UTC**; input/display dan jadwal operasional **WITA** (`Asia/Makassar`).
**Sumber requirement:** PRD §3.8, §8; `docs/Domain Rules.md`.

> Dokumen ini adalah kontrak implementasi. Jangan memakai `Asia/Jakarta`, offset `+8`, atau `time.FixedZone`. WITA adalah IANA `Asia/Makassar` (UTC+8, tanpa DST).

---

## 1. Prinsip

| Lapisan | Aturan |
|---|---|
| Server OS / container | `TZ=UTC` |
| PostgreSQL | Session/server `timezone=UTC`; simpan `TIMESTAMPTZ` |
| Backend Go | `APP_TIMEZONE=Asia/Makassar`; `time.LoadLocation`; tidak ada offset tetap |
| Storage | Event timestamp UTC; tanggal bisnis `DATE` |
| API | Timestamp ISO 8601 UTC (`Z`/offset); tanggal `YYYY-MM-DD` |
| Web | Input/display WITA via `Intl`/`date-fns-tz`; konversi ke UTC saat submit |
| Worker | Cron location `Asia/Makassar`; eksekusi job dalam UTC |
| Reporting | Bucket hari/minggu/bulan memakai batas WITA |

WITA tidak memiliki DST. Tetap gunakan IANA timezone agar kode tidak bergantung pada asumsi offset dan tetap benar jika aturan zona berubah.

---

## 2. Tipe data

### 2.1 `TIMESTAMPTZ` — event & audit

Gunakan untuk semua momen waktu:

* `created_at`, `updated_at`, `synced_at`, `detail_synced_at`
* `posted_at`, `reserved_at`, `released_at`, `consumed_at`, `used_at`
* `started_at`, `closed_at`, `last_completed_at`, `verified_at`, `overridden_at`
* `read_at`, `decided_at`, `changed_at`
* `recorded_at` (meter reading)

Aturan:

* Jangan memakai `TIMESTAMP WITHOUT TIME ZONE`.
* Jangan menyimpan string waktu.
* Default `now()` boleh dipakai; nilainya `timestamptz` (UTC secara internal).
* Saat menampilkan, konversi ke WITA di aplikasi—bukan mengubah nilai storage.

### 2.2 `DATE` — tanggal bisnis

Gunakan untuk tanggal kalender yang tidak punya jam:

* `p2h_reports.report_date`
* `work_orders.due_date`
* `pm_plans.effective_from`, `effective_to`
* `pm_schedules.due_date`
* filter laporan `date_from`, `date_to`

Opname tidak memakai kolom tanggal terpisah; gunakan `created_at`/`posted_at` (`TIMESTAMPTZ`).

Aturan:

* `DATE` **tidak** dikonversi timezone.
* `due_date = '2026-11-18'` berarti 18 November 2026 sepanjang hari WITA.
* Jangan mengubah `DATE` menjadi `TIMESTAMPTZ` untuk "memperjelas" zona—itu menciptakan bug.

### 2.3 Numeric / interval

* HM/KM: `numeric(12,1)`.
* Durasi: `numeric` jam/hari atau `interval`; simpan durasi, bukan string.
* Jangan menyimpan "WITA" di dalam timestamp; zona adalah metadata tampilan.

---

## 3. Konfigurasi

### 3.1 Environment

```env
# Target
TZ=UTC
APP_TIMEZONE=Asia/Makassar
PGTZ=UTC
```

* `TZ=UTC` untuk proses container (server, api, worker, web server-side).
* `APP_TIMEZONE=Asia/Makassar` untuk logika bisnis, scheduler, dan format.
* `PGTZ=UTC` untuk libpq; sebaiknya juga set session `timezone=UTC` di connection string:

```text
postgres://user:pass@db:5432/workorder?sslmode=disable&options=-c%20timezone%3DUTC
```

* Validasi `APP_TIMEZONE` saat startup:

```go
loc, err := time.LoadLocation(cfg.AppTimezone)
if err != nil {
    log.Fatal("invalid APP_TIMEZONE", "value", cfg.AppTimezone, "err", err)
}
```

### 3.2 Gap konfigurasi saat ini

M0/.env.example masih memakai:

```env
TZ=Asia/Jakarta
```

Ini **tidak sesuai target**. Perubahan `.env.example`, `docker-compose.yml`, dan README adalah perubahan konfigurasi—bukan bagian dari task dokumentasi ini. Perubahan tersebut harus dilakukan pada milestone implementation dengan approval terpisah.

Jangan mengubah nilai menjadi `TZ=Asia/Makassar` untuk proses container; target adalah `TZ=UTC` + `APP_TIMEZONE=Asia/Makassar`.

---

## 4. Backend Go

### 4.1 Load sekali

```go
type Clock struct {
    loc *time.Location
}

func NewClock(appTimezone string) (*Clock, error) {
    loc, err := time.LoadLocation(appTimezone) // "Asia/Makassar"
    if err != nil {
        return nil, fmt.Errorf("load timezone %q: %w", appTimezone, err)
    }
    return &Clock{loc: loc}, nil
}

func (c *Clock) Now() time.Time { return time.Now().In(c.loc) }
func (c *Clock) Today() time.Time {
    now := c.Now()
    return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, c.loc)
}
```

### 4.2 Parse tanggal bisnis

```go
func ParseBusinessDate(loc *time.Location, s string) (time.Time, error) {
    return time.ParseInLocation("2006-01-02", s, loc)
}
```

Jangan:

```go
// Salah: memakai UTC/local process
time.Parse("2006-01-02", s)

// Salah: hardcode offset
time.FixedZone("WITA", 8*3600)
```

### 4.3 Serialize API

* Timestamp: `t.UTC().Format(time.RFC3339)` → `2026-11-18T07:00:00Z`.
* Date: `t.Format("2006-01-02")`.
* Jangan mengirim `2026-11-18 07:00:00` tanpa offset.
* Untuk input `datetime-local` dari web, terima ISO 8601 dengan offset/UTC; server menyimpan UTC.

### 4.4 Business date

* `current_date` di Go = `clock.Now()` dalam WITA, bukan `time.Now().UTC()`.
* Perhitungan due, aging, compliance, dan report memakai tanggal WITA.
* Jangan memakai `time.Now().Format("2006-01-02")` tanpa `.In(loc)`.

---

## 5. PostgreSQL

### 5.1 Session timezone

Target: `timezone=UTC`. Cek:

```sql
SHOW timezone; -- harus UTC
```

Jika server default bukan UTC, set di database/role/connection. Jangan mengandalkan default OS.

### 5.2 `TIMESTAMPTZ` menyimpan instant

`TIMESTAMPTZ` menyimpan UTC secara internal; session timezone hanya mempengaruhi tampilan/parsing. Dengan session UTC, `created_at::date` menghasilkan tanggal UTC—**bukan** tanggal WITA. Karena itu:

```sql
-- SALAH untuk laporan WITA
SELECT created_at::date AS d, count(*) FROM work_orders GROUP BY 1;

-- BENAR
SELECT (created_at AT TIME ZONE 'Asia/Makassar')::date AS d, count(*)
FROM work_orders
GROUP BY 1;
```

### 5.3 Rentang hari WITA

```sql
-- 18 Nov 2026 00:00 WITA <= created_at < 19 Nov 2026 00:00 WITA
WHERE created_at >= (DATE '2026-11-18'::timestamp AT TIME ZONE 'Asia/Makassar')
  AND created_at <  ((DATE '2026-11-18' + 1)::timestamp AT TIME ZONE 'Asia/Makassar')
```

Gunakan rentang `>= start AND < end`, bukan `BETWEEN`, untuk menghindari batas inklusif ganda.

### 5.4 Bucket minggu/bulan

```sql
-- Minggu dimulai Senin, WITA
SELECT date_trunc('week', created_at AT TIME ZONE 'Asia/Makassar')::date AS week_start,
       count(*)
FROM work_orders
GROUP BY 1;

-- Bulan WITA
SELECT date_trunc('month', created_at AT TIME ZONE 'Asia/Makassar')::date AS month_start,
       count(*)
FROM work_orders
GROUP BY 1;
```

### 5.5 `current_date`

Dengan session UTC, `current_date` adalah tanggal UTC. Jangan memakai `current_date` untuk aturan bisnis WITA. Pilihan:

* Hitung tanggal WITA di Go dan kirim sebagai parameter `DATE`.
* Atau SQL eksplisit:

```sql
(now() AT TIME ZONE 'Asia/Makassar')::date
```

### 5.6 Index

* Query rentang timestamp memakai index biasa pada kolom timestamp.
* Jika sering filter tanggal WITA, pertimbangkan expression index:

```sql
CREATE INDEX idx_work_orders_business_date
  ON work_orders ((created_at AT TIME ZONE 'Asia/Makassar')::date);
```

* Ukur dulu; jangan menambah index tanpa query nyata.
* Untuk `due_date` (`DATE`), index biasa sudah tepat.

### 5.7 `DATE` + timezone

Jangan:

```sql
-- Salah: menggeser tanggal bisnis
due_date::timestamptz AT TIME ZONE 'UTC'
```

`DATE` dibandingkan dengan `DATE` WITA yang dihitung di aplikasi/SQL.

---

## 6. Web

### 6.1 Display

Gunakan `Intl.DateTimeFormat` atau `date-fns-tz`:

```ts
const fmt = new Intl.DateTimeFormat('id-ID', {
  timeZone: 'Asia/Makassar',
  dateStyle: 'medium',
  timeStyle: 'short',
});
fmt.format(new Date(isoUtc)); // 18 Nov 2026, 07.00
```

Jangan:

```ts
new Date(iso).toLocaleString('id-ID'); // mengikuti timezone browser, bukan WITA
```

### 6.2 Input `datetime-local`

`<input type="datetime-local">` menghasilkan wall time tanpa zona (mis. `2026-11-18T07:00`). Interpretasikan sebagai WITA, lalu kirim UTC:

```ts
import { fromZonedTime } from 'date-fns-tz';

const utc = fromZonedTime(`${value}:00`, 'Asia/Makassar');
const iso = utc.toISOString(); // 2026-11-17T23:00:00.000Z
```

Jangan menambahkan `+08:00` manual. Jika tidak ingin menambah `date-fns-tz`, gunakan `Temporal`/polyfill atau helper `Intl` yang tervalidasi—tetap tanpa hardcode offset.

### 6.3 Date-only

* `due_date`, `report_date`, `effective_from`: tampilkan apa adanya (`18 Nov 2026`).
* Jangan mengonversi date-only ke `Date` lalu memformat dengan timezone—bisa bergeser sehari.
* Parse sebagai string `YYYY-MM-DD` dan format manual/`date-fns`.

### 6.4 Hydration

* Format waktu di client component atau gunakan nilai terformat dari server.
* Jika SSR + client format berbeda, hydration mismatch. Pilih satu strategi dan konsisten.
* Jangan memakai `new Date().toLocaleString()` tanpa `timeZone`.

---

## 7. Worker & scheduler

### 7.1 Cron location

```go
cron := cron.New(cron.WithLocation(loc)) // Asia/Makassar
cron.AddFunc("0 7 * * *", jobPMScan)     // 07:00 WITA
cron.AddFunc("30 7 * * *", jobStockAlert) // 07:30 WITA
```

Jika library tidak mendukung location, hitung next run di WITA lalu konversi ke UTC sebelum sleep/register.

### 7.2 Job schedule

| Job | Waktu WITA | UTC |
|---|---|---|
| PM due scan / auto-WO | 07:00 | 23:00 hari sebelumnya |
| Stock alert | 07:30 | 23:30 hari sebelumnya |
| Candidate reminder | hourly | hourly |
| Weekly report | Senin 08:00 | Senin 00:00 |
| Daily report | 08:00 | 00:00 |

### 7.3 Eksekusi

* Log timestamp UTC dan WITA:

```json
{"ts":"2026-11-17T23:00:00Z","wita":"2026-11-18T07:00:00+08:00","job":"pm_scan"}
```

* Job harus idempoten; auto-WO memakai lock + partial unique.
* Jika job terlewat (server mati), jalankan catch-up sekali; jangan membuat duplikat.
* Jangan mengandalkan `time.Now().Hour()` untuk menentukan jadwal; cron menangani.

### 7.4 Konversi

| WITA | UTC |
|---|---|
| 00:00 | 16:00 hari sebelumnya |
| 00:30 | 16:30 hari sebelumnya |
| 07:00 | 23:00 hari sebelumnya |
| 07:30 | 23:30 hari sebelumnya |
| 12:00 | 04:00 |
| 23:30 | 15:30 |
| 23:59 | 15:59 |

WITA tidak DST; konversi selalu +8, tetapi tetap gunakan IANA.

---

## 8. Reporting

* "Hari ini" = tanggal WITA.
* Hari dimulai 00:00 WITA = 16:00 UTC hari sebelumnya.
* Minggu dimulai Senin 00:00 WITA.
* Bulan dimulai tanggal 1 00:00 WITA.
* Filter `date_from`/`date_to` adalah tanggal WITA inklusif; server mengubah ke rentang UTC.
* Export CSV mencantumkan `business_date` WITA dan `generated_at` UTC.
* Dashboard card "hari ini" memakai WITA; jangan memakai `current_date` UTC.

Contoh query:

```sql
-- Parameter $1 = DATE WITA, $2 = DATE WITA (inklusif)
WHERE created_at >= ($1::timestamp AT TIME ZONE 'Asia/Makassar')
  AND created_at <  (($2::date + 1)::timestamp AT TIME ZONE 'Asia/Makassar')
```

---

## 9. API contract

| Field | Format | Contoh |
|---|---|---|
| Timestamp | RFC3339 UTC | `2026-11-17T23:00:00Z` |
| Tanggal bisnis | `YYYY-MM-DD` | `2026-11-18` |
| Filter tanggal | `YYYY-MM-DD` WITA | `?date_from=2026-11-01&date_to=2026-11-18` |
| Input datetime | ISO 8601 UTC/offset | `2026-11-17T23:00:00Z` |
| Durasi | number (menit/jam) | `90` |

Aturan:

* Jangan mengirim timestamp tanpa offset.
* Jangan mengirim `DATE` sebagai timestamp tengah malam UTC.
* Error validasi tanggal harus menyebut format WITA.
* API tidak mengembalikan nama zona yang berbeda-beda; timestamp UTC, tanggal bisnis eksplisit.

---

## 10. Migrasi & kondisi saat ini

* M0 sudah memakai `TIMESTAMPTZ` untuk event dan `DATE` untuk `report_date`; tidak perlu konversi storage.
* Display berubah dari `Asia/Jakarta` ke `Asia/Makassar`; tidak ada perubahan nilai timestamp.
* `.env.example` masih `TZ=Asia/Jakarta`; target `TZ=UTC` + `APP_TIMEZONE=Asia/Makassar`.
* `docker-compose.yml` dan README mungkin menyebut timezone lama; perubahan konfigurasi harus melalui approval terpisah.
* Jika ditemukan `TIMESTAMP WITHOUT TIME ZONE` di masa depan:
  1. Identifikasi apakah nilainya WITA atau UTC.
  2. Migrasikan ke `TIMESTAMPTZ` dengan konversi eksplisit.
  3. Audit data sebelum/sesudah.
  4. Jangan menganggap semua timestamp naif adalah UTC.
* `DATE` existing tidak diubah.

---

## 11. Testing

### 11.1 Unit test Go

* `APP_TIMEZONE=Asia/Makassar` valid; invalid gagal startup.
* `07:00 WITA` job next run = `23:00 UTC` hari sebelumnya.
* `00:30 WITA` = `16:30 UTC` hari sebelumnya.
* `23:30 WITA` = `15:30 UTC` hari yang sama.
* Business date `2026-11-18` untuk timestamp `2026-11-17T23:30:00Z` (07:30 WITA) = `2026-11-18`.
* Business date untuk `2026-11-17T16:30:00Z` (00:30 WITA) = `2026-11-18`.
* Business date untuk `2026-11-18T15:30:00Z` (23:30 WITA) = `2026-11-18`.
* Parse `2026-11-18` dengan `ParseInLocation` menghasilkan 00:00 WITA.

### 11.2 DB test

* Insert `TIMESTAMPTZ` `2026-11-17T23:00:00Z`; query `AT TIME ZONE 'Asia/Makassar'` = `2026-11-18 07:00`.
* Daily bucket: transaksi 23:30 WITA masuk hari WITA yang benar, bukan hari UTC berikutnya.
* Range query `date_from/date_to` inklusif.
* `DATE` tidak bergeser.

### 11.3 Web test

* Timestamp UTC tampil 07:00 WITA.
* Input `2026-11-18T07:00` dikirim `2026-11-17T23:00:00Z`.
* Date-only `2026-11-18` tampil 18 Nov 2026, bukan 17/19.
* Tidak ada hydration mismatch.

### 11.4 CI guard

Cari dan larang:

```text
Asia/Jakarta
+08:00
+8
FixedZone
time.FixedZone
timestamp without time zone
created_at::date
current_date (tanpa konteks WITA)
```

Pengecualian: dokumentasi migrasi yang secara eksplisit menyebut nilai lama.

---

## 12. Do / Don't

| Do | Don't |
|---|---|
| `time.LoadLocation("Asia/Makassar")` | `time.FixedZone("WITA", 8*3600)` |
| `TIMESTAMPTZ` untuk event | `TIMESTAMP` tanpa zona |
| `DATE` untuk tanggal bisnis | Timestamp tengah malam untuk due date |
| `AT TIME ZONE 'Asia/Makassar'` di query | `created_at::date` untuk laporan WITA |
| Hitung `current_date` WITA eksplisit | `current_date` UTC untuk aturan bisnis |
| `Intl`/`date-fns-tz` di web | `toLocaleString()` tanpa `timeZone` |
| Cron `WithLocation(loc)` | `time.Now().Hour() == 7` |
| Log UTC + WITA | Log tanpa zona |
| Test boundary 00:00/23:59 WITA | Asumsi "tanggal sama dengan UTC" |

---

## 13. Keputusan terkait

* D-05: timezone global WITA `Asia/Makassar`, storage UTC.
* PRD §3.8 (TZ-01–TZ-08), §8.
* `docs/Domain Rules.md` §1.2, §2.3, §7.3.
* `docs/Data Model.md` §1, §5.

Jika ada pertanyaan timezone yang belum terjawab, default-nya adalah: **simpan UTC, tampilkan WITA, hitung bisnis di WITA, dan jangan hardcode offset.**
