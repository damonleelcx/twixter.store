"use client";

import { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { authApiRequestPasswordReset } from "@/lib/api";

export default function ForgotPasswordPage() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await authApiRequestPasswordReset(email);
      setSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setLoading(false);
    }
  }

  if (sent) {
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
          {t("forgotPasswordTitle")}
        </h1>
        <p className="mt-4 text-[var(--muted)]">
          {t("forgotPasswordSuccess", { email })}
        </p>
        <Link
          href={`/${locale}/auth/signin`}
          className="mt-6 inline-block font-medium text-[var(--accent)] hover:underline"
        >
          {t("backToSignIn")}
        </Link>
      </>
    );
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
        {t("forgotPasswordTitle")}
      </h1>
      <p className="mb-6 text-[var(--muted)]">
        {t("forgotPasswordSubtitle")}
      </p>
      <form onSubmit={handleSubmit} className="space-y-4">
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
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {loading ? "..." : t("submitSendLink")}
        </button>
      </form>
      <Link
        href={`/${locale}/auth/signin`}
        className="mt-6 block text-center text-sm font-medium text-[var(--accent)] hover:underline"
      >
        {t("backToSignIn")}
      </Link>
    </>
  );
}
