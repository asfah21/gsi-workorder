import { NextRequest, NextResponse } from "next/server";

import { isAllowedOrigin } from "@/lib/request-origin";
import { apiBaseUrl } from "@/lib/server-api";
import {
  clearSessionCookies,
  refreshSession,
  setSessionCookies,
  type BackendTokens,
} from "@/lib/server-session";
import { ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/session-cookies";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

const ALLOWED_PREFIXES = new Set([
  "units",
  "unit-models",
  "engine-models",
  "dashboard",
  "wo-candidates",
  "p2h",
  "notifications",
  "settings",
  "workorders",
  "users",
  "pm",
  "spareparts",
  "inventory",
  "warehouses",
  "warehouse-locations",
  "material-requirements",
  "components",
]);
const ALLOWED_METHODS = new Set(["GET", "POST", "PATCH", "DELETE"]);

export async function GET(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function POST(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function PATCH(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function DELETE(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

async function proxy(request: NextRequest, context: RouteContext): Promise<NextResponse> {
  if (!ALLOWED_METHODS.has(request.method)) {
    return errorResponse(405, "method_not_allowed", "Method tidak diizinkan");
  }

  if (request.method !== "GET" && !isAllowedOrigin(request)) {
    return errorResponse(403, "forbidden_origin", "Origin tidak diizinkan");
  }

  const { path } = await context.params;
  if (!path?.length || !ALLOWED_PREFIXES.has(path[0])) {
    return errorResponse(404, "not_found", "Endpoint tidak ditemukan");
  }
  if (path.some((part) => !/^[a-zA-Z0-9_-]+$/.test(part))) {
    return errorResponse(400, "invalid_path", "Path tidak valid");
  }

  let accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  let refreshed: BackendTokens | null = null;

  if (!accessToken && refreshToken) {
    refreshed = await refreshSession(refreshToken);
    if (!refreshed) {
      const response = errorResponse(401, "unauthorized", "Sesi tidak valid");
      clearSessionCookies(response);
      return response;
    }
    accessToken = refreshed.access_token;
  }

  if (!accessToken) {
    return errorResponse(401, "unauthorized", "Sesi tidak valid");
  }

  const body =
    request.method === "GET" || request.method === "DELETE"
      ? undefined
      : await request.arrayBuffer();
  const contentType = request.headers.get("content-type") || undefined;
  const target = new URL(`/api/${path.join("/")}`, apiBaseUrl());
  target.search = request.nextUrl.search;

  let upstream = await send(target, request.method, accessToken, body, contentType);
  if (upstream.status === 401 && refreshToken && !refreshed) {
    refreshed = await refreshSession(refreshToken);
    if (refreshed) {
      accessToken = refreshed.access_token;
      upstream = await send(target, request.method, accessToken, body, contentType);
    }
  }

  if (upstream.status === 401) {
    const response = errorResponse(401, "unauthorized", "Sesi tidak valid");
    clearSessionCookies(response);
    return response;
  }

  const buffer = upstream.status === 204 ? null : await upstream.arrayBuffer();
  const response = new NextResponse(buffer, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") || "application/json; charset=utf-8",
      "Cache-Control": "no-store",
      ...(upstream.headers.get("content-disposition")
        ? { "Content-Disposition": upstream.headers.get("content-disposition") as string }
        : {}),
    },
  });
  if (refreshed) {
    setSessionCookies(response, refreshed);
  }
  return response;
}

async function send(
  target: URL,
  method: string,
  accessToken: string,
  body: ArrayBuffer | undefined,
  contentType: string | undefined,
): Promise<Response> {
  return fetch(target, {
    method,
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${accessToken}`,
      ...(contentType ? { "Content-Type": contentType } : {}),
    },
    body,
    cache: "no-store",
  });
}

function errorResponse(status: number, code: string, message: string): NextResponse {
  return NextResponse.json({ error: { code, message } }, { status });
}
