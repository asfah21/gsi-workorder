"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useId } from "react";

import { Button } from "@/components/ui/button";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { PAGE_SIZE_OPTIONS } from "@/hooks/use-list-params";

export function TablePagination({
  page,
  pageSize,
  total,
  totalPages,
  loading,
  itemLabel,
  onPageChange,
  onPageSizeChange,
}: {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  loading: boolean;
  itemLabel: string;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}) {
  const selectId = useId();
  const safeTotalPages = Math.max(1, totalPages);

  useEffect(() => {
    if (!loading && total > 0 && page > safeTotalPages) {
      onPageChange(safeTotalPages);
    }
  }, [loading, onPageChange, page, safeTotalPages, total]);

  const start = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);
  const range =
    total === 0
      ? `Tidak ada ${itemLabel}`
      : `${start}–${end} dari ${total} ${itemLabel}`;

  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-muted-foreground" aria-live="polite">
        {range} · Halaman {page} dari {safeTotalPages}
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <label htmlFor={selectId} className="text-sm text-muted-foreground">
          Baris
        </label>
        <NativeSelect
          id={selectId}
          value={String(pageSize)}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
          disabled={loading}
          className="w-24"
        >
          {PAGE_SIZE_OPTIONS.map((size) => (
            <NativeSelectOption key={size} value={String(size)}>
              {size}
            </NativeSelectOption>
          ))}
        </NativeSelect>
        <Button
          type="button"
          variant="outline"
          disabled={loading || page <= 1}
          onClick={() => onPageChange(page - 1)}
          className="h-10 px-3"
          aria-label="Halaman sebelumnya"
        >
          <ChevronLeft aria-hidden="true" />
          Sebelumnya
        </Button>
        <Button
          type="button"
          variant="outline"
          disabled={loading || page >= safeTotalPages}
          onClick={() => onPageChange(page + 1)}
          className="h-10 px-3"
          aria-label="Halaman berikutnya"
        >
          Berikutnya
          <ChevronRight aria-hidden="true" />
        </Button>
      </div>
    </div>
  );
}
