import { fetchViewingToken } from "@/lib/api";

export const VIEWING_COOKIE = "twixter_viewing";
const VIEWING_MAX_AGE_DAYS = 365;

/** Cookie/URL value is opaque (encrypted on backend). Frontend never sees plain viewing_light/viewing_dark. */
export function getViewingCookie(): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(
    new RegExp("(?:^|;\\s*)" + VIEWING_COOKIE + "=([^;]*)")
  );
  const value = match ? match[1].trim() : null;
  return value || null;
}

export function setViewingCookie(value: string): void {
  if (typeof document === "undefined") return;
  const maxAge = VIEWING_MAX_AGE_DAYS * 24 * 60 * 60;
  document.cookie = `${VIEWING_COOKIE}=${encodeURIComponent(value)}; path=/; max-age=${maxAge}; SameSite=Lax`;
}

/**
 * Raw ?viewing= param (opaque token). Backend decrypts; frontend does not interpret.
 */
export function getViewingFromSearchParams(
  searchParams: URLSearchParams | Readonly<URLSearchParams>
): string | null {
  const raw = searchParams.get("viewing");
  return raw ? raw.trim() : null;
}

/** Build share URL with encrypted viewing param (fetched from backend). */
export async function buildShareUrl(
  baseUrl: string,
  locale: string,
  contentId: number,
  category: "light" | "dark"
): Promise<string> {
  const token = await fetchViewingToken(category);
  const path = `/${locale}/post/${contentId}`;
  return `${baseUrl}${path}?viewing=${encodeURIComponent(token)}`;
}

/** Build home page share URL with encrypted viewing_dark param (premium tab; can_view_nsfw via cookie; sign up required for purchase/membership, dark account only). */
export async function buildHomeShareUrl(
  baseUrl: string,
  locale: string,
  category: "light" | "dark"
): Promise<string> {
  const token = await fetchViewingToken(category);
  const path = `/${locale}`;
  return `${baseUrl}${path}?viewing=${encodeURIComponent(token)}`;
}
