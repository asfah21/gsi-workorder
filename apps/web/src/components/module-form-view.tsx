"use client";

import { CloudUpload, Info } from "lucide-react";
import Link from "next/link";

import { PageHeader } from "@/components/layout/page-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

const permissions = [
  "Baca data persediaan dan lokasi gudang",
  "Tulis faktur penjualan dan pengiriman",
  "Kirim notifikasi kepada pengguna",
];

const summaryRows = [
  { label: "Format", value: ".zip" },
  { label: "Ukuran maksimal", value: "50 MB" },
  { label: "Status awal", value: "Draft" },
  { label: "Calon pengguna", value: "18" },
];

export function ModuleFormView() {
  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        breadcrumb={
          <>
            <Link href="/dashboard">Beranda</Link>
            <span aria-hidden="true">/</span>
            <Link href="/modules">Modul</Link>
            <span aria-hidden="true">/</span>
            <span className="text-foreground">Tambah modul</span>
          </>
        }
        title="Tambah modul"
        description="Unggah paket modul dan tentukan informasi serta izin yang diminta."
        actions={
          <>
            <Link href="/modules" className={cn(buttonVariants({ variant: "outline" }))}>
              Batal
            </Link>
            <Button type="button" disabled title="Membutuhkan module registry/API">
              Pasang modul
            </Button>
          </>
        }
      />

      <Alert className="border-border bg-muted/40">
        <Info aria-hidden="true" />
        <AlertTitle>Data contoh dari template Figma</AlertTitle>
        <AlertDescription>
          Formulir ini hanya sebagai referensi visual. Belum ada module registry/API,
          sehingga unggahan dan izin tidak disimpan.
        </AlertDescription>
      </Alert>

      <div className="grid gap-4 xl:grid-cols-12">
        <div className="flex flex-col gap-4 xl:col-span-8">
          <Card>
            <CardHeader>
              <CardTitle>Berkas modul</CardTitle>
            </CardHeader>
            <CardContent>
              <label
                htmlFor="module-file"
                className="flex min-h-28 cursor-pointer flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border bg-muted/40 px-4 py-6 text-center hover:bg-muted/60"
              >
                <CloudUpload className="size-6 text-muted-foreground" aria-hidden="true" />
                <span className="text-sm font-medium">Seret berkas .zip modul ke sini</span>
                <span className="text-xs text-muted-foreground">
                  atau pilih dari komputer - maksimal 50 MB
                </span>
                <input
                  id="module-file"
                  type="file"
                  accept=".zip"
                  className="sr-only"
                  aria-label="Berkas modul .zip"
                />
              </label>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Informasi modul</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <Field>
                  <FieldLabel htmlFor="module-name">Nama modul</FieldLabel>
                  <Input id="module-name" defaultValue="Inventory" />
                </Field>
                <Field>
                  <FieldLabel htmlFor="module-version">Versi</FieldLabel>
                  <Input id="module-version" defaultValue="v1.0.0" />
                </Field>
                <Field>
                  <FieldLabel htmlFor="module-category">Kategori</FieldLabel>
                  <NativeSelect id="module-category" defaultValue="inventory">
                    <NativeSelectOption value="inventory">Inventory</NativeSelectOption>
                    <NativeSelectOption value="finance">Finance</NativeSelectOption>
                    <NativeSelectOption value="sales">Sales</NativeSelectOption>
                  </NativeSelect>
                </Field>
                <Field>
                  <FieldLabel htmlFor="module-compatibility">Kompatibilitas</FieldLabel>
                  <NativeSelect id="module-compatibility" defaultValue="all">
                    <NativeSelectOption value="all">Semua versi</NativeSelectOption>
                    <NativeSelectOption value="v2">Sanggar ERP v2</NativeSelectOption>
                  </NativeSelect>
                </Field>
              </div>
              <Field>
                <FieldLabel htmlFor="module-description">Deskripsi</FieldLabel>
                <Textarea
                  id="module-description"
                  rows={3}
                  defaultValue="Kelola stok, lokasi gudang, dan pergerakan barang masuk-keluar."
                />
              </Field>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Izin yang diminta</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              {permissions.map((permission) => (
                <label key={permission} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    defaultChecked
                    className="size-4 rounded border-border accent-primary"
                  />
                  {permission}
                </label>
              ))}
            </CardContent>
          </Card>
        </div>

        <div className="flex flex-col gap-4 xl:col-span-4">
          <Card>
            <CardHeader>
              <CardTitle>Ringkasan instalasi</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              {summaryRows.map((row) => (
                <div key={row.label} className="flex items-center justify-between gap-2">
                  <span className="text-sm text-muted-foreground">{row.label}</span>
                  <span className="text-sm font-medium">{row.value}</span>
                </div>
              ))}
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm text-muted-foreground">Status</span>
                <Badge variant="secondary" className="h-5 px-2">
                  Draft
                </Badge>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Info className="size-4" aria-hidden="true" />
                Alur instalasi
              </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <p className="text-sm text-muted-foreground">
                Modul berstatus Draft setelah dipasang. Aktifkan pada Daftar Modul agar
                tersedia untuk 18 pengguna.
              </p>
              <Alert className="border-border bg-muted/40 py-2">
                <Info aria-hidden="true" />
                <AlertDescription>
                  Unggahan tidak diproses sampai module registry tersedia.
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
