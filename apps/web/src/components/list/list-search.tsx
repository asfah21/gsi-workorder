import type * as React from "react";
import { Search } from "lucide-react";
import { cn } from "cn";

import { Input } from "@/components/ui/input";

export function SearchInput({
  className,
  type = "search",
  ...props
}: React.ComponentProps<"input">) {
  return (
    <div className="relative">
      <Search
        className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        aria-hidden="true"
      />
      <Input
        type={type}
        enterKeyHint="search"
        className={cn("h-9 pl-9", className)}
        {...props}
      />
    </div>
  );
}
