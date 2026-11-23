# Brief untuk opencode / coding agent

Pakai file ini bersama:

* `prds/2026-11-18_work-order-maintenance-gsi.md` — **apa & kenapa** (source of truth requirement).
* `docs/Data Model.md` — ERD, kamus tabel, proposal DDL.
* `docs/Domain Rules.md` — status transition, perhitungan, RBAC.
* `docs/Timezone.md` — desain UTC/WITA dan scheduler.

**Hierarki kebenaran:** migrasi M0 + PRD §5.2/A2 untuk kondisi implemented; PRD §3 dan companion docs untuk requirement/proposal M1+. Jika dokumen bertentangan, berhenti dan tanya—jangan menebak.

---

## 1. Status proyek

**Milestone 0 (scaffold, schema, seed, Docker) sudah implemented dan locked.** Jangan mengubah migrasi, schema, atau perilaku M0 tanpa approval eksplisit.

Fakta M0:

| Item | Nilai |
|---|---|
| Service | `web`, `api`, `worker`, `db` (4 service, Docker Compose) |
| Tabel aplikasi | **15** + `schema_migrations` |
| Seed setting | `p2h_auto_wo=false`, `candidate_sla_hours=24`, `candidate_backfill_days=30`, `pm_auto_wo=false`, `pm_upcoming_days=30`, `pm_due_soon_days=7` |
| Kandidat P2H | status `pending/approved/skipped/auto_created/archived`, kolom `notified_at` |
| Auto-WO P2H | default `false`; hanya jika setting `true` |
| Timezone saat ini | `APP_TIMEZONE` default `Asia/Makassar` dan divalidasi saat startup; `.env.example` dan `docker-compose.yml` masih memakai `TZ=Asia/Jakarta`. Target proses `TZ=UTC` + `APP_TIMEZONE=Asia/Makassar`; perubahan config butuh approval terpisah |

**Fase saat ini (status per 2026-11-23):** M1, M2, M3, dan M4 sudah **accepted**. Verifikasi lokal M1/M2/M3/M4 (Go test/vet/build, ESLint, Next build, dan integration test PostgreSQL) lulus. Sync Odoo nyata diverifikasi pada 2026-11-21: 4.454 report, 8.519 finding, 2.501 kandidat unik (587 pending, 1.914 archived), `work_orders` hasil sync = 0, dan `raw` tanpa foto. Sync kedua idempoten. Kontrak skip divalidasi sebagai 400 dan bulk tetap 422. M4 PM core accepted di server pada migration `000007_m4_pm_core`, termasuk job evaluasi PM 07:00 WITA. M5 Sparepart & Inventory sudah diimplementasikan lokal di migration `000008_m5_spareparts_inventory`: master sparepart/nomor/alternatif/compatibility, PM material template, warehouse/location, transaksi inventory immutable, saldo, stock policy, opname, rekonsiliasi, dan alert stok harian 07:30 WITA. M6 Material sudah diimplementasikan lokal di migration `000009_m6_material_requirements`: material requirement per WO, readiness/shortage, reservasi/release, issue/return + usage, PM snapshot, waiting-part guard, shortage aging, dan rekonsiliasi reserved. M7A (history & downtime unit) selesai lokal dan ter-commit di `fde629d` (migration `000010_m7_history_reporting`). M7B (dashboard operasional, laporan JSON/CSV, agregasi benchmark, audit hardening) selesai lokal di migration `000011_m7b_reporting` dengan endpoint `/dashboard/pm|material|inventory|maintenance` dan `/reports/:name`; seluruh verifikasi lokal lulus (Go test/vet/build dengan PostgreSQL, ESLint, Next build). M5, M6, M7A, dan M7B selesai lokal; acceptance server UAT tetap menunggu akses VPS. Baseline rilis tahap 1 ditetapkan pada 2026-11-23 sebagai **v0.1.7 (M7)**: UI/operasional tahap 1 sudah accepted, sementara acceptance server UAT M5/M6/M7A/M7B tetap menjadi gate terpisah. Angka `7` pada versi merujuk ke milestone M7; `M7B` adalah sub-milestone reporting. Setelah baseline, M8a discovery selesai di `docs/M8 Discovery.md`, dan M8b (master komponen, instalasi komponen per unit, lifetime view, dan API) diimplementasikan lokal di migration `000012_m8_components`. M8b bersifat additive: master `components` diperluas dengan kelas dan lifetime default, tabel `unit_component_installations` menyimpan serial/instalasi/pelepasan/transfer, dan view `v_unit_component_lifetime` menghitung pemakaian dari HM/KM unit. API baru mencakup `/components` (list/detail/create/update), `/units/:id/component-installations` (list/detail/install), `.../:installation_id/replace`, `.../:installation_id/transfer`, dan `/units/:id/component-lifetime`. M8b belum menambahkan UI dan belum mengimpor data komponen. M8c kemudian menambahkan ingestion HM/KM P2H di transaksi sync: `hm_awal` menjadi reading HM, `km_akhir` menjadi reading KM, `source_type='p2h'`, `source_id=p2h_reports.id`, dan `recorded_at` di awal tanggal laporan WITA. Migration `000013_m8c_meter_anomaly` menambahkan `unit_meter_readings.is_anomaly`/`anomaly_reason`; reading P2H yang lebih rendah dari reading valid sebelumnya ditandai anomali, tidak dipakai untuk `units.current_*`, dan dikirim sebagai notifikasi admin. Command backfill `backfill-p2h-readings` menghitung ulang reading/anomali dan merefresh cache unit. M8d menambahkan UI operasional: tab **Komponen** di profil unit untuk melihat lifetime dan melakukan install/replace/transfer, alert komponen `due`/`overdue`, serta tab **Komponen** di Master Unit untuk membuat/mengedit master komponen dan lifetime default. Verifikasi frontend (`npm run lint`, `npm run check:typography`, `npm run build`) lulus. Import master komponen/instalasi awal dan konfirmasi ambang lifetime tetap pending PAM. M9a/M9b kemudian mengubah Usulan WO menjadi **Work Request**: role `dispatch`, Work Request manual, status `validated`/`converted`, kategori Service/Scheduled/Unscheduled/Incident/UBOP, flag breakdown, filter backlog non-breakdown, serta aksi validasi dan konversi terpisah (aksi gabungan tetap tersedia). Migration `000014_m9_work_requests` additive; data P2H existing tetap valid. UI Work Request mencakup form manual, detail P2H/manual, validasi/konversi/skip, dan bulk sesuai status.

> **Verification note:** runtime Docker/Linux M0 dan sync Odoo nyata sudah terverifikasi di server UAT. Route auth, unit, P2H, candidate, notification, dan settings sudah terdaftar dan diuji di router API. Jangan mengubah migrasi atau kode M0.
>
> **README:** Tabel status milestone di README sudah diselaraskan dengan Brief ini pada 2026-11-23. Untuk rincian acceptance dan verifikasi tetap gunakan Brief ini; perubahan status README berikutnya memerlukan approval.

---

## 2. Guardrail non-negotiable

1. **M0 locked.** Perubahan schema hanya melalui migration baru yang additive; jangan edit migrasi M0.
2. **P2H tetap candidate-based.** Sync tidak boleh membuat `work_orders` kecuali `p2h_auto_wo=true`. Default `false`.
3. **PM auto-WO default mati.** Butuh **global `pm_auto_wo=true` + plan `auto_generate_wo=true`**. Hanya admin yang boleh mengubah global; audit di `app_settings_history`.
4. **Tidak ada procurement.** Jangan membuat PR, PO, vendor selection, approval pembelian, harga, cost, valuation, invoice, atau write ke Accurate. Itu domain Accurate.
5. **Inventory ledger immutable.** Posted transaction tidak boleh dihapus dan item/qty-nya tidak boleh diubah; satu-satunya update adalah status `posted → reversed` saat reversal. Koreksi lewat reversal/adjustment. Tidak ada update saldo langsung.
6. **Planning/reservation tidak mengurangi stok fisik.** Hanya issue yang mengurangi `on_hand`; return menambah. Reservation hanya menambah `reserved`.
7. **No negative stock.** `on_hand`, `reserved`, dan available tidak boleh negatif.
8. **Unit history dari transaksi aktual.** Jangan membuat tabel total statis atau menghitung history dari material requirement yang belum di-issue.
9. **UTC storage, WITA experience.** `TIMESTAMPTZ` untuk event, `DATE` untuk tanggal bisnis, `APP_TIMEZONE=Asia/Makassar`, `TZ=UTC` untuk proses. Dilarang hardcode `+8`/`FixedZone`.
10. **4 service, stack tetap.** Tanpa Redis, message broker, search engine, atau ORM (Prisma/GORM).
11. **No duplicate tables.** Lihat §6. Jangan membuat `unit_maintenance_history`, `material_usage`, `inventory_adjustments`, `pm_notifications`, `waiting_material`, `pm_executions`, atau `unit_downtime_events`.
12. **Satu milestone per sesi.** Jangan mengerjakan fitur di luar milestone aktif.
13. **Istilah “unit”.** Semua UI, dokumen, dan teks user-facing wajib memakai “unit” untuk kendaraan/alat; jangan memakai sinonim lama.

---

## 3. Peta modul

| Modul | PRD | Tabel utama | Milestone | Aturan kunci |
|---|---|---|---|---|
| Auth & role | §3.1, §7.9 | `users`, `employees` | M1 | Tambah `storekeeper`; RBAC server-side |
| Unit master | §5.3 | `unit_models`, `engine_models`, `units`, `unit_meter_readings` | M1 | Model/engine nullable; HM/KM dari reading |
| Component master & installation | §5.3 | `components`, `unit_component_installations`, `v_unit_component_lifetime` | M8 | Satu aktif per unit+tipe; serial aktif unik; lifetime dari HM/KM |
| P2H sync | §3.1, A4 | `p2h_reports`, `p2h_findings`, `wo_candidates` | M2 | Candidate-based; idempoten |
| Work Request | §3.1, A4 | `wo_candidates` | M9 | Manual/P2H; pending → validated → converted; kategori/backlog; dispatch |
| WO lifecycle | §7.3 | `work_orders`, `wo_updates`, `wo_attachments` | M3 | `waiting_part` reuse; audit |
| PM | §3.2, §7.1–7.2 | `pm_plans`, `pm_plan_checklist_items`, `pm_schedules`, `work_order_checklist_items` | M4 | Interval plan-level; first reached; auto-WO double control |
| PM material plan | §3.2 PM-08 | `pm_plan_materials` | M5 | Template; material requirement baru di M6 |
| Sparepart | §3.3, §7.6 | `spareparts`, `sparepart_numbers`, `sparepart_alternatives`, `sparepart_compatibility` | M5 | Nomor OEM/brand; compatibility status |
| Inventory | §3.4, §7.5 | `warehouses`, `warehouse_locations`, `inventory_transactions`, `inventory_transaction_items`, `inventory_balances`, `stock_opname_sessions`, `stock_opname_items`, `inventory_stock_policies` | M5 | Ledger immutable; balance cache |
| Material | §3.5, §7.4 | `material_requirements`, `material_requirement_items`, `material_requirement_updates`, `inventory_reservations` | M6 | Planning ≠ stock; issue/return via ledger |
| Usage/history | §3.6 | `work_order_sparepart_usage`, view history | M6/M7 | Actual usage only |
| Dashboard/report | §3.7, §9 | view/query | M7 | WITA buckets; no static totals |
| Timezone | §3.8, §8 | semua timestamp/date | M1+ | UTC storage; WITA display/scheduler |

### 3b. Frontend

- **Stack:** Next.js `15.5.29` App Router, React `19.1.0`, TypeScript, Tailwind CSS v4, shadcn/ui style `base-vega`; font Geist dibundel lokal.
- **Workflow:** gunakan skill `nextjs-15`. Port/parity dari template `next-shadcn-dashboard`; target visual `https://demos.shadcndashboard.dev/?mode=dark`.
- **Dependency:** dependency baru hanya boleh shadcn/base-vega, Recharts, `next-themes` bila diperlukan, dan Geist. Baseline yang sudah ada (`@base-ui/react`, `lucide-react`, `class-variance-authority`, `cn`) dipertahankan; jangan menambah library UI/state/chart lain tanpa approval. Saat ini `next-themes` belum dipakai; tema memakai script lokal dan `theme-toggle`.
- **Data:** jangan commit mock/demo data. Gunakan data API nyata; state loading/empty/error harus ditangani.
- **Brand (accepted 2026-11-23):** UI memakai `PAMS — Plant and Maintenance System`. Rebranding hanya pada frontend/metadata; jangan mengganti nama Go module, Docker service, atau path repo.
- **Versi (accepted 2026-11-23):** badge sidebar `v.0.1.7`, dengan `7` merujuk ke milestone M7; footer memakai `AZV-M7B` untuk menandai sub-milestone reporting.
- **Navigasi:** Beta badge amber kecil pada Laporan, Maintenance, Inventory, Sparepart, Master Unit, dan Work Request; badge disembunyikan saat sidebar collapsed; Work Request berada tepat di bawah Work Order.
- **Dashboard:** ikuti visual demo shadcn, gunakan data API nyata, dan jangan kembali ke data demo/ecommerce. Hero tidak menampilkan tanggal atau summary badge; ilustrasi dibuat lebih kecil.
- **Pola halaman list:** PageHeader → filter card → tabel full-width → pagination di footer card. State loading, empty, dan error wajib rapi di desktop maupun mobile.
- **Selesai bila:** `npm run lint` dan `npm run build` lulus; screenshot desktop + mobile, dark + light, dibandingkan dengan demo target; tidak ada regresi shell/tabel/form M1/M2.

---

## 4. Roadmap

| Milestone | Scope | Jangan lakukan | Uji terima |
|---|---|---|---|
| **M0** ✅ | Scaffold, 4 service, 15 tabel, 3 seed | Mengubah M0 | Compose healthy (verifikasi Linux pending); `/healthz`; seed benar |
| **M1** | Auth Odoo/JWT, RBAC + `storekeeper`, sync user/employee, unit model/engine, meter reading, dashboard shell | PM/inventory/material | Login; role; unit model/engine; HM/KM reading |
| **M2** | Sync P2H/findings, kandidat, notifikasi/reminder, approve/skip/bulk, kandidat → WO | WO otomatis dari sync | Sync 2x idempoten; kandidat pending; WO sync = 0; approve = 1 WO |
| **M3** ✅ | WO manual/lifecycle, assignee, supervisor, `maintenance_type`, `started_at`, attachments, waiting_part | PM/inventory | Transisi valid; audit; done/cancel |
| **M4** ✅ | PM plan/checklist/schedule/due, generate WO, completion/history/compliance | Material plan/requirement, stock check/reservation | Plan unit/model; combined; auto-WO 0 default; next due benar |
| **M5** 🟡 | Sparepart/compatibility, PM material plan (`pm_plan_materials`), warehouse/location, receipt/issue/return/transfer/adjustment/opname, ledger, balance, min stock | Material readiness/reservation | Ledger immutable; negative ditolak; balance reconcile |
| **M6** 🟡 | Material requirement/readiness, reserve/release, issue/return, waiting_part, shortage aging, PM material | Procurement | Planning ≠ stock; issue/return benar; readiness benar |
| **M7** ✅ tahap 1 | Unit profile/history, component interval, downtime, dashboard/report, export, performance/audit | Fitur baru di luar PRD | History dari transaksi; report WITA; p95 |
| **M8** 🟡 | M8a discovery; M8b master komponen/instalasi/lifetime API; M8c P2H HM/KM ingestion + anomaly + backfill; M8d UI unit & master komponen | UI sebelum data HM valid; mengarang data komponen | Migration additive; install/replace/transfer; HM/KM lifetime; anomaly P2H; UI lifetime |
| **M9** 🟡 | Work Request manual/P2H, dispatch role, validate/convert, kategori, breakdown/backlog, UI | Menghapus kandidat P2H atau bypass validasi | Migration additive; manual create; validasi/konversi; backlog non-breakdown |

**Dependensi:** M4 PM core tidak menjanjikan readiness stok; `pm_plan_materials` baru ada di M5, dan material requirement/readiness penuh di M6. Jika PM harus launch dengan material plan/readiness, urutan M4/M5/M6 perlu approval.

**Baseline tahap 1:** `v0.1.7` (M7); `M7B` adalah sub-milestone reporting. Acceptance server UAT tetap menjadi gate terpisah dan tidak membatalkan baseline UI tahap 1.

---

## 5. Definition of Done per milestone

### M1 — Foundation

* [ ] Login Odoo → JWT; refresh; logout; role dari DB.
* [ ] Role baseline `admin/planner/mechanic/storekeeper/viewer`; M9 menambah `dispatch`; otorisasi server-side.
* [ ] Sync user/employee idempoten; user nonaktif tidak dihapus.
* [ ] `unit_models`, `engine_models`, `units` extension, `unit_meter_readings`.
* [ ] Dashboard shell + halaman unit dasar; tidak ada fitur PM/inventory.
* [ ] Migration additive; M0 regression test lulus.
* [ ] `APP_TIMEZONE=Asia/Makassar` divalidasi saat startup.

### M2 — P2H

* [ ] Sync list + detail + findings; concurrency ≤4; ≤1 req/detik.
* [ ] Satu kandidat per laporan bertemuan; `archived` untuk laporan lama.
* [ ] Notifikasi kandidat + reminder 24 jam; `notified_at` sekali.
* [ ] Approve/skip/bulk; skip reason ≥5 karakter; approve ulang 409.
* [ ] `work_orders` hasil sync = 0 saat `p2h_auto_wo=false`.
* [ ] Sync 2x tidak menghasilkan duplikat.

### M3 — WO

* [x] WO manual; kode `WO-YYYYMM-####`.
* [x] Transisi `open → in_progress ↔ waiting_part → done`; waiting_part tidak langsung done; cancel beralasan.
* [x] `started_at`, `closed_at`, `supervisor_id`, `maintenance_type`.
* [x] `wo_updates` append-only; assignment/priority/note tercatat.
* [x] Attachment jpg/png ≤5 MB di volume lokal.
* [x] Waiting-part duration dapat dihitung dari `wo_updates`.

### M4 — PM

* [x] Plan per unit **atau** model; exactly one target.
* [x] Interval HM/KM/calendar; minimal satu; `first_reached` default.
* [x] Schedule per unit; due/upcoming/overdue; override beralasan.
* [x] Checklist snapshot ke WO; PM WO tidak done tanpa reading yang diwajibkan plan (HM/KM/date).
* [x] Generate WO manual; auto-WO hanya double control.
* [x] Completion update schedule, meter reading, next cycle (bila plan aktif/effective).
* [x] PM history/compliance dari WO; belum ada material plan/requirement (M5/M6).
* [x] Tidak ada interval hardcoded.

> Catatan M4: accepted di server pada migration `version=7`; job evaluasi PM berjalan 07:00 WITA. Evaluasi PM dapat dijalankan manual dengan `docker compose exec worker /app/worker evaluate-pm`.

### M5 — Sparepart & Inventory

* [x] Sparepart master + OEM/aftermarket numbers + brand + alternatives.
* [x] Compatibility per unit model/engine/config; status verifikasi dan verifier/timestamp.
* [x] `pm_plan_materials` dapat didefinisikan setelah sparepart master; belum ada material requirement/readiness.
* [x] Warehouse/location; receipt/issue/return/transfer/adjustment/opname.
* [x] Ledger immutable; reversal; balance cache; reconciliation.
* [x] No negative; min/critical stock; balance reconcile dari ledger.
* [x] Movement history; search by part number/unit/engine/compatibility.
* [x] Tidak ada harga/cost/procurement.
* [x] Alert stok low/critical harian 07:30 WITA dengan dedupe per sparepart/lokasi/status/hari.
* [x] Idempotency key transaksi; reversal memakai alasan dan hanya admin/planner.

> Catatan M5: migration `000008_m5_spareparts_inventory` additive, integration test PostgreSQL lokal lulus (termasuk idempotency, stok negatif, reversal, opname, alert dedupe, dan rekonsiliasi), API dapat start, serta ESLint/Next build lulus. Acceptance server dan penerapan migration `000008` masih pending. UI tersedia di `/spareparts`, `/inventory`, dan panel material pada `/pm`.

### M6 — Material

* [x] Material requirement per WO; lifecycle 10 status.
* [x] Planning/checking tidak mengubah stok.
* [x] Readiness Ready/Partially/Shortage; shortage aging.
* [x] Reserve/release (reservation tidak mengurangi `on_hand`); issue/return; usage linked ke ledger.
* [x] WO waiting_part ↔ material status.
* [x] PM material integration; cancel release reservations.
* [x] Tidak ada PR/PO/vendor/harga/Accurate.
* [x] Rekonsiliasi `reserved` terhadap reservasi aktif.
* [x] UI material di detail WO dan tab reservasi inventory.

> Catatan M6: migration `000009_m6_material_requirements` additive; M6 integration test PostgreSQL lokal lulus (readiness, reserve/release, issue/return, PM snapshot, waiting_part, cancel/complete, shortage aging, rekonsiliasi reserved). API startup, Go vet/build, ESLint, dan Next build lulus. Command worker tambahan: `/app/worker recompute-material`. Acceptance server masih pending bersama M5.

### M7 — History & Reporting

* [x] Unit profile, timeline, component history, HM/KM, downtime.
* [x] Component replacement interval dari usage.
* [x] Dashboard PM/material/stock/maintenance/downtime.
* [x] Report PM compliance, shortage aging, consumption, movement, component.
* [x] CSV export; WITA buckets.
* [x] Audit hardening: timezone IANA dari config tanpa fallback fixed offset; parameter SQL terikat, tidak ada hardcode zona.
* [ ] p95 list <300 ms, dashboard <500 ms — perlu pengukuran di server UAT setelah deployment.

> Catatan M7A: migration `000010_m7_history_reporting` menambahkan view timeline unit, history komponen (fallback sparepart), downtime, frequency/interval komponen, dan index performa. Unit profile memiliki tab Ringkasan, Timeline, Sparepart, HM/KM, Downtime, dan Interval; issue material dapat memilih komponen opsional.
>
> Catatan M7B: migration `000011_m7b_reporting` menambahkan view agregat `v_pm_due`, `v_pm_compliance`, `v_waiting_part_units`, `v_sparepart_consumption`, dan `v_inventory_movement` plus index. Paket `internal/reporting` bersifat read-only dan mengembalikan tabel generik `{columns, rows, meta}` di dalam envelope `{data}`. Semua filter tanggal memakai batas hari WITA yang dikonversi ke UTC (`to` = awal hari berikutnya, eksklusif). CSV memakai BOM, CRLF, timestamp WITA, dan escape formula injection. Halaman `/reports` dan widget dashboard operasional memakai komponen UI bersama. Tidak ada data harga, vendor, PR/PO, atau Accurate.

### M8 — Master Unit & Komponen

* [x] M8a discovery: audit data live, gap analysis, draft schema, keputusan, dan entry criteria di `docs/M8 Discovery.md`.
* [x] M8b migration `000012_m8_components`: `component_class` dan lifetime default pada `components`; `unit_component_installations`; `v_unit_component_lifetime`.
* [x] M8b API: master komponen list/detail/create/update; instalasi list/detail/install; replace; transfer; lifetime unit.
* [x] M8b guardrail: satu instalasi aktif per unit+komponen, serial aktif unik, transisi replace/transfer atomik, dan status `removed`/`transferred` menyimpan histori.
* [x] M8c: ingest `hm_awal`/`km_akhir` P2H ke `unit_meter_readings`, refresh `units.current_*`, tandai anomali, dan command backfill.
* [x] M8d: UI tab Komponen di profil unit (lifetime, install/replace/transfer, alert) dan tab master komponen.
* [ ] M8 acceptance: data komponen awal diimpor/disetujui PAM; lifetime threshold dikonfirmasi; UAT server.

> Catatan M8b: migration additive, tanpa seed komponen. Instalasi aktif memakai partial unique index `(unit_id, component_id)` dan serial aktif unik; komponen multi-instance harus dipecah menjadi kode komponen berbeda sampai PAM mengonfirmasi model lain. Nilai lifetime default berasal dari master komponen dan dapat dioverride per instalasi. State lifetime dihitung di Go dengan ambang default: `due` pada 10% tersisa atau 7 hari, `upcoming` pada 20% atau 30 hari; `unset` jika tidak ada lifetime. Replace menutup instalasi lama sebagai `removed`; transfer menutup sebagai `transferred` dan membuat instalasi baru di unit tujuan dalam satu transaksi. M8b tidak menambahkan UI dan tidak mengubah data existing.
>
> Catatan M8c: migration `000013_m8c_meter_anomaly` menambahkan `is_anomaly` dan `anomaly_reason` pada `unit_meter_readings`. Sync P2H meng-upsert HM/KM secara idempoten lewat unique source; `recorded_at` memakai awal tanggal laporan WITA. Reading P2H yang lebih rendah dari reading valid sebelumnya disimpan sebagai anomali, tidak dipakai untuk `units.current_*`, dan dikirim ke admin. Backfill existing dijalankan dengan `/app/worker backfill-p2h-readings`. Integration test PostgreSQL mencakup sync, koreksi, backfill, anomali, notifikasi, dan reset cache saat reading dikosongkan.
>
> Catatan M8d: tab Komponen di profil unit memakai endpoint lifetime dan menampilkan pemakaian/sisa HM/KM/hari, badge status, serta aksi install/replace/transfer lewat centered sheet. Alert muncul bila ada komponen `due`/`overdue`. Master Unit mendapat tab Komponen untuk create/edit master dan lifetime default. Replace/transfer tetap diizinkan untuk master komponen nonaktif; instalasi baru tidak. Import data awal dan threshold final masih pending PAM.

### M9 — Work Request

* [x] M9a migration `000014_m9_work_requests`: role `dispatch`, kolom manual/validasi/konversi/kategori/breakdown, status baru, index backlog.
* [x] M9a API: create manual Work Request, validate, convert, approve gabungan, skip, bulk validate/convert/skip, filter backlog.
* [x] M9b UI: halaman Work Request manual/P2H, filter status/backlog, form create, detail manual/P2H, validasi/konversi/skip, bulk sesuai status.
* [x] Guardrail: P2H existing tetap candidate-based; `approve` lama menjadi aksi gabungan dan menulis status `converted`; data lama `approved`/`auto_created` tetap terbaca.
* [ ] M9 acceptance: UAT dispatch/planner/mechanic dan konfirmasi mapping kategori ke maintenance type.

> Catatan M9: status Work Request adalah `pending → validated → converted`; `skipped` dapat dari pending/validated. `approved` dan `auto_created` dipertahankan sebagai status historis. Backlog default = `pending`/`validated` + `is_breakdown=false`. Work Request manual wajib unit, judul, dan kategori. Konversi manual membuat `work_orders.source='manual'`; konversi P2H tetap `source='p2h'` dan melampirkan findings. Mapping kategori ke `maintenance_type`: `Scheduled→preventive`, `Incident` atau `is_breakdown→breakdown`, lainnya `corrective`. Role `dispatch` dapat membuat, memvalidasi, mengonversi, dan menolak; mechanic dapat membuat manual; admin/planner tetap penuh.

---

## 6. Aturan bisnis ringkas

### 6.1 PM

* Interval di level plan; per item checklist tidak punya interval di v1.
* Combined: `first_reached` default; `all_required` opsional.
* Due calendar = last completed date + interval days; HM/KM = last completed HM/KM + interval.
* Auto-WO = global `pm_auto_wo` **dan** plan `auto_generate_wo`; default false.
* 1 schedule = 1 WO aktif; WO cancelled boleh diganti (partial unique index).
* Checklist di-snapshot ke WO; material plan disalin menjadi material requirement saat M6; perubahan template tidak mengubah WO.
* PM done wajib HM/KM sesuai interval plan; calendar-only cukup tanggal completion. Update next due dan `unit_meter_readings`.

### 6.2 Material

* Status: `Draft → Planned → Checking Stock → Ready|Partially Ready|Shortage → Reserved → Issued → Completed`, plus `Cancelled`.
* Prioritas status: Cancelled → Completed → Issued → Reserved → Shortage → Partially Ready → Ready → Checking Stock → Planned → Draft.
* Ready = semua required item available; Partially Ready = semua required ada sebagian; Shortage = ada required item coverage nol. Item `is_optional` tidak memblokir.
* Planning/checking/reservation **tidak** mengubah `on_hand`.
* Issue mengurangi `on_hand`; return menambah; keduanya via ledger + usage.
* Reserved = semua required item covered (available + reservasi + issue) dan minimal satu reservasi aktif.
* Issued = issue sudah dimulai dan tidak ada required item coverage nol (boleh belum penuh); gap tetap terlihat dari coverage.
* Completed = WO done + reservasi aktif nol.
* Bukan procurement.

### 6.3 Inventory

| Aksi | `on_hand` | `reserved` | Arah ledger |
|---|---|---|---|
| Receipt | + | — | `to_location_id` |
| Issue | − | − (jika reservasi) | `from_location_id` |
| Return | + | — | `to_location_id` |
| Transfer | − asal, + tujuan | — | both |
| Reserve | — | + | `from_location_id` |
| Release | — | − | `from_location_id` |
| Adjustment increase | + | — | `to_location_id` |
| Adjustment decrease | − | — | `from_location_id` |
| Opname | ± | — | adjustment per selisih |

* Posted transaction tidak dihapus; reversal membuat transaction baru.
* `inventory_balances` di-update dalam transaksi DB yang sama; lock baris.
* Tidak ada edit saldo langsung; tidak ada negative.

### 6.4 WO & history

* `waiting_part` hanya untuk material; tidak ada `waiting_material`.
* `maintenance_type`: preventive (planned downtime), breakdown/corrective (unplanned), inspection (tidak dihitung downtime).
* Downtime WO `done` dari `started_at` (fallback `created_at` untuk breakdown/corrective) sampai `closed_at`; waiting part dari `wo_updates`; cancelled tidak dihitung.
* Unit history = WO + checklist + usage + meter readings + downtime; bukan static totals.
* Component interval dari actual usage, bukan plan interval.

### 6.5 Compatibility

* `Pending Verification`, `Approved Compatible`, `Conditional Compatible`, `Not Compatible`.
* Conditional wajib catatan; Not Compatible memblokir pemilihan.
* Verifikasi admin/planner; `verified_by/at`.
* Alternatif hanya disarankan jika verified.

### 6.6 Notifikasi

* Reuse `notifications` + `dedupe_key`; jangan buat `pm_notifications`.
* PM upcoming default 30 hari, due soon default 7 hari, due/overdue harian; threshold per plan dapat override.
* Low/critical stock harian 07:30 WITA.
* Email fase 2; in-app dulu.

---

## 7. Timezone checklist

* [ ] `APP_TIMEZONE=Asia/Makassar`; invalid → fail fast.
* [ ] Container/server/Postgres `TZ=UTC`/`timezone=UTC`.
* [ ] Event `TIMESTAMPTZ`; business date `DATE`.
* [ ] Scheduler memakai `time.LoadLocation("Asia/Makassar")`, bukan `+8`.
* [ ] Job 07:00 WITA = 23:00 UTC hari sebelumnya.
* [ ] API mengembalikan ISO 8601 UTC/offset.
* [ ] Web memformat dengan `Intl.DateTimeFormat` + `timeZone`.
* [ ] Report harian memakai batas WITA.
* [ ] Tidak ada `time.FixedZone`, `+8`, atau penjumlahan jam manual.

Detail: `docs/Timezone.md`.

---

## 8. Review checklist sebelum merge

* [ ] Tidak ada request user yang memanggil Odoo langsung.
* [ ] Tidak ada WO dari sync kecuali `p2h_auto_wo=true`.
* [ ] Tidak ada PM auto-WO kecuali global + plan flag true.
* [ ] Tidak ada update `on_hand`/`reserved` di luar inventory transaction.
* [ ] Tidak ada posted transaction yang dihapus/diubah itemnya; status `posted → reversed` hanya saat reversal.
* [ ] Tidak ada negative stock.
* [ ] Tidak ada PR/PO/vendor/harga/Accurate.
* [ ] Tidak ada tabel duplikat (§2 poin 11).
* [ ] Tidak ada hardcoded timezone/offset.
* [ ] RBAC dicek di server, bukan hanya UI.
* [ ] Audit pelaku/waktu tercatat.
* [ ] Frontend: `npm run lint` dan `npm run build` lulus.
* [ ] Screenshot desktop + mobile, dark + light, dibandingkan dengan demo target; shell/tabel/form tidak meregresi M1/M2.
* [ ] Tidak ada mock/demo data yang di-commit; loading/empty/error state memakai data/API nyata.
* [ ] Tidak ada dependency UI baru tanpa approval; versi Next/React/Tailwind/shadcn sesuai §3b.
* [ ] Migrasi M0 tidak berubah; migration baru additive.
* [ ] Test/build/lint lulus; tidak ada secret di repo.

---

## 9. Keputusan terbuka — tanyakan, jangan menebak

| # | Pertanyaan | Default sementara jika tidak dijawab |
|---|---|---|
| Q-01 | Interval default per model/komponen? | Planner input dari manual/OEM; jangan hardcode |
| Q-02 | Mapping sparepart→komponen? | `components` master + pilihan manual saat usage |
| Q-03 | Default assignee/supervisor per plan? | Opsional; fallback planner |
| Q-04 | Siapa posting adjustment/opname? | Storekeeper counting opname; admin/planner posting adjustment/opname |
| Q-05 | Opening balance? | Stock opname awal; bukan edit saldo |
| Q-06 | Return tanpa issue asli? | Wajib referensi issue; exception admin beralasan |
| Q-07 | Format export? | CSV dulu |
| Q-08 | Penerima notifikasi per site? | Semua planner/admin |
| Q-09 | Auto-WO perlu approval supervisor? | Buat WO `open`; approval opsional |
| Q-10 | Email fase 1? | Tidak; in-app dulu |

Jika jawaban berbeda dari default, update PRD/decision log terlebih dahulu—jangan langsung mengode.

---

## 10. File map

| Path | Isi | Boleh diubah? |
|---|---|---|
| `prds/2026-11-18_work-order-maintenance-gsi.md` | PRD rev. 4 (baseline `v0.1.7` / M7) | Ya, dengan approval |
| `docs/Agent Brief.md` | Brief ini | Ya, dengan approval |
| `docs/Data Model.md` | ERD/DDL proposal | Ya, dengan approval |
| `docs/Domain Rules.md` | Status/business rules | Ya, dengan approval |
| `docs/Timezone.md` | Desain UTC/WITA | Ya, dengan approval |
| `migrations/` | Migrasi M0 | **Tidak** (M0 locked) |
| `docker-compose.yml`, `Dockerfile.*`, `.env.example` | Konfigurasi M0 | **Tidak** tanpa approval |
| `README.md` | Operasional M0 | **Tidak** tanpa approval |
| `apps/web/`, `services/api/` | Kode | Sesuai milestone aktif |

---

## 11. Protokol sesi

1. Baca PRD, Brief ini, Data Model, Domain Rules, dan Timezone.
2. Inspeksi repo dan migrasi yang ada; jangan mengasumsikan tabel sudah ada.
3. Konfirmasi milestone aktif dan scope-nya.
4. Buat rencana singkat; sebutkan file yang akan diubah dan migration baru.
5. Implementasikan **hanya** milestone aktif.
6. Jalankan test/build/lint dan verifikasi manual.
7. Laporkan: selesai, belum selesai, perintah verifikasi, risiko.
8. Jangan mulai milestone berikutnya tanpa instruksi.

---

## 12. Prompt template

### M1

```
Lanjut Milestone 1 (Foundation) sesuai PRD §12 dan Brief §5.
Scope: auth Odoo → JWT, RBAC + storekeeper, sync user/employee,
unit_models/engine_models/units extension/unit_meter_readings, dashboard shell.
Jangan membuat PM, inventory, material, atau report.
Jangan mengubah migrasi M0.
Verifikasi: login, role, sync idempoten, HM/KM reading, M0 regression.
```

### M2

```
Lanjut Milestone 2 (P2H) sesuai PRD §3.1/A4 dan Brief §5.
Scope: sync list/detail/findings, kandidat, notifikasi/reminder, approve/skip/bulk.
Syarat: kandidat unik per laporan; WO dari sync = 0 saat p2h_auto_wo=false;
sync 2x tidak duplikat; skip wajib alasan; approve ulang 409.
Jangan menyentuh PM/inventory/material.
```

### M3

```
Lanjut Milestone 3 (WO) sesuai PRD §7.3 dan Brief §5.
Scope: WO manual/lifecycle, assignee, supervisor, maintenance_type,
started_at, attachments, waiting_part, wo_updates.
Syarat: transisi valid; audit; done/cancel beralasan; tidak ada PM/inventory.
```

### M4

```
Lanjut Milestone 4 (PM core) sesuai PRD §3.2/§7.1–7.2 dan Brief §5.
Scope: pm_plans, checklist, schedules, due/upcoming/overdue,
generate WO, completion, history, compliance.
Syarat: plan per unit/model; interval HM/KM/calendar; first_reached default;
auto-WO hanya jika pm_auto_wo=true DAN auto_generate_wo=true; default 0;
PM done wajib reading yang diwajibkan plan (HM/KM/date); material plan/requirement baru di M5/M6.
```

### M5

```
Lanjut Milestone 5 (Sparepart & Inventory) sesuai PRD §3.3–3.4/§7.5–7.6.
Scope: sparepart master/numbers/alternatives/compatibility,
pm_plan_materials, warehouse/location,
receipt/issue/return/transfer/adjustment/opname,
ledger, balances, stock policy.
Syarat: ledger immutable; negative ditolak; tidak ada harga/cost/procurement;
tidak ada reservation/material readiness.
```

### M6

```
Lanjut Milestone 6 (Material) sesuai PRD §3.5/§7.4.
Scope: material requirement, readiness, reserve/release, issue/return,
waiting_part integration, shortage aging, PM material integration.
Syarat: planning/checking/reservation tidak mengubah on_hand;
issue mengurangi; return menambah; readiness benar; tidak ada PR/PO.
```

### M7

```
Lanjut Milestone 7 (History & Reporting) sesuai PRD §3.6–3.7/§9.
Scope: unit profile/history, component history/interval, downtime,
dashboard, report, CSV, performance/audit.
Syarat: semua history dari transaksi aktual; bucket WITA;
tidak ada static total; p95 target.
```

### Review

```
Review perubahan ini terhadap PRD dan Brief. Laporkan:
- requirement yang belum terpenuhi;
- field/tabel yang menyimpang dari Data Model;
- request user yang memanggil Odoo langsung (harus nol);
- WO dari sync tanpa p2h_auto_wo (harus nol);
- PM auto-WO tanpa double control (harus nol);
- update saldo di luar inventory transaction (harus nol);
- hardcoded timezone/+8 (harus nol);
- tabel duplikat/static history (harus nol).
Jangan ubah kode; cukup daftar temuan.
```

---

## 13. Yang harus dicek sendiri setelah agent selesai

* `select count(*) from p2h_reports` = 4.454; kandidat unik = 2.501 (587 pending, 1.914 archived); `work_orders` hasil sync = 0 sebelum approve.
* Approve 3 kandidat → 3 WO, masing-masing punya `candidate_id` + `wo_updates`.
* Skip tanpa alasan → 400; dengan alasan → `skipped`, WO tidak bertambah.
* `p2h_auto_wo=true` → kandidat baru `auto_created`; balik `false` → pending.
* PM default → tidak ada WO otomatis; aktifkan global + plan → WO `source='pm'`.
* Receipt +10, reserve 3 → on_hand 10, reserved 3, available 7; issue 3 → on_hand 7, reserved 0.
* Return 1 → on_hand 8; posted transaction tidak bisa dihapus.
* Job 07:00 WITA → log menunjukkan 23:00 UTC.
* Matikan Odoo → UI tetap jalan dengan data terakhir.

---

## 14. Hal yang paling sering salah

1. Membuat `waiting_material` padahal sudah ada `waiting_part`.
2. Mengurangi `on_hand` saat reservasi.
3. Membuat tabel history/usage duplikat.
4. Menghitung PM due dengan `+8` atau `Asia/Jakarta`.
5. Membuat WO otomatis dari P2H tanpa setting.
6. Menganggap `pm_auto_wo` cukup tanpa plan flag.
7. Menyimpan harga/cost di sparepart.
8. Mengedit `inventory_balances` langsung.
9. Mengubah migrasi M0.
10. Memakai total statis untuk unit history.
