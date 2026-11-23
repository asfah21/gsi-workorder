# M8a — Discovery & Decision Gate: Master Unit & Komponen

- **Tanggal:** 2026-11-23
- **Status:** M8a–M8d selesai; acceptance/import data pending
- **Baseline:** `v0.1.7` / M7
- **Scope:** M8 (Master Unit & Komponen) dan dependensi data untuk M9–M11

---

## 1. Tujuan

Mengunci kondisi awal aplikasi dan data existing sebelum mendesain M8, agar:

- tidak membuat modul yang sudah ada;
- tidak mengarang data komponen;
- tidak merusak kandidat P2H/Work Request existing;
- tidak membangun lifetime komponen di atas data HM yang belum valid.

---

## 2. Hasil audit data live

Audit dilakukan ke API `http://server:8080` pada 2026-11-23.

### 2.1 Unit

| Item | Nilai |
|---|---:|
| Total unit | 240 |
| Unit aktif | 240 |
| Unit dengan `serial_number` | 102 |
| Unit dengan `engine_serial_number` | 40 |
| Unit dengan `current_hm > 0` | 0 |
| Unit dengan `current_km > 0` | 0 |
| Unit breakdown | 0 |

Distribusi tipe unit:

| Tipe | Jumlah |
|---|---:|
| dump_truck | 140 |
| excavator | 41 |
| light_vehicle | 23 |
| articulated | 13 |
| bulldozer | 9 |
| tower_lamp | 7 |
| rock_breaker | 2 |
| compactor | 2 |
| genset | 1 |
| manhaul | 1 |
| motor_grader | 1 |

**Implikasi:** data identitas unit sebagian tersedia, tetapi HM/KM operasional belum terisi.

### 2.2 Komponen

| Item | Nilai |
|---|---:|
| Master `components` | 0 |
| History komponen | 0 |
| Component interval | 0 |

**Implikasi:** M8 tidak bisa mengandalkan data komponen existing. Master komponen dan instalasi awal harus diimpor atau diinput manual dari data PAM.

### 2.3 Work Request / P2H

| Item | Nilai |
|---|---:|
| P2H report | 4.512 |
| Kandidat/WR pending | 629 |
| Work Order | 2 |
| Work Order selesai | 1 |
| Work Order cancelled | 1 |

Contoh P2H memiliki `hm_awal` 225,6 dan 1.193,3, tetapi `units.current_hm` tetap 0 dan `unit_meter_readings` kosong.

**Implikasi:**

- Data WR existing wajib dipertahankan.
- Rename Work Request tidak boleh mengubah `candidate_id`/`wo_id`.
- Ada gap ingestion HM dari P2H ke `unit_meter_readings`/`units.current_hm`.

### 2.4 Maintenance Planning

| Item | Nilai |
|---|---:|
| PM plan | 0 |
| PM schedule aktif | 0 |
| PM compliance | 0 |

**Implikasi:** M11 akan mulai dari data plan yang kosong. Monitoring PM berbasis HM belum bisa berjalan sampai reading HM tersedia.

---

## 3. Gap utama

| Gap | Dampak | Milestone terkait |
|---|---|---|
| Tidak ada instalasi komponen per unit | Serial, tanggal/HM pemasangan, lifetime, transfer tidak bisa dilacak | M8b/M8d |
| Master komponen kosong | Tidak ada pilihan komponen untuk form/import | M8a/M8b |
| HM/KM unit 0 dan reading kosong | Lifetime komponen dan PM berbasis HM tidak valid | M8c/M11 |
| P2H HM tidak masuk ke reading unit | Data HM sudah ada tetapi tidak dipakai | M8c |
| `wo_candidates` hanya P2H | WR manual dan dispatch workflow belum bisa | M9b |
| `maintenance_type` mencampur kategori dan breakdown | Kategori WO baru dan durasi breakdown belum akurat | M10a |
| PM plan kosong | Target Hour/PCR/lifetime monitoring belum punya baseline | M11a |
| 250 unit belum diuji | Risiko performa worker/query | M11b |

---

## 4. Rekomendasi keputusan

| ID | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D-01 | Nama resmi PAM/PAMS | Pakai `PAM` di dokumen requirement; UI tetap `PAMS` sampai keputusan branding | Perlu konfirmasi |
| D-02 | PAM Dispatch | Tambah role `dispatch`; fallback ke `planner` bila user sync belum siap | Perlu konfirmasi |
| D-03 | Validasi WR vs konversi WO | Pisahkan `validated` dan `converted`; sediakan aksi gabungan | Perlu konfirmasi |
| D-04 | Pembuat WR manual | Operator/mechanic membuat; dispatch validasi | Perlu konfirmasi |
| D-05 | Definisi kategori WO | Definisikan Service/Scheduled/Unscheduled/Incident/UBOP bersama PAM | Perlu konfirmasi |
| D-06 | Breakdown timestamp | Simpan `breakdown_started_at`/`breakdown_ended_at`; backfill dari WO lama | Rekomendasi |
| D-07 | Backlog | Non-breakdown yang belum mulai/ditunda; bukan semua open | Perlu konfirmasi |
| D-08 | Identitas komponen | Serialized per unit, dengan riwayat instalasi | Dipakai untuk M8b (default) |
| D-09 | Lifetime pindah unit | Kumulatif per serial bila lifetime milik komponen | Dipakai untuk M8b (default); konfirmasi PAM pending |
| D-10 | Target Hour | Target HM komponen/plan; bukan target jam kerja user | Dipakai sebagai lifetime HM per instalasi; field Target Hour terpisah pending |
| D-11 | PCR vs Plant Component Request | PCR = planned replacement; Plant Component Request = request komponen | Perlu konfirmasi |
| D-12 | MR GSI | Deep link dulu; integrasi penuh setelah API/auth jelas | Rekomendasi |
| D-13 | Data awal komponen | Import Excel/manual dari PAM; jangan mengarang data | Sumber data belum diberikan; import pending |
| D-14 | Performa 250 unit | Benchmark `EvaluateDue`, index, batch, timeout worker | Rekomendasi |

---

## 5. Draft model M8b

> Draft ini menjadi dasar M8b. D-08, D-09, dan D-10 dipakai sebagai default implementasi lokal; konfirmasi PAM dan sumber data D-13 tetap pending.

### 5.1 `components` (extend)

Field tambahan yang diusulkan:

- `component_class` — `major` / `minor`
- `default_lifetime_hm`
- `default_lifetime_days`
- `default_lifetime_km`

Contoh major: Engine, Transmission, Final Drive, Injector, Alternator.

Contoh minor: Fuel Pump, Steering Pump, dan lainnya.

### 5.2 `unit_component_installations`

Field:

- `id`
- `unit_id`
- `component_id`
- `serial_number`
- `installed_at`
- `installed_hm`
- `installed_km`
- `removed_at`
- `removed_hm`
- `removed_km`
- `lifetime_hm`
- `lifetime_days`
- `lifetime_km`
- `status` — `active`, `removed`, `transferred`, `scrapped`
- `source_wo_id`
- `notes`
- `created_by`
- `created_at`
- `updated_at`

Constraint yang diusulkan:

- Satu unit hanya boleh punya satu instalasi aktif per `component_id`.
- Serial number unik untuk instalasi aktif.
- `removed_at >= installed_at`.
- `removed_hm >= installed_hm`.
- Lifetime boleh override default komponen.

### 5.3 View `v_unit_component_lifetime`

Output yang dibutuhkan:

- unit code
- component code/name/class
- serial number
- installed_at
- installed_hm
- current_hm
- used_hm = current_hm - installed_hm
- remaining_hm = lifetime_hm - used_hm
- used_days
- remaining_days
- status bucket: `safe`, `upcoming`, `due`, `overdue`

### 5.4 Transfer komponen

- Tutup instalasi lama: isi `removed_at`, `removed_hm`, `status='transferred'`.
- Buat instalasi baru di unit tujuan dengan serial yang sama.
- Jika lifetime kumulatif, sisa lifetime dihitung dari total pemakaian serial tersebut, bukan reset.

---

## 6. M8 breakdown yang diusulkan

| Milestone | Scope |
|---|---|
| **M8a** | Discovery, audit data, keputusan |
| **M8b** | Schema komponen + API |
| **M8c** | HM/KM reading ingestion + backfill |
| **M8d** | UI komponen, lifetime, transfer, alert |

M8c ditambahkan karena tanpa HM/KM reading, lifetime komponen dan PM HM tidak bisa valid.

---

## 7. Entry criteria M8b

M8b boleh dimulai jika:

- D-08, D-09, D-10, dan D-13 sudah disetujui.
- Sumber data master komponen sudah jelas.
- Aturan serial number dan transfer sudah disetujui.
- Rencana backfill HM/KM dari P2H sudah disetujui.
- Migration additive dan down migration sudah direview.

---

## 8. Open questions

1. Apakah komponen wajib serialized, atau cukup per tipe untuk komponen minor?
2. Apakah satu unit boleh punya lebih dari satu komponen aktif dengan tipe yang sama? (M8b default: satu aktif; multi-instance pending)
3. Apakah lifetime komponen dihitung kumulatif saat pindah unit? (M8b default: kumulatif per serial; konfirmasi PAM pending)
4. Apakah komponen yang sudah dilepas boleh dipasang kembali?
5. Apakah HM/KM awal komponen diambil dari reading unit saat instalasi? (M8b default: current unit, bisa override)
6. Apakah P2H `hm_awal` boleh menjadi sumber reading unit? (dipakai untuk M8c; konfirmasi PAM pending)
7. Siapa yang boleh mengubah lifetime default komponen?
8. Apakah Plant Component Request adalah bagian PM atau MR?
9. Apakah 250 unit adalah jumlah unit aktif atau seluruh unit termasuk nonaktif?
10. Apakah data komponen awal tersedia dalam Excel dengan serial dan HM?

---

## 9. Kesimpulan

M8a selesai pada level discovery. Data existing menunjukkan:

- aplikasi sudah punya 240 unit dan 629 WR pending;
- tetapi belum punya master komponen, instalasi komponen, reading HM/KM, atau PM plan.

Karena itu, M8 tidak boleh langsung membuat UI lifetime di atas data kosong. Urutan yang aman:

1. setujui keputusan D-08/D-09/D-10/D-13;
2. bangun schema komponen (M8b);
3. benahi ingestion/backfill HM/KM (M8c);
4. baru bangun UI lifetime dan alert (M8d).

---

## 10. Status implementasi M8b–M8d

M8b–M8d diimplementasikan pada 2026-11-23 dengan default berikut:

- komponen aktif diserialisasi per unit;
- satu unit hanya boleh punya satu instalasi aktif per tipe komponen;
- lifetime dihitung kumulatif per serial selama instalasi aktif;
- lifetime HM per instalasi dipakai sebagai target HM; belum ada field Target Hour terpisah;
- P2H `hm_awal`/`km_akhir` dipakai sebagai reading HM/KM; reading P2H yang lebih rendah dari reading valid sebelumnya ditandai anomali dan tidak menimpa current unit;
- master komponen dan instalasi awal belum diimpor karena sumber data PAM belum diberikan.

Yang sudah ada:

- migration additive `000012_m8_components` dan down migration;
- `components.component_class`, `default_lifetime_hm`, `default_lifetime_days`, `default_lifetime_km`;
- `unit_component_installations` untuk instalasi, pelepasan, replace, dan transfer;
- `v_unit_component_lifetime` untuk used/remaining HM, KM, dan hari;
- API master komponen, instalasi komponen, replace, transfer, dan lifetime;
- migration additive `000013_m8c_meter_anomaly`: `unit_meter_readings.is_anomaly` dan `anomaly_reason`;
- ingestion P2H idempoten lewat `source_type='p2h'` + `source_id=p2h_reports.id`, refresh `units.current_*`, notifikasi admin untuk anomali;
- command worker `/app/worker backfill-p2h-readings` untuk backfill DB-only;
- UI tab **Komponen** di profil unit: lifetime, badge status, alert due/overdue, install/replace/transfer via centered sheet;
- UI tab **Komponen** di Master Unit: list/create/edit master komponen dan lifetime default;
- unit test service/HTTP dan integration test PostgreSQL (skip jika `TEST_DATABASE_URL` tidak di-set).

Migration, integration test, dan frontend sudah divalidasi: seluruh migration up berhasil, view lifetime menghasilkan used/remaining yang benar, partial unique index menolak instalasi aktif ganda, ingestion P2H idempoten, anomali tidak menimpa current, notifikasi admin dibuat, down migration berhasil lalu up kembali, serta `npm run lint`, `npm run check:typography`, dan `npm run build` lulus.

Yang belum:

- import master komponen dan instalasi awal dari PAM;
- konfirmasi ambang lifetime dan keputusan komponen multi-instance;
- pemetaan shift P2H ke timestamp intra-hari (saat ini semua reading memakai awal tanggal laporan WITA);
- visual/UAT server untuk alur komponen.
