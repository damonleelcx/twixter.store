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
        <a
          href="#"
          className="flex flex-col items-center gap-1 py-2 px-4 text-[var(--muted)]"
        >
          <Icon
            d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
            className="w-6 h-6"
          />
          <span className="text-[10px]">{t("notifications")}</span>
        </a>
        <a
          href="#"
          className="flex flex-col items-center gap-1 py-2 px-4 text-[var(--muted)]"
        >
          <Icon
            d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
            className="w-6 h-6"
          />
          <span className="text-[10px]">{t("messages")}</span>
        </a>
      </nav>
      <div className="h-16 md:hidden" />
    </>
  );
}
