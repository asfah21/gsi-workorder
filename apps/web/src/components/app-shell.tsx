"use client";

import {
  Boxes,
  CircleAlert,
  LayoutDashboard,
  Search,
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

import { ProfileMenu } from "@/components/profile-menu";
import { ThemeToggle } from "@/components/theme-toggle";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
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

const navigationGroups = [
  {
    label: "Ringkasan",
    items: [{ href: "/dashboard", label: "Dashboard", icon: LayoutDashboard }],
  },
  {
    label: "Operasional",
    items: [{ href: "/units", label: "Unit", icon: Truck }],
  },
  {
    label: "Master data",
    items: [{ href: "/master", label: "Master Unit", icon: Boxes }],
  },
];

export function AppShell({ children }: { children: React.ReactNode }) {
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

  return (
    <SessionContext.Provider value={value as SessionContextValue}>
      <SidebarProvider className="min-h-dvh bg-muted/30">
        <AppShellFrame user={user}>{children}</AppShellFrame>
      </SidebarProvider>
    </SessionContext.Provider>
  );
}

function AppShellFrame({
  user,
  children,
}: {
  user: User;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { isMobile, setOpenMobile } = useSidebar();
  const [search, setSearch] = useState("");
  const [loggingOut, setLoggingOut] = useState(false);

  const roleLabel = user.role ? roleLabels[user.role] : "-";

  const closeMobileNav = () => {
    if (isMobile) {
      setOpenMobile(false);
    }
  };

  const handleLogout = async () => {
    setLoggingOut(true);
    try {
      await logout();
    } catch {
      // Redirect remains safe even if the local session is already invalid.
    } finally {
      setOpenMobile(false);
      router.replace("/login");
      router.refresh();
      setLoggingOut(false);
    }
  };

  const handleSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const term = search.trim();
    closeMobileNav();
    router.push(term ? `/units?q=${encodeURIComponent(term)}` : "/units");
  };

  return (
    <>
      <a
        href="#konten"
        className="fixed left-4 top-4 z-50 -translate-y-24 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-lg transition-transform focus:translate-y-0"
      >
        Lewati ke konten
      </a>

      <Sidebar
        variant="inset"
        collapsible="icon"
        className="**:data-[slot=sidebar-inner]:border **:data-[slot=sidebar-inner]:border-border **:data-[slot=sidebar-inner]:bg-background"
      >
        <SidebarHeader className="flex flex-row items-center justify-between gap-2 border-b border-border p-3 pe-12 md:pe-3">
          <Link
            href="/dashboard"
            onClick={closeMobileNav}
            aria-label="GSI Work Order"
            className="flex min-w-0 items-center gap-3"
          >
            <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-primary text-primary-foreground">
              <Wrench className="size-4" aria-hidden="true" />
            </span>
            <span className="hide-menu min-w-0">
              <span className="block truncate text-sm font-semibold tracking-tight">
                GSI Work Order
              </span>
              <span className="block truncate text-xs text-muted-foreground">
                Maintenance &amp; Unit
              </span>
            </span>
          </Link>
          <Badge
            variant="secondary"
            className="hide-menu hidden h-6 shrink-0 px-2 text-[0.65rem] md:inline-flex"
          >
            M1
          </Badge>
        </SidebarHeader>

        <SidebarContent>
          {navigationGroups.map((group) => (
            <SidebarGroup key={group.label} className="py-3">
              <SidebarGroupLabel className="px-3 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                {group.label}
              </SidebarGroupLabel>
              <SidebarMenu className="gap-1">
                {group.items.map((item) => {
                  const active = isActive(pathname, item.href);
                  return (
                    <SidebarMenuItem key={item.href}>
                      <SidebarMenuButton
                        render={
                          <Link
                            href={item.href}
                            aria-label={item.label}
                            aria-current={active ? "page" : undefined}
                            onClick={closeMobileNav}
                          />
                        }
                        isActive={active}
                        tooltip={item.label}
                        className="min-h-10 gap-3 rounded-lg px-3 text-sm font-medium data-active:bg-primary data-active:text-primary-foreground"
                      >
                        <item.icon className="size-4" aria-hidden="true" />
                        <span className="hide-menu min-w-0 flex-1 truncate">
                          {item.label}
                        </span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroup>
          ))}
        </SidebarContent>

        <SidebarFooter className="border-t border-border p-3">
          <div className="hide-menu rounded-xl border border-border bg-muted/40 p-3">
            <p className="text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              Milestone 1
            </p>
            <p className="mt-1 text-sm font-medium">Master unit &amp; HM/KM</p>
            <p className="mt-1 text-xs leading-5 text-muted-foreground">
              P2H, work order, dan inventory menyusul.
            </p>
          </div>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset
        id="konten"
        tabIndex={-1}
        className="m-2 overflow-hidden rounded-xl border border-border bg-background shadow-none"
      >
        <header className="sticky top-0 z-30 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85">
          <div className="mx-auto flex max-w-7xl items-center gap-2 px-3 py-3 sm:gap-3 lg:px-6">
            <Link
              href="/dashboard"
              className="flex min-w-0 items-center gap-2 md:hidden"
            >
              <span className="grid size-9 shrink-0 place-items-center rounded-xl bg-primary text-primary-foreground">
                <Wrench className="size-4" aria-hidden="true" />
              </span>
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold tracking-tight">
                  GSI Work Order
                </span>
                <span className="hidden truncate text-xs text-muted-foreground sm:block">
                  Maintenance &amp; Unit
                </span>
              </span>
            </Link>

            <SidebarTrigger
              aria-label="Buka menu navigasi"
              className="size-11 shrink-0 rounded-full"
            />

            <form
              role="search"
              onSubmit={handleSearch}
              className="ml-1 hidden min-w-0 flex-1 md:block md:max-w-sm"
            >
              <label htmlFor="global-unit-search" className="sr-only">
                Cari unit
              </label>
              <div className="relative">
                <Search
                  className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                  aria-hidden="true"
                />
                <Input
                  id="global-unit-search"
                  name="q"
                  type="search"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Cari unit..."
                  enterKeyHint="search"
                  className="h-10 pl-9"
                />
              </div>
            </form>

            <Separator
              orientation="vertical"
              className="mx-1 hidden h-6 self-center md:block"
            />

            <div className="ml-auto flex items-center gap-1.5 sm:gap-2">
              <ThemeToggle />
              <ProfileMenu
                user={user}
                roleLabel={roleLabel}
                onLogout={handleLogout}
                loggingOut={loggingOut}
              />
            </div>
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col px-4 py-6 lg:px-6 lg:py-8">
          <div className="min-h-[calc(100dvh-9rem)] flex-1">{children}</div>
          <footer className="mt-10 flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4 text-xs text-muted-foreground">
            <p>GSI Work Order · Milestone 1</p>
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <p>Operasional WITA · PT GSI</p>
              <span aria-hidden="true">·</span>
              <a
                href="https://shadcndashboard.dev/"
                target="_blank"
                rel="noreferrer"
                className="underline-offset-4 hover:text-foreground hover:underline"
              >
                UI reference: Shadcn Dashboard
              </a>
            </div>
          </footer>
        </div>
      </SidebarInset>
    </>
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

function AppShellSkeleton() {
  return (
    <div className="min-h-dvh bg-muted/30 p-2">
      <div className="flex min-h-[calc(100dvh-1rem)] gap-2">
        <aside className="hidden w-64 shrink-0 rounded-xl border border-border bg-background p-4 md:block">
          <div className="flex items-center gap-3">
            <Skeleton className="size-10 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-3 w-20" />
            </div>
          </div>
          <div className="mt-8 space-y-3">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-10 w-full rounded-lg" />
            <Skeleton className="h-10 w-full rounded-lg" />
            <Skeleton className="h-10 w-full rounded-lg" />
          </div>
        </aside>
        <div className="min-w-0 flex-1 overflow-hidden rounded-xl border border-border bg-background">
          <header className="border-b border-border px-4 py-3">
            <div className="flex items-center gap-3">
              <Skeleton className="size-10 rounded-full" />
              <Skeleton className="h-10 w-full max-w-sm" />
              <div className="ml-auto flex items-center gap-2">
                <Skeleton className="size-10 rounded-full" />
                <Skeleton className="size-10 rounded-full" />
              </div>
            </div>
          </header>
          <div className="px-4 py-6 lg:px-6 lg:py-8">
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
          </div>
        </div>
      </div>
    </div>
  );
}
