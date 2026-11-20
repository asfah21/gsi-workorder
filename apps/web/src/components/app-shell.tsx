"use client";

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

import { ApiError, getSession, logout } from "@/lib/api-client";
import type { User } from "@/lib/types";

type SessionContextValue = {
  user: User;
  refreshSession: () => Promise<void>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

const roleLabels: Record<User["role"], string> = {
  admin: "Admin",
  planner: "Planner",
  mechanic: "Mekanik",
  storekeeper: "Storekeeper",
  viewer: "Viewer",
};

const navigation = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/units", label: "Unit" },
  { href: "/master", label: "Master Unit" },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState("");

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
    await logout();
    router.replace("/login");
    router.refresh();
  };

  const value = useMemo<SessionContextValue | null>(
    () => (user ? { user, refreshSession: loadSession } : null),
    [user, loadSession],
  );

  if (error) {
    return (
      <main className="grid min-h-dvh place-items-center px-6">
        <div role="alert" className="max-w-md text-center">
          <h1 className="text-xl font-semibold">Tidak dapat memuat sesi</h1>
          <p className="mt-2 text-sm text-muted-foreground">{error}</p>
          <button
            type="button"
            onClick={() => void loadSession()}
            className="mt-5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            Coba lagi
          </button>
        </div>
      </main>
    );
  }

  if (!user) {
    return (
      <main className="grid min-h-dvh place-items-center px-6">
        <p role="status" className="text-sm text-muted-foreground">
          Memuat sesi...
        </p>
      </main>
    );
  }

  return (
    <SessionContext.Provider value={value as SessionContextValue}>
      <a
        href="#konten"
        className="fixed left-4 top-4 z-50 -translate-y-24 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-transform focus:translate-y-0"
      >
        Lewati ke konten
      </a>
      <div className="min-h-dvh bg-muted/30">
        <header className="border-b border-border bg-background">
          <div className="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-3 lg:px-8">
            <div className="min-w-0">
              <Link href="/dashboard" className="block truncate text-sm font-semibold tracking-tight">
                GSI Work Order
              </Link>
              <p className="truncate text-xs text-muted-foreground">Maintenance &amp; Unit</p>
            </div>
            <div className="flex items-center gap-3">
              <div className="hidden text-right sm:block">
                <p className="text-sm font-medium">{user.name || user.nik}</p>
                <p className="text-xs text-muted-foreground">{roleLabels[user.role]}</p>
              </div>
              <button
                type="button"
                onClick={() => void handleLogout()}
                className="min-h-10 rounded-md border border-border px-3 text-sm font-medium hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
              >
                Keluar
              </button>
            </div>
          </div>
          <nav aria-label="Navigasi utama" className="border-t border-border lg:hidden">
            <ul className="mx-auto flex max-w-7xl gap-1 overflow-x-auto px-3 py-2">
              {navigation.map((item) => (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    aria-current={isActive(pathname, item.href) ? "page" : undefined}
                    className={`flex min-h-10 items-center rounded-md px-3 text-sm font-medium ${
                      isActive(pathname, item.href)
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground"
                    }`}
                  >
                    {item.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        </header>

        <div className="mx-auto flex max-w-7xl">
          <aside className="hidden w-56 shrink-0 border-r border-border bg-background px-3 py-6 lg:block">
            <nav aria-label="Navigasi samping">
              <ul className="space-y-1">
                {navigation.map((item) => (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      aria-current={isActive(pathname, item.href) ? "page" : undefined}
                      className={`flex min-h-10 items-center rounded-md px-3 text-sm font-medium ${
                        isActive(pathname, item.href)
                          ? "bg-primary text-primary-foreground"
                          : "text-muted-foreground hover:bg-muted hover:text-foreground"
                      }`}
                    >
                      {item.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          </aside>
          <main id="konten" tabIndex={-1} className="min-w-0 flex-1 px-4 py-6 lg:px-8">
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
