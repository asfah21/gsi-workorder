export const APP_TIMEZONE = "Asia/Makassar";

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
