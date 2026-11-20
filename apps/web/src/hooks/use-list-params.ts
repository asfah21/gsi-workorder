"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef } from "react";

export const PAGE_SIZE_OPTIONS = [10, 25, 50, 100] as const;
export type ListOrder = "asc" | "desc";

export function useListParams({
  prefix = "",
  defaultSort,
  allowedSorts,
  defaultPageSize = 25,
}: {
  prefix?: string;
  defaultSort: string;
  allowedSorts: readonly string[];
  defaultPageSize?: number;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const pendingSort = useRef<string | null>(null);

  const key = (name: string) =>
    prefix ? `${prefix}${name[0].toUpperCase()}${name.slice(1)}` : name;
  const pageKey = key("page");
  const pageSizeKey = key("pageSize");
  const sortKey = key("sort");
  const orderKey = key("order");

  const page = readPositiveInt(searchParams.get(pageKey), 1);
  const pageSize = readPageSize(searchParams.get(pageSizeKey), defaultPageSize);
  const sort = allowedSorts.includes(searchParams.get(sortKey) ?? "")
    ? (searchParams.get(sortKey) as string)
    : defaultSort;
  const order: ListOrder = searchParams.get(orderKey) === "desc" ? "desc" : "asc";

  const update = (
    updates: Record<string, string | number | null | undefined>,
  ) => {
    const next = new URLSearchParams(searchParams.toString());
    for (const [name, value] of Object.entries(updates)) {
      if (value === null || value === undefined || value === "") {
        next.delete(name);
      } else {
        next.set(name, String(value));
      }
    }
    const query = next.toString();
    router.replace(query ? `${pathname}?${query}` : pathname, { scroll: false });
  };

  const setPage = (nextPage: number) => {
    update({ [pageKey]: nextPage <= 1 ? null : nextPage });
  };

  const setPageSize = (nextPageSize: number) => {
    update({
      [pageSizeKey]:
        nextPageSize === defaultPageSize ? null : normalizePageSize(nextPageSize, defaultPageSize),
      [pageKey]: null,
    });
  };

  const toggleSort = (nextSort: string) => {
    pendingSort.current = nextSort;
    const nextOrder: ListOrder =
      sort === nextSort && order === "asc" ? "desc" : "asc";
    update({
      [sortKey]: nextSort,
      [orderKey]: nextOrder,
      [pageKey]: null,
    });
  };

  const setFilter = (name: string, value: string | null) => {
    update({ [name]: value, [pageKey]: null });
  };

  const setFilters = (updates: Record<string, string | null>) => {
    update({ ...updates, [pageKey]: null });
  };

  useEffect(() => {
    const nextSort = pendingSort.current;
    if (!nextSort) return;
    pendingSort.current = null;
    const selector = `[data-sort-button="${prefix ? `${prefix}:` : ""}${nextSort}"]`;
    requestAnimationFrame(() => {
      document.querySelector<HTMLButtonElement>(selector)?.focus();
    });
  }, [order, prefix, sort]);

  return {
    page,
    pageSize,
    sort,
    order,
    pageKey,
    pageSizeKey,
    sortKey,
    orderKey,
    setPage,
    setPageSize,
    toggleSort,
    setFilter,
    setFilters,
  };
}

function readPositiveInt(raw: string | null, fallback: number): number {
  const value = Number(raw);
  return Number.isInteger(value) && value > 0 ? value : fallback;
}

function normalizePageSize(raw: number, fallback: number): number {
  return (PAGE_SIZE_OPTIONS as readonly number[]).includes(raw) ? raw : fallback;
}

function readPageSize(raw: string | null, fallback: number): number {
  const value = Number(raw);
  return normalizePageSize(value, fallback);
}
