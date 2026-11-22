"use client";

import { CircleAlert, Download, Loader2, Upload } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ApiError, proxyUpload } from "@/lib/api-client";
import type { DataResponse, UnitImportResult } from "@/lib/types";
import { downloadBlob } from "@/lib/download";

const MAX_FILE_BYTES = 5 * 1024 * 1024;

type UnitImportSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImported: (result: UnitImportResult) => void;
};

export function UnitImportSheet({
  open,
  onOpenChange,
  onImported,
}: UnitImportSheetProps) {
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState("");
  const [templateError, setTemplateError] = useState("");
  const [pending, setPending] = useState(false);
  const [downloading, setDownloading] = useState(false);

  const reset = () => {
    setFile(null);
    setError("");
    setTemplateError("");
  };

  const changeFile = (selected: File | null) => {
    setError("");
    if (!selected) {
      setFile(null);
      return;
    }
    if (!selected.name.toLowerCase().endsWith(".xlsx")) {
      setFile(null);
      setError("File harus berformat .xlsx.");
      return;
    }
    if (selected.size > MAX_FILE_BYTES) {
      setFile(null);
      setError("Ukuran file maksimal 5 MB.");
      return;
    }
    setFile(selected);
  };

  const downloadTemplate = async () => {
    setTemplateError("");
    setDownloading(true);
    try {
      const response = await fetch("/api/proxy/units/import-template", {
        credentials: "same-origin",
        cache: "no-store",
        headers: {
          Accept:
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        },
      });
      if (!response.ok) {
        throw new Error("Template tidak dapat diunduh.");
      }
      downloadBlob(await response.blob(), "template-import-unit.xlsx");
    } catch (downloadError) {
      setTemplateError(
        downloadError instanceof Error
          ? downloadError.message
          : "Template tidak dapat diunduh.",
      );
    } finally {
      setDownloading(false);
    }
  };

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!file) {
      setError("Pilih file XLSX terlebih dahulu.");
      return;
    }
    setPending(true);
    setError("");
    const formData = new FormData();
    formData.append("file", file);
    try {
      const response = await proxyUpload<DataResponse<UnitImportResult>>(
        "/units/import",
        formData,
      );
      onImported(response.data);
      reset();
      onOpenChange(false);
    } catch (uploadError) {
      if (uploadError instanceof ApiError) {
        setError(uploadError.fields.file || uploadError.message);
      } else {
        setError("File tidak dapat diimpor.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          reset();
        }
        onOpenChange(nextOpen);
      }}
    >
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Impor unit</SheetTitle>
          <SheetDescription>
            Unggah file XLSX sesuai template. Seluruh baris harus valid sebelum data disimpan.
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={(event) => void submit(event)} className="space-y-4" noValidate>
          <Field data-invalid={Boolean(error)}>
            <FieldLabel htmlFor="unit-import-file">File XLSX</FieldLabel>
            <FieldDescription>
              Maksimal 5 MB dan 1000 baris data. Kolom harus sesuai template.
            </FieldDescription>
            <Input
              id="unit-import-file"
              name="file"
              type="file"
              accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
              onChange={(event) => changeFile(event.target.files?.[0] ?? null)}
              aria-invalid={Boolean(error)}
              aria-describedby={error ? "unit-import-file-error" : undefined}
            />
            {error ? (
              <FieldError id="unit-import-file-error">
                <CircleAlert className="size-3.5" aria-hidden="true" />
                {error}
              </FieldError>
            ) : null}
            {file ? (
              <p className="text-xs text-muted-foreground" role="status">
                {file.name} — {(file.size / 1024).toFixed(1)} KB
              </p>
            ) : null}
          </Field>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void downloadTemplate()}
            disabled={downloading}
          >
            {downloading ? (
              <Loader2 className="animate-spin" aria-hidden="true" />
            ) : (
              <Download aria-hidden="true" />
            )}
            Unduh template XLSX
          </Button>
          {templateError ? (
            <p className="text-sm text-destructive" role="alert">
              {templateError}
            </p>
          ) : null}

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
            <Button type="submit" size="sm" disabled={pending || !file}>
              {pending ? (
                <>
                  <Loader2 className="animate-spin" aria-hidden="true" />
                  Mengimpor...
                </>
              ) : (
                <>
                  <Upload aria-hidden="true" />
                  Unggah
                </>
              )}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
