"use client";

import { routing } from "@/i18n/routing";
import { useLocale, useTranslations } from "next-intl";
import Link from "next/link";
import { usePathname } from "next/navigation";

export function LanguageSwitcher() {
  const t = useTranslations("locale");
  const locale = useLocale();
  const pathname = usePathname() ?? "";
  const pathnameWithoutLocale = pathname.replace(/^\/[a-z]{2}/, "") || "";

  return (
    <div className="flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--background)] px-3 py-2">
      <span className="text-sm text-[var(--muted)]" title={t("switchTo")}>
        {locale === "zh" ? "中文" : "EN"}
      </span>
      {routing.locales.map((loc) =>
        loc === locale ? null : (
          <Link
            key={loc}
            href={`/${loc}${pathnameWithoutLocale}`}
            className="text-sm font-medium text-[var(--accent)] hover:underline"
          >
            {loc === "zh" ? "中文" : "English"}
          </Link>
        )
      )}
    </div>
  );
}
