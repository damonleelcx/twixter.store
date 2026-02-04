const AUTH_TOKEN_KEY = "twixter_access_token";
const REFRESH_TOKEN_KEY = "twixter_refresh_token";
/** Cookie name for access token (used for SSR; set when client stores token). */
export const AUTH_TOKEN_COOKIE = "twixter_token";
const COOKIE_MAX_AGE = 60 * 60 * 24 * 7; // 7 days

/** Prefer relative /api so Next.js rewrites proxy to backend; set NEXT_PUBLIC_API_URL for direct backend URL (e.g. production). No trailing slash. */
export function getApiBase(): string {
  if (typeof window !== "undefined") {
    // On HTTPS pages always use relative /api to avoid mixed content (browser warning)
    if (window.location.protocol === "https:") return "/api";
    const raw = process.env.NEXT_PUBLIC_API_URL || "";
    const base = raw.replace(/\/+$/, "");
    return base ? `${base}/api` : "/api";
  }
  return "/api";
}

function authErrorFromResponse(data: { error?: string; details?: string }, fallback: string): string {
  const msg = data?.error || fallback;
  const details = data?.details;
  return details ? `${msg}: ${details}` : msg;
}

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(AUTH_TOKEN_KEY);
}

export function setAccessToken(token: string): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(AUTH_TOKEN_KEY, token);
  document.cookie = `${AUTH_TOKEN_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=${COOKIE_MAX_AGE}; SameSite=Lax`;
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setRefreshToken(token: string): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(REFRESH_TOKEN_KEY, token);
}

export function clearAccessToken(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(AUTH_TOKEN_KEY);
}

export function clearRefreshToken(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

/** Clear both access and refresh tokens (e.g. after refresh failed or logout). */
export function clearTokens(): void {
  clearAccessToken();
  clearRefreshToken();
  if (typeof window !== "undefined") {
    document.cookie = `${AUTH_TOKEN_COOKIE}=; path=/; max-age=0`;
  }
}

export function getAuthHeaders(): HeadersInit {
  const token = getAccessToken();
  const headers: HeadersInit = { "Content-Type": "application/json" };
  if (token) {
    (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }
  return headers;
}

/** Call refresh API and update stored access token. Throws on failure. */
export async function authApiRefresh(): Promise<{ access_token: string; refresh_token?: string }> {
  const base = getApiBase();
  const refreshToken = getRefreshToken();
  if (!refreshToken) throw new Error("No refresh token");
  const res = await fetch(`${base}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    clearTokens();
    throw new Error(authErrorFromResponse(data, "Token refresh failed"));
  }
  const session = data?.session;
  if (!session?.access_token) {
    clearTokens();
    throw new Error("Invalid refresh response");
  }
  setAccessToken(session.access_token);
  if (session.refresh_token) setRefreshToken(session.refresh_token);
  return session;
}

/** Single in-flight refresh promise so multiple 401s trigger only one refresh. Resolves to new access_token. */
let refreshPromise: Promise<string> | null = null;

/**
 * Fetch with Bearer token. On 401, tries to refresh (deduped) and retries the request once
 * using the new access token from the refresh response (not localStorage) to avoid races.
 * If refresh fails or no refresh token, clears tokens and throws.
 */
async function fetchWithAuth(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const options: RequestInit = {
    ...init,
    headers: { ...getAuthHeaders(), ...init?.headers },
  };
  let res = await fetch(input, options);
  if (res.status === 401 && getRefreshToken()) {
    try {
      if (!refreshPromise) {
        refreshPromise = authApiRefresh()
          .then((session) => session.access_token)
          .finally(() => {
            refreshPromise = null;
          });
      }
      const pendingRefresh = refreshPromise;
      const newAccessToken = await pendingRefresh;
      res = await fetch(input, {
        ...init,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${newAccessToken}`,
          ...init?.headers,
        },
      });
    } catch {
      refreshPromise = null;
      throw new Error("Session expired. Please sign in again.");
    }
  }
  return res;
}

export type CurrentUser = {
  id: number;
  email: string;
  username: string;
  account_type: string;
  email_verified: boolean;
  referral_code: string;
  created_at: string;
  permissions?: string[];
  wallet_balance?: number;
  membership_status?: "active" | "none";
  membership_expires_at?: string | null;
};

export async function fetchCurrentUser(): Promise<CurrentUser | null> {
  const base = getApiBase();
  if (!getAccessToken() && !getRefreshToken()) return null;
  try {
    const res = await fetchWithAuth(`${base}/auth/me`);
    if (!res.ok) return null;
    const data = await res.json();
    return data?.user ?? null;
  } catch {
    return null;
  }
}

/** Fetch encrypted viewing token for URL/cookie (mode=light|dark). Backend-only decrypts. */
export async function fetchViewingToken(mode: "light" | "dark"): Promise<string> {
  const base = getApiBase();
  const res = await fetch(`${base}/content/viewing-token?mode=${mode}`);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || "Failed to get viewing token");
  return typeof data?.token === "string" ? data.token : "";
}

export async function authApiRegister(body: {
  email: string;
  password: string;
  username?: string;
  referral_code?: string;
  viewing?: "viewing_light" | "viewing_dark";
  viewing_token?: string;
}): Promise<{ user: unknown; session: { access_token: string; refresh_token?: string } }> {
  const base = getApiBase();
  const res = await fetch(`${base}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "Registration failed"));
  return data;
}

export async function authApiLogin(body: {
  email: string;
  password: string;
}): Promise<{ user: unknown; session: { access_token: string; refresh_token?: string } }> {
  const base = getApiBase();
  const res = await fetch(`${base}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "Login failed"));
  return data;
}

export async function authApiRequestPasswordReset(email: string): Promise<void> {
  const base = getApiBase();
  const res = await fetch(`${base}/auth/password/reset/request`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(authErrorFromResponse(data, "Request failed"));
  }
}

export async function authApiResetPassword(
  token: string,
  new_password: string
): Promise<void> {
  const base = getApiBase();
  const res = await fetch(`${base}/auth/password/reset`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token, new_password }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "Reset failed"));
}

/** Verify checkout session (call after Stripe redirect). Triggers backend to apply credits/membership if paid. */
export async function verifyCheckout(sessionId: string): Promise<{
  status: string;
  payment_status: string;
  payment_intent_id?: string;
  payment_intent_status?: string;
}> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/verify`, {
    method: "POST",
    body: JSON.stringify({ session_id: sessionId }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "Verify failed"));
  return data;
}

/** Create credits checkout session. Returns clientSecret for Stripe embedded checkout. */
export async function createCreditsCheckout(amount: number, credits: number): Promise<{
  clientSecret: string;
  url?: string;
  checkout_session_id?: string;
}> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/credits/checkout`, {
    method: "POST",
    body: JSON.stringify({ amount, credits }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "Checkout failed"));
  return data;
}

/** Create membership checkout session. months: 1, 3, or 6. Returns clientSecret for Stripe embedded checkout. */
export async function createMembershipCheckout(months: 1 | 3 | 6): Promise<{
  clientSecret: string;
  url?: string;
  checkout_session_id?: string;
}> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/membership/checkout`, {
    method: "POST",
    body: JSON.stringify({ months }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "Membership checkout failed"));
  return data;
}

/** Create PayPal order for membership (backup payment). months: 1, 3, or 6. Returns PayPal order id. */
export async function createPayPalMembershipOrder(months: 1 | 3 | 6): Promise<{ id: string }> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/membership/paypal/order`, {
    method: "POST",
    body: JSON.stringify({ months }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "PayPal order failed"));
  return data;
}

/** Create PayPal order for credits (backup payment). Returns PayPal order id. */
export async function createPayPalCreditsOrder(amount: number, credits: number): Promise<{ id: string }> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/credits/paypal/order`, {
    method: "POST",
    body: JSON.stringify({ amount, credits }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "PayPal credits order failed"));
  return data;
}

/** Error message substring indicating insufficient credits (redirect user to credits page). */
export const INSUFFICIENT_CREDITS_KEY = "insufficient credits";

/** Purchase content with credits. Requires auth. */
export async function purchaseContent(contentId: number): Promise<{ message?: string }> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/content`, {
    method: "POST",
    body: JSON.stringify({ content_id: contentId }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = authErrorFromResponse(data, "Purchase failed");
    throw new Error(msg);
  }
  return data;
}

/** Returns true if the error is due to insufficient credits. */
export function isInsufficientCreditsError(err: unknown): boolean {
  const message = err instanceof Error ? err.message : String(err);
  return message.toLowerCase().includes(INSUFFICIENT_CREDITS_KEY);
}

/** Capture PayPal order after buyer approval. Completes payment (membership or credits). */
export async function capturePayPalOrder(orderID: string): Promise<{ status: string }> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/purchase/membership/paypal/capture`, {
    method: "POST",
    body: JSON.stringify({ orderID }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(authErrorFromResponse(data, "PayPal capture failed"));
  return data;
}

/** Call backend logout to revoke current session. Safe to call even without token. */
export async function authApiLogout(): Promise<void> {
  const base = getApiBase();
  try {
    const res = await fetchWithAuth(`${base}/auth/logout`, { method: "POST" });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(authErrorFromResponse(data, "Logout failed"));
    }
  } finally {
    clearTokens();
  }
}

export type TagOption = { id: number; name: string; slug: string };

/** Content feed item from list API (author, timestamp, tags, purchased, bookmarked, preview gif). */
export type ContentFeedItem = {
  id: number;
  name: string;
  description: string;
  category: string;
  price: number;
  preview_gif_url: string;
  /** 列表预取的模糊 GIF（base64），有则直接用作 data URL，无需请求 /preview */
  preview_gif_base64?: string;
  first_file_id: number;
  purchased: boolean;
  bookmarked?: boolean;
  created_at: string;
  author_username: string;
  author_avatar?: string;
  tags?: string[];
};

/** Content detail from GET /content/:id (with files, purchased, bookmarked, price, author). */
export type ContentDetail = {
  content: {
    id: number;
    name: string;
    description: string;
    type: string;
    status: string;
    category: string;
    price: number;
    created_at: string;
    purchased: boolean;
    bookmarked?: boolean;
    author_username?: string;
    author_avatar?: string;
    tags?: string[];
  };
  files: Array<{
    id: number;
    file_name: string;
    file_index: number;
    original_file_url: string;
    gif_file_url: string;
    transcoded_file_url: string;
    stage: string;
    width?: number;
    height?: number;
    duration?: number;
  }>;
};

/** Get content by id. Auth or viewing_dark cookie for dark. Optional viewing param for unauthenticated dark access. */
export async function getContentById(
  id: number,
  viewingToken?: string | null,
  options?: { cache?: RequestCache }
): Promise<ContentDetail> {
  const base = getApiBase();
  const url = viewingToken ? `${base}/content/${id}?viewing=${encodeURIComponent(viewingToken)}` : `${base}/content/${id}`;
  const res = await fetchWithAuth(url, { cache: options?.cache });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error || data?.details || res.statusText || "Content not found");
  }
  return data as ContentDetail;
}

/** Server-only: get content by id with optional token and viewing param. Returns null on 404/401/error. */
export async function getContentByIdServer(
  id: number,
  token: string | undefined,
  viewingToken?: string | null
): Promise<ContentDetail | null> {
  const base = getServerApiBase();
  if (!base) return null;
  const headers: HeadersInit = { "Content-Type": "application/json" };
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  const url = viewingToken ? `${base}/content/${id}?viewing=${encodeURIComponent(viewingToken)}` : `${base}/content/${id}`;
  const res = await fetch(url, { headers, cache: "no-store" });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) return null;
  return data as ContentDetail;
}

/** Admin video analytics (requires can_view_analytics). */
export type AdminVideoAnalytics = {
  start_date: string;
  end_date: string;
  total_views: number;
  total_play_count: number;
  total_watch_time: number;
  top_contents: Array<{
    content_id: number;
    name: string;
    total_views: number;
    total_play_count: number;
    total_watch_time: number;
    total_revenue: number;
  }>;
};

export async function fetchAdminVideoAnalytics(
  startDate?: string,
  endDate?: string
): Promise<AdminVideoAnalytics> {
  const base = getApiBase();
  const params = new URLSearchParams();
  if (startDate) params.set("start_date", startDate);
  if (endDate) params.set("end_date", endDate);
  const q = params.toString();
  const url = q ? `${base}/admin/analytics/video?${q}` : `${base}/admin/analytics/video`;
  const res = await fetchWithAuth(url);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || data?.details || "Failed to load video analytics");
  return data as AdminVideoAnalytics;
}

/** Admin revenue analytics (requires can_view_analytics). */
export type AdminRevenueAnalytics = {
  start_date: string;
  end_date: string;
  content_revenue: number;
  credits_revenue: number;
  total_revenue: number;
};

export async function fetchAdminRevenueAnalytics(
  startDate?: string,
  endDate?: string
): Promise<AdminRevenueAnalytics> {
  const base = getApiBase();
  const params = new URLSearchParams();
  if (startDate) params.set("start_date", startDate);
  if (endDate) params.set("end_date", endDate);
  const q = params.toString();
  const url = q ? `${base}/admin/analytics/revenue?${q}` : `${base}/admin/analytics/revenue`;
  const res = await fetchWithAuth(url);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || data?.details || "Failed to load revenue analytics");
  return data as AdminRevenueAnalytics;
}

/** Update content metadata (requires can_edit_content). */
export async function updateContent(
  contentId: number,
  body: { name: string; description: string; category: "light" | "dark"; price: number; tags: string[] }
): Promise<void> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/content/${contentId}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || data?.details || "Failed to update content");
}

/** Delete content (requires can_delete_content, admin only). */
export async function deleteContent(contentId: number): Promise<void> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/content/${contentId}`, { method: "DELETE" });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || data?.details || "Failed to delete content");
}

/** Read access token from cookie string (for server: pass request cookies). */
export function getAccessTokenFromCookie(cookieHeader: string | null): string | undefined {
  if (!cookieHeader) return undefined;
  const match = cookieHeader.match(new RegExp(`${AUTH_TOKEN_COOKIE}=([^;]+)`));
  if (!match) return undefined;
  try {
    return decodeURIComponent(match[1].trim());
  } catch {
    return undefined;
  }
}

/** Response shape for content feed list (shared by client and server fetch). */
export type ContentFeedResponse = {
  items: ContentFeedItem[];
  next_cursor: number;
  has_more: boolean;
};

/** Fetch content feed by category. Optional auth (token sent when available). Dark requires can_view_nsfw. */
export async function fetchContentFeed(
  category: "light" | "dark",
  cursor: number,
  limit = 20
): Promise<ContentFeedResponse> {
  const base = getApiBase();
  const params = new URLSearchParams({
    category,
    cursor: String(cursor),
    limit: String(limit),
  });
  const headers: HeadersInit = {};
  const token = getAccessToken();
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${base}/content/list?${params}`, {
    headers,
    cache: "no-store",
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error || data?.details || res.statusText || "Failed to load feed");
  }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    next_cursor: typeof data?.next_cursor === "number" ? data.next_cursor : cursor + 1,
    has_more: Boolean(data?.has_more),
  };
}

/** Fetch content feed by tag name. Same response shape as fetchContentFeed. */
export async function fetchContentFeedByTag(
  tag: string,
  cursor: number,
  limit = 20
): Promise<ContentFeedResponse> {
  const base = getApiBase();
  const params = new URLSearchParams({
    tag: tag.trim(),
    cursor: String(cursor),
    limit: String(limit),
  });
  const headers: HeadersInit = {};
  const token = getAccessToken();
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${base}/content/list?${params}`, {
    headers,
    cache: "no-store",
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error || data?.details || res.statusText || "Failed to load feed");
  }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    next_cursor: typeof data?.next_cursor === "number" ? data.next_cursor : cursor + 1,
    has_more: Boolean(data?.has_more),
  };
}

/** Fetch content feed by search query (fuzzy name/description). category "" = all. */
export async function fetchContentFeedSearch(
  q: string,
  category: "light" | "dark" | "",
  cursor: number,
  limit = 20
): Promise<ContentFeedResponse> {
  const base = getApiBase();
  const params = new URLSearchParams({
    cursor: String(cursor),
    limit: String(limit),
  });
  if (q.trim()) params.set("q", q.trim());
  if (category) params.set("category", category);
  const headers: HeadersInit = {};
  const token = getAccessToken();
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${base}/content/list?${params}`, {
    headers,
    cache: "no-store",
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error || data?.details || res.statusText || "Failed to load feed");
  }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    next_cursor: typeof data?.next_cursor === "number" ? data.next_cursor : cursor + 1,
    has_more: Boolean(data?.has_more),
  };
}

/** Add bookmark for content. Requires auth. */
export async function addBookmark(contentId: number): Promise<void> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/content/${contentId}/bookmark`, {
    method: "POST",
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || "Failed to add bookmark");
}

/** Remove bookmark for content. Requires auth. */
export async function removeBookmark(contentId: number): Promise<void> {
  const base = getApiBase();
  const res = await fetchWithAuth(`${base}/content/${contentId}/bookmark`, {
    method: "DELETE",
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || "Failed to remove bookmark");
}

/** Record watch progress for analytics (TotalWatchTime, AverageWatchTime, CompletionRate). Requires auth. */
export async function recordWatchProgress(
  contentId: number,
  watchTimeSeconds: number,
  durationSeconds?: number
): Promise<void> {
  const base = getApiBase();
  const body: { watch_time_seconds: number; duration_seconds?: number } = {
    watch_time_seconds: watchTimeSeconds,
  };
  if (durationSeconds != null && durationSeconds > 0) body.duration_seconds = durationSeconds;
  const res = await fetchWithAuth(`${base}/content/${contentId}/watch-progress`, {
    method: "POST",
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error || "Failed to record watch progress");
}

/** Fetch user's bookmarked content (Bookmarks page). Requires auth. */
export async function fetchBookmarks(
  cursor: number,
  limit = 20
): Promise<ContentFeedResponse> {
  const base = getApiBase();
  const params = new URLSearchParams({
    cursor: String(cursor),
    limit: String(limit),
  });
  const res = await fetchWithAuth(`${base}/content/bookmarks?${params}`);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error || data?.details || res.statusText || "Failed to load bookmarks");
  }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    next_cursor: typeof data?.next_cursor === "number" ? data.next_cursor : cursor + 1,
    has_more: Boolean(data?.has_more),
  };
}

/** Fetch user's purchased content (Library). Requires auth. */
export async function fetchLibraryContent(
  cursor: number,
  limit = 20
): Promise<ContentFeedResponse> {
  const base = getApiBase();
  const params = new URLSearchParams({
    cursor: String(cursor),
    limit: String(limit),
  });
  const res = await fetchWithAuth(`${base}/content/library?${params}`);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data?.error || data?.details || res.statusText || "Failed to load library");
  }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    next_cursor: typeof data?.next_cursor === "number" ? data.next_cursor : cursor + 1,
    has_more: Boolean(data?.has_more),
  };
}

/** Server-only: base URL for API (absolute). Use in server components / server actions. */
/**
 * Server-only API base for fetch from Node (SSR). In K8s, set BACKEND_URL (e.g. http://twixter-backend:8080)
 * so the frontend pod can reach the backend; NEXT_PUBLIC_API_URL (e.g. api.twixter.local) is for the browser
 * and may not resolve inside the cluster.
 * When BACKEND_URL and NEXT_PUBLIC_API_URL are both unset (e.g. production without env), use NEXT_PUBLIC_APP_URL
 * so SSR fetches same-origin /api and Next.js rewrites proxy to the backend (avoids ECONNREFUSED on localhost).
 */
export function getServerApiBase(): string {
  if (typeof window !== "undefined") return "";
  const url =
    process.env.BACKEND_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    (process.env.NEXT_PUBLIC_APP_URL ? process.env.NEXT_PUBLIC_APP_URL.replace(/\/+$/, "") : "") ||
    "http://localhost:8080";
  const base = url.replace(/\/$/, "");
  return base.startsWith("http") ? `${base}/api` : `http://${base}/api`;
}

/** Server-only: fetch current user using token (e.g. from cookies().get(AUTH_TOKEN_COOKIE)?.value). Returns null if no token or API error. */
export async function fetchCurrentUserServer(
  token: string | undefined | null
): Promise<CurrentUser | null> {
  const base = getServerApiBase();
  if (!base || !token) return null;
  try {
    const res = await fetch(`${base}/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (!res.ok) return null;
    const data = await res.json();
    return data?.user ?? null;
  } catch {
    return null;
  }
}

/** Trending tag item from /api/tags/trending */
export type TrendingTagItem = { id: number; name: string; slug: string; post_count: number };

/** Server-only: fetch trending tags. category=light for light content only; category=all requires can_view_nsfw (pass token). */
export async function fetchTrendingTagsServer(
  category: "light" | "all",
  limit = 10,
  token?: string | null
): Promise<TrendingTagItem[]> {
  const base = getServerApiBase();
  if (!base) return [];
  const params = new URLSearchParams({ category, limit: String(limit) });
  const headers: HeadersInit = { "Content-Type": "application/json" };
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  try {
    const res = await fetch(`${base}/tags/trending?${params}`, { headers, cache: "no-store" });
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data?.tags) ? data.tags : [];
  } catch {
    return [];
  }
}

/** Server-side fetch for content feed. Pass token from cookie when user is signed in so purchased is correct. */
export async function fetchContentFeedServer(
  category: "light" | "dark",
  cursor: number,
  limit = 20,
  token?: string | null
): Promise<ContentFeedResponse> {
  const base = getServerApiBase();
  if (!base) return { items: [], next_cursor: 0, has_more: false };
  const params = new URLSearchParams({
    category,
    cursor: String(cursor),
    limit: String(limit),
  });
  const headers: HeadersInit = { "Content-Type": "application/json" };
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  const res = await fetch(`${base}/content/list?${params}`, {
    cache: "no-store",
    headers,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    return { items: [], next_cursor: 0, has_more: false };
  }
  return {
    items: Array.isArray(data?.items) ? data.items : [],
    next_cursor: typeof data?.next_cursor === "number" ? data.next_cursor : cursor + 1,
    has_more: Boolean(data?.has_more),
  };
}

export async function searchTags(
  q: string,
  limit = 20
): Promise<TagOption[]> {
  const base = getApiBase();
  if (!getAccessToken() && !getRefreshToken()) return [];
  try {
    const params = new URLSearchParams({ limit: String(limit) });
    if (q.trim()) params.set("q", q.trim());
    const res = await fetchWithAuth(`${base}/tags/search?${params}`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data?.tags) ? data.tags : [];
  } catch {
    return [];
  }
}

/** Client: fetch trending tags. category=all requires can_view_nsfw (use when user has permission). */
export async function fetchTrendingTags(
  category: "light" | "all",
  limit = 10
): Promise<TrendingTagItem[]> {
  const base = getApiBase();
  const params = new URLSearchParams({ category, limit: String(limit) });
  const headers: HeadersInit = {};
  const token = getAccessToken();
  if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  try {
    const res = await fetch(`${base}/tags/trending?${params}`, { headers, cache: "no-store" });
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data?.tags) ? data.tags : [];
  } catch {
    return [];
  }
}
