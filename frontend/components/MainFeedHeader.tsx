"use client";

import { ThemeSwitcher } from "@/app/[locale]/ThemeSwitcher";
import { useTranslations } from "next-intl";
import { LanguageSwitcher } from "./LanguageSwitcher";

export function MainFeedHeader() {
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
        <a
          href="#"
          className="flex-1 py-4 text-center font-semibold text-[var(--foreground)] underline decoration-[var(--accent)] decoration-2 underline-offset-8"
        >
          {tFeed("forYou")}
        </a>
        <a
          href="#"
          className="flex-1 py-4 text-center font-semibold text-[var(--muted)] hover:bg-[var(--hover)]"
        >
          {tFeed("following")}
        </a>
      </div>
    </header>
  );
}
