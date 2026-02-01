"use client";

import {
  fetchAdminVideoAnalytics,
  fetchCurrentUser,
  getAccessToken,
} from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

function formatSeconds(s: number): string {
  if (s < 60) return `${Math.round(s)}s`;
  const m = Math.floor(s / 60);
  const sec = Math.round(s % 60);
  if (m < 60) return `${m}m ${sec}s`;
  const h = Math.floor(m / 60);
  const min = m % 60;
  return `${h}h ${min}m ${sec}s`;
}

export default function AdminVideoAnalyticsPage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("admin.analytics");
  const [allowed, setAllowed] = useState<boolean | null>(null);
  const [data, setData] = useState<Awaited<ReturnType<typeof fetchAdminVideoAnalytics>> | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const formatYMD = (d: Date) =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const now = new Date();
  const [startDate, setStartDate] = useState(() =>
    formatYMD(new Date(now.getFullYear(), now.getMonth(), 1))
  );
  const [endDate, setEndDate] = useState(() =>
    formatYMD(new Date(now.getFullYear(), now.getMonth() + 1, 0))
  );

  const load = useCallback(async () => {
    if (!getAccessToken()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetchAdminVideoAnalytics(startDate || undefined, endDate || undefined);
      setData(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate]);

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

  useEffect(() => {
    if (allowed === true) load();
  }, [allowed, load]);

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
      <div className="mx-auto max-w-4xl px-4 py-8">
        <div className="mb-6 flex flex-wrap items-center gap-4">
          <Link
            href={`/${locale}/admin/analytics`}
            className="rounded-lg border border-[var(--border)] px-3 py-2 text-sm hover:bg-[var(--hover)]"
          >
            ← {t("back")}
          </Link>
          <h1 className="text-2xl font-bold">{t("videoAnalytics")}</h1>
        </div>

        <div className="mb-6 flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1">
            <span className="text-sm text-[var(--muted)]">{t("startDate")}</span>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)]"
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm text-[var(--muted)]">{t("endDate")}</span>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-[var(--foreground)]"
            />
          </label>
          <button
            type="button"
            onClick={load}
            disabled={loading}
            className="rounded-lg bg-[var(--accent)] px-4 py-2 text-sm font-medium text-[var(--accent-foreground)] disabled:opacity-50 hover:opacity-90"
          >
            {loading ? t("loading") : t("apply")}
          </button>
        </div>

        {error && (
          <div
            className="mb-4 rounded-lg border border-red-500/50 bg-red-500/10 px-4 py-3 text-sm text-red-600 dark:text-red-400"
            role="alert"
          >
            {error}
          </div>
        )}

        {data && (
          <>
            <div className="mb-6 grid gap-4 sm:grid-cols-3">
              <div className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-4">
                <p className="text-sm text-[var(--muted)]">{t("totalViews")}</p>
                <p className="text-2xl font-bold">{data.total_views.toLocaleString()}</p>
              </div>
              <div className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-4">
                <p className="text-sm text-[var(--muted)]">{t("totalPlayCount")}</p>
                <p className="text-2xl font-bold">{data.total_play_count.toLocaleString()}</p>
              </div>
              <div className="rounded-xl border border-[var(--border)] bg-[var(--background)] p-4">
                <p className="text-sm text-[var(--muted)]">{t("totalWatchTime")}</p>
                <p className="text-2xl font-bold">{formatSeconds(data.total_watch_time)}</p>
              </div>
            </div>
            <div className="rounded-xl border border-[var(--border)] bg-[var(--background)] overflow-hidden">
              <h2 className="border-b border-[var(--border)] px-4 py-3 font-semibold">{t("topContents")}</h2>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-[var(--border)] bg-[var(--hover)]">
                      <th className="px-4 py-2 text-left font-medium">{t("content")}</th>
                      <th className="px-4 py-2 text-right font-medium">{t("views")}</th>
                      <th className="px-4 py-2 text-right font-medium">{t("plays")}</th>
                      <th className="px-4 py-2 text-right font-medium">{t("watchTime")}</th>
                      <th className="px-4 py-2 text-right font-medium">{t("revenue")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.top_contents.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-4 py-6 text-center text-[var(--muted)]">
                          {t("noData")}
                        </td>
                      </tr>
                    ) : (
                      data.top_contents.map((row) => (
                        <tr key={row.content_id} className="border-b border-[var(--border)] last:border-0">
                          <td className="px-4 py-2">
                            <Link
                              href={`/${locale}/post/${row.content_id}`}
                              className="text-[var(--accent)] hover:underline"
                            >
                              {row.name || `#${row.content_id}`}
                            </Link>
                          </td>
                          <td className="px-4 py-2 text-right">{row.total_views.toLocaleString()}</td>
                          <td className="px-4 py-2 text-right">{row.total_play_count.toLocaleString()}</td>
                          <td className="px-4 py-2 text-right">{formatSeconds(row.total_watch_time)}</td>
                          <td className="px-4 py-2 text-right">{row.total_revenue.toFixed(2)}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
            <p className="mt-2 text-xs text-[var(--muted)]">
              {t("period")}: {data.start_date} — {data.end_date}
            </p>
          </>
        )}
      </div>
    </div>
  );
}
