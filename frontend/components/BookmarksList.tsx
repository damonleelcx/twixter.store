"use client";

import type { ContentFeedItem } from "@/lib/api";
import { fetchBookmarks } from "@/lib/api";
import { useLocale, useTranslations } from "next-intl";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { ContentCard } from "./ContentCard";

const PAGE_SIZE = 20;

export function BookmarksList() {
  const locale = useLocale();
  const tFeed = useTranslations("feed");
  const tBookmarks = useTranslations("bookmarks");
  const tAuth = useTranslations("auth");
  const [items, setItems] = useState<ContentFeedItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [cursor, setCursor] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [isAuthError, setIsAuthError] = useState(false);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  const loadPage = useCallback(async (pageCursor: number, append: boolean) => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    if (pageCursor === 0) {
      setIsLoading(true);
      setError(null);
      setIsAuthError(false);
    } else {
      setIsLoadingMore(true);
    }
    try {
      const { items: nextItems, next_cursor, has_more } =
        await fetchBookmarks(pageCursor, PAGE_SIZE);
      setItems((prev) => (append ? [...prev, ...nextItems] : nextItems));
      setCursor(next_cursor);
      setHasMore(has_more);
    } catch (err) {
      const raw = err instanceof Error ? err.message : "Failed to load bookmarks";
      const authErr =
        raw.toLowerCase().includes("session") ||
        raw.toLowerCase().includes("sign in") ||
        raw.toLowerCase().includes("expired");
      setIsAuthError(authErr);
      setError(authErr ? tBookmarks("signInToView") : raw);
      if (pageCursor === 0) setItems([]);
    } finally {
      loadingRef.current = false;
      setIsLoading(false);
      setIsLoadingMore(false);
    }
  }, [tBookmarks]);

  useEffect(() => {
    loadPage(0, false);
  }, [loadPage]);

  const loadMore = useCallback(() => {
    if (!hasMore || loadingRef.current) return;
    loadPage(cursor, true);
  }, [cursor, hasMore, loadPage]);

  useEffect(() => {
    const el = loadMoreRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (
          !entries[0]?.isIntersecting ||
          isLoading ||
          isLoadingMore ||
          !hasMore
        )
          return;
        loadMore();
      },
      { root: null, rootMargin: "200px 0px", threshold: 0 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [loadMore, isLoading, isLoadingMore, hasMore]);

  const handleBookmarkedChange = useCallback((item: ContentFeedItem, bookmarked: boolean) => {
    if (!bookmarked) {
      setItems((prev) => prev.filter((i) => i.id !== item.id));
    }
  }, []);

  if (isLoading && items.length === 0) {
    return (
      <div className="flex min-h-[200px] items-center justify-center border-b border-[var(--border)] py-12">
        <div className="flex items-center gap-2 text-[var(--muted)]">
          <svg
            className="h-5 w-5 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
            aria-hidden
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          <span className="text-sm">{tFeed("loading")}</span>
        </div>
      </div>
    );
  }

  if (error && items.length === 0) {
    return (
      <div className="border-b border-[var(--border)] px-4 py-8 text-center">
        <p className="text-sm text-[var(--muted)]">{error}</p>
        {isAuthError ? (
          <Link
            href={`/${locale}/auth/signin`}
            className="mt-4 inline-block rounded-lg bg-[var(--accent)] px-4 py-2 text-sm text-[var(--accent-foreground)] hover:opacity-90"
          >
            {tAuth("signIn")}
          </Link>
        ) : (
          <button
            type="button"
            onClick={() => loadPage(0, false)}
            className="mt-4 rounded-lg border border-[var(--border)] px-4 py-2 text-sm hover:bg-[var(--hover)]"
          >
            {tBookmarks("retry")}
          </button>
        )}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="border-b border-[var(--border)] px-4 py-12 text-center">
        <p className="text-sm text-[var(--muted)]">{tBookmarks("empty")}</p>
      </div>
    );
  }

  return (
    <>
      {items.map((item) => (
        <ContentCard
          key={item.id}
          item={{ ...item, bookmarked: true }}
          locale={locale}
          onBookmarkedChange={handleBookmarkedChange}
        />
      ))}
      <div
        ref={loadMoreRef}
        className="flex min-h-[72px] flex-col items-center justify-center border-b border-[var(--border)] py-6"
      >
        {isLoadingMore && (
          <div className="flex items-center gap-2 text-[var(--muted)]">
            <svg
              className="h-5 w-5 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
              aria-hidden
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
            <span className="text-sm">{tFeed("loading")}</span>
          </div>
        )}
        {!hasMore && !isLoadingMore && (
          <p className="text-sm text-[var(--muted)]">{tFeed("allCaughtUp")}</p>
        )}
      </div>
    </>
  );
}
