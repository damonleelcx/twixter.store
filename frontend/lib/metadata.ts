/**
 * Base URL for canonical and Open Graph metadata.
 * Always returns HTTPS so that metadata never causes "some parts are not secure" on HTTPS sites.
 */
export function getBaseUrlForMetadata(): string {
  let base: string;
  if (typeof process.env.NEXT_PUBLIC_APP_URL === "string" && process.env.NEXT_PUBLIC_APP_URL) {
    base = process.env.NEXT_PUBLIC_APP_URL.replace(/\/+$/, "");
  } else if (typeof process.env.VERCEL_URL === "string" && process.env.VERCEL_URL) {
    base = `https://${process.env.VERCEL_URL}`;
  } else {
    base = "https://twixter.store";
  }
  // Ensure HTTPS so canonical/og:url never trigger mixed content on HTTPS pages
  if (base.startsWith("http://")) {
    base = "https://" + base.slice(7);
  }
  return base;
}

/** Returns true if url is safe for og:image / meta tags (https or relative). */
export function isSecureUrlForMeta(url: string | undefined): boolean {
  if (!url || typeof url !== "string") return false;
  return url.startsWith("https://") || url.startsWith("/");
}

/**
 * 若 URL 无协议（如 S3 endpoint 只配了 host），补上 https://，避免被浏览器当相对路径请求到当前站导致 404。
 */
export function ensureAbsoluteUrl(url: string | undefined): string | undefined {
  if (!url || typeof url !== "string") return undefined;
  const t = url.trim();
  if (!t) return undefined;
  if (t.startsWith("http://") || t.startsWith("https://")) return t;
  if (t.startsWith("//")) return "https:" + t;
  return "https://" + t;
}
