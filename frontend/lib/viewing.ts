const VIEWING_COOKIE = "twixter_viewing";
const VIEWING_MAX_AGE_DAYS = 365;
type Viewing = "viewing_light" | "viewing_dark";

export function getViewingCookie(): Viewing | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(
    new RegExp("(?:^|;\\s*)" + VIEWING_COOKIE + "=([^;]*)")
  );
  const value = match ? match[1].trim() : null;
  if (value === "viewing_light" || value === "viewing_dark") return value;
  return null;
}

export function setViewingCookie(value: Viewing): void {
  if (typeof document === "undefined") return;
  const maxAge = VIEWING_MAX_AGE_DAYS * 24 * 60 * 60;
  document.cookie = `${VIEWING_COOKIE}=${value}; path=/; max-age=${maxAge}; SameSite=Lax`;
}

/**
 * Resolve viewing from URL search params.
 * viewing_dark when: ?viewing_dark or ?viewing=viewing_dark
 */
export function getViewingFromSearchParams(
  searchParams: URLSearchParams | Readonly<URLSearchParams>
): Viewing | null {
  if (searchParams.get("viewing_dark") !== null) return "viewing_dark";
  if (searchParams.get("viewing") === "viewing_dark") return "viewing_dark";
  return null;
}
