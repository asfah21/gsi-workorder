"use client";

import { CircleAlert, Truck } from "lucide-react";
import { Cell, Pie, PieChart } from "recharts";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";

const chartConfig = {
  healthy: { label: "Aktif sehat", color: "var(--chart-1)" },
  breakdown: { label: "Breakdown", color: "var(--chart-3)" },
  inactive: { label: "Nonaktif", color: "var(--chart-4)" },
} satisfies ChartConfig;

export function FleetStatusChart({
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
      <div className="space-y-4" role="status" aria-label="Memuat status Unit">
        <Skeleton className="mx-auto size-44 rounded-full" />
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
          <EmptyTitle>Status Unit tidak tersedia</EmptyTitle>
          <EmptyDescription>
            Ringkasan Unit gagal dimuat. Coba muat ulang ringkasan.
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
            Chart status Unit muncul setelah unit ditambahkan ke master.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  const breakdown = Math.max(0, unitsBreakdown);
  const healthy = Math.max(0, unitsActive - breakdown);
  const inactive = Math.max(0, unitsTotal - unitsActive);
  const data = [
    { status: "healthy" as const, label: "Aktif sehat", value: healthy },
    { status: "breakdown" as const, label: "Breakdown", value: breakdown },
    { status: "inactive" as const, label: "Nonaktif", value: inactive },
  ];
  const summary = `Status Unit dari ${unitsTotal} unit: ${healthy} aktif sehat, ${breakdown} breakdown, ${inactive} nonaktif.`;

  return (
    <div className="space-y-4">
      <p className="sr-only">{summary}</p>
      <div className="relative">
        <ChartContainer
          config={chartConfig}
          aria-hidden="true"
          className="mx-auto aspect-square h-56"
        >
          <PieChart>
            <ChartTooltip
              content={<ChartTooltipContent nameKey="status" hideLabel />}
            />
            <Pie
              data={data}
              dataKey="value"
              nameKey="status"
              innerRadius="62%"
              outerRadius="88%"
              paddingAngle={2}
              cornerRadius={4}
              stroke="var(--background)"
              strokeWidth={2}
              isAnimationActive={false}
            >
              {data.map((item) => (
                <Cell
                  key={item.status}
                  fill={`var(--color-${item.status})`}
                />
              ))}
            </Pie>
          </PieChart>
        </ChartContainer>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-2xl font-semibold tabular-nums">
            {unitsTotal}
          </span>
          <span className="text-[0.65rem] font-medium uppercase tracking-wide text-muted-foreground">
            Total unit
          </span>
        </div>
      </div>

      <ul className="grid gap-2">
        {data.map((item) => (
          <li
            key={item.status}
            className="flex items-center justify-between gap-3 text-sm"
          >
            <span className="flex items-center gap-2 text-muted-foreground">
              <span
                className="size-2.5 rounded-full"
                style={{ backgroundColor: chartConfig[item.status].color }}
                aria-hidden="true"
              />
              {item.label}
            </span>
            <span className="font-medium tabular-nums">{item.value}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
