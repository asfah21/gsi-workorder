# Domain Rules — GSI Work Order & Maintenance

**Status:** Proposal untuk M1+ (kontrak bisnis, bukan kode)
**Sumber requirement:** `prds/2026-11-18_work-order-maintenance-gsi.md`
**Model data:** `docs/Data Model.md`
**Timezone:** `docs/Timezone.md`

> Dokumen ini menjawab pertanyaan "boleh/tidak" dan "bagaimana dihitung". Jika implementasi berbeda, implementasi yang salah—kecuali PRD diubah dengan approval.

---

## 1. Konvensi

### 1.1 Aktor

| Aktor | Arti |
|---|---|
| `admin` | Akses penuh, setting, sync, master, posting adjustment/opname |
| `planner` | PM, WO, material planning, verifikasi compatibility |
| `mechanic` | Eksekusi WO miliknya, checklist, HM/KM, request material |
| `storekeeper` | Receipt, issue, return, transfer, reservasi, counting opname |
| `viewer` | Read-only |
| `system` | Worker/job; transisi tanpa user tercatat `user_id=NULL` |

### 1.2 Waktu

* Semua timestamp disimpan UTC (`TIMESTAMPTZ`).
* Tanggal bisnis (`due_date`, `effective_from`, `report_date`) adalah `DATE` kalender WITA.
* "Hari ini" untuk aturan bisnis = tanggal WITA yang dihitung eksplisit, mis. `(now() AT TIME ZONE 'Asia/Makassar')::date`; bukan `current_date` UTC.
* Scheduler memakai `Asia/Makassar`; contoh 07:00 WITA = 23:00 UTC hari sebelumnya.

### 1.3 Audit & append-only

* `wo_updates` append-only; setiap transisi WO wajib menulis baris.
* `material_requirement_updates` append-only; setiap transisi status material wajib menulis baris.
* `inventory_transactions` posted tidak boleh dihapus/diubah item/qty-nya; satu-satunya update status adalah `posted → reversed` saat reversal.
* `app_settings_history` mencatat setting lama → baru.
* `wo_candidates` mencatat `decided_by/at`, `skip_reason`.
* `pm_schedules` mencatat override (`override_reason`, `overridden_by`, `overridden_at`).
* `sparepart_compatibility` mencatat `verified_by/at`; perubahan status tanpa history table adalah keterbatasan v1.

### 1.4 Idempotency

* Sync P2H idempoten per `odoo_report_id`; kandidat unik per `p2h_report_id`.
* PM auto-WO idempoten per schedule aktif (partial unique index).
* Inventory posting menerima `idempotency_key` unik; retry dengan key sama mengembalikan transaksi yang sama, bukan duplikat.
* Nomor dokumen (`WO-`, `TRX-`, `MR-`) dibuat dengan retry on unique conflict.

---

## 2. Preventive Maintenance

### 2.1 Status plan

| Dari | Ke | Aktor | Guard | Efek |
|---|---|---|---|---|
| `draft` | `active` | planner/admin | Target tepat satu; minimal satu interval; `effective_from`; minimal satu checklist aktif | Buat `pm_schedules` untuk unit target |
| `active` | `inactive` | planner/admin | — | Tidak membuat schedule baru; schedule aktif tetap ada; auto-WO berhenti |
| `inactive` | `active` | planner/admin | Validasi sama | Buat schedule yang belum ada; jangan duplikat |
| `active` | `draft` | planner/admin | Tidak ada schedule/WO | Hanya untuk plan yang belum pernah jalan |
| `draft`/`inactive` | — | — | — | Tidak ada delete jika sudah punya schedule/WO; gunakan `inactive` |

Aturan:

* Plan per unit (`unit_id`) membuat satu schedule.
* Plan per model (`unit_model_id`) membuat schedule untuk semua unit `is_active=true` pada model tersebut.
* Unit baru aktif pada model setelah plan aktif: worker harian membuat schedule bila belum ada active schedule; due awal dihitung dari reading/effective date saat pembuatan.
* Plan `active` tanpa unit aktif valid tetapi menghasilkan nol schedule; tampilkan warning.
* `auto_generate_wo` hanya boleh `true` jika global `pm_auto_wo=true`; validasi saat simpan dan saat eksekusi.
* Perubahan interval/ambang/checklist setelah plan aktif hanya berlaku untuk WO/siklus baru; schedule aktif tidak dihitung ulang otomatis. Jika due harus berubah, gunakan override due beralasan.

### 2.2 Status schedule

| Dari | Ke | Aktor | Guard | Efek |
|---|---|---|---|---|
| `scheduled` | `due` | system | Threshold tercapai (lihat §2.3) | Notifikasi due |
| `scheduled`/`due` | `overdue` | system | Threshold terlewati | Notifikasi overdue |
| `due` | `overdue` | system | Hari/reading bergerak melewati due | Notifikasi overdue |
| `scheduled`/`due`/`overdue` | `wo_created` | planner/admin/system | Tidak ada WO aktif; plan aktif; unit aktif | Buat WO + snapshot checklist; material plan/requirement mengikuti milestone (M5/M6) |
| `wo_created` | `completed` | system | WO `done` + guard completion §2.6 | Update last/next; buat siklus berikutnya |
| `wo_created` | `due`/`overdue` | system | WO `cancelled` | Kembalikan status sesuai threshold; WO pengganti boleh dibuat |
| `scheduled`/`due`/`overdue` | `skipped` | planner/admin | Alasan ≥5 karakter; tidak ada WO aktif | Siklus ditutup; compliance dihitung missed |
| `scheduled`/`due`/`overdue`/`wo_created` | `cancelled` | planner/admin | Alasan; tidak ada WO aktif (atau WO sudah cancelled) | Siklus ditutup; tidak ada next cycle |
| `completed`/`skipped`/`cancelled` | — | — | Terminal | — |

Aturan:

* `due_soon` dan `upcoming` adalah **bucket tampilan**, bukan status DB. Status DB hanya `scheduled/due/overdue/wo_created/completed/skipped/cancelled`.
* Status dapat kembali dari `overdue` ke `due` hanya bila koreksi reading HM/KM disetujui admin dan tercatat; selain itu tidak.
* Satu schedule hanya boleh punya satu WO non-`cancelled` (partial unique index). WO `cancelled` boleh diganti.
* Jika plan `inactive`, schedule aktif tidak auto-WO; planner dapat menyelesaikan/membatalkan.

### 2.3 Perhitungan due

Untuk setiap dimensi yang dikonfigurasi:

| Dimensi | Remaining | Due | Overdue |
|---|---|---|---|
| Calendar | `due_date - tanggal WITA` | `remaining = 0` | `remaining < 0` |
| HM | `due_hm - units.current_hm` | `remaining = 0` | `remaining < 0` |
| KM | `due_km - units.current_km` | `remaining = 0` | `remaining < 0` |

Bucket per dimensi:

* `overdue` bila remaining < 0.
* `due` bila remaining = 0.
* `due_soon` bila `0 < remaining <= due_soon` (hari/HM/KM).
* `upcoming` bila `due_soon < remaining <= upcoming`.
* `scheduled` bila remaining > upcoming.

Aturan combined:

* `first_reached`: ambil bucket **paling parah** di antara dimensi yang dikonfigurasi (overdue > due > due_soon > upcoming > scheduled).
* `all_required`: due hanya bila **semua** dimensi minimal `due`; overdue bila **semua** overdue; due_soon bila semua minimal due_soon; upcoming bila semua minimal upcoming; selain itu scheduled.
* Jika ambang HM/KM `upcoming`/`due_soon` null, dimensi tersebut hanya menghasilkan `due`/`overdue`.
* Ambang harus lebih kecil dari interval: `upcoming < interval` dan `due_soon < upcoming`; jika tidak, plan tidak dapat diaktifkan (menghindari PM selalu "upcoming").
* Jika plan hanya calendar, dimensi HM/KM diabaikan.
* `due_date`/`due_hm`/`due_km` yang null tidak dievaluasi.

Due awal saat aktivasi:

* Calendar: `effective_from + interval_days`.
* HM: `units.current_hm + interval_hm` (jika reading 0/null, gunakan 0 + interval).
* KM: `units.current_km + interval_km`.
* Override awal dicatat dengan alasan minimal 5 karakter.

Next due setelah completion:

* Calendar: `completion_date + interval_days`.
* HM: `completed_hm + interval_hm`.
* KM: `completed_km + interval_km`.
* Completion date = tanggal WO `done` dalam WITA.

### 2.4 Auto-WO

Guard lengkap (semua harus true):

1. Global `app_settings.pm_auto_wo = true`.
2. Plan `auto_generate_wo = true`.
3. Plan `active` dan tanggal WITA berada dalam `effective_from`/`effective_to` (jika diisi).
4. Unit `is_active = true`.
5. Schedule `due`/`overdue`.
6. Tidak ada WO non-`cancelled` untuk schedule tersebut.
7. Schedule tidak sedang di-lock oleh proses lain.

Prosedur:

1. `SELECT ... FOR UPDATE` pada schedule.
2. Validasi guard.
3. Generate kode WO.
4. Insert WO: `source='pm'`, `maintenance_type='preventive'`, `status='open'`, `pm_schedule_id`, priority/assignee dari plan, `due_date = GREATEST(schedule.due_date, tanggal WITA)` bila ada. Jika default assignee null/nonaktif, biarkan unassigned dan notifikasi planner.
5. Snapshot checklist dari plan.
6. Jika modul material (M6) tersedia: buat `material_requirements` status `draft` + items dari `pm_plan_materials` (termasuk `is_optional`). Sebelum M6, lewati langkah ini.
7. Set schedule `wo_created`.
8. Insert `wo_updates` note "Dibuat otomatis dari PM".
9. Notifikasi planner/admin + assignee.
10. Commit.

Jika guard gagal, tidak ada WO; cukup notifikasi. Jika global false atau plan flag false, **tidak ada** auto-WO—hanya notifikasi.

### 2.5 Generate manual

Planner/admin dapat memanggil generate untuk schedule `due`/`overdue`. Prosedur sama, aktor tercatat, tanpa guard global/plan flag. Tetap tidak boleh ada WO aktif kedua.

### 2.6 Completion PM

WO PM boleh `done` hanya jika:

* Semua checklist item pada WO memiliki `result != 'pending'` (`ok/not_ok/na`).
* Reading wajib tersedia:
  * Plan punya interval HM → completed HM wajib.
  * Plan punya interval KM → completed KM wajib.
  * Plan calendar-only → tanggal completion wajib; HM/KM opsional.
  * Combined → semua meter yang dipakai plan wajib.
* Tidak ada required material item berstatus shortage. Item `is_optional` diabaikan.
* Jika shortage, admin dapat override dengan alasan tercatat di `wo_updates`; status tetap `done` tetapi ditandai.

Efek completion:

1. Insert `unit_meter_readings` untuk HM/KM (source `wo`, source_id WO).
2. Update `units.current_hm/current_km` hanya jika reading >= current; jika lebih rendah, tolak atau jalankan koreksi admin.
3. Update schedule: `last_hm`, `last_km`, `last_completed_at`, status `completed`.
4. Buat schedule baru `cycle_no + 1`, status `scheduled`, due dari §2.3.
5. Jika plan `inactive` atau tanggal WITA di luar `effective_from`/`effective_to`, jangan buat siklus baru.
6. WO `closed_at = now()`.

### 2.7 Skip & cancel

* Skip hanya untuk schedule tanpa WO aktif; alasan ≥5 karakter; status `skipped`; catat `skip_reason`, `skipped_by`, `skipped_at`.
* Next cycle setelah skip: `due = previous due + interval` (cadence dipertahankan). Jika due sudah jauh lewat, tetap satu interval per siklus; jangan loncat beberapa siklus sekaligus.
* Cancel schedule hanya jika tidak ada WO aktif; alasan ≥5 karakter; catat `cancel_reason`, `cancelled_by`, `cancelled_at`; tidak ada next cycle.
* WO PM yang cancelled: schedule kembali `due`/`overdue`; WO pengganti boleh dibuat.

### 2.8 Compliance

* `due_count` = schedule dengan due date/reading pada periode.
* `completed_count` = schedule due dalam periode yang completed, on time atau late.
* `completed_on_time` = completion tidak melewati due; untuk combined mengikuti `interval_rule`:
  * `first_reached`: on-time jika completion tidak melewati threshold pertama yang memicu due.
  * `all_required`: on-time hanya jika completion tidak melewati seluruh threshold.
* `skipped` = dihitung missed, bukan completed.
* `overdue` = belum completed dan melewati due.
* `compliance_pct = completed_on_time / due_count` (0 jika due_count=0).
* `completion_rate = completed_count / due_count` (0 jika due_count=0).
* PM compliance tidak memakai material readiness sebagai faktor; itu metrik terpisah.

---

## 3. Work Order

### 3.1 Status transition

| Dari | Ke | Aktor | Guard | Efek |
|---|---|---|---|---|
| `open` | `in_progress` | mechanic/planner/admin | Assignee ada; unit aktif | `started_at=now()`; `wo_updates` |
| `in_progress` | `waiting_part` | mechanic/planner/admin | Material requirement ada dan status `shortage`/`partially_ready` | `wo_updates`; durasi mulai |
| `waiting_part` | `in_progress` | mechanic/planner/admin | Material status `ready`/`reserved`/`issued`/`completed`/`cancelled` | `wo_updates`; durasi selesai |
| `in_progress` | `done` | mechanic/planner/admin | PM: guard §2.6; non-PM: alasan penutupan wajib | `closed_at`; material completion; usage tetap |
| `open`/`in_progress`/`waiting_part` | `cancelled` | planner/admin | Alasan ≥5 karakter | `closed_at`; release reservasi; PM schedule revert |
| `done` | — | — | Terminal | — |
| `cancelled` | — | — | Terminal | — |

Aturan:

* `waiting_part` tidak boleh langsung `done`; harus kembali `in_progress`.
* Tidak ada reopen; buat WO baru.
* `source='pm'` wajib `pm_schedule_id`; `source='p2h'` wajib `p2h_report_id`; `source='manual'` tidak wajib.
* `maintenance_type` default `corrective`; PM `preventive`; P2H default `corrective`, dapat diubah ke `breakdown` bila unit breakdown.
* Setiap transisi menulis `wo_updates` (`status_from`, `status_to`, `user_id`, `note`).
* `started_at` diisi saat pertama kali `in_progress`; tidak di-reset bila kembali dari `waiting_part`.
* `closed_at` diisi saat `done`/`cancelled`.

### 3.2 Waiting part

* Hanya jika ada material requirement; WO tanpa material tidak boleh `waiting_part`.
* Material requirement status yang memicu: `shortage` atau `partially_ready`.
* Material readiness **tidak** otomatis memindahkan WO; sistem mengirim notifikasi, planner/mechanic memindahkan ke `in_progress`.
* Durasi waiting = jumlah interval `waiting_part` → status berikutnya dari `wo_updates`; beberapa episode dijumlahkan.
* Jika WO `cancelled` saat waiting, reservasi dilepas.

### 3.3 Completion non-PM

* `done` boleh tanpa checklist; alasan penutupan wajib (note di `wo_updates`).
* Jika ada material requirement, tidak boleh `shortage` (kecuali admin override beralasan); saat `done`, release sisa reservasi lalu completed/cancelled sesuai §4.8.
* Jika ada issue yang belum di-return, tetap boleh done; usage tercatat.

### 3.4 Cancellation

* WO `cancelled` wajib alasan.
* Reservasi aktif dilepas.
* Material requirement di-cancel; issue yang sudah posted tidak dihapus.
* Jika ada issue > return, admin harus menyetujui cancellation atau meminta return/reversal terlebih dahulu.
* WO PM: schedule kembali `due`/`overdue`; WO pengganti boleh dibuat.
* WO P2H: kandidat tetap `approved`; tidak kembali ke pending.

### 3.5 Downtime

* Unplanned: `maintenance_type IN ('breakdown','corrective')`, status `done`.
* Planned: `maintenance_type='preventive'`, status `done`.
* Start = `started_at`; fallback `created_at` untuk breakdown/corrective.
* End = `closed_at`.
* Cancelled tidak dihitung downtime.
* Waiting part dihitung terpisah dari `wo_updates`.
* Untuk unit yang punya beberapa WO overlap, downtime dihitung dari **union interval**, bukan penjumlahan (menghindari double count).
* `units.is_breakdown` adalah flag operasional manual; tidak mengubah history.

---

## 4. Material Requirement

### 4.1 Definisi status

| Status | Arti | Stok |
|---|---|---|
| `draft` | Header dibuat, item diedit | Tidak berubah |
| `planned` | Item final, siap dicek | Tidak berubah |
| `checking_stock` | Pengecekan berjalan | Tidak berubah |
| `ready` | Semua required item tersedia | Tidak berubah |
| `partially_ready` | Semua required item punya sebagian coverage, belum penuh | Tidak berubah |
| `shortage` | Ada required item dengan coverage nol | Tidak berubah |
| `reserved` | Semua required item tercakup (available + reservasi + issue); minimal satu reservasi aktif | `reserved` naik; `on_hand` tetap |
| `issued` | Issue sudah dimulai dan tidak ada required item coverage nol; boleh belum penuh | `on_hand` turun |
| `completed` | WO selesai, reservasi aktif nol | Tidak berubah |
| `cancelled` | Dibatalkan | Reservasi dilepas |

### 4.2 Transisi manual

| Dari | Ke | Aktor | Guard |
|---|---|---|---|
| `draft` | `planned` | planner/admin | Minimal satu item; qty > 0 |
| `planned` | `checking_stock` | planner/storekeeper | — |
| `checking_stock` | `checking_stock` | planner/storekeeper | Recheck |
| `ready`/`partially_ready`/`shortage` | `checking_stock` | planner/storekeeper | Recheck setelah stock berubah |
| `ready`/`partially_ready` | `reserved` | planner/storekeeper | Semua required item `coverage >= qty_required`; minimal satu reservasi aktif; jika belum, tetap status semula |
| `ready`/`partially_ready`/`reserved` | `issued` | storekeeper/admin | Issue pertama dan tidak ada required item coverage nol |
| semua non-terminal kecuali `shortage` | `completed` | system | WO done; release sisa reservasi; tidak shortage |
| `shortage` | `completed` | admin | WO done; override alasan |
| semua non-terminal | `cancelled` | planner/admin | Alasan; release reservasi; issue > return butuh approval |

### 4.3 Recompute sistem

Sistem menghitung ulang status pada event:

* Stock receipt/issue/return/transfer/adjustment/opname.
* Reservation/release.
* Issue/return material.
* WO done/cancelled.
* Recheck manual.

Status yang boleh berubah otomatis: `ready ↔ partially_ready ↔ shortage`, `ready/partially_ready/reserved → issued`, `issued → partially_ready/shortage` (bila return/koreksi), dan penutupan `completed` saat WO done dari status non-terminal (kecuali `shortage` yang butuh override admin). Pengecualian penutupan WO ini bukan skip manual; `draft/planned/checking_stock` tidak boleh dilompati oleh aksi manual biasa.

### 4.4 Formula readiness

Per item:

```text
net_issued        = qty_issued - qty_returned
available         = on_hand - reserved        (scope lokasi/warehouse)
coverage          = available + qty_reserved + net_issued
shortage          = GREATEST(qty_required - coverage, 0)
```

* `available` dihitung dari `v_stock_balance`; jika `preferred_warehouse_id` diisi, hanya lokasi di warehouse tersebut.
* `qty_reserved` adalah cache reservasi aktif untuk item tersebut.
* `coverage` menjumlahkan stok yang masih tersedia, yang sudah direservasi, dan yang sudah di-issue (net).
* Item `is_optional=true` **tidak** memblokir status; shortage optional tidak membuat header `shortage`.

Status header readiness (prioritas; status workflow `draft/planned/checking_stock` adalah fallback):

1. `cancelled` — status cancelled.
2. `completed` — status completed.
3. `issued` — `net_issued > 0` dan tidak ada required item `coverage <= 0` (boleh belum penuh).
4. `reserved` — `qty_reserved > 0` dan semua required item `coverage >= qty_required`.
5. `shortage` — ada required item `coverage <= 0`.
6. `partially_ready` — semua required item `coverage > 0`, tetapi ada yang `coverage < qty_required`.
7. `ready` — semua required item `coverage >= qty_required`, `qty_reserved = 0`, `net_issued = 0`.
8. `checking_stock`.
9. `planned`.
10. `draft`.

Jika tidak ada required item (semua optional), status mengikuti `ready` setelah check. Perubahan status lifecycle manual tetap mengikuti §4.2; view readiness dapat menampilkan status derivatif yang lebih tinggi.

### 4.5 Reservasi

* Qty reservasi <= `qty_required - qty_reserved - net_issued` (remaining).
* Lokasi harus punya `available` cukup; jika `preferred_warehouse_id` diisi, hanya lokasi di warehouse tersebut.
* Reservasi membuat/menambah `inventory_reservations` status `active` dan transaksi `reserve`; `on_hand` tidak berubah, `reserved` bertambah.
* Partial reservation diperbolehkan; status tetap `partially_ready`/`shortage` sampai coverage penuh.
* Release mengurangi `reserved`, menandai reservation `released`, dan transaksi `release`.
* Reservasi tidak boleh melebihi available; jika stok turun karena issue lain, reservasi yang ada tetap, tetapi reservasi baru ditolak.

### 4.6 Issue

* Qty issue <= `qty_required - net_issued`.
* Konsumsi reservasi aktif terlebih dahulu (FIFO `reserved_at`), lalu available.
* Issue mengurangi `on_hand`; jika dari reservasi, `reserved` dan `reservation.qty` berkurang; reservation menjadi `consumed` saat `qty = 0`.
* Setiap issue membuat `work_order_sparepart_usage` (`movement_type='issue'`) yang menautkan `inventory_transaction_item_id` dan mencatat snapshot `hm/km` saat issue bila tersedia.
* Issue parsial diperbolehkan; status menjadi `issued` bila tidak ada shortage.
* Issue tanpa reservasi diperbolehkan bila available cukup.
* Issue melebihi required ditolak (kecuali fitur override baru dengan approval).

### 4.7 Return

* Qty return <= `net_issued`.
* Return menambah `on_hand`; membuat usage `movement_type='return'`; tidak otomatis reservasi ulang.
* Jika requirement belum completed, status dihitung ulang; jika menjadi shortage, `shortage_since` di-set.
* Return setelah completed diperbolehkan; status tetap completed, reporting memakai net usage.
* Return harus untuk WO dan sparepart yang sama dengan issue asal; qty <= sisa issue yang belum diretur.
* Return tanpa referensi issue asli ditolak; `return_of_usage_id` wajib diisi dan harus menunjuk usage `issue`; return of return ditolak. Exception admin harus beralasan dan tercatat.

### 4.8 Complete & cancel

Complete:

* WO `done`.
* Saat WO done, sistem release sisa reservasi aktif (transaksi `release`) sebelum menutup material requirement.
* Reservasi aktif = 0.
* `net_issued <= qty_required` (issue guard).
* Item optional boleh belum di-issue; required yang tidak terpakai dicatat sebagai variance di reporting, bukan blocker.
* Status `completed`.

Cancel:

* Release semua reservasi aktif; reservation `qty = 0` dan ditandai `cancelled` (bukan `released`, karena dibatalkan).
* Status `cancelled`.
* Jika `net_issued > 0`, block kecuali admin override; return/reversal tetap tercatat.
* WO cancel memicu cancel material requirement.

### 4.9 Shortage aging

* `shortage_since` di-set saat status berubah menjadi `shortage` dari status non-shortage.
* `shortage_since` dipertahankan selama masih shortage; di-clear saat keluar shortage.
* Jika masuk shortage lagi setelah resolved, timestamp baru (episode baru).
* Aging = `(now() AT TIME ZONE 'Asia/Makassar')::date - (shortage_since AT TIME ZONE 'Asia/Makassar')::date`; bucket 0–3/4–7/8–14/>14 hari.
* Laporan aging memakai episode saat ini, bukan akumulasi historis.

### 4.10 PM material snapshot

* `pm_plan_materials` tersedia mulai M5 (setelah master sparepart).
* Saat WO PM dibuat setelah M6, `pm_plan_materials` disalin ke `material_requirement_items`; `is_optional` ikut disalin.
* WO PM yang dibuat sebelum M6 tidak memiliki material requirement; material plan tetap template.
* Perubahan plan setelah WO dibuat tidak mengubah snapshot.
* Material requirement PM mulai `draft`; stock check mengikuti lifecycle M6.
* Jika plan tidak punya material, tidak ada material requirement.

### 4.11 Concurrency

* Lock `material_requirements` + item rows saat reserve/issue/return.
* Lock `inventory_balances` urut `(sparepart_id, warehouse_location_id)` untuk menghindari deadlock.
* Gunakan `idempotency_key` untuk posting inventory.
* Retry pada unique conflict nomor dokumen.

---

## 5. Inventory

### 5.1 Matriks transaksi

| Type | From | To | `on_hand` | `reserved` | Usage |
|---|---|---|---|---|---|
| `receipt` | — | ✅ | + | — | — |
| `issue` | ✅ | — | − | − jika reservasi | issue |
| `return` | — | ✅ | + | — | return |
| `reserve` | ✅ | — | — | + | — |
| `release` | ✅ | — | — | − | — |
| `transfer` | ✅ | ✅ | − asal, + tujuan | — | — |
| `adjustment` increase | — | ✅ | + | — | — |
| `adjustment` decrease | ✅ | — | − | — | — |
| `opname` increase | — | ✅ | + | — | — |
| `opname` decrease | ✅ | — | − | — | — |

### 5.2 Posting algorithm

Dalam satu DB transaction:

1. Validasi input: qty > 0; sparepart/location aktif; tipe transaksi valid; issue manual tanpa WO/material requirement wajib alasan.
2. Lock balance rows terkait (`SELECT ... FOR UPDATE`), urutkan key.
3. Validasi:
   * Issue: `available + reserved_for_this_requirement >= qty`; konsumsi reservasi lebih dulu.
   * Transfer/adjustment decrease/reserve: `available >= qty`.
   * Adjustment decrease: `on_hand - qty >= reserved`.
   * Transfer: from ≠ to; available cukup.
   * Reservation: reservation aktif/valid.
4. Insert `inventory_transactions` (status `posted`, `posted_by`, `posted_at`, `idempotency_key`).
5. Insert items.
6. Update `inventory_balances`.
7. Update reservation (jika issue/release).
8. Insert usage (jika issue/return WO).
9. Commit.

Jika gagal di langkah mana pun, rollback semua. Tidak ada posting parsial.

### 5.3 Saldo

* `on_hand >= 0`.
* `reserved >= 0`.
* `reserved <= on_hand`.
* `available = on_hand - reserved`.
* `inventory_balances` adalah cache; sumber kebenaran ledger.
* Tidak ada endpoint yang meng-update `on_hand`/`reserved` langsung.

### 5.4 Reservasi & release

* Reserve hanya untuk `material_requirement_item`.
* Qty <= available di lokasi.
* `reserved += qty`; reservation `active` dengan `qty` = sisa yang direservasi; transaksi `reserve`.
* Release: `reserved -= reservation.qty`; reservation `qty = 0`, status `released`; transaksi `release`.
* Satu item/lokasi punya satu active reservation (partial unique); qty diagregasi.
* Reservasi tidak mengubah `on_hand`.

### 5.5 Transfer

* From dan to harus berbeda; sparepart sama; qty > 0.
* Hanya boleh memindahkan **available** (`on_hand - reserved`), bukan stok yang sudah direservasi.
* `on_hand` from −, to +; `reserved` tidak berubah.
* Satu transaksi, dua item rows (from/to).

### 5.6 Adjustment

* Alasan wajib.
* Increase: `to_location`, `on_hand += qty`.
* Decrease: `from_location`, `on_hand -= qty`; tidak boleh di bawah `reserved`.
* Approver admin/planner; `posted_by` mencatat pelaku.
* Tidak ada edit saldo langsung.

### 5.7 Stock opname

Status session: `draft → counting → review → posted`, plus `cancelled`.

1. Draft: pilih warehouse; belum snapshot.
2. Counting: snapshot `system_qty` dari balance; isi `counted_qty`.
3. Review: lihat difference; tidak ada perubahan stok.
4. Posted: buat adjustment/opname transaction per selisih; update balance; session posted.
5. Cancelled: tidak ada perubahan.

Aturan:

* `counted_qty >= 0`.
* Item opname hanya untuk lokasi dalam warehouse session; lokasi lain ditolak.
* Jika `counted_qty < reserved`, posting ditolak sampai reservasi dilepas/disesuaikan; stok fisik tidak boleh lebih kecil dari yang sudah dijanjikan.
* Difference = counted - system; positif increase, negatif decrease.
* Tidak ada edit `inventory_balances` langsung.
* Opname ulang setelah posted membuat session baru.

### 5.8 Reversal

* Posted transaction tidak dihapus dan item/qty-nya tidak diubah. Satu-satunya update yang diizinkan adalah status `posted → reversed` saat reversal; item tidak berubah.
* Reversal membuat transaction baru dengan item mirror (from/to ditukar), `reversal_of_id` diisi, status `posted`.
* Original status menjadi `reversed`.
* Issue reversal menghasilkan return usage; original usage tetap ada (net consumption = issue - return).
* Receipt reversal = decrease; transfer reversal = transfer balik; reserve reversal = release; release reversal = reserve (jika available); adjustment reversal = opposite.
* Opname reversal: reverse adjustment yang dihasilkan, atau buat opname baru; jangan edit session posted.
* Reversal tidak boleh membuat negative/reserved violation.

### 5.9 Minimum stock

* Policy per sparepart per lokasi: `min_qty`, `critical_qty`, `reorder_qty`.
* `low` bila `available < min_qty`.
* `critical` bila `available <= critical_qty`.
* Tidak ada policy → tidak ada alert.
* Lokasi dengan policy tetapi belum punya baris `inventory_balances` dianggap `on_hand=0`, `reserved=0`; alert low/critical tetap dihitung.
* Alert harian 07:30 WITA; dedupe per sparepart/lokasi/status/hari.
* Alert bukan procurement; tidak membuat PR/PO.

### 5.10 Reconciliation

* `on_hand` harus sama dengan hasil ledger transaksi asli + mirror (receipt/return/transfer in/adjustment increase/opname increase − issue/transfer out/adjustment decrease/opname decrease). Status `reversed` adalah penanda audit, bukan filter ledger; menghitung mirror saja akan menghasilkan selisih sebesar transaksi asli.
* `reserved` harus sama dengan sum active reservations.
* Reconcile query harus nol selisih; jika tidak, perbaiki bug dan buat adjustment beralasan—jangan edit manual.
* `inventory_balances` dapat di-rebuild dari ledger.

### 5.11 Concurrency & idempotency

* Lock balance rows dengan urutan key konsisten.
* Unique `idempotency_key` mencegah double posting.
* Retry nomor dokumen pada unique conflict.
* Jangan menahan lock saat memanggil service eksternal (tidak ada service eksternal di path inventory).

---

## 6. Sparepart & Compatibility

### 6.1 Status compatibility

| Dari | Ke | Aktor | Guard |
|---|---|---|---|
| `pending_verification` | `approved_compatible` | admin/planner | Target valid |
| `pending_verification` | `conditional_compatible` | admin/planner | `condition_note` wajib |
| `pending_verification` | `not_compatible` | admin/planner | Alasan disarankan |
| `approved_compatible` | `conditional_compatible`/`not_compatible` | admin/planner | Re-verifikasi |
| `conditional_compatible` | `approved_compatible`/`not_compatible` | admin/planner | Re-verifikasi |
| `not_compatible` | `pending_verification` | admin/planner | Data berubah |
| Semua | `pending_verification` | system | Target/part berubah; verifikasi di-reset |

Aturan:

* `Not Compatible` memblokir pemilihan part untuk unit target (hard block).
* `Conditional Compatible` menampilkan warning + note; planner harus mengonfirmasi.
* `Pending Verification` menampilkan warning; boleh dipilih dengan konfirmasi planner.
* Verifikasi mencatat `verified_by`, `verified_at`, `source`; status selain `pending_verification` wajib punya `verified_by` dan `verified_at`.
* Perubahan target/part setelah verifikasi mengembalikan status ke pending.

### 6.2 Alternatif

* Relation: `equivalent`, `superseded`, `substitute`.
* Status verifikasi alternatif mengikuti §6.1; status selain `pending_verification` wajib punya verifier.
* Hanya alternatif dengan status approved/conditional yang disarankan.
* `superseded` searah (lama → baru); jangan otomatis membuat reverse.
* Alternatif tidak otomatis mengganti part di material requirement; planner memilih.

### 6.3 Pemilihan part

* Cari berdasarkan kode internal, nama, nomor OEM/aftermarket, brand.
* Filter compatibility berdasarkan unit model/engine/configuration.
* Jika tidak ada compatibility record: warning "belum terverifikasi"; planner boleh lanjut dengan konfirmasi.
* Jika `not_compatible`: block.
* Jika `conditional`: warning + note.

### 6.4 Master lifecycle

* Sparepart yang sudah dipakai tidak dihapus; `is_active=false`.
* Nomor part yang sudah dipakai tidak dihapus; nonaktifkan via catatan atau ganti primary.
* Compatibility/alternatif yang sudah dipakai tetap ada untuk history.
* Tidak ada harga/cost/vendor.

---

## 7. Unit History & Downtime

### 7.1 Sumber

| Data | Sumber |
|---|---|
| Maintenance timeline | `work_orders` + `wo_updates` + `work_order_checklist_items` |
| Sparepart/component history | `work_order_sparepart_usage` |
| Actual usage | usage `movement_type='issue'` net return |
| HM/KM | `unit_meter_readings`; `units.current_*` cache |
| Downtime | WO `started_at`/`created_at`/`closed_at` + `wo_updates` |
| Component interval | usage issue per unit+component |
| Mechanic/supervisor | `work_orders.assignee_id`/`supervisor_id` |

Tidak ada tabel total statis. Semua agregasi query/view.

### 7.2 HM/KM reading

* Value >= 0.
* Reading baru normalnya >= reading terakhir.
* Reading lebih rendah:
  * Dari P2H: tandai anomali, jangan update `units.current_*`; notifikasi admin.
  * Manual/WO: butuh alasan; admin dapat menerima sebagai koreksi; tetap simpan reading baru, jangan hapus lama.
* `units.current_*` = reading valid terbaru berdasarkan `recorded_at`.
* P2H sync insert reading source `p2h`; WO completion source `wo`; manual source `manual`.
* Unique per source mencegah duplikat.

### 7.3 Downtime

* Unplanned: `breakdown`/`corrective`; planned: `preventive`; inspection tidak dihitung.
* Start = `started_at`; fallback `created_at` untuk breakdown/corrective.
* End = `closed_at`.
* Hanya WO `done`; `cancelled` tidak dihitung.
* Waiting part = interval `wo_updates` status `waiting_part` → berikutnya; dijumlahkan.
* Overlap: union interval per unit, bukan sum.
* MTTR = average downtime unplanned pada periode.
* `is_breakdown` flag tidak menggantikan perhitungan.

### 7.4 Component replacement interval

* Event = issue usage dengan `component_id` tidak null; group per `wo_id` + `component_id` (satu WO = satu event).
* Jika `component_id` null, fallback ke `sparepart_id` (ditandai).
* Interval = `used_at`/HM/KM event sekarang − event sebelumnya untuk unit+component.
* HM/KM event diambil dari snapshot `work_order_sparepart_usage.hm/km`; fallback reading WO terkait atau reading terdekat.
* Frequency = jumlah event per periode.
* Bukan plan interval; plan interval adalah rencana, ini actual.

### 7.5 Visibilitas

* Semua role authenticated dapat membaca unit history; mechanic dapat melihat WO sendiri dan unit terkait.
* Data biaya tidak ada.
* Export mengikuti RBAC report.

---

## 8. Notifications

| Event | Penerima | Trigger | Dedupe key |
|---|---|---|---|
| PM upcoming | planner/admin | Scan harian | `pm:{schedule}:upcoming:{date}` |
| PM due soon | planner/admin | Scan harian | `pm:{schedule}:due_soon:{date}` |
| PM due | planner/admin | Scan harian | `pm:{schedule}:due:{date}` |
| PM overdue | planner/admin | Scan harian | `pm:{schedule}:overdue:{date}` |
| PM auto-WO | planner/admin + assignee | WO dibuat | `pm:{schedule}:wo:{wo_id}` |
| Material shortage/partially ready | planner/storekeeper | Status shortage/partially_ready | `mat:{requirement}:{status}:{date}` |
| Material ready | planner/mechanic | Status ready/reserved | `mat:{requirement}:{status}:{date}` |
| WO waiting part | planner/admin + `supervisor_id` WO | Status waiting_part | `wo:{wo}:waiting:{date}` |
| Low stock | storekeeper/admin | 07:30 WITA | `stock:{sparepart}:{location}:low:{date}` |
| Critical stock | storekeeper/admin | 07:30 WITA | `stock:{sparepart}:{location}:critical:{date}` |
| Adjustment/opname posted | admin/planner | Posting | `inv:{transaction}:posted` |
| Candidate P2H baru | planner/admin | Sync | existing M0 |
| Candidate reminder | planner/admin | Hourly | existing `notified_at` |

Aturan:

* In-app dulu; email fase 2.
* `dedupe_key` unik per user.
* Notifikasi tidak boleh menggantikan audit transaksi.
* Recipient default semua planner/admin; filter site fase berikut.

---

## 9. RBAC

| Aksi | admin | planner | mechanic | storekeeper | viewer |
|---|---|---|---|---|---|
| Settings & sync | ✅ | — | — | — | — |
| User/role management | ✅ | — | — | — | — |
| Unit master | ✅ | ✅ | — | — | read |
| PM plan create/update/activate | ✅ | ✅ | — | — | read |
| PM schedule generate/override/skip | ✅ | ✅ | — | — | read |
| PM checklist execution | ✅ | ✅ | ✅ own | — | read |
| PM completion | ✅ | ✅ | ✅ own | — | — |
| Approve/skip kandidat P2H | ✅ | ✅ | — | — | — |
| WO create/assign/cancel | ✅ | ✅ | — | — | — |
| WO start/update/complete | ✅ | ✅ | ✅ own | — | — |
| Sparepart master | ✅ | ✅ | — | read | read |
| Compatibility verify | ✅ | ✅ | — | — | read |
| Warehouse/location master | ✅ | ✅ | — | read | read |
| Receipt | ✅ | — | — | ✅ | — |
| Issue | ✅ | — | — | ✅ | — |
| Return | ✅ | — | — | ✅ | — |
| Transfer | ✅ | — | — | ✅ | — |
| Reservation | ✅ | ✅ | — | ✅ | — |
| Adjustment create | ✅ | ✅ | — | — | — |
| Adjustment post | ✅ | ✅ | — | — | — |
| Opname create/count | ✅ | — | — | ✅ | — |
| Opname post | ✅ | ✅ | — | — | — |
| Material requirement create/update | ✅ | ✅ | — | — | read |
| Material reserve | ✅ | ✅ | — | ✅ | — |
| Material issue/return | ✅ | — | — | ✅ | — |
| Material cancel | ✅ | ✅ | — | — | — |
| Dashboard/report | ✅ | ✅ | limited | ✅ | read |
| Unit history | ✅ | ✅ | ✅ | ✅ | ✅ |
| Audit/history | ✅ | ✅ | — | — | — |

Aturan:

* "own" = WO dengan `assignee_id = user.id`.
* Otorisasi dicek di server, bukan hanya UI.
* `storekeeper` tidak boleh mengubah PM plan/WO/material requirement.
* `planner` tidak boleh posting receipt/issue; boleh reservasi.
* `admin` dapat melakukan semua, tetapi tetap melalui guard transisi.

---

## 10. Audit

| Perubahan | Tempat audit |
|---|---|
| Setting | `app_settings_history` |
| Kandidat P2H | `wo_candidates.decided_by/at/skip_reason` |
| WO status/assignment/note | `wo_updates` |
| PM override/skip/cancel | `pm_schedules.override_*`, `skip_*`, `cancel_*` |
| PM completion | `pm_schedules.last_*`, `unit_meter_readings` |
| Material status | `material_requirement_updates` |
| Inventory posting | `inventory_transactions.posted_by/at`, `notes` |
| Reservation | `inventory_reservations.reserved_by/at`, `released_at`, `consumed_at` |
| Actual usage | `work_order_sparepart_usage.used_by/at` + ledger |
| Opname | `stock_opname_sessions.posted_by/at` |
| Compatibility | `verified_by/at`, `updated_at` |
| Sync | `sync_runs` |

Tidak ada generic audit log di v1. Jika audit penuh semua field dibutuhkan, tambahkan tabel audit terpisah di milestone berikut dengan approval—jangan mengandalkan log aplikasi sebagai satu-satunya sumber.

---

## 11. Edge cases

| Kasus | Aturan |
|---|---|
| PM due saat WO sebelumnya masih open | Tidak buat WO kedua; schedule tetap `wo_created` |
| Auto-WO dua worker bersamaan | Lock schedule + partial unique; satu WO |
| WO PM cancelled | Schedule kembali due/overdue; WO pengganti boleh |
| Plan inactive saat due | Tidak auto-WO; schedule tetap; planner putuskan |
| Unit model berubah setelah schedule ada | Schedule existing tetap; plan model baru berlaku untuk schedule baru |
| HM/KM reading turun | Quarantine/flag; koreksi admin; jangan hapus history |
| Material shortage saat WO in_progress | WO boleh `waiting_part`; tidak otomatis |
| Material ready saat waiting_part | Notifikasi; transisi manual ke in_progress |
| Issue tanpa reservasi | Boleh jika available cukup |
| Return tanpa issue | Ditolak; exception admin beralasan |
| Cancel WO dengan issue > return | Block kecuali admin override; return/reversal tetap tercatat |
| Cancel material requirement dengan issue | Sama seperti cancel WO |
| Transfer stok reserved | Ditolak; hanya available |
| Opname counted < reserved | Posting ditolak sampai reservasi dilepas |
| Compatibility berubah setelah reservasi | Reservasi existing tetap; planning baru mengikuti status baru |
| Part Not Compatible sudah terlanjur direservasi | Release/return; jangan issue baru |
| Stock opname saat ada transaksi | Lock balance; serialisasi |
| Retry posting inventory | Idempotency key mengembalikan transaksi sama |
| Duplikasi nomor dokumen | Retry on unique conflict |
| Qty 0/negatif | Ditolak |
| UOM berbeda | Tidak ada konversi di v1; qty dalam UOM sparepart |
| Over-issue/over-return | Ditolak |
| Negative stock | Ditolak |
| Posted ledger dihapus | Dilarang; reversal |
| Master dipakai lalu dihapus | Dilarang; nonaktifkan |
| Timezone 23:30 WITA | Tanggal bisnis tetap hari WITA; query UTC memakai batas WITA |
| Report harian | Bucket WITA, bukan UTC |

---

## 12. Referensi keputusan

* D-01 M0 locked.
* D-02 P2H candidate-based.
* D-04 Accurate procurement out.
* D-05 WITA/UTC.
* D-06/D-07 PM interval plan-level, first reached.
* D-08 PM auto-WO double control.
* D-09 1 PM schedule = 1 WO aktif.
* D-10 no duplicate tables.
* D-11 immutable ledger.
* D-12 planning/reservation ≠ stock.
* D-13 compatibility statuses.
* D-14 storekeeper role.
* D-15 quantity-only inventory.

Open questions dan default ada di PRD §13.2 / Brief §9. Jika implementasi membutuhkan keputusan baru, catat di PRD decision log terlebih dahulu.
