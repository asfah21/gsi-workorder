import { NextRequest, NextResponse } from "next/server";

import { publicOrigin } from "@/lib/request-origin";
import { ACCESS_COOKIE, REFRESH_COOKIE } from "@/lib/session-cookies";

export function middleware(request: NextRequest): NextResponse {
  const pathname = request.nextUrl.pathname;
  const hasSession =
    Boolean(request.cookies.get(ACCESS_COOKIE)?.value) ||
    Boolean(request.cookies.get(REFRESH_COOKIE)?.value);

  const origin = publicOrigin(request) ?? request.url;

  if (pathname === "/login") {
    if (hasSession) {
      return NextResponse.redirect(new URL("/dashboard", origin));
    }
    return NextResponse.next();
  }

  if (!hasSession) {
    const loginUrl = new URL("/login", origin);
    loginUrl.searchParams.set("next", pathname + request.nextUrl.search);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/login", "/dashboard/:path*", "/units/:path*", "/master/:path*"],
};
