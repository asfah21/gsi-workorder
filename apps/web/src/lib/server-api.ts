const DEFAULT_API_URL = "http://localhost:8080";

/**
 * Server-only API base URL. The browser never needs the internal Docker
 * service name; all browser requests go through the same-origin BFF routes.
 */
export function apiBaseUrl(): string {
  if (process.env.API_INTERNAL_URL) {
    return process.env.API_INTERNAL_URL.replace(/\/$/, "");
  }
  if (process.env.NODE_ENV === "production") {
    return "http://api:8080";
  }
  return (process.env.NEXT_PUBLIC_API_URL || DEFAULT_API_URL).replace(/\/$/, "");
}
