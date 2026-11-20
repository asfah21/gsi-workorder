import type { NextRequest } from "next/server";

const HTTP_PROTOCOLS = new Set(["http:", "https:"]);

/**
 * Origin check for same-origin BFF calls. Next.js may build nextUrl from the
 * server bind address (for example http://0.0.0.0:3000), which is not the
 * origin the browser sees. Prefer the public host/proxy headers, while keeping
 * nextUrl.origin as a fallback for local development.
 */
export function isAllowedOrigin(request: NextRequest): boolean {
  const rawOrigin = request.headers.get("origin");
  if (!rawOrigin) {
    return true;
  }

  const origin = normalizeOrigin(rawOrigin);
  return origin !== null && allowedOrigins(request).has(origin);
}

/**
 * Returns the browser-facing origin derived from the request. This is used for
 * redirects so a client is not sent to the container bind address (0.0.0.0).
 */
export function publicOrigin(request: NextRequest): string | null {
  const protocol = requestProtocol(request);
  for (const host of [
    firstHeaderValue(request.headers.get("x-forwarded-host")),
    firstHeaderValue(request.headers.get("host")),
  ]) {
    if (!host) {
      continue;
    }
    const origin = normalizeOrigin(withProtocol(host, protocol));
    if (origin) {
      return origin;
    }
  }

  return normalizeOrigin(process.env.APP_PUBLIC_URL) ?? normalizeOrigin(request.nextUrl.origin);
}

function allowedOrigins(request: NextRequest): Set<string> {
  const origins = new Set<string>();
  const add = (value: string | null | undefined) => {
    const origin = normalizeOrigin(value);
    if (origin) {
      origins.add(origin);
    }
  };

  add(process.env.APP_PUBLIC_URL);
  add(publicOrigin(request));
  add(request.nextUrl.origin);

  const protocol = requestProtocol(request);
  for (const host of [
    firstHeaderValue(request.headers.get("x-forwarded-host")),
    firstHeaderValue(request.headers.get("host")),
  ]) {
    if (host) {
      add(withProtocol(host, protocol));
    }
  }

  return origins;
}

function requestProtocol(request: NextRequest): "http" | "https" {
  const forwarded = firstHeaderValue(request.headers.get("x-forwarded-proto"))?.toLowerCase();
  if (forwarded === "https") {
    return "https";
  }
  if (forwarded === "http") {
    return "http";
  }
  return request.nextUrl.protocol === "https:" ? "https" : "http";
}

function withProtocol(host: string, protocol: string): string {
  return host.includes("://") ? host : `${protocol}://${host}`;
}

function firstHeaderValue(value: string | null): string | null {
  return value?.split(",")[0]?.trim() || null;
}

function normalizeOrigin(value: string | null | undefined): string | null {
  if (!value) {
    return null;
  }
  try {
    const url = new URL(value);
    return HTTP_PROTOCOLS.has(url.protocol) ? url.origin : null;
  } catch {
    return null;
  }
}
