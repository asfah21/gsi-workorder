import type * as React from "react";
import { cn } from "cn";

export function FilterBar({
  className,
  ...props
}: React.ComponentProps<"form">) {
  return (
    <form
      className={cn(
        "flex flex-col gap-3 md:flex-row md:flex-wrap md:items-end",
        className,
      )}
      {...props}
    />
  );
}
