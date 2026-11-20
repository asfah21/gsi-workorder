"use client";

import {
  Boxes,
  CircleAlert,
  LayoutDashboard,
  Loader2,
  LogOut,
  Truck,
  Wrench,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, getSession, logout } from "@/lib/api-client";
import type { User } from "@/lib/types";

type SessionContextValue = {
  user: User;
  refreshSession: () => Promise<void>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

export const roleLabels: Record<User["role"], string> = {
  admin: "Admin",
  planner: "Planner",
  mechanic: "Mekanik",
  storekeeper: "Storekeeper",
  viewer: "Viewer",
};

const navigation = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/units", label: "Unit", icon: Truck },
  { href: "/master", label: "Master Unit", icon: Boxes },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState("");
  const [loggingOut, setLoggingOut] = useState(false);

  const loadSession = useCallback(async () => {
    try {
      setUser(await getSession());
      setError("");
    } catch (sessionError) {
      if (sessionError instanceof ApiError && sessionError.status === 401) {
        router.replace("/login");
        return;
      }
      setError("Sesi tidak dapat dimuat. Coba lagi.");
    }
  }, [router]);

  useEffect(() => {
    void loadSession();
  }, [loadSession]);

  const handleLogout = async () => {
    setLoggingOut(true);
    try {
      await logout();
    } catch {
      // Redirect remains safe even if the local session is already invalid.
    } finally {
      router.replace("/login");
      router.refresh();
      setLoggingOut(false);
    }
  };

  const value = useMemo<SessionContextValue | null>(
    () => (user ? { user, refreshSession: loadSession } : null),
    [user, loadSession],
  );

  if (error) {
    return (
      <main className="grid min-h-dvh place-items-center bg-muted/30 px-4 py-10">
        <div className="w-full max-w-md rounded-2xl border border-border bg-background p-6 shadow-sm">
          <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Tidak dapat memuat sesi</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
          <Button
            type="button"
            onClick={() => void loadSession()}
            className="mt-4 h-11 w-full px-4"
          >
            Coba lagi
          </Button>
        </div>
      </main>
    );
  }

  if (!user) {
    return <AppShellSkeleton />;
  }

  const initials = userInitials(user);

  return (
    <SessionContext.Provider value={value as SessionContextValue}>
      <a
        href="#konten"
        className="fixed left-4 top-4 z-50 -translate-y-24 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-lg transition-transform focus:translate-y-0"
      >
        Lewati ke konten
      </a>
      <div className="min-h-dvh bg-muted/30">
        <header className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85">
          <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-3 lg:px-8">
            <Link href="/dashboard" className="flex min-w-0 items-center gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-primary text-primary-foreground shadow-sm">
                <Wrench className="size-5" aria-hidden="true" />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold tracking-tight">
                  GSI Work Order
                </span>
                <span className="block truncate text-xs text-muted-foreground">
                  Maintenance &amp; Unit
                </span>
              </span>
            </Link>

            <div className="flex items-center gap-2 sm:gap-3">
              <div className="hidden text-right sm:block">
                <p className="text-sm font-medium">{user.name || user.nik || "Pengguna"}</p>
                <p className="text-xs text-muted-foreground">
                  {user.role ? roleLabels[user.role] : "-"}
                </p>
              </div>
              <span
                className="grid size-9 shrink-0 place-items-center rounded-full bg-muted text-xs font-semibold text-muted-foreground"
                aria-hidden="true"
              >
                {initials}
              </span>
              <Button
                type="button"
                variant="outline"
                onClick={() => void handleLogout()}
                disabled={loggingOut}
                className="h-10 px-3"
                aria-label="Keluar dari aplikasi"
              >
                {loggingOut ? (
                  <Loader2 className="animate-spin" aria-hidden="true" />
                ) : (
                  <LogOut aria-hidden="true" />
                )}
                <span className="hidden sm:inline">Keluar</span>
              </Button>
            </div>
          </div>

          <nav aria-label="Navigasi utama" className="border-t border-border lg:hidden">
            <ul className="mx-auto flex max-w-7xl gap-1 overflow-x-auto px-3 py-2">
              {navigation.map((item) => {
                const active = isActive(pathname, item.href);
                return (
                  <li key={item.href} className="shrink-0">
                    <Link
                      href={item.href}
                      aria-current={active ? "page" : undefined}
                      className={`flex min-h-10 items-center gap-2 rounded-lg px-3 text-sm font-medium transition-colors ${
                        active
                          ? "bg-primary text-primary-foreground shadow-sm"
                          : "text-muted-foreground hover:bg-muted hover:text-foreground"
                      }`}
                    >
                      <item.icon className="size-4" aria-hidden="true" />
                      {item.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </nav>
        </header>

        <div className="mx-auto flex max-w-7xl">
          <aside className="hidden w-60 shrink-0 border-r border-border bg-background px-3 py-6 lg:block">
            <p className="px-3 pb-2 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              Menu
            </p>
            <nav aria-label="Navigasi samping">
              <ul className="space-y-1">
                {navigation.map((item) => {
                  const active = isActive(pathname, item.href);
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        aria-current={active ? "page" : undefined}
                        className={`flex min-h-10 items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors ${
                          active
                            ? "bg-primary text-primary-foreground shadow-sm"
                            : "text-muted-foreground hover:bg-muted hover:text-foreground"
                        }`}
                      >
                        <item.icon className="size-4" aria-hidden="true" />
                        <span className="min-w-0 flex-1 truncate">{item.label}</span>
                        {active ? (
                          <span
                            className="size-1.5 rounded-full bg-primary-foreground"
                            aria-hidden="true"
                          />
                        ) : null}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </nav>

            <div className="mt-6 rounded-xl border border-border bg-muted/40 p-4">
              <p className="text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                Milestone 1
              </p>
              <p className="mt-1 text-sm font-medium">Master unit &amp; HM/KM</p>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                P2H, work order, dan inventory menyusul.
              </p>
            </div>
          </aside>

          <main id="konten" tabIndex={-1} className="min-w-0 flex-1 px-4 py-6 lg:px-8 lg:py-8">
            {children}
          </main>
        </div>
      </div>
    </SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error("useSession must be used inside AppShell");
  }
  return context;
}

function isActive(pathname: string, href: string): boolean {
  return pathname === href || pathname.startsWith(`${href}/`);
}

function userInitials(user: User): string {
  return (
    (user.name || user.nik || "?")
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0])
      .join("")
      .toUpperCase() || "?"
  );
}

function AppShellSkeleton() {
  return (
    <div className="min-h-dvh bg-muted/30">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-3 lg:px-8">
          <div className="flex items-center gap-3">
            <Skeleton className="size-10 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-3 w-24" />
            </div>
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden space-y-2 sm:block">
              <Skeleton className="ml-auto h-4 w-28" />
              <Skeleton className="ml-auto h-3 w-16" />
            </div>
            <Skeleton className="size-9 rounded-full" />
            <Skeleton className="h-10 w-24 rounded-lg" />
          </div>
        </div>
      </header>
      <div className="mx-auto flex max-w-7xl">
        <aside className="hidden w-60 shrink-0 border-r border-border bg-background p-4 lg:block">
          <Skeleton className="mb-4 h-3 w-16" />
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </aside>
        <main className="min-w-0 flex-1 px-4 py-6 lg:px-8 lg:py-8">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="mt-3 h-4 w-80 max-w-full" />
          <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-32 rounded-xl" />
            ))}
          </div>
          <div className="mt-4 grid gap-4 lg:grid-cols-2">
            <Skeleton className="h-48 rounded-xl" />
            <Skeleton className="h-48 rounded-xl" />
          </div>
        </main>
      </div>
    </div>
  );
}
