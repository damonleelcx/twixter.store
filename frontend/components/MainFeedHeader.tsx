"use client";

import { ThemeSwitcher } from "@/app/[locale]/ThemeSwitcher";
import { buildHomeShareUrl } from "@/lib/viewing";
import { useLocale, useTranslations } from "next-intl";
import Link from "next/link";
import { useState } from "react";
import { Icon } from "./Icon";
import { LanguageSwitcher } from "./LanguageSwitcher";

export type FeedTab = "forYou" | "premium";

export type MainFeedHeaderProps = {
  activeTab: FeedTab;
  onTabChange: (tab: FeedTab) => void;
  canViewNsfw?: boolean;
};

const PROFILE_ICON =
  "M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z";
const SHARE_ICON =
  "M4 12v8a2 2 0 002 2h12a2 2 0 002-2v-8M16 6l-4-4-4 4M12 2v13";

export function MainFeedHeader({ activeTab, onTabChange, canViewNsfw }: MainFeedHeaderProps) {
  const locale = useLocale();
  const tFeed = useTranslations("feed");
  const tNav = useTranslations("nav");
  const [shareStatus, setShareStatus] = useState<"idle" | "loading" | "copied" | "error">("idle");

  async function handleSharePremiumLink() {
    setShareStatus("loading");
    try {
      const baseUrl = typeof window !== "undefined" ? window.location.origin : "";
      const url = await buildHomeShareUrl(baseUrl, locale, "dark");
      if (navigator.share) {
        await navigator.share({
          title: "Premium content",
          text: tFeed("premiumContent"),
          url,
        });
        setShareStatus("copied");
      } else {
        await navigator.clipboard.writeText(url);
        setShareStatus("copied");
      }
      setTimeout(() => setShareStatus("idle"), 2000);
    } catch {
      setShareStatus("error");
      setTimeout(() => setShareStatus("idle"), 2000);
    }
  }

  return (
    <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[var(--background)]/80 backdrop-blur-md">
      <div className="flex items-center justify-between px-4 py-2 md:justify-start">
        <h1 className="text-xl font-bold">{tFeed("home")}</h1>
        <div className="flex items-center gap-2 md:hidden">
          <Link
            href={`/${locale}/profile`}
            className="rounded-full p-2 text-[var(--muted)] transition-colors hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
            aria-label={tNav("profile")}
          >
            <Icon d={PROFILE_ICON} className="h-5 w-5" />
          </Link>
          <ThemeSwitcher />
          <LanguageSwitcher />
        </div>
      </div>
      <div className="flex items-center">
        <button
          type="button"
          onClick={() => onTabChange("forYou")}
          className="flex-1 py-4 text-center font-semibold transition-colors hover:bg-[var(--hover)]"
          style={{
            color: activeTab === "forYou" ? "var(--foreground)" : "var(--muted)",
            textDecoration: activeTab === "forYou" ? "underline" : "none",
            textDecorationColor: activeTab === "forYou" ? "var(--accent)" : "transparent",
            textDecorationThickness: "2px",
            textUnderlineOffset: "8px",
          }}
        >
          {tFeed("forYou")}
        </button>
        <button
          type="button"
          onClick={() => onTabChange("premium")}
          className="flex-1 py-4 text-center font-semibold transition-colors hover:bg-[var(--hover)]"
          style={{
            color: activeTab === "premium" ? "var(--foreground)" : "var(--muted)",
            textDecoration: activeTab === "premium" ? "underline" : "none",
            textDecorationColor: activeTab === "premium" ? "var(--accent)" : "transparent",
            textDecorationThickness: "2px",
            textUnderlineOffset: "8px",
          }}
        >
          {tFeed("premiumContent")}
        </button>
        {activeTab === "premium" && canViewNsfw && (
          <button
            type="button"
            onClick={handleSharePremiumLink}
            disabled={shareStatus === "loading"}
            className="rounded-full p-2 text-[var(--muted)] transition-colors hover:bg-[var(--hover)] hover:text-[var(--foreground)] disabled:opacity-50"
            title={tFeed("sharePremiumLink")}
            aria-label={tFeed("sharePremiumLink")}
          >
            {shareStatus === "loading" ? (
              <span className="inline-block h-5 w-5 animate-spin rounded-full border-2 border-[var(--muted)] border-t-transparent" />
            ) : shareStatus === "copied" ? (
              <span className="text-xs text-[var(--accent)]">✓</span>
            ) : (
              <Icon d={SHARE_ICON} className="h-5 w-5" />
            )}
          </button>
        )}
      </div>
    </header>
  );
}
