import type { LucideIcon } from "lucide-react";
import Link from "next/link";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatNumber } from "@/lib/time";
import { cn } from "@/lib/utils";

export function KpiCard({
  label,
  value,
  hint,
  icon: Icon,
  href,
  loading = false,
  formatValue,
  className,
}: {
  label: string;
  value?: number;
  hint?: string;
  icon: LucideIcon;
  href?: string;
  loading?: boolean;
  formatValue?: (value: number) => string;
  className?: string;
}) {
  const card = (
    <Card
      className={cn(
        "h-full",
        href && "transition-colors hover:border-foreground/20",
        className,
      )}
    >
      <CardHeader className="flex flex-row items-center justify-between gap-3 pb-0">
        <CardTitle className="text-sm font-medium">{label}</CardTitle>
        <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <p className="text-2xl font-semibold">
            {typeof value === "number"
              ? (formatValue ?? formatNumber)(value)
              : "-"}
          </p>
        )}
        {hint ? (
          <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
        ) : null}
      </CardContent>
    </Card>
  );

  if (!href) {
    return card;
  }

  return (
    <Link
      href={href}
      className="block rounded-xl focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
    >
      {card}
    </Link>
  );
}
