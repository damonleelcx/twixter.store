export const VIEWING_COOKIE = "twixter_viewing";
const VIEWING_MAX_AGE_DAYS = 365;
export type Viewing = "viewing_light" | "viewing_dark";

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

/** Encode viewing for share URL (base64). */
export function encodeViewingParam(value: Viewing): string {
  if (typeof btoa === "undefined") return value;
  return btoa(value);
}

/** Decode viewing from URL param (base64). Returns null if invalid. */
export function decodeViewingParam(token: string): Viewing | null {
  if (!token || typeof atob === "undefined") return null;
  try {
    const decoded = atob(token);
    if (decoded === "viewing_light" || decoded === "viewing_dark") return decoded;
    return null;
  } catch {
    return null;
  }
}

/**
 * Resolve viewing from URL search params.
 * Supports: ?viewing_dark, ?viewing=viewing_dark, ?viewing=<base64-encoded>
 */
export function getViewingFromSearchParams(
  searchParams: URLSearchParams | Readonly<URLSearchParams>
): Viewing | null {
  if (searchParams.get("viewing_dark") !== null) return "viewing_dark";
  const raw = searchParams.get("viewing");
  if (raw === "viewing_dark" || raw === "viewing_light") return raw;
  if (raw) return decodeViewingParam(raw);
  return null;
}

/** Build share URL for content detail with encoded viewing param. */
export function buildShareUrl(
  baseUrl: string,
  locale: string,
  contentId: number,
  category: "light" | "dark"
): string {
  const viewing: Viewing = category === "dark" ? "viewing_dark" : "viewing_light";
  const path = `/${locale}/post/${contentId}`;
  const encoded = encodeViewingParam(viewing);
  return `${baseUrl}${path}?viewing=${encodeURIComponent(encoded)}`;
}
