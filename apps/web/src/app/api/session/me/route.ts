import { NextRequest, NextResponse } from "next/server";

import { apiBaseUrl } from "@/lib/server-api";
import {
  clearSessionCookies,
  refreshSession,
  setSessionCookies,
} from "@/lib/server-session";
import { ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/session-cookies";

export async function GET(request: NextRequest): Promise<NextResponse> {
  const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;

  if (accessToken) {
    const response = await fetch(`${apiBaseUrl()}/api/me`, {
      headers: { Accept: "application/json", Authorization: `Bearer ${accessToken}` },
      cache: "no-store",
    });
    if (response.ok) {
      return NextResponse.json(await response.json());
    }
    if (response.status !== 401 && response.status !== 403) {
      return NextResponse.json(
        { error: { code: "session_unavailable", message: "Sesi tidak dapat diperiksa" } },
        { status: 502 },
      );
    }
  }

  if (refreshToken) {
    const tokens = await refreshSession(refreshToken);
    if (tokens) {
      const result = NextResponse.json(tokens.user);
      setSessionCookies(result, tokens);
      return result;
    }
  }

  const result = NextResponse.json(
    { error: { code: "unauthorized", message: "Sesi tidak valid" } },
    { status: 401 },
  );
  clearSessionCookies(result);
  return result;
}
