"use client";

import { ThemeSwitcher } from "@/app/[locale]/ThemeSwitcher";
import { fetchCurrentUser } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useEffect, useState } from "react";
import { Icon } from "./Icon";
import { LanguageSwitcher } from "./LanguageSwitcher";
import { NavItem } from "./NavItem";
import { navItems } from "./constants";

type LeftSidebarProps = {
  locale: string;
};

export function LeftSidebar({ locale }: LeftSidebarProps) {
  const t = useTranslations("nav");
  const tAuth = useTranslations("auth");
  const [user, setUser] = useState<Awaited<ReturnType<typeof fetchCurrentUser>> | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled) setUser(u ?? null);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const canUpload = Boolean(user?.permissions?.includes("can_upload_content"));
  const canViewAnalytics = Boolean(user?.permissions?.includes("can_view_analytics"));
  const isLoggedIn = user != null;

  const items = navItems.filter((item) => {
    if (item.key === "adminUpload") return canUpload;
    if (item.key === "adminAnalytics") return canViewAnalytics;
    return true;
  });

  return (
    <aside className="sticky top-0 hidden h-screen shrink-0 flex-col border-r border-[var(--border)] pl-4 pr-2 md:flex md:w-[68px] xl:w-[275px]">
      <div className="flex items-center justify-between xl:justify-start xl:px-1">
        <Link
          href={`/${locale}`}
          className="inline-flex items-center justify-center p-3 xl:px-4"
        >
          <span className="text-2xl font-bold text-[var(--accent)]">𝕋</span>
        </Link>
        <div className="flex items-center gap-2">
          <ThemeSwitcher />
          <div className="hidden xl:block">
            <LanguageSwitcher />
          </div>
        </div>
      </div>
      <nav className="mt-1 flex flex-1 flex-col gap-1">
        {items.map((item, i) => (
          <NavItem
            key={item.key}
            label={t(item.key)}
            icon={item.icon}
            active={i === 0}
            href={
              item.key === "home"
                ? `/${locale}`
                : item.key === "explore"
                  ? `/${locale}/explore`
                  : item.key === "adminUpload"
                    ? `/${locale}/admin/upload`
                    : item.key === "adminAnalytics"
                      ? `/${locale}/admin/analytics`
                      : item.key === "profile"
                      ? `/${locale}/profile`
                      : item.key === "bookmarks"
                        ? `/${locale}/bookmarks`
                        : item.key === "library"
                          ? `/${locale}/library`
                          : undefined
            }
          />
        ))}
      </nav>
      {isLoggedIn && canUpload ? (
        <Link
          href={`/${locale}/admin/upload`}
          className="my-3 flex w-full items-center justify-center gap-2 rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 xl:max-w-[90%]"
        >
          <span className="hidden xl:inline">{t("post")}</span>
          <Icon d="M12 4v16m8-8H4" className="h-6 w-6 xl:mx-auto xl:hidden" />
        </Link>
      ) : isLoggedIn ? null : (
        <div className="my-3 flex flex-col gap-2 xl:max-w-[90%]">
          <Link
            href={`/${locale}/auth/signin`}
            className="w-full rounded-full border border-[var(--border)] py-3 text-center font-bold text-[var(--foreground)] transition-colors hover:bg-[var(--hover)]"
          >
            <span className="hidden xl:inline">{tAuth("signIn")}</span>
            <Icon d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" className="mx-auto h-6 w-6 xl:hidden" />
          </Link>
          <Link
            href={`/${locale}/auth/signup`}
            className="w-full rounded-full bg-[var(--accent)] py-3 text-center font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
          >
            <span className="hidden xl:inline">{tAuth("signUp")}</span>
            <Icon d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" className="mx-auto h-6 w-6 xl:hidden" />
          </Link>
        </div>
      )}
    </aside>
  );
}
