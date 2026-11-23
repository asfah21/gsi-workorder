"use client";

import { CircleAlert, Truck } from "lucide-react";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { formatNumber } from "@/lib/time";

const TICKS = 64;

const chartConfig = {
  healthy: { label: "Aktif sehat", color: "var(--foreground)" },
  breakdown: { label: "Breakdown", color: "var(--chart-1)" },
  inactive: { label: "Nonaktif", color: "var(--muted-foreground)" },
} as const;

export function UnitStatusChart({
  unitsTotal,
  unitsActive,
  unitsBreakdown,
  loading = false,
  failed = false,
}: {
  unitsTotal: number;
  unitsActive: number;
  unitsBreakdown: number;
  loading?: boolean;
  failed?: boolean;
}) {
  if (loading) {
    return (
      <div className="space-y-4" role="status" aria-label="Memuat status unit">
        <Skeleton className="mx-auto size-52 rounded-full" />
        <div className="grid gap-2">
          <Skeleton className="h-5 w-full" />
          <Skeleton className="h-5 w-full" />
          <Skeleton className="h-5 w-full" />
        </div>
      </div>
    );
  }

  if (failed) {
    return (
      <Empty className="border-0 py-6">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <CircleAlert aria-hidden="true" />
          </EmptyMedia>
          <EmptyTitle>Status unit tidak tersedia</EmptyTitle>
          <EmptyDescription>
            Ringkasan unit gagal dimuat. Coba muat ulang dashboard.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  if (unitsTotal <= 0) {
    return (
      <Empty className="border-0 py-6">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Truck aria-hidden="true" />
          </EmptyMedia>
          <EmptyTitle>Belum ada unit</EmptyTitle>
          <EmptyDescription>
            Chart status unit muncul setelah unit ditambahkan ke master.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  const breakdown = Math.max(0, Math.min(unitsBreakdown, unitsTotal));
  const active = Math.max(0, Math.min(unitsActive, unitsTotal));
  const healthy = Math.max(0, active - breakdown);
  const inactive = Math.max(0, unitsTotal - active);

  const data = [
    {
      key: "healthy" as const,
      label: "Aktif sehat",
      value: healthy,
      color: chartConfig.healthy.color,
    },
    {
      key: "breakdown" as const,
      label: "Breakdown",
      value: breakdown,
      color: chartConfig.breakdown.color,
    },
    {
      key: "inactive" as const,
      label: "Nonaktif",
      value: inactive,
      color: chartConfig.inactive.color,
    },
  ];

  const healthyTicks = Math.min(
    TICKS,
    Math.round((healthy / unitsTotal) * TICKS),
  );
  const breakdownTicks = Math.min(
    TICKS - healthyTicks,
    Math.round((breakdown / unitsTotal) * TICKS),
  );
  const summary = `Status unit dari ${unitsTotal} unit: ${healthy} aktif sehat, ${breakdown} breakdown, ${inactive} nonaktif.`;

  return (
    <div className="flex flex-1 flex-col justify-between gap-6">
      <p className="sr-only">{summary}</p>
      <div className="relative mx-auto aspect-square w-full max-w-56">
        <svg
          viewBox="0 0 200 200"
          className="absolute inset-0 size-full"
          aria-hidden="true"
        >
          {Array.from({ length: TICKS }).map((_, index) => {
            const color =
              index < healthyTicks
                ? chartConfig.healthy.color
                : index < healthyTicks + breakdownTicks
                  ? chartConfig.breakdown.color
                  : chartConfig.inactive.color;
            const opacity =
              index < healthyTicks + breakdownTicks ? 1 : 0.35;
            return (
              <line
                key={index}
                x1="100"
                y1="8"
                x2="100"
                y2="18"
                stroke={color}
                strokeWidth="2.5"
                strokeLinecap="round"
                opacity={opacity}
                transform={`rotate(${index * (360 / TICKS)} 100 100)`}
              />
            );
          })}
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-1">
          <div className="mb-1 grid size-9 place-items-center rounded-md border border-border">
            <Truck className="size-4" aria-hidden="true" />
          </div>
          <span className="type-overline-xs text-muted-foreground">Total unit</span>
          <span className="text-2xl font-semibold tracking-tight">
            {formatNumber(unitsTotal)}
          </span>
        </div>
      </div>

      <ul className="grid gap-3">
        {data.map((item) => {
          const percentage =
            unitsTotal > 0 ? Math.round((item.value / unitsTotal) * 100) : 0;
          return (
            <li
              key={item.key}
              className="flex items-center justify-between gap-3 text-sm"
            >
              <span className="flex items-center gap-2 text-muted-foreground">
                <span
                  className="size-2.5 rounded-full"
                  style={{ backgroundColor: item.color }}
                  aria-hidden="true"
                />
                {item.label}
              </span>
              <span className="flex items-center gap-2">
                <span className="font-medium">{formatNumber(item.value)}</span>
                <span className="text-xs text-muted-foreground">
                  ({percentage}%)
                </span>
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
