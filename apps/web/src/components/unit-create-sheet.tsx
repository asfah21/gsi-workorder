"use client";

import { Loader2 } from "lucide-react";

import { UnitForm } from "@/components/unit-form";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { Unit } from "@/lib/types";

type UnitCreateSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: (unit: Unit) => void;
};

export function UnitCreateSheet({
  open,
  onOpenChange,
  onSaved,
}: UnitCreateSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>Tambah unit</SheetTitle>
          <SheetDescription>
            Lengkapi identitas unit. Data tersimpan ke master unit setelah tombol Buat unit ditekan.
          </SheetDescription>
        </SheetHeader>
        <UnitForm
          idPrefix="unit-create-sheet"
          onCancel={() => onOpenChange(false)}
          onSuccess={(unit) => {
            onSaved(unit);
            onOpenChange(false);
          }}
          footer={({ pending }) => (
            <SheetFooter className="px-0 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => onOpenChange(false)}
                disabled={pending}
              >
                Batal
              </Button>
              <Button type="submit" size="sm" disabled={pending}>
                {pending ? (
                  <>
                    <Loader2 className="animate-spin" aria-hidden="true" />
                    Menyimpan...
                  </>
                ) : (
                  "Buat unit"
                )}
              </Button>
            </SheetFooter>
          )}
        />
      </SheetContent>
    </Sheet>
  );
}
