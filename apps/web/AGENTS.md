# Frontend Instructions

Sebelum mengubah UI di folder ini:

1. **Wajib** baca `docs/Frontend Design Contract.md` dan `apps/web/AGENTS.md` ini. Baca ulang jika menyentuh filter, field, atau kartu metrik.
2. **Wajib** baca template Figma `Sanggar ERP` via Figma MCP sebelum mengubah UI: https://www.figma.com/design/MKBk3um5mQT2M4mP0vSLxj/Sanggar-ERP?node-id=4-2
3. Gunakan `PageHeader`, `SearchInput`, `FilterBar`, dan komponen `components/ui` yang sudah ada.
4. Jangan membuat ukuran search, input, select, button, page title, atau nilai metrik baru per halaman.
5. Semua kontrol form memakai `text-sm`; semua nilai KPI/metrik memakai `text-2xl font-semibold`; semua label metrik memakai `text-sm text-muted-foreground`. Jangan mencampur ukuran teks untuk elemen yang sejenis.
6. Jika komponen shared menyimpang dari kontrak, perbaiki komponen shared tersebut. Jangan meng-override ukuran teks per halaman.
7. Semua modal popup wajib berada di tengah layar, memakai `SheetContent` yang sudah terpusat. Jangan membuat side sheet kanan untuk form/detail.
8. Referensi visual pendukung: https://demos.shadcndashboard.dev/?mode=dark

Sebelum dinyatakan selesai:

1. Jalankan `npm run lint`, `npm run build`, dan `npm run check:typography`; semuanya harus lulus.
2. Audit `rg -n "text-base|text-lg|type-overline" src` dan pastikan hanya pengecualian yang tercantum di kontrak (SheetTitle, FieldLegend, CardTitle, label grafik, form login).
3. Cek layout di mobile dan desktop, dark dan light, termasuk filter bar dengan date input dan select.
