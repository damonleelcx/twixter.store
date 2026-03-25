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
  const tSlide = useTranslations("slideshow");
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
      <div className="ml-auto flex items-center gap-2 shrink-0">
        <Link
          href={`/${locale}/slideshow`}
          className="rounded-full border border-[var(--border)] bg-[var(--background)] px-4 py-2 text-sm font-semibold text-[var(--foreground)] hover:bg-[var(--hover)]"
        >
          {tSlide("freeSlideshow")}
        </Link>
        {isLoggedIn ? (
          canUpload ? (
            <Link
              href={`/${locale}/admin/upload`}
              className="rounded-full bg-[var(--accent)] px-5 py-2 font-bold text-[var(--accent-foreground)] hover:opacity-90"
            >
              {tFeed("post")}
            </Link>
          ) : null
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
  );

  // After sign up: no "What is happening?!" compose area; sort + post buttons at top only
  return (
    <div className="border-b border-[var(--border)] px-4 py-3">
      {sortRow}
    </div>
  );
}
