export const APP_TIMEZONE = "Asia/Makassar";

function witaCalendarParts(value: Date): { year: number; month: number; day: number } {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: APP_TIMEZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(value);
  const find = (type: string) => Number(parts.find((part) => part.type === type)?.value ?? 0);
  return { year: find("year"), month: find("month"), day: find("day") };
}

/** WITA calendar date for today, formatted as YYYY-MM-DD. */
export function todayWitaDate(): string {
  const { year, month, day } = witaCalendarParts(new Date());
  return `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

/** WITA calendar date N days before today, formatted as YYYY-MM-DD. */
export function daysAgoWitaDate(days: number): string {
  const { year, month, day } = witaCalendarParts(new Date());
  const value = new Date(Date.UTC(year, month - 1, day - days));
  return value.toISOString().slice(0, 10);
}

export function formatWita(
  value: string | number | Date,
  options: Intl.DateTimeFormatOptions = {},
): string {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat("id-ID", {
    timeZone: APP_TIMEZONE,
    ...options,
  }).format(date);
}

export function formatDateTimeWita(value?: string | null): string {
  if (!value) {
    return "-";
  }
  return formatWita(value, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

export function formatNumber(value: number): string {
  return new Intl.NumberFormat("id-ID", {
    maximumFractionDigits: 1,
    minimumFractionDigits: 0,
  }).format(value);
}

export function formatDurationMinutes(value?: number | null): string {
  if (value === undefined || value === null || !Number.isFinite(value)) {
    return "-";
  }
  const total = Math.max(0, Math.round(value));
  if (total < 60) {
    return `${total} mnt`;
  }
  const hours = Math.floor(total / 60);
  const minutes = total % 60;
  if (hours < 24) {
    return minutes > 0 ? `${hours}j ${minutes}m` : `${hours}j`;
  }
  const days = Math.floor(hours / 24);
  const rest = hours % 24;
  return rest > 0 ? `${days}h ${rest}j` : `${days}h`;
}

export function toDatetimeLocalWita(date: Date = new Date()): string {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: APP_TIMEZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).formatToParts(date);

  const get = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((part) => part.type === type)?.value || "00";

  return `${get("year")}-${get("month")}-${get("day")}T${get("hour")}:${get("minute")}`;
}

/**
 * Convert a datetime-local value interpreted in Asia/Makassar to UTC ISO.
 * The offset is derived from Intl so no fixed +08:00 offset is hardcoded.
 */
export function witaLocalToISOString(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/.exec(value);
  if (!match) {
    throw new Error("Format waktu tidak valid");
  }
  const [, year, month, day, hour, minute] = match;
  const wallClockAsUTC = Date.UTC(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
  );

  let timestamp = wallClockAsUTC;
  for (let i = 0; i < 2; i += 1) {
    const offset = timezoneOffsetMs(new Date(timestamp));
    timestamp = wallClockAsUTC - offset;
  }
  return new Date(timestamp).toISOString();
}

function timezoneOffsetMs(date: Date): number {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: APP_TIMEZONE,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).formatToParts(date);

  const get = (type: Intl.DateTimeFormatPartTypes) =>
    Number(parts.find((part) => part.type === type)?.value || 0);

  const asUTC = Date.UTC(
    get("year"),
    get("month") - 1,
    get("day"),
    get("hour"),
    get("minute"),
    get("second"),
  );
  return asUTC - date.getTime();
}
