"use client";

import { ArrowDown, ArrowUp, ChevronsUpDown } from "lucide-react";

import { TableHead } from "@/components/ui/table";
import type { ListOrder } from "@/hooks/use-list-params";
import { cn } from "@/lib/utils";

export function TableSortHeader({
  label,
  sortKey,
  prefix = "",
  currentSort,
  currentOrder,
  onSort,
  align = "start",
}: {
  label: string;
  sortKey: string;
  prefix?: string;
  currentSort: string;
  currentOrder: ListOrder;
  onSort: (key: string) => void;
  align?: "start" | "end";
}) {
  const active = currentSort === sortKey;
  const ariaSort = active
    ? currentOrder === "asc"
      ? "ascending"
      : "descending"
    : undefined;
  const Icon = active ? (currentOrder === "asc" ? ArrowUp : ArrowDown) : ChevronsUpDown;

  return (
    <TableHead scope="col" aria-sort={ariaSort}>
      <button
        type="button"
        data-sort-button={prefix ? `${prefix}:${sortKey}` : sortKey}
        onClick={() => onSort(sortKey)}
        className={cn(
          "inline-flex min-h-11 w-full items-center gap-1.5 text-left font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring",
          align === "end" ? "justify-end" : "justify-start",
        )}
      >
        {label}
        <Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
      </button>
    </TableHead>
  );
}
