"use client";

import { Activity, CircleAlert } from "lucide-react";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";

export type InventoryMovementDatum = {
  label: string;
  masuk: number;
  keluar: number;
};

const config = {
  masuk: { label: "Masuk", color: "var(--chart-1)" },
  keluar: { label: "Keluar", color: "var(--chart-2)" },
} satisfies ChartConfig;

export function InventoryMovementChart({
  data,
  loading,
  failed,
}: {
  data: InventoryMovementDatum[];
  loading: boolean;
  failed: boolean;
}) {
  if (loading) {
    return <Skeleton className="h-[220px] w-full" />;
  }

  if (failed) {
    return (
      <ChartEmpty
        title="Grafik tidak tersedia"
        description="Data pergerakan gagal dimuat. Coba muat ulang dashboard."
        icon={CircleAlert}
      />
    );
  }

  if (data.length === 0 || data.every((item) => item.masuk === 0 && item.keluar === 0)) {
    return (
      <ChartEmpty
        title="Belum ada pergerakan"
        description="Transaksi stok yang diposting pada 7 hari terakhir akan muncul di sini."
        icon={Activity}
      />
    );
  }

  return (
    <ChartContainer config={config} className="h-[220px] w-full">
      <BarChart data={data} margin={{ left: 0, right: 8, top: 8, bottom: 0 }}>
        <CartesianGrid
          vertical={false}
          stroke="var(--border)"
          strokeDasharray="4 4"
        />
        <XAxis
          dataKey="label"
          tickLine={false}
          axisLine={false}
          tickMargin={10}
          tick={{ fontSize: 12, fill: "var(--muted-foreground)" }}
        />
        <YAxis
          width={32}
          allowDecimals={false}
          tickLine={false}
          axisLine={false}
          tickMargin={4}
          tick={{ fontSize: 12, fill: "var(--muted-foreground)" }}
        />
        <ChartTooltip
          content={<ChartTooltipContent indicator="line" />}
          cursor={{ fill: "var(--muted)" }}
        />
        <Bar
          dataKey="masuk"
          fill="var(--color-masuk)"
          radius={[2, 2, 0, 0]}
          maxBarSize={20}
          isAnimationActive
          animationDuration={700}
          animationEasing="ease-out"
        />
        <Bar
          dataKey="keluar"
          fill="var(--color-keluar)"
          radius={[2, 2, 0, 0]}
          maxBarSize={20}
          isAnimationActive
          animationBegin={100}
          animationDuration={700}
          animationEasing="ease-out"
        />
      </BarChart>
    </ChartContainer>
  );
}

function ChartEmpty({
  title,
  description,
  icon: Icon,
}: {
  title: string;
  description: string;
  icon: typeof Activity;
}) {
  return (
    <Empty className="border-0 py-8">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Icon aria-hidden="true" />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
