export const ACCESS_COOKIE = "gsi_workorder_access";
export const REFRESH_COOKIE = "gsi_workorder_refresh";

export const ACCESS_MAX_AGE_SECONDS = 12 * 60 * 60;
export const REFRESH_MAX_AGE_SECONDS = 7 * 24 * 60 * 60;

export const sessionCookieOptions = {
  httpOnly: true,
  sameSite: "lax" as const,
  // Set COOKIE_SECURE=true when the web app is served over HTTPS. The
  // default is false so an internal HTTP deployment can still log in.
  secure: process.env.COOKIE_SECURE === "true",
  path: "/",
};
