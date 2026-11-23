"use client";

import { Activity, CircleAlert } from "lucide-react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  XAxis,
  YAxis,
} from "recharts";

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
import { formatNumber } from "@/lib/time";

export type ActivityDatum = {
  label: string;
  created: number;
  completed: number;
};

export type BucketDatum = {
  label: string;
  value: number;
  fill?: string;
};

const activityConfig = {
  created: { label: "Dibuat", color: "var(--foreground)" },
  completed: {
    label: "Selesai",
    color: "color-mix(in srgb, var(--foreground) 55%, transparent)",
  },
} satisfies ChartConfig;

export function MaintenanceActivityChart({
  data,
  loading,
  failed,
}: {
  data: ActivityDatum[];
  loading: boolean;
  failed: boolean;
}) {
  if (loading) {
    return <Skeleton className="h-[260px] w-full" />;
  }

  if (failed) {
    return (
      <ChartEmpty
        title="Aktivitas tidak tersedia"
        description="Data maintenance gagal dimuat. Coba muat ulang dashboard."
        icon={CircleAlert}
      />
    );
  }

  if (data.length === 0) {
    return (
      <ChartEmpty
        title="Belum ada aktivitas"
        description="Work order yang dibuat pada periode ini akan muncul di sini."
        icon={Activity}
      />
    );
  }

  return (
    <ChartContainer config={activityConfig} className="h-[260px] w-full">
      <BarChart data={data} margin={{ left: 0, right: 12, top: 8, bottom: 0 }}>
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
          dataKey="created"
          stackId="activity"
          fill="var(--color-created)"
          maxBarSize={48}
          isAnimationActive
          animationDuration={700}
          animationEasing="ease-out"
        />
        <Bar
          dataKey="completed"
          stackId="activity"
          fill="var(--color-completed)"
          radius={[2, 2, 0, 0]}
          maxBarSize={48}
          isAnimationActive
          animationBegin={100}
          animationDuration={700}
          animationEasing="ease-out"
        />
      </BarChart>
    </ChartContainer>
  );
}

export function BucketBarChart({
  data,
  loading,
  failed,
  valueLabel,
  formatValue,
  emptyTitle,
  emptyDescription,
}: {
  data: BucketDatum[];
  loading: boolean;
  failed: boolean;
  valueLabel: string;
  formatValue?: (value: number) => string;
  emptyTitle: string;
  emptyDescription: string;
}) {
  const config = {
    value: { label: valueLabel, color: "var(--chart-1)" },
  } satisfies ChartConfig;

  if (loading) {
    return <Skeleton className="h-[220px] w-full" />;
  }

  if (failed) {
    return (
      <ChartEmpty
        title="Data tidak tersedia"
        description="Ringkasan gagal dimuat. Coba muat ulang dashboard."
        icon={CircleAlert}
      />
    );
  }

  if (data.length === 0 || data.every((item) => item.value <= 0)) {
    return (
      <ChartEmpty
        title={emptyTitle}
        description={emptyDescription}
        icon={Activity}
      />
    );
  }

  return (
    <ChartContainer config={config} className="h-[220px] w-full">
      <BarChart
        data={data}
        layout="vertical"
        margin={{ left: 0, right: 8, top: 4, bottom: 4 }}
      >
        <CartesianGrid horizontal={false} />
        <XAxis type="number" hide />
        <YAxis
          type="category"
          dataKey="label"
          width={96}
          tickLine={false}
          axisLine={false}
          tick={{ fontSize: 12 }}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              formatter={(value) =>
                formatValue
                  ? formatValue(Number(value))
                  : formatNumber(Number(value))
              }
            />
          }
          cursor={{ fill: "var(--muted)" }}
        />
        <Bar dataKey="value" barSize={16} radius={[0, 4, 4, 0]}>
          {data.map((item) => (
            <Cell key={item.label} fill={item.fill ?? "var(--color-value)"} />
          ))}
        </Bar>
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
