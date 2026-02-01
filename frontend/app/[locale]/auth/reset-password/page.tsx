"use client";

import { useState, useEffect, Suspense } from "react";
import Link from "next/link";
import { useRouter, useParams, useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { PasswordInput } from "@/components/PasswordInput";
import { authApiResetPassword } from "@/lib/api";

function ResetPasswordForm() {
  const router = useRouter();
  const params = useParams();
  const searchParams = useSearchParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const token = searchParams.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!token) setError(t("errorMissingToken"));
  }, [token, t]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (password.length < 8) {
      setError(t("errorPasswordMin"));
      return;
    }
    if (password !== confirm) {
      setError(t("errorPasswordMismatch"));
      return;
    }
    setLoading(true);
    try {
      await authApiResetPassword(token, password);
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setLoading(false);
    }
  }

  if (done) {
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
          {t("resetPassword")}
        </h1>
        <p className="mt-4 text-[var(--muted)]">{t("successReset")}</p>
        <Link
          href={`/${locale}/auth/signin`}
          className="mt-6 inline-block rounded-full bg-[var(--accent)] px-6 py-3 font-bold text-[var(--accent-foreground)] hover:opacity-90"
        >
          {t("signIn")}
        </Link>
      </>
    );
  }

  if (!token) {
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
          {t("resetPasswordTitle")}
        </h1>
        <p className="mt-4 text-red-600 dark:text-red-400">{error}</p>
        <Link
          href={`/${locale}/auth/forgot-password`}
          className="mt-6 inline-block font-medium text-[var(--accent)] hover:underline"
        >
          {t("requestNewLink")}
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
        {t("resetPasswordTitle")}
      </h1>
      <form onSubmit={handleSubmit} className="mt-8 space-y-4">
        {error && (
          <p className="rounded-lg bg-red-500/10 px-3 py-2 text-sm text-red-600 dark:text-red-400">
            {error}
          </p>
        )}
        <PasswordInput
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={t("newPasswordPlaceholder")}
          required
          minLength={8}
          autoComplete="new-password"
        />
        <PasswordInput
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          placeholder={t("confirmPassword")}
          required
          minLength={8}
          autoComplete="new-password"
        />
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {loading ? "..." : t("submitReset")}
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

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <div className="text-center text-[var(--muted)]">Loading...</div>
      }
    >
      <ResetPasswordForm />
    </Suspense>
  );
}
