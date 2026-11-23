# Frontend UI Guidelines

Dokumen ini adalah kontrak visual dan layout untuk `apps/web`.

## Source of truth

- Referensi visual utama: https://demos.shadcndashboard.dev/?mode=dark
- Referensi komponen: `apps/web/src/components/ui/*`
- Token warna, radius, font, dan dark mode: `apps/web/src/app/globals.css`
- Font: Geist melalui `next/font/local`
- Stack: Next.js 15 App Router, React 19, TypeScript, Tailwind CSS v4, shadcn/Base UI

Jangan menambah library UI, state, chart, atau icon baru tanpa approval.

## Aturan utama

1. Gunakan komponen yang sudah ada di `components/ui` dan shared component di `components/layout` serta `components/list`.
2. Jangan membuat ukuran input, search, select, atau button baru per halaman.
3. Jangan memakai hardcoded color Tailwind seperti `text-gray-500`, `bg-zinc-900`, atau hex.
4. Dark mode harus berjalan tanpa perubahan khusus per halaman.
5. Semua halaman list memakai pola: `PageHeader` → filter card → table card → pagination.
6. Semua search field memakai `SearchInput`.
7. Semua filter form memakai `FilterBar`.
8. Semua form memakai `Field`, `FieldLabel`, `FieldError`, `Input`, `NativeSelect`, dan `Textarea`.
9. Semua halaman harus menangani loading, empty, error, dan unauthorized state.
10. Jangan memakai `<a>` untuk navigasi internal; gunakan `next/link`.
11. Semua kontrol form dan labelnya memakai `text-sm`; jangan mencampur `text-base` atau `text-lg` dalam satu field, filter bar, atau kartu metrik.

## Page shell

Root halaman:

```tsx
<div className="flex flex-col gap-4">
  <PageHeader ... />
  {/* filter card */}
  {/* table/card */}
</div>
```

Jangan membungkus page header di dalam Card kecuali halaman dashboard yang memang berupa kumpulan widget.

## PageHeader

Gunakan `components/layout/page-header.tsx`.

```tsx
<PageHeader
  title="Work Order"
  description="Kelola work order manual, P2H, dan PM."
  breadcrumb={
    <nav aria-label="Breadcrumb" className="flex items-center gap-2">
      <Link href="/dashboard">Dashboard</Link>
      <span aria-hidden="true">/</span>
      <span aria-current="page">Work Order</span>
    </nav>
  }
  actions={
    <>
      <Badge variant="secondary">12 data</Badge>
      <Button>Tambah</Button>
    </>
  }
/>
```

Aturan:

- Judul selalu `text-2xl font-semibold tracking-tight` melalui `PageHeader`.
- Deskripsi selalu `text-sm text-muted-foreground`, maksimal `max-w-3xl`.
- Action berada di kanan pada desktop dan wrap di mobile.
- Jangan membuat `<h1>` manual dengan ukuran berbeda.

## Search

Gunakan `components/list/list-search.tsx`.

```tsx
<Field className="min-w-0 flex-1">
  <FieldLabel htmlFor="unit-search">Cari unit</FieldLabel>
  <SearchInput
    id="unit-search"
    name="q"
    value={query}
    onChange={(event) => setQuery(event.target.value)}
    placeholder="Kode, nopol, lokasi, atau merk"
  />
</Field>
```

Aturan:

- Semua search field: tinggi `h-9`, ikon kiri, `pl-9`, teks `text-sm`.
- Placeholder harus menjelaskan cakupan search.
- Search form harus memiliki label yang terlihat atau `sr-only`.
- Jangan membuat input search manual dengan `Search` icon sendiri.
- Jangan memakai ukuran `h-8`, `text-base`, atau `pl-10` untuk search.

## Filter bar

Gunakan `components/list/list-filter-bar.tsx`.

```tsx
<FilterBar onSubmit={submitFilters} aria-label="Cari dan filter unit">
  <Field className="min-w-0 flex-1">
    <FieldLabel htmlFor="unit-search">Cari unit</FieldLabel>
    <SearchInput id="unit-search" name="q" ... />
  </Field>
  <Field className="md:w-44">
    <FieldLabel htmlFor="unit-status">Status</FieldLabel>
    <NativeSelect id="unit-status" ...>
      <NativeSelectOption value="">Semua status</NativeSelectOption>
    </NativeSelect>
  </Field>
  <div className="flex gap-2 md:ml-auto">
    <Button type="submit" variant="outline">
      <Filter aria-hidden="true" />
      Terapkan
    </Button>
    <Button type="button" variant="ghost" onClick={reset}>
      <RotateCcw aria-hidden="true" />
      Reset
    </Button>
  </div>
</FilterBar>
```

Aturan:

- Tinggi select, input, dan button dalam satu baris harus sama (`h-9`).
- Di mobile, filter boleh bertumpuk; di desktop memakai baris horizontal.
- Tombol submit memakai `variant="outline"`.
- Tombol reset memakai `variant="ghost"`.
- Jangan memakai custom `<label className="h-9">`; gunakan `Field`.
- Untuk filter kompleks (4 field atau lebih), boleh memakai `grid gap-3 md:grid-cols-2 xl:grid-cols-6`; semua search tetap memakai `SearchInput` dan semua kontrol tetap `h-9`.
- Semua label (`FieldLabel`) dan kontrol (`Input`, `NativeSelect`, `SearchInput`) dalam satu filter bar wajib memakai `text-sm`. Date picker memakai `Input type="date"`, bukan `<input>` native.
- Jangan meng-override ukuran teks kontrol per halaman; perbaiki komponen shared di `components/ui` jika ada penyimpangan.

## Card dan table

Pola list:

```tsx
<Card>
  <CardHeader>
    <CardTitle>Daftar unit</CardTitle>
    <CardDescription>...</CardDescription>
  </CardHeader>
  <CardContent className="flex flex-col gap-3">
    {/* toolbar jika diperlukan */}
    <div className="overflow-x-auto">
      <Table>...</Table>
    </div>
    <TablePagination ... />
  </CardContent>
</Card>
```

Aturan:

- Card memakai komponen `Card`, bukan `div` dengan border manual.
- Table selalu dibungkus `overflow-x-auto`.
- Header tabel tidak memakai font lebih besar dari body.
- Row action memakai `Button size="xs"`.
- Pagination selalu memakai `TablePagination`.
- Loading memakai `Skeleton`, bukan teks “Loading...”.
- Empty state memakai `Empty`.

## Form dan sheet

```tsx
<Field>
  <FieldLabel htmlFor="name">Nama</FieldLabel>
  <Input id="name" ... />
  <FieldError>{error}</FieldError>
</Field>
```

Aturan:

- Form desktop memakai grid 2 kolom jika field lebih dari 3.
- Field full-width memakai `className="md:col-span-2"`.
- Semua input memakai komponen `Input`; textarea memakai `Textarea`.
- Semua modal wajib berada di tengah layar, bukan panel kanan/kiri.
- Gunakan `SheetContent` yang sudah dioverride menjadi dialog terpusat.
- Modal memakai `w-full`, `sm:max-w-*` sesuai kompleksitas, `max-h-[calc(100dvh-2rem)]`, dan scroll internal.
- Overlay wajib menutup area belakang dan mendukung light dismiss.
- Tombol simpan berada di `SheetFooter`.
- Button utama memakai `size="sm"` di sheet/form compact, atau default di halaman.
- Jangan mencampur `h-8` dan `h-9` dalam satu form.
- Teks kontrol (`Input`, `Textarea`, `NativeSelect`) selalu `text-sm`; `Input` dan `NativeSelect` memakai `leading-5`, `Textarea` memakai `leading-6`.
- Jangan menimpa ukuran teks kontrol di level halaman; gunakan default komponen shared.
- Jangan membuat side sheet baru dengan `side="right"` untuk form/detail.

### Impor dan ekspor data

- Tombol `Impor XLSX` dan `Ekspor XLSX` berada di `PageHeader actions`, bukan di dalam tabel.
- Form unggah file memakai centered `Sheet`; input file tetap memakai `Field` + `FieldLabel` + `Input type="file"`.
- Ekspor mengikuti filter aktif pada halaman list, tanpa pagination.
- Template unduhan dan hasil impor memakai file `.xlsx`; backend yang membuat dan mem-parsing file.
- Ringkasan hasil impor ditampilkan dengan `Alert`; baris gagal ditampilkan dengan `Table` di dalam `overflow-x-auto`.

### Dashboard

- Bagian atas memakai sapaan, tanggal WITA, dan empat `KpiCard` dengan nilai `text-2xl font-semibold`.
- Bagian bawah memetakan data nyata sebagai berikut: downtime/MTTR, status work order, kesiapan stok, unit aktif, kondisi unit, dan penunggu part.
- `Cuplikan unit` memakai tabel lebar penuh; `Akses cepat` menjadi kartu grid di bawahnya.
- Jangan menambah data tren palsu atau angka dummy untuk menyerupai demo; semua nilai harus berasal dari endpoint dashboard.

## Typography

Satu jenis elemen hanya boleh memakai satu ukuran teks. Jangan mencampur `text-sm`, `text-base`, dan `text-lg` untuk elemen yang setara.

| Elemen | Class |
|---|---|
| Page title | `text-2xl font-semibold tracking-tight` |
| Section title halaman (`h2`) | `text-base font-medium` |
| Card title | `CardTitle` default |
| Card section label (di dalam `Card`) | `text-sm font-medium` |
| Form label | `FieldLabel` default (`text-sm font-medium leading-5`) |
| Form control (`Input`, `NativeSelect`) | `text-sm leading-5` |
| Form control (`Textarea`) | `text-sm leading-6` |
| Form description | `text-sm text-muted-foreground` |
| Body | `text-sm` |
| Muted description | `text-sm text-muted-foreground` |
| KPI/metric label | `text-sm text-muted-foreground` |
| KPI/metric value | `text-2xl font-semibold` |
| KPI/metric hint | `text-xs text-muted-foreground` |
| Table metadata | `text-xs text-muted-foreground` |
| Chart/legend label | `type-overline-xs` (khusus label grafik) |
| Badge | `h-5` atau `h-6` dengan padding konsisten |

Aturan tambahan:

- Jangan memakai `text-xl font-medium` untuk page title.
- Semua field dalam satu filter bar harus memiliki ukuran teks yang sama; date input tidak boleh lebih besar dari select.
- Nilai KPI tidak boleh memakai `text-lg`; gunakan `text-2xl font-semibold` secara konsisten, termasuk untuk kartu metrik ringkas.
- `type-overline` dan `type-overline-xs` hanya untuk label grafik atau metadata, bukan untuk label form maupun label KPI.
- `Skeleton` harus mengikuti tinggi teks final (misalnya `h-8` untuk KPI `text-2xl`).
- Pengecualian yang sudah disepakati: `SheetTitle` (`text-lg font-semibold`), `FieldLegend`, judul kartu bawaan, label grafik, dan form login yang mengikuti referensi demo. Jangan menambah pengecualian baru tanpa approval.

## Dashboard dan metrik

- Urutan dashboard: `PageHeader` → KPI row → filter periode → chart utama → chart pendukung → table/panel.
- KPI row memakai 4 card dengan pola yang sama: label `text-sm`, value `text-2xl font-semibold`, hint `text-xs text-muted-foreground`.
- Jangan menduplikasi metrik yang sama pada beberapa card atau section.
- Chart memakai `ChartContainer` dan token `--chart-*`; jangan memakai warna hardcoded.
- Setiap chart wajib memiliki loading (`Skeleton`), empty (`Empty`), dan failed state.
- Chart utama memakai grid 8/4 pada desktop; chart pendukung memakai grid 3 kolom.
- Satu kontrol periode harus memengaruhi semua chart dan tabel operasional.
- Table pada dashboard tetap dibungkus `overflow-x-auto`.

## Button

- Action utama halaman: default size.
- Action dalam tabel: `size="xs"`.
- Action dalam toolbar: default atau `size="sm"` jika toolbar padat.
- Semua tombol ikon harus memiliki `aria-label`.
- Jangan membuat `<button>` custom jika `Button` sudah cukup.

## Status dan badge

- Status aktif/nonaktif: `ActiveBadge`.
- Status WO/PM/inventory: badge lokal harus mengikuti pola `h-5 px-2` atau `h-6 px-2.5`.
- Warna status hanya memakai token semantik atau kelas status yang sudah disepakati.
- Jangan memakai warna hardcoded selain pola status yang ada.

## Empty, loading, error

- Loading: `Skeleton` dengan tinggi yang sama dengan konten final.
- Empty: `Empty`, `EmptyHeader`, `EmptyMedia`, `EmptyTitle`, `EmptyDescription`.
- Error: `Alert variant="destructive"` atau `FieldError` untuk field.
- Success: `Alert` dengan border emerald.
- Jangan menampilkan error mentah dari server tanpa pesan yang bisa dipahami.

## Responsive

- Target minimum: 390 px.
- Grid default: 1 kolom; `sm:grid-cols-2`; `xl:grid-cols-4`.
- Table harus scroll horizontal, bukan memaksa viewport melebar.
- Search/filter bar: vertikal di mobile, horizontal di `md`.
- Sheet: full width di mobile.
- Cek `scrollWidth` tidak melebihi `innerWidth`.

## Dark mode

- Semua warna memakai token `bg-background`, `text-foreground`, `border-border`, `text-muted-foreground`, `bg-muted`, `bg-card`, dan seterusnya.
- Jangan memakai `bg-white`, `text-black`, `gray-*`, `slate-*`, atau hex.
- Status color boleh memakai pola `bg-emerald-500/10 text-emerald-700 dark:text-emerald-300` yang sudah ada.

## Forbidden patterns

- Search input manual dengan icon absolut sendiri.
- `h-8`, `h-10`, `text-base`, atau `pl-10` untuk search field.
- Page title `text-xl`.
- `div` dengan border/shadow manual sebagai pengganti Card.
- `<button>` atau `<input>` native tanpa alasan.
- Warna hardcoded.
- Form tanpa label atau `aria-label`.
- Table tanpa `overflow-x-auto`.
- Empty state teks polos tanpa `Empty`.
- Link internal memakai `<a>`.
- `text-base` atau `text-lg` pada `Input`, `Textarea`, `NativeSelect`, atau `SearchInput`.
- Mencampur ukuran teks label dan kontrol dalam satu field atau filter bar.
- `text-lg` untuk nilai KPI/metrik.
- Memakai `type-overline`/`type-overline-xs` sebagai label form atau label KPI.
- Modal/form/detail yang muncul dari kanan layar.
- `SheetContent` dengan positioning side sheet manual.

## Definition of done

- `npm run lint` lulus.
- `npm run build` lulus.
- Tidak ada error TypeScript.
- Halaman dicek di 390, 768, 1280, dan 1536 px.
- Dark dan light mode dicek.
- Tidak ada overflow horizontal.
- Search, filter, table, pagination, empty state, dan error state konsisten.
- Audit `rg -n "text-base|text-lg|type-overline" apps/web/src` hanya menampilkan pengecualian yang terdokumentasi.
- Label dan nilai KPI memakai satu skala tipografi yang sama di seluruh dashboard.
- `npm run check:typography` lulus.
- Screenshot perbandingan dengan demo target bila memungkinkan.
