"use client";

import {
  Boxes,
  CalendarClock,
  CircleAlert,
  ClipboardCheck,
  ClipboardList,
  LayoutDashboard,
  ListChecks,
  PackageSearch,
  Search,
  Settings2,
  Truck,
  Warehouse,
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

import { NotificationMenu } from "@/components/notification-menu";
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

type NavigationItem = {
  href: string;
  label: string;
  icon: typeof LayoutDashboard;
  roles?: User["role"][];
};

const navigationGroups: { label: string; items: NavigationItem[] }[] = [
  {
    label: "Ringkasan",
    items: [{ href: "/dashboard", label: "Dashboard", icon: LayoutDashboard }],
  },
  {
    label: "Operasional",
    items: [
      { href: "/units", label: "Unit", icon: Truck },
      { href: "/p2h", label: "P2H", icon: ClipboardList },
      {
        href: "/wo-candidates",
        label: "Usulan WO",
        icon: ListChecks,
        roles: ["admin", "planner"],
      },
      {
        href: "/workorders",
        label: "Work Order",
        icon: ClipboardCheck,
      },
      {
        href: "/pm",
        label: "Preventive Maintenance",
        icon: CalendarClock,
      },
      {
        href: "/inventory",
        label: "Inventory",
        icon: Warehouse,
      },
    ],
  },
  {
    label: "Master data",
    items: [
      { href: "/master", label: "Master Unit", icon: Boxes },
      { href: "/spareparts", label: "Sparepart", icon: PackageSearch },
    ],
  },
  {
    label: "Administrasi",
    items: [
      {
        href: "/settings",
        label: "Pengaturan",
        icon: Settings2,
        roles: ["admin"],
      },
    ],
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
      <main className="grid min-h-dvh place-items-center bg-background px-4 py-10">
        <div className="w-full max-w-md rounded-xl border border-border bg-background p-6">
          <Alert variant="destructive" className="border-destructive/30 bg-destructive/5">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Tidak dapat memuat sesi</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
          <Button
            type="button"
            onClick={() => void loadSession()}
            className="mt-4 w-full"
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
      <SidebarProvider className="min-h-svh">
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

      <Sidebar variant="inset" collapsible="icon">
        <SidebarHeader className="flex flex-row items-center justify-between gap-2 p-2 pb-4">
          <Link
            href="/dashboard"
            onClick={closeMobileNav}
            aria-label="GSI Work Order"
            className="flex min-w-0 items-center gap-2 overflow-hidden"
          >
            <span className="grid size-8 shrink-0 place-items-center rounded-md bg-primary text-primary-foreground">
              <Wrench className="size-4" aria-hidden="true" />
            </span>
            <span className="hide-menu min-w-0">
              <span className="block truncate text-sm font-semibold">
                GSI Work Order
              </span>
              <span className="block truncate text-xs text-muted-foreground">
                Maintenance &amp; Unit
              </span>
            </span>
          </Link>
          <Badge
            variant="secondary"
            className="hide-menu h-5 shrink-0 px-2"
          >
            M6
          </Badge>
        </SidebarHeader>

        <SidebarContent>
          {navigationGroups.map((group) => {
            const items = group.items.filter(
              (item) => !item.roles || item.roles.includes(user.role),
            );
            if (items.length === 0) {
              return null;
            }
            return (
              <SidebarGroup key={group.label}>
                <SidebarGroupLabel className="mb-2 h-auto px-0 text-xs font-semibold uppercase text-sidebar-foreground/70">
                  {group.label}
                </SidebarGroupLabel>
                <SidebarMenu className="gap-0.5">
                  {items.map((item) => {
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
                              className="cursor-pointer"
                            />
                          }
                          isActive={active}
                          tooltip={item.label}
                          className="h-9 select-none gap-3 rounded-md px-3 text-sm font-medium data-active:bg-primary data-active:text-background data-active:hover:bg-primary data-active:hover:text-background"
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
            );
          })}
        </SidebarContent>
      </Sidebar>

      <SidebarInset
        id="konten"
        tabIndex={-1}
        className="m-2 rounded-none border border-border bg-background md:rounded-xl md:shadow-sm"
      >
        <header className="sticky top-0 z-20 min-h-14 border-b border-border bg-background md:rounded-t-xl">
          <div className="flex items-center justify-between gap-2 p-2">
            <div className="flex h-9 min-w-0 items-center gap-2">
              <Link
                href="/dashboard"
                className={`${isMobile ? "flex" : "hidden"} min-w-0 items-center gap-2`}
              >
                <span className="grid size-8 shrink-0 place-items-center rounded-md bg-primary text-primary-foreground">
                  <Wrench className="size-4" aria-hidden="true" />
                </span>
                <span className="hidden min-w-0 sm:block">
                  <span className="block truncate text-sm font-semibold">
                    GSI Work Order
                  </span>
                  <span className="block truncate text-xs text-muted-foreground">
                    Maintenance &amp; Unit
                  </span>
                </span>
              </Link>

              <SidebarTrigger
                aria-label="Buka menu navigasi"
                className="size-9 shrink-0"
              />

              <Separator
                orientation="vertical"
                className="mx-0.5 h-6 shrink-0 self-center"
              />

              <form
                role="search"
                onSubmit={handleSearch}
                className="hidden min-w-0 sm:block"
              >
                <label htmlFor="global-unit-search" className="sr-only">
                  Cari unit
                </label>
                <div className="relative w-80 max-w-full">
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
                    className="pl-9 text-sm leading-5"
                  />
                </div>
              </form>
            </div>

            <div className="ml-auto flex items-center gap-1.5">
              <NotificationMenu />
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

        <div className="flex flex-1 flex-col gap-4 p-4">
          <div className="container mx-auto flex min-h-[calc(100dvh-140px)] w-full flex-1 flex-col">
            {children}
          </div>
          <div className="container mx-auto w-full pt-6">
            <footer className="flex items-center justify-center border-t border-border/60 pt-4 text-xs text-muted-foreground sm:justify-between">
              <p className="hidden sm:block">
                © 2026 - PT Gunung Samudera Internasional.
              </p>
              <p className="sm:hidden">© 2026 - PT GSI</p>
              <p className="hidden text-muted-foreground/70 sm:block">AZV - V.0.1.1</p>
            </footer>
          </div>
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
    <div className="min-h-svh bg-sidebar p-2">
      <aside className="fixed inset-y-2 left-2 hidden w-64 rounded-xl bg-sidebar p-3 lg:block">
        <div className="flex items-center gap-2 px-1">
          <Skeleton className="size-8 rounded-md" />
          <div className="space-y-1.5">
            <Skeleton className="h-3.5 w-28" />
            <Skeleton className="h-3 w-20" />
          </div>
        </div>
        <div className="mt-8 space-y-2">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-full rounded-md" />
        </div>
      </aside>
      <div className="ml-0 min-w-0 overflow-hidden rounded-none border border-border bg-background lg:ml-64 lg:rounded-xl">
        <header className="min-h-14 border-b border-border p-2">
          <div className="flex items-center gap-2">
            <Skeleton className="size-9 rounded-md" />
            <Skeleton className="h-9 w-80 max-w-full" />
            <div className="ml-auto flex items-center gap-1.5">
              <Skeleton className="size-9 rounded-md" />
              <Skeleton className="h-9 w-24 rounded-md" />
            </div>
          </div>
        </header>
        <div className="p-4">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="mt-3 h-4 w-80 max-w-full" />
          <div className="mt-4 grid gap-4 lg:grid-cols-12">
            <Skeleton className="h-56 rounded-xl lg:col-span-8" />
            <Skeleton className="h-56 rounded-xl lg:col-span-4" />
            <Skeleton className="h-80 rounded-xl lg:col-span-8" />
            <Skeleton className="h-80 rounded-xl lg:col-span-4" />
          </div>
        </div>
      </div>
    </div>
  );
}
