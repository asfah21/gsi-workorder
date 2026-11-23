# M8a — Discovery & Decision Gate: Master Unit & Komponen

- **Tanggal:** 2026-11-23
- **Status:** Selesai untuk discovery; menunggu keputusan sebelum coding
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
| D-08 | Identitas komponen | Serialized per unit, dengan riwayat instalasi | Rekomendasi |
| D-09 | Lifetime pindah unit | Kumulatif per serial bila lifetime milik komponen | Perlu konfirmasi |
| D-10 | Target Hour | Target HM komponen/plan; bukan target jam kerja user | Perlu konfirmasi |
| D-11 | PCR vs Plant Component Request | PCR = planned replacement; Plant Component Request = request komponen | Perlu konfirmasi |
| D-12 | MR GSI | Deep link dulu; integrasi penuh setelah API/auth jelas | Rekomendasi |
| D-13 | Data awal komponen | Import Excel/manual dari PAM; jangan mengarang data | Rekomendasi |
| D-14 | Performa 250 unit | Benchmark `EvaluateDue`, index, batch, timeout worker | Rekomendasi |

---

## 5. Draft model M8b

> Ini draft desain, bukan migration. Jangan implementasi sebelum D-08, D-09, D-10, dan D-13 disetujui.

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
2. Apakah satu unit boleh punya lebih dari satu komponen aktif dengan tipe yang sama?
3. Apakah lifetime komponen dihitung kumulatif saat pindah unit?
4. Apakah komponen yang sudah dilepas boleh dipasang kembali?
5. Apakah HM/KM awal komponen diambil dari reading unit saat instalasi?
6. Apakah P2H `hm_awal` boleh menjadi sumber reading unit?
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
