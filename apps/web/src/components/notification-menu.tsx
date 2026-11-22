"use client";

import {
  Bell,
  CalendarClock,
  CheckCircle2,
  Clock3,
  Loader2,
  RefreshCw,
  Wrench,
  XCircle,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, proxyGet, proxyPost } from "@/lib/api-client";
import type { NotificationItem, NotificationListResponse } from "@/lib/types";
import { formatDateTimeWita } from "@/lib/time";
import { cn } from "@/lib/utils";

export function NotificationMenu() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<NotificationItem[]>([]);
  const [unread, setUnread] = useState(0);
  const [loading, setLoading] = useState(true);
  const [marking, setMarking] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async (quiet = false) => {
    if (!quiet) {
      setLoading(true);
    }
    setError("");
    try {
      const result = await proxyGet<NotificationListResponse>(
        "/notifications?limit=20",
      );
      setItems(result.data);
      setUnread(result.meta.unread);
    } catch (loadError) {
      if (!quiet) {
        setError(
          loadError instanceof ApiError
            ? loadError.message
            : "Notifikasi tidak dapat dimuat.",
        );
      }
    } finally {
      if (!quiet) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        void load(true);
      }
    }, 30_000);
    return () => window.clearInterval(timer);
  }, [load]);

  useEffect(() => {
    if (open) {
      void load();
    }
  }, [load, open]);

  const markRead = async (ids: number[], all = false) => {
    setMarking(true);
    try {
      await proxyPost("/notifications/read", all ? { all: true } : { ids });
      if (all) {
        setUnread(0);
        setItems((current) =>
          current.map((item) => ({ ...item, read_at: item.read_at || new Date().toISOString() })),
        );
      } else {
        const target = new Set(ids);
        const changed = items.filter((item) => target.has(item.id) && !item.read_at).length;
        setItems((current) =>
          current.map((item) =>
            target.has(item.id) && !item.read_at
              ? { ...item, read_at: new Date().toISOString() }
              : item,
          ),
        );
        setUnread((current) => Math.max(0, current - changed));
      }
    } catch (markError) {
      setError(
        markError instanceof ApiError
          ? markError.message
          : "Notifikasi tidak dapat ditandai dibaca.",
      );
    } finally {
      setMarking(false);
    }
  };

  const openNotification = async (item: NotificationItem) => {
    if (!item.read_at) {
      await markRead([item.id]);
    }
    setOpen(false);
    if (item.ref_type === "wo_candidate" && item.ref_id) {
      router.push(`/wo-candidates?candidate=${item.ref_id}`);
    } else if (item.ref_type === "work_order") {
      router.push("/wo-candidates?status=all");
    } else if (item.ref_type === "pm_schedule") {
      router.push("/pm");
    }
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger
        type="button"
        aria-label={unread > 0 ? `Notifikasi, ${unread} belum dibaca` : "Notifikasi"}
        className="relative flex size-10 items-center justify-center rounded-full transition-colors hover:bg-primary/5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        <Bell className="size-4" aria-hidden="true" />
        {unread > 0 ? (
          <span className="absolute right-0.5 top-0.5 grid min-w-4 place-items-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-4 text-primary-foreground">
            {unread > 99 ? "99+" : unread}
          </span>
        ) : null}
      </SheetTrigger>

      <SheetContent className="w-full gap-0 bg-background p-0 sm:max-w-md">
        <SheetHeader className="border-b border-border p-6">
          <div className="flex items-start justify-between gap-4 pr-10">
            <div>
              <SheetTitle className="text-lg font-semibold">Notifikasi</SheetTitle>
              <SheetDescription>
                {unread > 0 ? `${unread} notifikasi belum dibaca` : "Semua notifikasi sudah dibaca"}
              </SheetDescription>
            </div>
            <Badge variant="secondary" className="mt-1 h-6 px-2.5">
              {items.length} terbaru
            </Badge>
          </div>
        </SheetHeader>

        <div className="flex items-center gap-2 border-b border-border px-6 py-3">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void load()}
            disabled={loading}
          >
            {loading ? (
              <Loader2 className="animate-spin" aria-hidden="true" />
            ) : (
              <RefreshCw aria-hidden="true" />
            )}
            Muat ulang
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => void markRead([], true)}
            disabled={marking || unread === 0}
          >
            Tandai semua dibaca
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {error ? (
            <div className="p-6">
              <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
                <XCircle aria-hidden="true" />
                <AlertTitle>Gagal memuat</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            </div>
          ) : null}

          {loading && items.length === 0 ? (
            <div className="space-y-3 p-6">
              <Skeleton className="h-16 w-full rounded-lg" />
              <Skeleton className="h-16 w-full rounded-lg" />
              <Skeleton className="h-16 w-full rounded-lg" />
            </div>
          ) : null}

          {!loading && items.length === 0 && !error ? (
            <div className="flex flex-col items-center gap-2 px-6 py-16 text-center">
              <span className="grid size-12 place-items-center rounded-full bg-muted text-muted-foreground">
                <Bell className="size-5" aria-hidden="true" />
              </span>
              <p className="text-sm font-medium">Belum ada notifikasi</p>
              <p className="max-w-xs text-sm text-muted-foreground">
                Usulan P2H dan keputusan terbaru akan tampil di sini.
              </p>
            </div>
          ) : null}

          {items.length > 0 ? (
            <ul className="divide-y divide-border">
              {items.map((item) => (
                <li key={item.id}>
                  <button
                    type="button"
                    onClick={() => void openNotification(item)}
                    className={cn(
                      "flex w-full gap-3 px-6 py-4 text-left transition-colors hover:bg-primary/5 focus-visible:bg-primary/5 focus-visible:outline-none",
                      !item.read_at && "bg-primary/[0.03]",
                    )}
                  >
                    <span
                      className={cn(
                        "mt-0.5 grid size-9 shrink-0 place-items-center rounded-full bg-muted text-muted-foreground",
                        !item.read_at && "bg-primary/10 text-primary",
                      )}
                    >
                      <NotificationIcon type={item.type} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="flex items-start gap-2">
                        <span className="line-clamp-2 flex-1 text-sm font-semibold">
                          {item.title}
                        </span>
                        {!item.read_at ? (
                          <span
                            className="mt-1.5 size-2 shrink-0 rounded-full bg-primary"
                            aria-label="Belum dibaca"
                          />
                        ) : null}
                      </span>
                      <span className="mt-1 line-clamp-2 block text-sm text-muted-foreground">
                        {item.body}
                      </span>
                      <span className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
                        <Clock3 className="size-3.5" aria-hidden="true" />
                        {formatDateTimeWita(item.created_at)}
                      </span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function NotificationIcon({ type }: { type: string }) {
  if (type.includes("approved") || type.includes("auto_wo")) {
    return <CheckCircle2 className="size-4" aria-hidden="true" />;
  }
  if (type.includes("skipped")) {
    return <XCircle className="size-4" aria-hidden="true" />;
  }
  if (type.includes("reminder")) {
    return <Clock3 className="size-4" aria-hidden="true" />;
  }
  if (type.includes("pm")) {
    return <CalendarClock className="size-4" aria-hidden="true" />;
  }
  if (type.includes("assigned")) {
    return <Wrench className="size-4" aria-hidden="true" />;
  }
  return <Bell className="size-4" aria-hidden="true" />;
}
