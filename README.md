# GSI Work Order & Maintenance

Aplikasi work order dan maintenance untuk PT GSI. Repository ini menggunakan
mono-repo dengan empat service Docker: `web`, `api`, `worker`, dan `db`.

**Status:** Milestone 0 (scaffold) — autentikasi, sinkronisasi Odoo, usulan WO,
dan fitur bisnis belum diimplementasikan. Verifikasi runtime berbasis Docker
belum dijalankan karena Docker tidak tersedia di mesin pengembangan; langkah
verifikasi di bawah harus dijalankan di server Linux.

## Arsitektur

```
Browser
  │
  ▼
web (Next.js 15) ──► api (Go + Gin) ──► PostgreSQL 16
                         ▲
                         │
worker (Go + cron) ──────┘
  │
  └──► Odoo (read-only, mulai Milestone 2; browser tidak pernah memanggil Odoo)
```

- `web` — Next.js 15 App Router, TypeScript, Tailwind CSS, shadcn/ui.
- `api` — REST API Go/Gin, pgx/v5, sqlc; menjalankan migrasi saat startup.
- `worker` — proses Go terpisah dengan cron internal; belum memiliki job di M0.
- `db` — PostgreSQL 16, satu instance, schema `public`.

## Struktur repository

```
gsi-workorder/
├── apps/web/                  # Next.js
├── services/api/
│   ├── cmd/api/               # server HTTP
│   ├── cmd/worker/            # worker
│   ├── internal/              # config, db, httpx, store
│   ├── queries/               # query sqlc
│   └── sqlc.yaml
├── migrations/                # golang-migrate SQL
├── docker-compose.yml
├── Dockerfile.api
├── Dockerfile.web
└── .env.example
```

## Prasyarat

### Server (verifikasi M0)

- Linux dengan Docker Engine 24+ dan Docker Compose v2.
- Port `3000` dan `8080` tersedia, atau gunakan SSH tunnel.
- RAM minimal 2 GB, disarankan 4 GB, untuk build image Next.js.
- Akses internet untuk menarik image Docker, dependency Go, dan dependency npm
  saat build pertama.
- Direktori deploy, misalnya `/opt/gsi-workorder`.

### Pengembangan lokal (opsional)

- Go 1.22+
- Node.js 22+
- PostgreSQL 16 bila ingin menjalankan API di luar Docker
- sqlc hanya diperlukan jika mengubah query

Docker tidak wajib untuk mengembangkan frontend atau menjalankan unit test Go,
tetapi DoD M0 memerlukan verifikasi Docker di server.

## Konfigurasi

Salin contoh konfigurasi, lalu ubah secret:

```bash
cp .env.example .env
```

Minimal ubah:

1. `POSTGRES_PASSWORD` — password database yang kuat dan URL-safe.
2. `DATABASE_URL` — gunakan password yang sama dengan `POSTGRES_PASSWORD`.
3. `JWT_SECRET` — 32+ karakter acak. Belum dipakai di M0, tetapi wajib disiapkan
   untuk Milestone 1.

Contoh membuat secret:

```bash
openssl rand -base64 32
```

Jika password mengandung karakter khusus seperti `@`, `:`, `/`, atau `#`,
URL-encode karakter tersebut di `DATABASE_URL`, atau gunakan password yang hanya
berisi huruf, angka, `-`, dan `_`.

Jangan commit file `.env`. File tersebut sudah diabaikan oleh `.gitignore`.

## Menjalankan aplikasi

```bash
docker compose up -d --build
docker compose ps
```

Tunggu sampai keempat service berstatus `healthy`:

- `db`
- `api`
- `worker`
- `web`

Akses:

- Web: `http://<server>:3000`
- API health: `http://<server>:8080/healthz`

Jika port tidak diekspos ke publik, gunakan SSH tunnel:

```bash
ssh -L 3000:127.0.0.1:3000 -L 8080:127.0.0.1:8080 <user>@<server>
```

Lalu buka `http://localhost:3000` dan `http://localhost:8080/healthz`.

## Verifikasi Milestone 0

### 1. Healthcheck API

```bash
curl -fsS http://127.0.0.1:8080/healthz
```

Ekspektasi:

```json
{"database":"up","status":"ok"}
```

Jika database tidak dapat dihubungi, endpoint mengembalikan HTTP 503.

### 2. Halaman web

```bash
curl -fsS http://127.0.0.1:3000 | grep -F "GSI Work Order"
```

Buka juga halaman di browser dan pastikan DevTools Console tidak menampilkan
error.

### 3. Empat service healthy

```bash
docker compose ps
```

Semua service harus berstatus `healthy`.

### 4. Tabel database

```bash
docker compose exec -T db psql -U gsi -d workorder -c "
select count(*) as app_tables
from information_schema.tables
where table_schema = 'public'
  and table_name <> 'schema_migrations';
"
```

Ekspektasi baseline M0: **15** tabel aplikasi. Setelah migration M8B `000012_m8_components`, ada **16** tabel aplikasi karena tambahan `unit_component_installations`.

Daftar tabel:

| Kelompok | Tabel |
|---|---|
| Identitas/master | `users`, `employees`, `units` |
| Komponen | `unit_component_installations` |
| P2H | `p2h_reports`, `p2h_findings` |
| Usulan & WO | `wo_candidates`, `work_orders`, `wo_findings`, `wo_updates`, `wo_attachments` |
| Notifikasi & setting | `notifications`, `app_settings`, `app_settings_history` |
| Sinkronisasi | `sync_runs`, `sync_state` |

`golang-migrate` juga membuat tabel teknis `schema_migrations`, sehingga `\dt`
akan menampilkan 17 tabel setelah M8B (16 tabel saat baseline M0). Angka “14 tabel” pada brief M0 adalah typo; Appendix
A2 memuat 15 tabel aplikasi baseline.

### 5. Seed setting

```bash
docker compose exec -T db psql -U gsi -d workorder -c "
select key, value from app_settings order by key;
"
```

Ekspektasi:

| key | value |
|---|---|
| `candidate_backfill_days` | `30` |
| `candidate_sla_hours` | `24` |
| `p2h_auto_wo` | `false` |

### 6. Guardrail WO

```bash
docker compose exec -T db psql -U gsi -d workorder -c "select count(*) from work_orders;"
```

Ekspektasi: `0`. Milestone 0 tidak boleh membuat work order.

### 7. Ekstensi schema yang disetujui

```bash
docker compose exec -T db psql -U gsi -d workorder -c "
select column_name, data_type, is_nullable
from information_schema.columns
where table_name = 'wo_candidates'
  and column_name = 'notified_at';
"
```

Ekspektasi: satu baris `notified_at | timestamp with time zone | YES`.

### 8. Migrasi idempoten

```bash
docker compose restart api
docker compose logs --tail=50 api
```

Ekspektasi: log migrasi menyebut schema sudah terbaru / `no change`, bukan
error.

### 9. Fresh start

```bash
docker compose down -v
docker compose up -d --build
```

**Peringatan:** `down -v` menghapus volume `pgdata` dan `uploads`. Semua data
hilang. Setelah fresh start, ulangi pemeriksaan di atas.

## Migrasi database

- File migrasi berada di `migrations/` dan memakai format golang-migrate.
- Migrasi dijalankan otomatis oleh service `api` saat startup.
- Service `worker` **tidak** menjalankan migrasi.
- Migrasi bersifat versioned; menjalankan ulang tidak mengulang migrasi yang
  sudah diterapkan.
- Jika migrasi gagal, container API berhenti dan healthcheck gagal. Periksa log
  `docker compose logs api`.

File saat ini:

```
000001_init_schema.up.sql / .down.sql
000002_seed_app_settings.up.sql / .down.sql
```

Down migration tersedia sebagai file, tetapi belum ada perintah rollback di
runtime M0. Untuk pengembangan, gunakan golang-migrate CLI dengan database
scratch, bukan database produksi.

## Setting aplikasi

Sumber kebenaran setting adalah tabel `app_settings`, bukan environment
variable. Default `p2h_auto_wo = false` berarti temuan P2H hanya menghasilkan
usulan dan menunggu keputusan pengguna.

Perubahan resmi (mulai Milestone 4):

```
PATCH /api/settings
```

Hanya role `admin` yang boleh mengubah, dan perubahan dicatat di
`app_settings_history`.

Untuk verifikasi M0, perubahan manual dapat dilakukan lewat SQL:

```bash
docker compose exec -T db psql -U gsi -d workorder -c "
update app_settings
set value = 'true', updated_at = now()
where key = 'p2h_auto_wo';
"
```

**Peringatan:** SQL manual tidak menulis `app_settings_history` dan belum
mengubah perilaku apa pun karena sinkronisasi baru tersedia di M2/M4. Gunakan
hanya untuk verifikasi. Kembalikan ke `false` setelah selesai:

```sql
update app_settings
set value = 'false', updated_at = now()
where key = 'p2h_auto_wo';
```

Environment variable `CANDIDATE_AUTO_APPROVE`, `CANDIDATE_SLA_HOURS`, dan
`CANDIDATE_BACKFILL_DAYS` di `.env.example` hanya dokumentasi/reserved dan tidak
dibaca aplikasi.

## Sinkronisasi

Pada Milestone 0, `worker` hanya:

- membuka koneksi database,
- menjalankan scheduler cron kosong,
- menyediakan subcommand `health` untuk healthcheck.

Belum ada panggilan ke Odoo dan belum ada job sinkronisasi.

Mulai Milestone 2:

```bash
docker compose run --rm worker /app/worker sync-full
```

Perintah tersebut akan melakukan backfill penuh. Sinkronisasi periodik akan
menggunakan `SYNC_INTERVAL` (default `15m`) dan `SYNC_CONCURRENCY` (default `4`).
Odoo hanya diakses oleh worker; semua endpoint baca aplikasi hanya query
PostgreSQL.

Sejak M8C, setiap sync P2H juga meng-upsert reading HM/KM (`hm_awal`/`km_akhir`)
ke `unit_meter_readings` dan merefresh `units.current_hm`/`current_km`. Untuk
backfill data P2H yang sudah ada tanpa memanggil Odoo:

```bash
docker compose run --rm worker /app/worker backfill-p2h-readings
```

Reading P2H yang lebih rendah dari reading valid sebelumnya ditandai anomali,
tidak dipakai untuk current unit, dan dikirim sebagai notifikasi admin.

## Logs & troubleshooting

Melihat log:

```bash
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f web
docker compose logs -f db
```

Masalah umum:

| Gejala | Penyebab / solusi |
|---|---|
| `POSTGRES_PASSWORD wajib diisi` | Belum membuat `.env` atau variabel kosong. |
| API gagal connect database | Password di `POSTGRES_PASSWORD` dan `DATABASE_URL` berbeda; karakter khusus belum di-URL-encode. |
| Port 3000/8080 sudah dipakai | Hentikan proses lain atau ubah mapping port di compose. |
| Web tidak healthy | Cek `docker compose logs web`; pastikan build Next.js selesai. |
| Migrasi dirty | Database pernah gagal di tengah migrasi. Perbaiki manual atau reset database development dengan `down -v`. |
| Perubahan password DB tidak berlaku | Volume `pgdata` menyimpan password lama. Reset dengan `down -v` untuk development. |

## Backup dan restore

Backup manual:

```bash
docker compose exec -T db pg_dump -U gsi -d workorder -Fc > workorder-$(date +%F).dump
```

Restore (hati-hati, menimpa objek yang ada):

```bash
cat workorder-YYYY-MM-DD.dump | docker compose exec -T db pg_restore -U gsi -d workorder --clean --if-exists
```

Backup harian otomatis belum termasuk Milestone 0.

## Keamanan

- Jangan commit `.env`, token Odoo, atau password apa pun.
- Database tidak mengekspos port ke host secara default.
- Ganti password contoh sebelum deploy.
- `JWT_SECRET` wajib acak dan panjang; jangan memakai nilai contoh.
- Browser tidak boleh memanggil Odoo; integrasi Odoo hanya lewat worker.
- Semua query baca aplikasi harus tetap berjalan saat Odoo tidak tersedia.

## Pengembangan lokal

### Backend

```bash
cd services/api
go vet ./...
go test ./...
go build ./...
```

Menjalankan API tanpa Docker memerlukan PostgreSQL lokal dan `DATABASE_URL`.
Dari direktori `services/api`, arahkan migrasi ke root repo:

```bash
MIGRATIONS_DIR=../../migrations go run ./cmd/api
```

Di PowerShell:

```powershell
$env:MIGRATIONS_DIR="../../migrations"
go run ./cmd/api
```

### Frontend

```bash
cd apps/web
npm ci
npm run dev
```

Pemeriksaan:

```bash
npm run lint
npm run build
```

### Regenerasi sqlc

Generated code sudah disertakan di repository, sehingga Docker build tidak
membutuhkan sqlc. Jika schema atau query berubah:

```bash
cd services/api
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

sqlc v1.31.1 membutuhkan toolchain Go yang lebih baru daripada Go 1.22. Dengan
pengaturan default `GOTOOLCHAIN=auto`, Go akan mengunduh toolchain yang
dibutuhkan secara otomatis. Generated code sudah disertakan di repository, jadi
Docker build tidak memerlukan langkah ini.

Konfigurasi sqlc hanya memakai file schema `000001_init_schema.up.sql`; file seed
tidak ikut diparsing.

## Status milestone

Status lokal per 2026-11-23. Lihat `docs/Agent Brief.md` untuk rincian acceptance.

| Milestone | Status |
|---|---|
| M0 — scaffold, compose, migrasi, seed, healthcheck, web | Diimplementasikan; terverifikasi di server UAT Linux |
| M1 — auth Odoo, JWT, sinkron user | Accepted |
| M2 — sinkron P2H, temuan, usulan | Accepted |
| M3 — UI approve/skip, bulk | Accepted |
| M4 — notifikasi, reminder, toggle, PM core | Accepted di server (`000007_m4_pm_core`) |
| M5 — sparepart, compatibility, inventory | Selesai lokal (`000008_m5_spareparts_inventory`); menunggu acceptance server |
| M6 — material requirement & readiness | Selesai lokal (`000009_m6_material_requirements`); menunggu acceptance server |
| M7A — unit history, timeline, downtime | Accepted tahap 1 / `v0.1.7` (`000010_m7_history_reporting`); acceptance server UAT terpisah |
| M7B — dashboard, report CSV, audit hardening | Accepted tahap 1 / `v0.1.7` (`000011_m7b_reporting`); acceptance server UAT terpisah |
| M8A — discovery master unit & komponen | Selesai discovery (`docs/M8 Discovery.md`) |
| M8B — master komponen, instalasi per unit, lifetime API | Selesai lokal (`000012_m8_components`); M8D belum |
| M8C — P2H HM/KM ingestion, anomali, backfill | Selesai lokal (`000013_m8c_meter_anomaly`); backfill via worker |
| M8D — UI komponen unit, lifetime, install/replace/transfer, master komponen | Selesai lokal; import data & threshold final pending PAM |

Baseline rilis tahap 1: **`v0.1.7` (M7)**. UI/operasional tahap 1 sudah accepted; acceptance server UAT M5/M6/M7A/M7B tetap dicatat terpisah. Angka `7` merujuk ke milestone M7; `M7B` adalah sub-milestone reporting. M8B–M8D adalah pekerjaan pasca-baseline dan belum menjadi bagian rilis tahap 1.

Jangan melanjutkan ke milestone berikutnya tanpa instruksi eksplisit.
