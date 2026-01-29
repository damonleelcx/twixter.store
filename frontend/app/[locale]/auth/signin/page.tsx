"use client";

import { PasswordInput } from "@/components/PasswordInput";
import { authApiLogin, setAccessToken, setRefreshToken } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";

export default function SignInPage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const { session } = await authApiLogin({ email, password });
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
        {t("signIn")}
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
          autoComplete="current-password"
        />
        <Link
          href={`/${locale}/auth/forgot-password`}
          className="block text-sm text-[var(--accent)] hover:underline"
        >
          {t("forgotPassword")}
        </Link>
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {loading ? "..." : t("submit")}
        </button>
      </form>
      <p className="mt-6 text-center text-sm text-[var(--muted)]">
        {t("noAccount")}{" "}
        <Link
          href={`/${locale}/auth/signup`}
          className="font-medium text-[var(--accent)] hover:underline"
        >
          {t("signUp")}
        </Link>
      </p>
    </>
  );
}
