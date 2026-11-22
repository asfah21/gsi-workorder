# Frontend Instructions

Sebelum mengubah UI di folder ini:

1. **Wajib** baca `docs/Frontend UI Guidelines.md`, terutama bagian **Typography** dan **Form dan sheet**. Baca ulang jika menyentuh filter, field, atau kartu metrik.
2. Gunakan `PageHeader`, `SearchInput`, `FilterBar`, dan komponen `components/ui` yang sudah ada.
3. Jangan membuat ukuran search, input, select, button, page title, atau nilai metrik baru per halaman.
4. Semua kontrol form memakai `text-sm`; semua nilai KPI/metrik memakai `text-2xl font-semibold`; semua label metrik memakai `text-sm text-muted-foreground`. Jangan mencampur ukuran teks untuk elemen yang sejenis.
5. Jika komponen shared menyimpang dari guideline, perbaiki komponen shared tersebut. Jangan meng-override ukuran teks per halaman.
6. Semua modal popup wajib berada di tengah layar, memakai `SheetContent` yang sudah terpusat. Jangan membuat side sheet kanan untuk form/detail.
7. Referensi visual: https://demos.shadcndashboard.dev/?mode=dark

Sebelum dinyatakan selesai:

1. Jalankan `npm run lint`, `npm run build`, dan `npm run check:typography`; semuanya harus lulus.
2. Audit `rg -n "text-base|text-lg|type-overline" src` dan pastikan hanya pengecualian yang tercantum di guideline (SheetTitle, FieldLegend, CardTitle, label grafik, form login).
3. Cek layout di mobile dan desktop, dark dan light, termasuk filter bar dengan date input dan select.
