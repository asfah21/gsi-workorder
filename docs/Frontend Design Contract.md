# Frontend Design Contract — Sanggar ERP / PAMS

**Status:** Berlaku di branch `pams`  
**Mulai berlaku:** 2026-11-23  
**Scope:** Seluruh UI di `apps/web`. File ini menggantikan `docs/Frontend UI Guidelines.md`.

Dokumen ini adalah **satu-satunya patokan desain frontend**. Setiap kali menyentuh frontend, agen wajib membaca dokumen ini **dan** membaca template Figma yang menjadi sumbernya. Jika keduanya bertentangan, aturan resolusi di bagian 3 yang berlaku.

---

## 1. Source of truth

| Prioritas | Sumber | Peran |
|---:|---|---|
| 1 | Figma `Sanggar ERP` — https://www.figma.com/design/MKBk3um5mQT2M4mP0vSLxj/Sanggar-ERP?node-id=4-2 | Sumber visual dan token desain. Wajib dibaca ulang via Figma MCP setiap kali mengerjakan frontend. |
| 2 | Dokumen ini (`docs/Frontend Design Contract.md`) | Terjemahan aturan Figma ke kontrak implementasi project. |
| 3 | `apps/web/AGENTS.md` | Instruksi operasional agen di folder frontend. |
| 4 | `apps/web/src/components/ui/*`, `components/layout/*`, `components/list/*` | Komponen yang harus dipakai ulang. |
| 5 | `apps/web/src/app/globals.css` | Token warna, radius, font, dan dark mode project. |

### Kewajiban membaca Figma

Sebelum mengubah UI, lakukan:

1. Muat skill `figma-design-to-code` (fallback: resource `skill://figma/figma-design-to-code/SKILL.md`).
2. `create_session` dengan `fileKey = MKBk3um5mQT2M4mP0vSLxj`.
3. `get_design_context` untuk node yang relevan (lihat peta di bagian 2). Gunakan `get_metadata`/`get_screenshot` bila perlu.
4. Bandingkan hasilnya dengan kontrak ini. Jika desain Figma berubah, **ikuti Figma terbaru** dan perbarui dokumen ini.
5. Jangan menyalin kode Tailwind hasil Figma apa adanya. Terjemahkan ke token, komponen shared, dan konvensi project.

> Link Figma di atas mengarah ke halaman `01 · Foundations`. Untuk layar aplikasi, gunakan node pada halaman `03 · Core ERP` atau `04 · Modul Inventory`.

---

## 2. Peta node Figma

| Node | Nama | Dipakai untuk |
|---|---|---|
| `0:1` | Cover | Identitas file, bukan implementasi. |
| `4:2` | 01 · Foundations | Halaman induk token/style guide. |
| `8:2` | 01 · Foundations V2 — Sanggar ERP | Frame induk (1440×4570). |
| `8:6` | Token Warna | Daftar variable `Sanggar/Colors`. |
| `8:128` | Tipografi & Data Mono | Skala tipografi dan mono. |
| `8:186` | Aturan Flat, Minimalist & Compact | Prinsip layout, radius, ikon, angka. |
| `122:2` | Pustaka Ikon | Daftar ikon dan ukurannya. |
| `122:190` | Ukuran, Jarak, dan Radius | Tabel ukuran komponen. |
| `122:226` | Cara Pakai Komponen | Pemetaan komponen dan aksi. |
| `4:3` | 02 · Components | Perpustakaan komponen Figma. |
| `4:4` | 03 · Core ERP | Layar aplikasi inti. |
| `4:5` | 04 · Modul Inventory | Layar modul inventory. |

---

## 3. Resolusi konflik Figma vs project

Keputusan yang berlaku di `pams`: **guidelines project menang** bila bertentangan dengan Figma. Figma tetap menjadi acuan untuk niat visual, ritme layout, dan semantik status.

| Aspek | Figma | Project | Keputusan |
|---|---|---|---|
| Font | Plus Jakarta Sans + JetBrains Mono | Geist via `next/font/local` | **Geist**. Jangan menambah font/library baru. |
| Page title | 20px bold | `text-2xl font-semibold tracking-tight` via `PageHeader` | **PageHeader**. |
| Nilai KPI | 24/30 bold | `text-2xl font-semibold` | **Project**, beda weight diterima. |
| Body | 12px | `text-sm` (14px) | **Project**. |
| Warna primary | Hijau `#3F7A1C` | Primary netral (`oklch`) | **Token project**. Hijau Figma hanya referensi visual. |
| Radius kartu | 16px | `--radius` 10px + skala token | **Token project**. Jangan pakai px arbitrary. |
| Ikon | Tabler Icons | `lucide-react` | **lucide-react**, 16px default. Jangan tambah library ikon. |
| Modal | — | `SheetContent` terpusat | **Project**. Tidak ada side sheet kanan. |

---

## 4. Layout & spacing

Nilai Figma di bawah adalah target visual. Implementasikan dengan kelas/token yang sudah ada; jangan membuat ukuran baru per halaman.

| Elemen | Target Figma | Implementasi project |
|---|---|---|
| Canvas | `#F4F7FA` | `bg-background` (token). |
| Kartu | Putih, border 1px `#EEF2F6`, tanpa shadow tebal | `Card` + pengaturan shadow yang sudah ada. |
| Jarak antar section | 16px | `gap-4`. |
| Jarak antar kartu | 12–16px | `gap-3` / `gap-4`. |
| Padding kartu | 14–24px | Padding bawaan `Card`/`CardContent`. |
| Skala jarak | 4, 8, 12, 16, 24, 32px | `gap-1/2/3/4/6/8`. |
| Button & input | 36px | `h-9`. |
| Button kecil / row action | 28px | `Button size="xs"`. |
| Header tabel | 36px | `h-9` pada header `Table`. |
| Baris tabel | 40px | Perilaku tabel existing; angka mono rata kanan. |
| Radius kontrol | 6–8px | `rounded-md` / token `--radius-md`. |
| Radius kartu & dialog | 10–16px | `rounded-lg` / `rounded-xl` sesuai token project. |

Prinsip: **flat, crisp hairline, compact**. Tidak ada drop-shadow tebal, tidak ada dekorasi berlebih, tidak ada campuran skala ukuran pada elemen sejenis.

---

## 5. Warna

Tabel berikut adalah token Figma (`Sanggar/Colors`) sebagai referensi visual. **Jangan menulis hex ini di kode.** Gunakan token semantik project (`bg-background`, `text-foreground`, `border-border`, `text-muted-foreground`, `bg-muted`, `bg-card`) dan pola status yang sudah ada.

| Grup | Token Figma | Hex | Peran |
|---|---|---|---|
| Brand | `brand/green` | `#77B844` | Hijau logo. |
| Brand | `brand/green-deep` | `#3F7A1C` | Tombol primer/aktif (referensi). |
| Brand | `brand/green-soft` | `#ECFDF3` | Latar hijau lembut. |
| Brand | `brand/blue` | `#2462AF` | Biru logo/info. |
| Brand | `brand/blue-soft` | `#EFF6FF` | Latar biru lembut. |
| Brand | `surface/dark` | `#1E5296` | Kartu hero gelap. |
| Surface | `surface/canvas` | `#F4F7FA` | Kanvas dashboard. |
| Surface | `surface/card` | `#FFFFFF` | Permukaan kartu. |
| Surface | `surface/subtle` | `#F7F9FC` | Permukaan halus. |
| Border | `border/subtle` | `#EEF2F6` | Hairline 1px. |
| Border | `border/strong` | `#E2E8F0` | Pemisah kuat. |
| Text | `text/primary` | `#111827` | Judul. |
| Text | `text/secondary` | `#475569` | Isi. |
| Text | `text/muted` | `#94A3B8` | Sekunder. |
| Text | `text/on-dark` | `#FFFFFF` | Teks di gelap. |
| Status | `status/safe-text` / `safe-bg` | `#4E8A25` / `#EEF7E6` | Aman. |
| Status | `status/warn-text` / `warn-bg` | `#B45309` / `#FEF3C7` | Waspada. |
| Status | `status/danger-text` / `danger-bg` | `#B91C1C` / `#FEE2E2` | Perhatian. |
| Status | `status/stale-text` / `stale-bg` | `#4B5563` / `#F1F5F9` | Stale. |

Aturan status: pakai pola status yang sudah disepakati, misalnya `bg-emerald-500/10 text-emerald-700 dark:text-emerald-300`. Warna hanya dipakai untuk menyampaikan masalah/status, bukan dekorasi.

---

## 6. Tipografi

Kolom Figma adalah referensi visual; kolom Project yang wajib dipakai.

| Gaya Figma | Spesifikasi Figma | Implementasi project |
|---|---|---|
| KPI Metric | Plus Jakarta Sans Bold 24/30, -0.48 | `text-2xl font-semibold` |
| Page Title | Plus Jakarta Sans Bold 20/26, -0.2 | `PageHeader` → `text-2xl font-semibold tracking-tight` |
| Card Title | SemiBold 14/20, -0.5 | `CardTitle` default |
| Body Strong | SemiBold 12/16 | `text-sm font-medium` |
| Body Medium | Medium 12/16 | `text-sm` |
| Label | Medium 11/15 | `FieldLabel` default (`text-sm font-medium leading-5`) |
| Badge | SemiBold 11/14 | Badge `h-5` / `h-6` |
| Helper | Regular 10/14 | `text-xs text-muted-foreground` |
| Data Mono | JetBrains Mono Medium 12/16 | `font-mono text-sm`, angka rata kanan di tabel |
| Data Mono SM | JetBrains Mono Medium 11/15 | `font-mono text-xs` |
| Section Caps | SemiBold 10/14, +0.6px | `type-overline-xs`, **khusus label grafik/metadata** |

Aturan tambahan:

- Satu jenis elemen hanya memakai satu ukuran teks. Jangan mencampur `text-sm`, `text-base`, dan `text-lg` untuk elemen setara.
- `type-overline` / `type-overline-xs` hanya untuk label grafik atau metadata; bukan label form maupun label KPI.
- Pengecualian yang diizinkan: `SheetTitle`, `FieldLegend`, `CardTitle`, label grafik, dan form login. Jangan menambah pengecualian baru.
- `npm run check:typography` harus lulus.

---

## 7. Komponen

| Konsep Figma | Komponen project |
|---|---|
| Button Primary / Secondary / Ghost | `Button` dengan variant yang sesuai. |
| Panel, kartu metrik | `Card`, `CardHeader`, `CardContent`, `KpiCard`. |
| Judul halaman + breadcrumb + aksi | `PageHeader`. |
| Kolom pencarian | `SearchInput` di dalam `Field`. |
| Baris filter | `FilterBar` + `NativeSelect` + `Button` `outline`/`ghost`. |
| Tabel, header caps, baris, pagination | `Table` dalam `overflow-x-auto` + `TablePagination`. |
| Badge status | `Badge`, `ActiveBadge`, atau badge lokal `h-5 px-2` / `h-6 px-2.5`. |
| Dialog / form / detail | `SheetContent` terpusat, `w-full sm:max-w-*`, `max-h-[calc(100dvh-2rem)]`, light dismiss. |
| Empty / loading / error | `Empty` + turunannya, `Skeleton`, `Alert` / `FieldError`. |
| Grafik | `ChartContainer` + token `--chart-*`; loading, empty, failed state wajib ada. |
| Ikon | `lucide-react`, `size-4` (16px) default, `aria-label` pada tombol ikon. |

Dilarang:

- Membuat search input manual dengan ikon absolut sendiri.
- `div` dengan border/shadow manual sebagai pengganti `Card`.
- `<button>` / `<input>` native tanpa alasan.
- Modal atau panel detail yang muncul dari samping layar.
- Menambah library UI, state, chart, icon, atau font tanpa approval.

---

## 8. Responsive & dark mode

- Target minimum **390px**.
- Grid default 1 kolom → `sm:grid-cols-2` → `xl:grid-cols-4`.
- Filter bar vertikal di mobile, horizontal di `md`.
- Tabel selalu `overflow-x-auto`; jangan memaksa viewport melebar.
- Sheet full width di mobile.
- Cek `scrollWidth` tidak melebihi `innerWidth`.
- Uji di 390, 768, 1280, dan 1536px, dalam mode light dan dark.
- Semua warna memakai token; tidak ada `bg-white`, `text-black`, `gray-*`, `slate-*`, atau hex hardcoded.

---

## 9. Checklist setiap kali menyentuh frontend

1. Baca `apps/web/AGENTS.md` dan dokumen ini.
2. Baca node Figma yang relevan via Figma MCP (bagian 1–2).
3. Gunakan komponen shared yang sudah ada; jangan membuat ukuran baru per halaman.
4. Terjemahkan nilai Figma ke token/kelas project (bagian 3–7).
5. Tangani state loading, empty, error, dan unauthorized.
6. Gunakan `next/link` untuk navigasi internal.
7. Pakai istilah **unit** untuk kendaraan/alat.
8. Jalankan `npm run lint`, `npm run build`, `npm run check:typography`.
9. Audit `rg -n "text-base|text-lg|type-overline" apps/web/src` — hanya pengecualian terdokumentasi yang boleh muncul.
10. Cek desktop/mobile dan dark/light, termasuk filter bar dengan date input dan select.

---

## 10. Definition of done

- Lint, build, dan typography check lulus.
- Tidak ada error TypeScript.
- Tidak ada overflow horizontal di 390/768/1280/1536px.
- Dark dan light mode konsisten.
- Search, filter, tabel, pagination, empty state, dan error state konsisten di seluruh halaman.
- Perubahan tetap berada di branch `pams`; `main`/`origin/main` tidak disentuh atau di-merge tanpa konfirmasi eksplisit.

---

## 11. Catatan implementasi node `32:2` — Dashboard Inventory

Keputusan adaptasi berikut berlaku untuk implementasi di `apps/web`:

| Elemen Figma | Keputusan implementasi |
|---|---|
| `AppTopBar` + sidebar inventory | Memakai `AppShell` existing; tidak membuat shell baru. |
| Breadcrumb `Beranda / Inventory / Dashboard` | `PageHeader` di route `/inventory/dashboard`. |
| 4 `MetricCard` | Total SKU aktif (`/spareparts?active_only=true`), Pergerakan 7 hari (`summary.movements`), Stok menipis (`summary.low` + badge `summary.critical`), Reservasi aktif (`/inventory/reservations?status=active`). |
| `Nilai Persediaan` | Tidak dipakai: tidak ada endpoint valuasi. Diganti "Pergerakan 7 hari". |
| `PO Terbuka` dan tombol `Pesan` | Dihilangkan: project tidak memiliki alur procurement/PO. |
| Badge tren `+4% vs Sep`, `+2,1% vs Sep` | Tidak dipakai: tidak ada data pembanding; diganti konteks data aktual. |
| `imgPlot` SVG statis | Tidak dipakai; grafik dirender dari data nyata lewat `ChartContainer` (dua seri: Masuk/Keluar). |
| Kolom `Oleh` pada tabel | Tidak tersedia di payload; diganti `Referensi` (`reference_no`/`transaction_code`). |
| Ikon Tabler | Tetap `lucide-react`. |
| Warna seri grafik | Memakai `--chart-1` / `--chart-2` project, bukan hijau/biru Figma. |

File terkait: `components/inventory-dashboard-view.tsx`, `components/inventory-movement-chart.tsx`, dan route `(app)/inventory/dashboard/page.tsx`.

> Catatan implementasi: pemanggilan ulang Figma MCP saat eksekusi terkena batas kuota Starter. Acuan yang dipakai adalah `get_design_context` node `32:2` yang sudah berhasil diambil pada sesi yang sama sebelum batas kuota tercapai.

---

## 12. Catatan implementasi node `25:1071` — Detail Modul

Node ini adalah halaman katalog/detail modul, bukan layar operasional inventory.

| Elemen Figma | Keputusan implementasi |
|---|---|
| `AppTopBar` + `AppSidebarV2` | Memakai `AppShell` existing; tidak dibuat ulang. |
| Breadcrumb | `PageHeader` di route `/modules` dengan `Beranda / Modul / Inventory`. |
| Kartu identitas modul + status `Terpasang` | `Card` + `Badge`, ikon `Warehouse` dari `lucide-react`. |
| Metadata Versi / Penerbit / Diperbarui / Ukuran | Grid `1 → sm:2 → xl:4`, label `text-sm text-muted-foreground`, nilai `text-sm font-medium`. |
| 3 metrik modul | `Card` grid `1 → sm:3`, nilai `text-2xl font-semibold`. |
| Riwayat Versi | `Table` dalam `overflow-x-auto`, 3 baris statis. |
| Dependensi Finance & Sales | `Card` + ikon `Coins`/`ShoppingCart`, badge status. |
| Panel Aksi | `Button` **disabled** karena tidak ada module registry/API; diberi `title` penjelas. |
| Data modul | Konstanta statis dari Figma, diberi alert "Data contoh dari template Figma". |
| Ikon 24px Figma | Tetap `lucide-react` `size-4` sesuai kontrak. |

File terkait: `components/module-detail-view.tsx`, route `(app)/modules/page.tsx`, dan menu `Modul` di `components/app-shell.tsx`.

> Catatan implementasi: Figma MCP masih terkena batas kuota Starter. Acuan yang dipakai adalah metadata `25:1071` yang diekstrak dari `get_metadata` sebelum kuota habis. Label tombol aksi diperkirakan dari ukuran/konvensi tombol Figma karena teksnya tidak terbaca di metadata.

---

## 13. Catatan implementasi node `26:1280` — Tambah Modul

| Elemen Figma | Keputusan implementasi |
|---|---|
| `AppTopBar` + `AppSidebarV2` | Memakai `AppShell` existing. |
| Breadcrumb | `PageHeader` di route `/modules/new` dengan `Beranda / Modul / Tambah modul`. |
| Dropzone `Berkas Modul` | `<label>` untuk `<input type="file" accept=".zip" className="sr-only">`; ikon `CloudUpload`, teks maksimal 50 MB. |
| 2 `TextInput` + `SelectField` + `Textarea` | `Field` + `FieldLabel` + `Input`/`NativeSelect`/`Textarea`; semua kontrol `text-sm`. |
| 3 `Checkbox` izin | `<input type="checkbox">` dengan kelas Tailwind; exception karena project belum punya primitive `Checkbox`. |
| `KeyValueInfoCard` | `Card` "Ringkasan instalasi" dengan pasangan label/nilai + badge `Draft`. |
| `Alur Instalasi` + `ToastAlert` | `Card` + `Info` + `Alert`; teks status Draft dan informasi belum terhubung API. |
| Tombol aksi | `Button` **disabled** untuk "Pasang modul"; tidak ada mutasi. |
| Data formulir | `defaultValue` statis dari Figma + alert "Data contoh dari template Figma". |

File terkait: `components/module-form-view.tsx` dan route `(app)/modules/new/page.tsx`.

> Catatan implementasi: Figma MCP masih terkena batas kuota Starter. Acuan yang dipakai adalah metadata `26:1280` yang diekstrak dari `get_metadata` sebelum kuota habis. Label dua `TextInput` diperkirakan sebagai "Nama modul" dan "Versi" karena teks label tidak terbaca di metadata.

---

## 14. Riwayat

| Tanggal | Perubahan |
|---|---|
| 2026-11-23 | Dibuat di branch `pams` sebagai pengganti `docs/Frontend UI Guidelines.md`. Menetapkan Figma Sanggar ERP sebagai template wajib dan resolusi konflik dengan guidelines project. |
| 2026-11-23 | Menambahkan keputusan adaptasi node `32:2` (Dashboard Inventory) ke kontrak. |
| 2026-11-23 | Menambahkan keputusan adaptasi node `25:1071` (Detail Modul) ke kontrak. |
| 2026-11-23 | Menambahkan keputusan adaptasi node `26:1280` (Tambah Modul) ke kontrak. |
