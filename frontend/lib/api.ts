const AUTH_TOKEN_KEY = "twixter_access_token";
const REFRESH_TOKEN_KEY = "twixter_refresh_token";

/** Prefer relative /api so Next.js rewrites proxy to backend; set NEXT_PUBLIC_API_URL for direct backend URL (e.g. production). */
export function getApiBase(): string {
  if (typeof window !== "undefined") {
    const base = process.env.NEXT_PUBLIC_API_URL || "";
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

export async function authApiRegister(body: {
  email: string;
  password: string;
  username?: string;
  referral_code?: string;
  viewing: "viewing_light" | "viewing_dark";
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

/** Create membership checkout session. months: 1, 3, or 9. Returns clientSecret for Stripe embedded checkout. */
export async function createMembershipCheckout(months: 1 | 3 | 9): Promise<{
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
