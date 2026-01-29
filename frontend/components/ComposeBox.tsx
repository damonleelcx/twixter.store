"use client";

import { fetchCurrentUser } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { Icon } from "./Icon";

export function ComposeBox() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const tFeed = useTranslations("feed");
  const tCompose = useTranslations("compose");
  const tAuth = useTranslations("auth");
  const [composeText, setComposeText] = useState("");
  const [user, setUser] = useState<Awaited<ReturnType<typeof fetchCurrentUser>>>(null);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled) setUser(u ?? null);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const isLoggedIn = user != null;
  const canUpload = Boolean(user?.permissions?.includes("can_upload_content"));

  return (
    <div className="flex gap-3 border-b border-[var(--border)] p-4">
      <div className="h-10 w-10 shrink-0 rounded-full bg-[var(--muted)]/30 flex items-center justify-center">
        <span className="text-sm font-semibold text-[var(--muted)]">U</span>
      </div>
      <div className="min-w-0 flex-1">
        <textarea
          placeholder={tFeed("composePlaceholder")}
          value={composeText}
          onChange={(e) => setComposeText(e.target.value)}
          className="w-full resize-none border-none bg-transparent py-3 text-[20px] placeholder:text-[var(--muted)] focus:outline-none"
          rows={2}
        />
        <div className="flex items-center justify-between border-t border-[var(--border)] pt-3">
          <div className="flex gap-1 text-[var(--accent)]">
            <button
              type="button"
              className="rounded-full p-2 hover:bg-[var(--accent)]/10"
              aria-label={tCompose("media")}
            >
              <Icon
                d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14"
                className="w-5 h-5"
              />
            </button>
            <button
              type="button"
              className="rounded-full p-2 hover:bg-[var(--accent)]/10"
              aria-label={tCompose("gif")}
            >
              <Icon
                d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z"
                className="w-5 h-5"
              />
            </button>
            <button
              type="button"
              className="rounded-full p-2 hover:bg-[var(--accent)]/10"
              aria-label={tCompose("poll")}
            >
              <Icon
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                className="w-5 h-5"
              />
            </button>
          </div>
          {isLoggedIn ? (
            canUpload ? (
              <Link
                href={`/${locale}/admin/upload`}
                className="rounded-full bg-[var(--accent)] px-5 py-2 font-bold text-[var(--accent-foreground)] hover:opacity-90"
              >
                {tFeed("post")}
              </Link>
            ) : (
             <></>
            )
          ) : (
            <Link
              href={`/${locale}/auth/signin`}
              className="rounded-full bg-[var(--accent)] px-5 py-2 font-bold text-[var(--accent-foreground)] hover:opacity-90"
            >
              {tAuth("signIn")}
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}
