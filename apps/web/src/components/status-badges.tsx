import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export function UnitStatusBadge({
  active,
  breakdown,
}: {
  active: boolean;
  breakdown: boolean;
}) {
  if (!active) {
    return (
      <Badge variant="outline" className="h-5 gap-1.5 px-2 text-muted-foreground">
        <span className="size-1.5 rounded-full bg-muted-foreground" aria-hidden="true" />
        Nonaktif
      </Badge>
    );
  }
  if (breakdown) {
    return (
      <Badge variant="destructive" className="h-5 gap-1.5 px-2">
        <span className="size-1.5 rounded-full bg-current" aria-hidden="true" />
        Breakdown
      </Badge>
    );
  }
  return (
    <Badge
      variant="secondary"
      className="h-6 gap-1.5 border-transparent bg-emerald-500/10 px-2.5 text-emerald-700 dark:text-emerald-300"
    >
      <span className="size-1.5 rounded-full bg-current" aria-hidden="true" />
      Aktif
    </Badge>
  );
}

export function ActiveBadge({ active }: { active: boolean }) {
  return (
    <Badge
      variant={active ? "secondary" : "outline"}
      className={cn(
        "h-5 gap-1.5 px-2",
        active
          ? "border-transparent bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
          : "text-muted-foreground",
      )}
    >
      <span
        className={cn(
          "size-1.5 rounded-full",
          active ? "bg-emerald-600 dark:bg-emerald-300" : "bg-muted-foreground",
        )}
        aria-hidden="true"
      />
      {active ? "Aktif" : "Nonaktif"}
    </Badge>
  );
}
