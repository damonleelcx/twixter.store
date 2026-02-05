"use client";

import { ThemeSwitcher } from "@/app/[locale]/ThemeSwitcher";
import { useLocale, useTranslations } from "next-intl";
import Link from "next/link";
import { Icon } from "./Icon";
import { LanguageSwitcher } from "./LanguageSwitcher";

export type FeedTab = "forYou" | "premium";

type MainFeedHeaderProps = {
  activeTab: FeedTab;
  onTabChange: (tab: FeedTab) => void;
};

const PROFILE_ICON =
  "M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z";

export function MainFeedHeader({ activeTab, onTabChange }: MainFeedHeaderProps) {
  const locale = useLocale();
  const tFeed = useTranslations("feed");
  const tNav = useTranslations("nav");

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
      <div className="flex">
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
      </div>
    </header>
  );
}
