"use client";

import { Loader2, LogOut, ShieldCheck } from "lucide-react";

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
  const initials = userInitials(user);
  const displayName = user.name || user.nik || "Pengguna";

  return (
    <Sheet>
      <SheetTrigger
        type="button"
        aria-label="Buka menu pengguna"
        className="flex h-9 items-center gap-2 rounded-md border border-transparent p-1 text-left transition-colors hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring sm:pr-2"
      >
        <Avatar className="size-7">
          <AvatarFallback className="bg-muted text-xs font-semibold text-muted-foreground">
            {initials}
          </AvatarFallback>
        </Avatar>
        <span className="hidden min-w-0 sm:block">
          <span className="block max-w-36 truncate text-sm font-medium">
            {displayName}
          </span>
          <span className="block text-xs text-muted-foreground">{roleLabel}</span>
        </span>
      </SheetTrigger>

      <SheetContent side="right" className="w-full sm:max-w-sm">
        <SheetHeader className="items-center pt-8 text-center">
          <Avatar size="lg" className="size-16">
            <AvatarFallback className="bg-primary text-lg font-semibold text-primary-foreground">
              {initials}
            </AvatarFallback>
          </Avatar>
          <SheetTitle className="mt-3 text-lg">{displayName}</SheetTitle>
          <SheetDescription>
            <span className="block">{roleLabel}</span>
            <span className="block font-mono">{user.nik}</span>
            {user.email ? (
              <span className="block truncate">{user.email}</span>
            ) : null}
          </SheetDescription>
        </SheetHeader>

        <div className="mx-4 rounded-xl border border-border bg-muted/40 p-4 text-sm">
          <div className="flex items-center gap-2 font-medium">
            <ShieldCheck className="size-4" aria-hidden="true" />
            Sesi aman
          </div>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">
            Token disimpan di cookie httpOnly dan tidak disimpan di localStorage.
          </p>
        </div>

        <SheetFooter className="mt-auto">
          <Button
            type="button"
            variant="outline"
            onClick={() => void onLogout()}
            disabled={loggingOut}
            className="w-full"
          >
            {loggingOut ? (
              <Loader2 className="animate-spin" aria-hidden="true" />
            ) : (
              <LogOut aria-hidden="true" />
            )}
            {loggingOut ? "Keluar..." : "Keluar dari aplikasi"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
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
