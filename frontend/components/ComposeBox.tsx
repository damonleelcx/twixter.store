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
          <div className="flex items-center gap-2">
            <span className="text-sm text-[var(--muted)]">{tFeed("sortBy")}</span>
            <div className="flex rounded-lg border border-[var(--border)] p-0.5">
              <button
                type="button"
                onClick={() => onSortChange("created_at")}
                className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                  sort === "created_at"
                    ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                    : "bg-[var(--background)] text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
                }`}
              >
                {tFeed("sortByUploadDate")}
              </button>
              <button
                type="button"
                onClick={() => onSortChange("view_count")}
                className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                  sort === "view_count"
                    ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                    : "bg-[var(--background)] text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
                }`}
              >
                {tFeed("sortByViewCount")}
              </button>
            </div>
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
