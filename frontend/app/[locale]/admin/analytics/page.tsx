"use client";

import { fetchCurrentUser, getAccessToken } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

export default function AdminAnalyticsIndexPage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("admin.analytics");
  const [allowed, setAllowed] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!getAccessToken()) {
        setAllowed(false);
        return;
      }
      const user = await fetchCurrentUser();
      if (cancelled) return;
      const hasPermission = (user?.permissions ?? []).includes("can_view_analytics");
      setAllowed(!!hasPermission);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (allowed === false) {
      router.replace(`/${locale}`);
    }
  }, [allowed, locale, router]);

  if (allowed === null) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[var(--background)]">
        <p className="text-[var(--muted)]">{t("loading")}</p>
      </div>
    );
  }

  if (allowed === false) return null;

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto max-w-2xl px-4 py-8">
        <div className="mb-6 flex items-center gap-4">
          <Link
            href={`/${locale}`}
            className="rounded-lg border border-[var(--border)] px-3 py-2 text-sm hover:bg-[var(--hover)]"
          >
            ← {t("back")}
          </Link>
          <h1 className="text-2xl font-bold">{t("title")}</h1>
        </div>
        <p className="mb-6 text-[var(--muted)]">{t("subtitle")}</p>
        <div className="grid gap-4 sm:grid-cols-2">
          <Link
            href={`/${locale}/admin/analytics/video`}
            className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-6 transition hover:border-[var(--accent)] hover:bg-[var(--hover)]"
          >
            <h2 className="text-lg font-semibold">{t("videoAnalytics")}</h2>
            <p className="mt-1 text-sm text-[var(--muted)]">{t("videoAnalyticsDesc")}</p>
          </Link>
          <Link
            href={`/${locale}/admin/analytics/revenue`}
            className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-6 transition hover:border-[var(--accent)] hover:bg-[var(--hover)]"
          >
            <h2 className="text-lg font-semibold">{t("revenueAnalytics")}</h2>
            <p className="mt-1 text-sm text-[var(--muted)]">{t("revenueAnalyticsDesc")}</p>
          </Link>
        </div>
      </div>
    </div>
  );
}
