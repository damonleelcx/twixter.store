"use client";

import type { FeedSort } from "@/lib/api";
import { fetchCurrentUser } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

type ComposeBoxProps = {
  sort: FeedSort;
  onSortChange: (sort: FeedSort) => void;
};

export function ComposeBox({ sort, onSortChange }: ComposeBoxProps) {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const tFeed = useTranslations("feed");
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
    <div className="w-full min-w-0 overflow-x-hidden border-b border-[var(--border)] p-4">
      <div className="flex w-full min-w-0 gap-3">
        <div className="h-10 w-10 shrink-0 rounded-full bg-[var(--muted)]/30 flex items-center justify-center">
          <span className="text-sm font-semibold text-[var(--muted)]">U</span>
        </div>
        <div className="w-0 min-w-0 flex-1">
          <textarea
            placeholder={tFeed("composePlaceholder")}
            value={composeText}
            onChange={(e) => setComposeText(e.target.value)}
            className="w-full min-w-0 max-w-full resize-none border-none bg-transparent py-3 text-[20px] placeholder:text-[var(--muted)] focus:outline-none"
            rows={2}
          />
          <div className="flex w-full min-w-0 items-center gap-2 border-t border-[var(--border)] pt-3">
            <div className="min-w-0 flex-1 overflow-hidden">
              <div className="flex items-center gap-1.5 sm:gap-2">
                <span className="shrink-0 text-sm text-[var(--muted)]">{tFeed("sortBy")}</span>
                <div className="flex shrink-0 rounded-lg border border-[var(--border)] p-0.5">
                  <button
                    type="button"
                    onClick={() => onSortChange("created_at")}
                    className={`whitespace-nowrap rounded-md px-2 py-1 text-xs transition-colors sm:px-3 sm:py-1.5 sm:text-sm ${
                      sort === "created_at"
                        ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                        : "bg-[var(--background)] text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
                    }`}
                  >
                    <span className="sm:hidden">{tFeed("sortByUploadDateShort")}</span>
                    <span className="hidden sm:inline">{tFeed("sortByUploadDate")}</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => onSortChange("view_count")}
                    className={`whitespace-nowrap rounded-md px-2 py-1 text-xs transition-colors sm:px-3 sm:py-1.5 sm:text-sm ${
                      sort === "view_count"
                        ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                        : "bg-[var(--background)] text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
                    }`}
                  >
                    <span className="sm:hidden">{tFeed("sortByViewCountShort")}</span>
                    <span className="hidden sm:inline">{tFeed("sortByViewCount")}</span>
                  </button>
                </div>
              </div>
            </div>
            {isLoggedIn ? (
              canUpload ? (
                <Link
                  href={`/${locale}/admin/upload`}
                  className="shrink-0 rounded-full bg-[var(--accent)] px-3 py-1.5 text-sm font-bold text-[var(--accent-foreground)] hover:opacity-90 sm:px-5 sm:py-2"
                >
                  {tFeed("post")}
                </Link>
              ) : (
               <></>
              )
            ) : (
              <Link
                href={`/${locale}/auth/signin`}
                className="shrink-0 rounded-full bg-[var(--accent)] px-3 py-1.5 text-sm font-bold text-[var(--accent-foreground)] hover:opacity-90 sm:px-5 sm:py-2"
              >
                {tAuth("signIn")}
              </Link>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
