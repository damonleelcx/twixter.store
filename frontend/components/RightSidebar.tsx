"use client";

import {
  fetchCurrentUser,
  fetchTrendingTags,
  type CurrentUser,
  type TrendingTagItem,
} from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

type RightSidebarProps = {
  /** SSR 传入的热门标签；未传时在客户端按权限请求 */
  initialTrendingTags?: TrendingTagItem[] | null;
};

export function RightSidebar({ initialTrendingTags }: RightSidebarProps) {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const tSidebar = useTranslations("sidebar");
  const tAuth = useTranslations("auth");
  const [user, setUser] = useState<CurrentUser | null | undefined>(undefined);
  const [clientTrending, setClientTrending] = useState<TrendingTagItem[]>([]);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled) setUser(u ?? null);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // When server didn't provide tags (null) or returned empty (e.g. server-side fetch failed in K8s), fetch on client
  const shouldFetchTags =
    initialTrendingTags == null || (Array.isArray(initialTrendingTags) && initialTrendingTags.length === 0);
  useEffect(() => {
    if (!shouldFetchTags) return;
    let cancelled = false;
    const category = user === undefined ? "light" : (user?.permissions?.includes("can_view_nsfw") ? "all" : "light");
    fetchTrendingTags(category, 10).then((list) => {
      if (!cancelled) setClientTrending(list);
    });
    return () => {
      cancelled = true;
    };
  }, [shouldFetchTags, user]);

  const walletBalance = user != null && typeof user.wallet_balance === "number" ? user.wallet_balance : null;
  const hasActiveMembership = user != null && user.membership_status === "active";

  const trendingTags =
    initialTrendingTags != null && initialTrendingTags.length > 0 ? initialTrendingTags : clientTrending;

  return (
    <aside className="sticky top-0 hidden h-screen w-[350px] shrink-0 overflow-y-auto p-4 xl:block">
      {walletBalance !== null && (
        <div className="rounded-2xl bg-[var(--hover)] p-4 mb-4">
          <h2 className="text-sm font-medium text-[var(--muted)]">{tAuth("walletBalance")}</h2>
          <p className="text-xl font-bold mt-0.5">{walletBalance} {tAuth("credits")}</p>
          <Link
            href={`/${locale}/credits`}
            className="mt-2 block w-full rounded-full bg-[var(--accent)] py-2 text-center text-sm font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
          >
            {tAuth("buyCredits")}
          </Link>
        </div>
      )}
      <div className="rounded-2xl bg-[var(--hover)] p-4">
        {hasActiveMembership ? (
          <>
            <h2 className="text-xl font-bold">{tSidebar("premiumMember")}</h2>
            <p className="mt-1 text-sm text-[var(--muted)]">
              {tAuth("membershipStatus")}: {tAuth("membershipActive")}
            </p>
            {user?.membership_expires_at && (
              <p className="mt-0.5 text-sm text-[var(--muted)]">
                {tAuth("membershipExpiresAt")}: {new Date(user.membership_expires_at).toLocaleDateString()}
              </p>
            )}
          </>
        ) : (
          <>
            <h2 className="text-xl font-bold">{tSidebar("premiumTitle")}</h2>
            <p className="mt-1 text-sm text-[var(--muted)]">
              {tSidebar("premiumDescription")}
            </p>
            <Link
              href={`/${locale}/subscribe`}
              className="mt-3 block w-full rounded-full bg-[var(--accent)] py-2 text-center font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
            >
              {tSidebar("subscribe")}
            </Link>
          </>
        )}
      </div>
      <div className="mt-4 rounded-2xl bg-[var(--hover)] p-4">
        <h2 className="text-xl font-bold">{tSidebar("whatsHappening")}</h2>
        <div className="mt-3 space-y-3">
          {trendingTags.map((t) => (
            <Link
              key={t.id}
              href={`/${locale}/tag/${encodeURIComponent(t.slug || t.name)}`}
              className="block rounded-lg p-2 hover:bg-[var(--border)]/50"
            >
              <span className="text-sm text-[var(--muted)]">
                {tSidebar("trending")}
              </span>
              <p className="font-bold">#{t.name}</p>
              <span className="text-sm text-[var(--muted)]">
                {tSidebar("posts", { count: t.post_count })}
              </span>
            </Link>
          ))}
        </div>
      </div>
      <footer className="mt-6 flex flex-wrap gap-x-3 border-t border-[var(--border)] pt-4">
        <Link
          href={`/${locale}/terms`}
          className="text-xs text-[var(--muted)] hover:text-[var(--accent)] hover:underline"
        >
          {tSidebar("termsOfUse")}
        </Link>
        <Link
          href={`/${locale}/privacy`}
          className="text-xs text-[var(--muted)] hover:text-[var(--accent)] hover:underline"
        >
          {tSidebar("privacyPolicy")}
        </Link>
      </footer>
    </aside>
  );
}
