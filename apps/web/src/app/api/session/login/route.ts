import { NextRequest, NextResponse } from "next/server";

import { isAllowedOrigin } from "@/lib/request-origin";
import { apiBaseUrl } from "@/lib/server-api";
import type { BackendTokens } from "@/lib/server-session";
import { setSessionCookies } from "@/lib/server-session";

export async function POST(request: NextRequest): Promise<NextResponse> {
  if (!isAllowedOrigin(request)) {
    return NextResponse.json(
      { error: { code: "forbidden_origin", message: "Origin tidak diizinkan" } },
      { status: 403 },
    );
  }

  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json(
      { error: { code: "invalid_request", message: "Body login tidak valid" } },
      { status: 400 },
    );
  }

  const response = await fetch(`${apiBaseUrl()}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });

  const payload = (await response.json().catch(() => null)) as
    | BackendTokens
    | { error?: { code?: string; message?: string } }
    | null;

  if (!response.ok || !payload || !("access_token" in payload)) {
    return NextResponse.json(payload ?? {
      error: { code: "login_failed", message: "Login gagal" },
    }, { status: response.status });
  }

  const tokens = payload as BackendTokens;
  const result = NextResponse.json({
    user: tokens.user,
    expires_at: tokens.expires_at,
  });
  setSessionCookies(result, tokens);
  return result;
}
