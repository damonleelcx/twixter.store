"use client";

import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Icon } from "./Icon";

export function MobileBottomNav() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("nav");

  return (
    <>
      <nav className="fixed bottom-0 left-0 right-0 z-20 flex items-center justify-around border-t border-[var(--border)] bg-[var(--background)] safe-bottom md:hidden">
        <Link
          href={`/${locale}`}
          className="flex flex-col items-center gap-1 py-2 px-4 text-[var(--foreground)]"
        >
          <Icon
            d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
            className="w-6 h-6"
          />
          <span className="text-[10px]">{t("home")}</span>
        </Link>
        <a
          href="#"
          className="flex flex-col items-center gap-1 py-2 px-4 text-[var(--muted)]"
        >
          <Icon
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            className="w-6 h-6"
          />
          <span className="text-[10px]">{t("explore")}</span>
        </a>
        <Link
          href={`/${locale}/bookmarks`}
          className="flex flex-col items-center gap-1 py-2 px-4 text-[var(--muted)]"
        >
          <Icon
            d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z"
            className="w-6 h-6"
          />
          <span className="text-[10px]">{t("bookmarks")}</span>
        </Link>
        <Link
          href={`/${locale}/library`}
          className="flex flex-col items-center gap-1 py-2 px-4 text-[var(--muted)]"
        >
          <Icon
            d="M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z"
            className="w-6 h-6"
          />
          <span className="text-[10px]">{t("library")}</span>
        </Link>
      </nav>
      <div className="h-16 md:hidden" />
    </>
  );
}
