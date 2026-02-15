"use client";

import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import {
  authApiLogout,
  clearTokens,
  fetchCurrentUser,
  fetchViewingTokenForReferral,
  type CurrentUser,
} from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

export default function ProfilePage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const tNav = useTranslations("nav");
  const tFeed = useTranslations("feed");
  const [user, setUser] = useState<CurrentUser | null | undefined>(undefined);
  const [shareableUrl, setShareableUrl] = useState<string | null>(null);
  const [copySuccess, setCopySuccess] = useState(false);
  const [shareAlreadyClaimed, setShareAlreadyClaimed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled) setUser(u ?? null);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // Build shareable signup link with referral code + encrypted viewing token (same account type as parent). Once per month per user.
  useEffect(() => {
    if (!user?.referral_code) return;
    if (user.shareable_link_claimed) {
      setShareAlreadyClaimed(true);
      return;
    }
    const mode = user.account_type === "dark" ? "dark" : "light";
    let cancelled = false;
    fetchViewingTokenForReferral(mode)
      .then((token) => {
        if (cancelled || !token) return;
        const origin = typeof window !== "undefined" ? window.location.origin : "";
        const url = `${origin}/${locale}/auth/signup?ref=${encodeURIComponent(user.referral_code)}&viewing=${encodeURIComponent(token)}`;
        setShareableUrl(url);
      })
      .catch((err) => {
        if (cancelled) return;
        if (err instanceof Error && err.message === "already_shared") {
          setShareAlreadyClaimed(true);
        } else {
          setShareableUrl(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [user?.referral_code, user?.account_type, user?.shareable_link_claimed, locale]);

  const copyShareableLink = useCallback(() => {
    if (!shareableUrl) return;
    navigator.clipboard?.writeText(shareableUrl).then(() => {
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    });
  }, [shareableUrl]);

  async function handleLogOut() {
    try {
      await authApiLogout();
    } finally {
      clearTokens();
      router.push(`/${locale}`);
      router.refresh();
    }
  }

  if (user === undefined) {
    return (
      <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
        <div className="mx-auto flex max-w-[1280px]">
          <LeftSidebar locale={locale} />
          <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px] p-6">
            <p className="text-[var(--muted)]">{tFeed("loading")}</p>
          </main>
          <RightSidebar />
        </div>
        <MobileBottomNav />
      </div>
    );
  }

  if (user === null) {
    router.replace(`/${locale}/auth/signin`);
    return null;
  }

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <div className="border-b border-[var(--border)] p-4">
            <h1 className="text-xl font-bold">{tNav("profile")}</h1>
          </div>
          <div className="p-6 space-y-6">
            <div className="rounded-xl border border-[var(--border)] p-4 space-y-2">
              <p className="text-sm text-[var(--muted)]">{t("email")}</p>
              <p className="font-medium">{user.email}</p>
              {user.username && (
                <>
                  <p className="text-sm text-[var(--muted)] pt-2">{t("usernamePlaceholder")}</p>
                  <p className="font-medium">{user.username}</p>
                </>
              )}
              <div className="pt-2 border-t border-[var(--border)] mt-2">
                <p className="text-sm text-[var(--muted)]">{t("walletBalance")}</p>
                <p className="font-medium">{typeof user.wallet_balance === "number" ? user.wallet_balance : 0} {t("credits")}</p>
                <Link
                  href={`/${locale}/credits`}
                  className="mt-2 inline-block rounded-full bg-[var(--accent)] px-4 py-2 text-sm font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
                >
                  {t("buyCredits")}
                </Link>
              </div>
              <div className="pt-2 border-t border-[var(--border)] mt-2">
                <p className="text-sm text-[var(--muted)]">{t("membershipStatus")}</p>
                <p className="font-medium">
                  {user.membership_status === "active"
                    ? t("membershipActive")
                    : t("membershipNone")}
                  {user.membership_status === "active" && user.membership_expires_at && (
                    <span className="block text-sm font-normal text-[var(--muted)] mt-1">
                      {t("membershipExpiresAt")}: {new Date(user.membership_expires_at).toLocaleDateString()}
                    </span>
                  )}
                </p>
                <Link
                  href={`/${locale}/subscribe`}
                  className="mt-2 inline-block rounded-full border border-[var(--border)] px-4 py-2 text-sm font-bold text-[var(--foreground)] transition-opacity hover:opacity-90"
                >
                  {tNav("subscribe")}
                </Link>
              </div>
              <div className="pt-4 border-t border-[var(--border)] mt-4">
                <p className="text-sm font-medium text-[var(--foreground)]">{t("shareableProfileLink")}</p>
                <p className="text-sm text-[var(--muted)] mt-1">{t("shareableProfileHint")}</p>
                {user.shareable_link_claimed || shareAlreadyClaimed ? (
                  <p className="mt-2 text-sm text-[var(--muted)]">
                    {t("shareableLinkAlreadyClaimed")}
                    <span className="block mt-1 text-[var(--muted)]">{t("shareableLinkResetsNextMonth")}</span>
                  </p>
                ) : shareableUrl ? (
                  <div className="mt-2 flex gap-2">
                    <input
                      type="text"
                      readOnly
                      value={shareableUrl}
                      className="min-w-0 flex-1 rounded-lg border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-sm text-[var(--foreground)]"
                      aria-label={t("shareableProfileLink")}
                    />
                    <button
                      type="button"
                      onClick={copyShareableLink}
                      className="shrink-0 rounded-lg border border-[var(--border)] bg-[var(--background)] px-4 py-2 text-sm font-medium text-[var(--foreground)] hover:bg-[var(--hover)]"
                    >
                      {copySuccess ? t("copied") : t("copyLink")}
                    </button>
                  </div>
                ) : (
                  <p className="mt-2 text-sm text-[var(--muted)]">…</p>
                )}
              </div>
            </div>
            <button
              type="button"
              onClick={handleLogOut}
              className="w-full rounded-full border border-[var(--border)] py-3 text-center font-bold text-[var(--foreground)] transition-colors hover:bg-[var(--hover)]"
            >
              {t("logOut")}
            </button>
          </div>
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
