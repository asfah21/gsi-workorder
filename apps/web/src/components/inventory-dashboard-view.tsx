"use client";

import {
  ArrowDownToLine,
  ArrowUpRight,
  Boxes,
  CircleAlert,
  Download,
  PackageSearch,
  RefreshCw,
  ShieldCheck,
  TriangleAlert,
} from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useSession } from "@/components/app-shell";
import {
  InventoryMovementChart,
  type InventoryMovementDatum,
} from "@/components/inventory-movement-chart";
import { PageHeader } from "@/components/layout/page-header";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError, proxyGet } from "@/lib/api-client";
import { daysAgoWitaDate, formatNumber, formatWita } from "@/lib/time";
import type {
  DashboardInventory,
  DataResponse,
  InventoryReservation,
  ListResponse,
  Sparepart,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const DAY_LABELS = ["Min", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"];
const RECENT_SIZE = 6;
const CHART_SIZE = 1000;

const movementTypeLabels: Record<string, string> = {
  receipt: "Penerimaan",
  issue: "Pengeluaran",
  return: "Pengembalian",
  transfer: "Transfer",
  adjustment: "Adjustment",
  opname: "Stock Opname",
};

function shortDayLabel(date: string) {
  const [year, month, day] = date.split("-").map(Number);
  return DAY_LABELS[new Date(Date.UTC(year, month - 1, day)).getUTCDay()];
}

export function InventoryDashboardView() {
  const { user } = useSession();
  const [dashboard, setDashboard] = useState<DashboardInventory | null>(null);
  const [skuTotal, setSkuTotal] = useState(0);
  const [reservationTotal, setReservationTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  const from = useMemo(() => daysAgoWitaDate(6), []);
  const to = useMemo(() => daysAgoWitaDate(0), []);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = new URLSearchParams({
        bucket: "day",
        from,
        to,
        status: "posted",
        limit: String(CHART_SIZE),
      });
      const [dashboardResult, sparepartResult, reservationResult] = await Promise.all([
        proxyGet<DataResponse<DashboardInventory>>(`/dashboard/inventory?${params}`),
        proxyGet<ListResponse<Sparepart>>("/spareparts?page=1&limit=1&active_only=true"),
        proxyGet<ListResponse<InventoryReservation>>(
          "/inventory/reservations?status=active&page=1&limit=1",
        ),
      ]);
      setDashboard(dashboardResult.data);
      setSkuTotal(sparepartResult.meta.total);
      setReservationTotal(reservationResult.meta.total);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError && (loadError.status === 401 || loadError.status === 403)
          ? "Akses ditolak. Hubungi administrator untuk membuka dashboard inventory."
          : loadError instanceof ApiError
            ? loadError.message
            : "Dashboard inventory tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [from, to]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  const downloadCSV = useCallback(async () => {
    setDownloading(true);
    try {
      const params = new URLSearchParams({
        bucket: "day",
        from,
        to,
        status: "posted",
        format: "csv",
      });
      const response = await fetch(`/api/proxy/reports/inventory-movement?${params}`, {
        credentials: "same-origin",
        cache: "no-store",
        headers: { Accept: "text/csv" },
      });
      if (!response.ok) {
        throw new Error("CSV tidak dapat dibuat.");
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "inventory-movement.csv";
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    } catch (downloadError) {
      setError(
        downloadError instanceof Error
          ? downloadError.message
          : "CSV tidak dapat dibuat.",
      );
    } finally {
      setDownloading(false);
    }
  }, [from, to]);

  const summary: Record<string, number> = useMemo(
    () => dashboard?.summary ?? {},
    [dashboard],
  );
  const lowStockRows = useMemo(() => dashboard?.low_stock.rows ?? [], [dashboard]);
  const movementRows = useMemo(() => dashboard?.movements.rows ?? [], [dashboard]);

  const chartData = useMemo<InventoryMovementDatum[]>(() => {
    const buckets = Array.from({ length: 7 }, (_, index) => daysAgoWitaDate(6 - index));
    return buckets.map((date) => {
      const rows = movementRows.filter((row) => String(row.bucket) === date);
      const masuk = rows
        .filter((row) => row.type === "receipt" || row.type === "return")
        .reduce((total, row) => total + Number(row.qty ?? 0), 0);
      const keluar = rows
        .filter((row) => row.type === "issue")
        .reduce((total, row) => total + Number(row.qty ?? 0), 0);
      return { label: shortDayLabel(date), masuk, keluar };
    });
  }, [movementRows]);

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        breadcrumb={
          <>
            <Link href="/dashboard">Beranda</Link>
            <span aria-hidden="true">/</span>
            <Link href="/inventory">Inventory</Link>
            <span aria-hidden="true">/</span>
            <span className="text-foreground">Dashboard</span>
          </>
        }
        title="Dashboard Inventory"
        description="Ringkasan stok, pergerakan 7 hari, dan kondisi sparepart kritis."
        actions={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => void downloadCSV()}
              disabled={downloading}
            >
              <Download aria-hidden="true" />
              {downloading ? "Mengunduh..." : "Unduh laporan"}
            </Button>
            <Link
              href="/spareparts"
              className={cn(buttonVariants({ variant: "default" }))}
            >
              <PackageSearch aria-hidden="true" />
              Master sparepart
            </Link>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={() => setRefreshKey((current) => current + 1)}
              disabled={loading}
              aria-label="Muat ulang dashboard inventory"
              title="Muat ulang dashboard inventory"
            >
              <RefreshCw className={loading ? "animate-spin" : undefined} aria-hidden="true" />
            </Button>
          </>
        }
      />

      {error ? (
        <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Data tidak dapat dimuat</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Total SKU aktif"
          value={skuTotal}
          loading={loading}
          icon={Boxes}
          hint="Sparepart aktif terdaftar"
        />
        <MetricCard
          label="Pergerakan 7 hari"
          value={Number(summary.movements ?? 0)}
          loading={loading}
          icon={ArrowUpRight}
          hint="Transaksi stok diposting"
        />
        <MetricCard
          label="Stok menipis"
          value={Number(summary.low ?? 0)}
          loading={loading}
          icon={ArrowDownToLine}
          hint={`${formatNumber(Number(summary.critical ?? 0))} kritis`}
          danger
        />
        <MetricCard
          label="Reservasi aktif"
          value={reservationTotal}
          loading={loading}
          icon={ShieldCheck}
          hint={`${user.role === "admin" ? "Admin" : "Pengguna"} · inventory`}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-12">
        <Card className="xl:col-span-7">
          <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle>Pergerakan stok - 7 hari</CardTitle>
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1.5">
                <span className="size-2 rounded-sm bg-[var(--chart-1)]" aria-hidden="true" />
                Masuk
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="size-2 rounded-sm bg-[var(--chart-2)]" aria-hidden="true" />
                Keluar
              </span>
            </div>
          </CardHeader>
          <CardContent>
            <InventoryMovementChart
              data={chartData}
              loading={loading}
              failed={Boolean(error)}
            />
          </CardContent>
        </Card>

        <Card className="xl:col-span-5">
          <CardHeader>
            <CardTitle>Stok kritis & menipis</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            {loading ? (
              <div className="flex flex-col gap-2">
                {Array.from({ length: 4 }).map((_, index) => (
                  <Skeleton key={index} className="h-11 w-full" />
                ))}
              </div>
            ) : lowStockRows.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                Tidak ada sparepart kritis atau menipis pada periode ini.
              </p>
            ) : (
              lowStockRows.slice(0, 4).map((row, index) => {
                const critical = row.stock_status === "critical";
                return (
                  <div
                    key={`${String(row.sparepart_code)}-${String(row.location_code)}-${index}`}
                    className="flex items-center gap-2"
                  >
                    <span
                      className={cn(
                        "flex size-7 shrink-0 items-center justify-center rounded-md",
                        critical
                          ? "bg-destructive/10 text-destructive"
                          : "bg-amber-500/10 text-amber-700 dark:text-amber-300",
                      )}
                    >
                      <TriangleAlert className="size-4" aria-hidden="true" />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {String(row.sparepart_name ?? "-")}
                      </p>
                      <p className="font-mono text-xs text-muted-foreground">
                        {formatNumber(Number(row.on_hand ?? 0))} dari min.{" "}
                        {formatNumber(Number(row.min_qty ?? 0))}
                      </p>
                    </div>
                    <Badge
                      variant={critical ? "destructive" : "outline"}
                      className="h-5 px-2"
                    >
                      {critical ? "Kritis" : "Menipis"}
                    </Badge>
                    <Link
                      href="/inventory"
                      className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
                    >
                      Lihat stok
                    </Link>
                  </div>
                );
              })
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <CardTitle>Pergerakan stok terbaru</CardTitle>
          <Link href="/inventory" className={cn(buttonVariants({ variant: "ghost", size: "sm" }))}>
            Semua pergerakan <ArrowUpRight aria-hidden="true" />
          </Link>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Waktu</TableHead>
                  <TableHead>Produk</TableHead>
                  <TableHead>Tipe</TableHead>
                  <TableHead className="text-right">Qty</TableHead>
                  <TableHead>Lokasi</TableHead>
                  <TableHead>Referensi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  Array.from({ length: 4 }).map((_, index) => (
                    <TableRow key={index}>
                      <TableCell colSpan={6}>
                        <Skeleton className="h-5 w-full" />
                      </TableCell>
                    </TableRow>
                  ))
                ) : movementRows.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-sm text-muted-foreground">
                      Belum ada pergerakan stok yang diposting pada 7 hari terakhir.
                    </TableCell>
                  </TableRow>
                ) : (
                  movementRows.slice(0, RECENT_SIZE).map((row, index) => {
                    const type = String(row.type ?? "");
                    return (
                      <TableRow key={`${String(row.transaction_code)}-${index}`}>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {formatWita(String(row.posted_at ?? ""), {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </TableCell>
                        <TableCell className="text-sm">
                          {String(row.sparepart_name ?? "-")}
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={type === "issue" ? "outline" : "secondary"}
                            className="h-5 px-2"
                          >
                            {movementTypeLabels[type] ?? type}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right font-mono text-xs">
                          {formatNumber(Number(row.qty ?? 0))}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {String(row.location || row.from_location || row.to_location || "-")}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {String(row.reference_no || row.transaction_code || "-")}
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <p className="text-xs text-muted-foreground">
        Data dihitung dari transaksi inventory yang diposting; seluruh tanggal menggunakan
        batas hari WITA.
      </p>
    </div>
  );
}

function MetricCard({
  label,
  value,
  hint,
  loading,
  icon: Icon,
  danger,
}: {
  label: string;
  value: number;
  hint: string;
  loading: boolean;
  icon: typeof Boxes;
  danger?: boolean;
}) {
  return (
    <Card className="gap-2 py-4">
      <CardHeader className="flex flex-row items-center justify-between px-4">
        <CardTitle className="text-sm font-medium">{label}</CardTitle>
        <Icon
          className={cn(
            "size-4",
            danger ? "text-destructive" : "text-muted-foreground",
          )}
          aria-hidden="true"
        />
      </CardHeader>
      <CardContent className="px-4">
        <p className="text-2xl font-semibold">
          {loading ? "..." : formatNumber(value)}
        </p>
        <p className={cn("text-xs", danger ? "text-destructive" : "text-muted-foreground")}>
          {hint}
        </p>
      </CardContent>
    </Card>
  );
}
