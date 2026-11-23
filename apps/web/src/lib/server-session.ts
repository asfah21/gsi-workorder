import { NextResponse } from "next/server";

import { apiBaseUrl } from "@/lib/server-api";
import {
  ACCESS_COOKIE,
  ACCESS_MAX_AGE_SECONDS,
  REFRESH_COOKIE,
  REFRESH_MAX_AGE_SECONDS,
  sessionCookieOptions,
} from "@/lib/session-cookies";

export type BackendUser = {
  id: number;
  nik: string;
  name: string;
  email: string;
  role: "admin" | "planner" | "mechanic" | "storekeeper" | "viewer" | "dispatch";
  is_active: boolean;
};

export type BackendTokens = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  expires_at: string;
  user: BackendUser;
};

export type RefreshResult = BackendTokens;

export function setSessionCookies(response: NextResponse, tokens: BackendTokens): void {
  response.cookies.set(ACCESS_COOKIE, tokens.access_token, {
    ...sessionCookieOptions,
    maxAge: ACCESS_MAX_AGE_SECONDS,
  });
  response.cookies.set(REFRESH_COOKIE, tokens.refresh_token, {
    ...sessionCookieOptions,
    maxAge: REFRESH_MAX_AGE_SECONDS,
  });
}

export function clearSessionCookies(response: NextResponse): void {
  response.cookies.set(ACCESS_COOKIE, "", { ...sessionCookieOptions, maxAge: 0 });
  response.cookies.set(REFRESH_COOKIE, "", { ...sessionCookieOptions, maxAge: 0 });
}

export async function refreshSession(refreshToken: string): Promise<BackendTokens | null> {
  if (!refreshToken) {
    return null;
  }
  const response = await fetch(`${apiBaseUrl()}/api/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
    cache: "no-store",
  });
  if (!response.ok) {
    return null;
  }
  return (await response.json()) as BackendTokens;
}
