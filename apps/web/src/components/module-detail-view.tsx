"use client";

import {
  Activity,
  ArrowLeftRight,
  Coins,
  Info,
  LayoutGrid,
  ShoppingCart,
  Users,
  Warehouse,
} from "lucide-react";
import Link from "next/link";

import { PageHeader } from "@/components/layout/page-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

const moduleInfo = {
  name: "Inventory",
  description:
    "Gudang & Persediaan - Kelola stok, lokasi gudang, dan pergerakan barang masuk-keluar.",
  status: "Terpasang",
};

const moduleMeta = [
  { label: "Versi", value: "v2.3.1" },
  { label: "Penerbit", value: "Sanggar Labs" },
  { label: "Terakhir diperbarui", value: "12 Nov 2026" },
  { label: "Ukuran", value: "2,1 MB" },
];

const moduleMetrics = [
  { label: "Pengguna aktif", value: "18", icon: Users },
  { label: "Transaksi / bulan", value: "2.418", icon: ArrowLeftRight },
  { label: "Waktu aktif 30 hari", value: "99,8%", icon: Activity },
];

const versionHistory = [
  { version: "v2.3.1", date: "12 Nov", note: "Perbaikan transfer antar gudang" },
  { version: "v2.3.0", date: "28 Okt", note: "Kolom harga jual pada daftar produk" },
  { version: "v2.2.4", date: "9 Okt", note: "Ekspor stok ke CSV" },
];

const dependencies = [
  { name: "Finance", icon: Coins, status: "Terpasang" },
  { name: "Sales", icon: ShoppingCart, status: "Terpasang" },
];

export function ModuleDetailView() {
  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        breadcrumb={
          <>
            <Link href="/dashboard">Beranda</Link>
            <span aria-hidden="true">/</span>
            <Link href="/modules">Modul</Link>
            <span aria-hidden="true">/</span>
            <span className="text-foreground">{moduleInfo.name}</span>
          </>
        }
        title={moduleInfo.name}
        description={moduleInfo.description}
        actions={
          <Link href="/modules/new" className={cn(buttonVariants({ variant: "outline" }))}>
            <LayoutGrid aria-hidden="true" /> Tambah modul
          </Link>
        }
      />

      <Alert className="border-border bg-muted/40">
        <Info aria-hidden="true" />
        <AlertTitle>Data contoh dari template Figma</AlertTitle>
        <AlertDescription>
          Detail modul ini menampilkan konten statis dari desain. Belum ada module registry
          atau API, sehingga tombol aksi tidak melakukan perubahan.
        </AlertDescription>
      </Alert>

      <div className="grid gap-4 xl:grid-cols-12">
        <div className="flex flex-col gap-4 xl:col-span-8">
          <Card>
            <CardContent className="flex flex-col gap-4 px-4 py-4 sm:flex-row sm:items-start">
              <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted">
                <Warehouse className="size-4" aria-hidden="true" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-2xl font-semibold tracking-tight">
                    {moduleInfo.name}
                  </h2>
                  <InstalledBadge status={moduleInfo.status} />
                </div>
                <p className="mt-1 text-sm text-muted-foreground">
                  {moduleInfo.description}
                </p>
              </div>
            </CardContent>
            <CardContent className="px-4 pb-4">
              <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                {moduleMeta.map((item) => (
                  <div key={item.label} className="flex flex-col gap-1">
                    <dt className="text-sm text-muted-foreground">{item.label}</dt>
                    <dd className="text-sm font-medium">{item.value}</dd>
                  </div>
                ))}
              </dl>
            </CardContent>
          </Card>

          <div className="grid gap-3 sm:grid-cols-3">
            {moduleMetrics.map((item) => {
              const Icon = item.icon;
              return (
                <Card key={item.label} className="gap-2 py-4">
                  <CardHeader className="flex flex-row items-center justify-between px-4">
                    <CardTitle className="text-sm font-medium">{item.label}</CardTitle>
                    <Icon className="size-4 text-muted-foreground" aria-hidden="true" />
                  </CardHeader>
                  <CardContent className="px-4">
                    <p className="text-2xl font-semibold">{item.value}</p>
                  </CardContent>
                </Card>
              );
            })}
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Riwayat versi</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Versi</TableHead>
                      <TableHead>Tanggal</TableHead>
                      <TableHead>Catatan</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {versionHistory.map((item) => (
                      <TableRow key={item.version}>
                        <TableCell className="font-mono text-xs">{item.version}</TableCell>
                        <TableCell className="text-sm text-muted-foreground">
                          {item.date}
                        </TableCell>
                        <TableCell className="text-sm">{item.note}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="flex flex-col gap-4 xl:col-span-4">
          <Card>
            <CardHeader>
              <CardTitle>Dependensi</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-2">
              {dependencies.map((item) => {
                const Icon = item.icon;
                return (
                  <div key={item.name} className="flex items-center gap-2">
                    <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted">
                      <Icon className="size-4" aria-hidden="true" />
                    </span>
                    <span className="text-sm font-medium">{item.name}</span>
                    <span className="h-px flex-1 bg-border" aria-hidden="true" />
                    <InstalledBadge status={item.status} />
                  </div>
                );
              })}
              <p className="text-xs text-muted-foreground">
                Semua dependensi terpenuhi - modul siap diperbarui.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Aksi</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col items-start gap-2">
              <Button type="button" disabled title="Membutuhkan module registry/API">
                Perbarui modul
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled
                title="Membutuhkan module registry/API"
              >
                Nonaktifkan
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled
                title="Membutuhkan module registry/API"
              >
                Hapus modul
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

function InstalledBadge({ status }: { status: string }) {
  return (
    <Badge
      variant="secondary"
      className="h-5 gap-1.5 border-transparent bg-emerald-500/10 px-2 text-emerald-700 dark:text-emerald-300"
    >
      <span className="size-1.5 rounded-full bg-current" aria-hidden="true" />
      {status}
    </Badge>
  );
}
