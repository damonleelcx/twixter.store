"use client";

import { PasswordInput } from "@/components/PasswordInput";
import { authApiRegister, setAccessToken, setRefreshToken } from "@/lib/api";
import {
  getViewingCookie,
  getViewingFromSearchParams,
  setViewingCookie,
} from "@/lib/viewing";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

export function SignUpContent() {
  const router = useRouter();
  const params = useParams();
  const searchParams = useSearchParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [referralCode, setReferralCode] = useState("");
  const [viewingToken, setViewingToken] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  // Resolve viewing token from URL or cookie (opaque; backend decrypts). Persist to cookie for registration.
  useEffect(() => {
    const fromUrl = getViewingFromSearchParams(searchParams);
    const fromCookie = getViewingCookie();
    const token = fromUrl ?? fromCookie;
    setViewingToken(token);
    if (token) setViewingCookie(token);
  }, [searchParams]);

  // Prefill referral code from ?ref= (e.g. from parent's shareable profile link)
  useEffect(() => {
    const ref = searchParams.get("ref");
    if (ref && typeof ref === "string") setReferralCode(ref.trim());
  }, [searchParams]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (password.length < 8) {
      setError(t("errorPasswordMin"));
      return;
    }
    setLoading(true);
    try {
      const body: Parameters<typeof authApiRegister>[0] = {
        email,
        password,
        username: username.trim() || undefined,
        referral_code: referralCode.trim() || undefined,
      };
      if (viewingToken) {
        body.viewing_token = viewingToken;
      } else {
        body.viewing = "viewing_light";
      }
      const { session } = await authApiRegister(body);
      setAccessToken(session.access_token);
      if (session.refresh_token) setRefreshToken(session.refresh_token);
      router.push(`/${locale}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="mb-8 flex justify-center">
        <Link
          href={`/${locale}`}
          className="text-4xl font-bold text-[var(--accent)]"
        >
          𝕋
        </Link>
      </div>
      <h1 className="mb-2 text-3xl font-bold tracking-tight text-[var(--foreground)]">
        {t("createAccount")}
      </h1>
      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        {error && (
          <p className="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-600 dark:text-red-400">
            {error}
          </p>
        )}
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={t("emailPlaceholder")}
          required
          autoComplete="email"
          className="w-full rounded-2xl border border-[var(--border)] bg-transparent px-4 py-3 text-[var(--foreground)] placeholder:text-[var(--muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
        />
        <PasswordInput
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={t("passwordPlaceholder")}
          required
          minLength={8}
          autoComplete="new-password"
        />
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder={t("usernamePlaceholder")}
          autoComplete="username"
          className="w-full rounded-2xl border border-[var(--border)] bg-transparent px-4 py-3 text-[var(--foreground)] placeholder:text-[var(--muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
        />
        <input
          type="text"
          value={referralCode}
          onChange={(e) => setReferralCode(e.target.value)}
          placeholder={t("referralCode")}
          className="w-full rounded-2xl border border-[var(--border)] bg-transparent px-4 py-3 text-[var(--foreground)] placeholder:text-[var(--muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
        />
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {loading ? "..." : t("signUp")}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-[var(--muted)]">
        {t("hasAccount")}{" "}
        <Link
          href={`/${locale}/auth/signin`}
          className="font-medium text-[var(--accent)] hover:underline"
        >
          {t("signIn")}
        </Link>
      </p>
      <p className="mt-4 text-center text-xs text-[var(--muted)]">
        {t("agreeToTerms")}{" "}
        <Link
          href={`/${locale}/terms`}
          className="font-medium text-[var(--accent)] hover:underline"
        >
          {t("termsOfUse")}
        </Link>
        {" "}{t("andPrivacyPolicy")}{" "}
        <Link
          href={`/${locale}/privacy`}
          className="font-medium text-[var(--accent)] hover:underline"
        >
          {t("privacyPolicy")}
        </Link>
      </p>
    </>
  );
}
