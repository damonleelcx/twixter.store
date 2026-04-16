"use client";

import type { FeedSort } from "@/lib/api";
import { fetchCurrentUser } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { FeedSortSelect } from "./FeedSortSelect";

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
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-4">
      <div className="flex min-w-0 items-center gap-2">
        <span className="hidden shrink-0 text-sm text-[var(--muted)] sm:inline">{tFeed("sortBy")}</span>
        <FeedSortSelect value={sort} onChange={onSortChange} />
      </div>
      <div className="flex shrink-0 items-center justify-end gap-2 sm:ml-auto">
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
