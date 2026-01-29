"use client";

import { ThemeSwitcher } from "@/app/[locale]/ThemeSwitcher";
import { useTranslations } from "next-intl";
import { LanguageSwitcher } from "./LanguageSwitcher";

export type FeedTab = "forYou" | "premium";

type MainFeedHeaderProps = {
  activeTab: FeedTab;
  onTabChange: (tab: FeedTab) => void;
};

export function MainFeedHeader({ activeTab, onTabChange }: MainFeedHeaderProps) {
  const tFeed = useTranslations("feed");

  return (
    <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[var(--background)]/80 backdrop-blur-md">
      <div className="flex items-center justify-between px-4 py-2 md:justify-start">
        <h1 className="text-xl font-bold">{tFeed("home")}</h1>
        <div className="flex items-center gap-2 md:hidden">
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
