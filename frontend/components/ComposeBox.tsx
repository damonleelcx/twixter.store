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

  const sortRow = (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-3">
      <div className="flex items-center gap-2 min-w-0 shrink-0">
        <span className="text-sm text-[var(--muted)] shrink-0">{tFeed("sortBy")}</span>
        <div className="flex rounded-lg border border-[var(--border)] p-0.5 shrink-0">
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
          <button
            type="button"
            onClick={() => onSortChange("free")}
            className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
              sort === "free"
                ? "bg-[var(--accent)] text-[var(--accent-foreground)]"
                : "bg-[var(--background)] text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
            }`}
          >
            {tFeed("sortByFree")}
          </button>
        </div>
      </div>
      {isLoggedIn ? (
        canUpload ? (
          <Link
            href={`/${locale}/admin/upload`}
            className="rounded-full bg-[var(--accent)] px-5 py-2 font-bold text-[var(--accent-foreground)] hover:opacity-90 shrink-0 ml-auto"
          >
            {tFeed("post")}
          </Link>
        ) : (
          <></>
        )
      ) : (
        <Link
          href={`/${locale}/auth/signin`}
          className="rounded-full bg-[var(--accent)] px-5 py-2 font-bold text-[var(--accent-foreground)] hover:opacity-90 shrink-0 ml-auto"
        >
          {tAuth("signIn")}
        </Link>
      )}
    </div>
  );

  if (!isLoggedIn) {
    return (
      <div className="border-b border-[var(--border)] px-4 py-3">
        {sortRow}
      </div>
    );
  }

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
        <div className="flex flex-wrap items-center gap-x-4 gap-y-3 border-t border-[var(--border)] pt-3">
          {sortRow}
        </div>
      </div>
    </div>
  );
}
