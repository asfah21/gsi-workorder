"use client";

import { CircleAlert, Repeat2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError, proxyGet } from "@/lib/api-client";
import type { DataResponse, UnitComponentInterval } from "@/lib/types";
import { formatDateTimeWita, formatNumber } from "@/lib/time";

export function UnitIntervalPanel({ unitId }: { unitId: number }) {
  const [items, setItems] = useState<UnitComponentInterval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const result = await proxyGet<DataResponse<UnitComponentInterval[]>>(
        `/units/${unitId}/component-intervals`,
      );
      setItems(result.data);
    } catch (loadError) {
      setError(
        loadError instanceof ApiError
          ? loadError.message
          : "Interval komponen tidak dapat dimuat.",
      );
    } finally {
      setLoading(false);
    }
  }, [unitId]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <Card className="gap-0 overflow-hidden">
      <CardHeader className="border-b">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <CardTitle>Interval penggantian komponen</CardTitle>
            <p className="text-sm text-muted-foreground">
              Rata-rata jarak antar penggantian berdasarkan actual usage yang telah di-issue.
            </p>
          </div>
          <Badge variant="secondary" className="h-6 px-2.5">
            {items.length} komponen
          </Badge>
        </div>
      </CardHeader>

      {error ? (
        <CardContent className="py-6">
          <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Interval tidak dapat dimuat</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        </CardContent>
      ) : loading ? (
        <CardContent className="flex flex-col gap-2 py-6">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-12 w-full" />
          ))}
        </CardContent>
      ) : items.length === 0 ? (
        <CardContent className="py-10">
          <Empty className="border-0">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Repeat2 aria-hidden="true" />
              </EmptyMedia>
              <EmptyTitle>Belum ada interval komponen.</EmptyTitle>
              <EmptyDescription>
                Interval dihitung setelah ada minimal dua event penggantian.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      ) : (
        <div className="overflow-x-auto">
          <Table>
            <TableCaption className="sr-only">Interval penggantian komponen unit</TableCaption>
            <TableHeader>
              <TableRow>
                <TableHead>Komponen</TableHead>
                <TableHead>Penggantian</TableHead>
                <TableHead>Interval hari</TableHead>
                <TableHead>Interval HM</TableHead>
                <TableHead>Interval KM</TableHead>
                <TableHead>Total qty</TableHead>
                <TableHead>Terakhir</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item) => (
                <TableRow key={`${item.component_id ?? "s"}-${item.fallback_sparepart_id ?? ""}-${item.component_name}`}>
                  <TableCell className="min-w-56">
                    <div className="flex flex-wrap items-center gap-1">
                      <p className="font-medium">{item.component_name || "-"}</p>
                      {item.is_fallback ? <Badge variant="secondary">Sparepart</Badge> : null}
                    </div>
                    <p className="text-xs text-muted-foreground">{item.component_code || "-"}</p>
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm">
                    <p>{item.replacement_count} kali</p>
                    <p className="text-xs text-muted-foreground">
                      Sejak {formatDateTimeWita(item.first_replaced_at)}
                    </p>
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm">
                    {item.avg_interval_days !== undefined
                      ? `${formatNumber(item.avg_interval_days)} hari`
                      : "-"}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm">
                    {item.avg_interval_hm !== undefined
                      ? formatNumber(item.avg_interval_hm)
                      : "-"}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm">
                    {item.avg_interval_km !== undefined
                      ? formatNumber(item.avg_interval_km)
                      : "-"}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm">
                    {formatNumber(item.total_qty)}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-sm">
                    <p>{formatDateTimeWita(item.last_replaced_at)}</p>
                    <p className="text-xs text-muted-foreground">
                      HM {item.last_hm !== undefined ? formatNumber(item.last_hm) : "-"} · KM{" "}
                      {item.last_km !== undefined ? formatNumber(item.last_km) : "-"}
                    </p>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </Card>
  );
}
