"use client";

import {
  Boxes,
  Home,
  Loader2,
  Mail,
  ShieldCheck,
  Truck,
  UserRound,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import type { User } from "@/lib/types";

const accountLinks = [
  { href: "/dashboard", label: "Dashboard", icon: Home },
  { href: "/units", label: "Unit", icon: Truck },
  { href: "/master", label: "Master unit", icon: Boxes },
] as const;

const menuRowBaseClassName =
  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-semibold text-muted-foreground";

const menuRowClassName =
  `${menuRowBaseClassName} transition-colors hover:bg-primary/5 hover:text-primary focus-visible:bg-primary/5 focus-visible:text-primary focus-visible:outline-none`;

export function ProfileMenu({
  user,
  roleLabel,
  onLogout,
  loggingOut,
}: {
  user: User;
  roleLabel: string;
  onLogout: () => Promise<void> | void;
  loggingOut: boolean;
}) {
  const [open, setOpen] = useState(false);
  const initials = userInitials(user);
  const displayName = user.name || user.nik || "Pengguna";
  const accountLine = user.email?.trim() || `NIK ${user.nik}`;

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger
        type="button"
        aria-label="Buka menu pengguna"
        className="flex size-10 items-center justify-center rounded-full transition-colors hover:bg-primary/5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        <Avatar className="size-8">
          <AvatarFallback className="bg-muted text-xs font-semibold text-muted-foreground">
            {initials}
          </AvatarFallback>
        </Avatar>
      </SheetTrigger>

      <SheetContent
        side="right"
        className="w-full gap-4 border-l-0 bg-background p-0 sm:max-w-sm"
      >
        <SheetHeader className="items-center p-6 text-center">
          <div className="flex flex-col items-center gap-4 pt-10">
            <Avatar className="size-20">
              <AvatarFallback className="bg-muted text-2xl font-semibold text-muted-foreground">
                {initials}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0 text-center">
              <SheetTitle className="text-lg font-semibold">
                {displayName}
              </SheetTitle>
              <SheetDescription className="flex items-center justify-center gap-2">
                <Mail className="size-4 shrink-0" aria-hidden="true" />
                <span className="truncate">{accountLine}</span>
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <div className="border-t border-dashed border-border">
          <ul className="flex flex-col gap-2 p-6">
            {accountLinks.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={() => setOpen(false)}
                  className={menuRowClassName}
                >
                  <item.icon className="size-5 shrink-0" aria-hidden="true" />
                  <span>{item.label}</span>
                </Link>
              </li>
            ))}
            <li>
              <InfoRow icon={UserRound} label="Peran" value={roleLabel} />
            </li>
            <li>
              <InfoRow icon={ShieldCheck} label="Sesi aman" value="Aktif" />
            </li>
          </ul>
        </div>

        <SheetFooter className="mt-auto border-t border-dashed border-border px-6 pt-6 pb-4">
          <Button
            type="button"
            variant="destructive"
            className="mx-auto h-9 px-2.5"
            onClick={() => {
              setOpen(false);
              void onLogout();
            }}
            disabled={loggingOut}
          >
            {loggingOut ? (
              <Loader2 className="animate-spin" aria-hidden="true" />
            ) : null}
            {loggingOut ? "Keluar..." : "Keluar"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function InfoRow({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  return (
    <div className={menuRowBaseClassName}>
      <Icon className="size-5 shrink-0" aria-hidden="true" />
      <span>{label}</span>
      <span className="ml-auto flex h-5 min-w-6 items-center justify-center rounded-sm bg-primary/5 px-1.5 text-xs font-medium text-primary">
        {value}
      </span>
    </div>
  );
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
