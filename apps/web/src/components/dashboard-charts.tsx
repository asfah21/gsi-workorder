"use client";

import { Activity, CircleAlert } from "lucide-react";
import {
  Area,
  AreaChart,
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
  created: { label: "WO dibuat", color: "var(--chart-1)" },
  completed: { label: "WO selesai", color: "var(--chart-2)" },
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
      <AreaChart data={data} margin={{ left: 0, right: 12, top: 8, bottom: 0 }}>
        <defs>
          <linearGradient id="activity-created" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="var(--color-created)" stopOpacity={0.35} />
            <stop offset="95%" stopColor="var(--color-created)" stopOpacity={0.02} />
          </linearGradient>
          <linearGradient id="activity-completed" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="var(--color-completed)" stopOpacity={0.3} />
            <stop offset="95%" stopColor="var(--color-completed)" stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey="label"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          tick={{ fontSize: 12 }}
        />
        <YAxis
          width={32}
          allowDecimals={false}
          tickLine={false}
          axisLine={false}
          tick={{ fontSize: 12 }}
        />
        <ChartTooltip
          content={<ChartTooltipContent indicator="line" />}
          cursor={{ stroke: "var(--border)" }}
        />
        <Area
          type="monotone"
          dataKey="created"
          stroke="var(--color-created)"
          strokeWidth={2}
          fill="url(#activity-created)"
          isAnimationActive={false}
        />
        <Area
          type="monotone"
          dataKey="completed"
          stroke="var(--color-completed)"
          strokeWidth={2}
          fill="url(#activity-completed)"
          isAnimationActive={false}
        />
      </AreaChart>
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
