"use client";

import type { ContentFeedItem, ContentFeedResponse } from "@/lib/api";
import {
  type FeedSort,
  fetchContentFeedByTag,
  fetchContentFeedSearch,
} from "@/lib/api";
import { useLocale, useTranslations } from "next-intl";
import { useCallback, useEffect, useRef, useState } from "react";
import { ContentCard } from "./ContentCard";

const PAGE_SIZE = 20;

type SearchOrTagFeedListProps =
  | { mode: "search"; query: string; category?: "light" | "dark" | "" }
  | { mode: "tag"; tag: string };

export function SearchOrTagFeedList(props: SearchOrTagFeedListProps) {
  const locale = useLocale();
  const tFeed = useTranslations("feed");
  const [sort, setSort] = useState<FeedSort>("created_at");
  const [items, setItems] = useState<ContentFeedItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [cursor, setCursor] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  const loadPage = useCallback(
    async (pageCursor: number, append: boolean) => {
      if (loadingRef.current) return;
      loadingRef.current = true;
      if (pageCursor === 0) {
        setIsLoading(true);
        setError(null);
      } else {
        setIsLoadingMore(true);
      }
      try {
        let result: ContentFeedResponse;
        if (props.mode === "tag") {
          result = await fetchContentFeedByTag(props.tag, pageCursor, PAGE_SIZE, sort);
        } else {
          result = await fetchContentFeedSearch(
            props.mode === "search" ? props.query : "",
            props.mode === "search" ? (props.category ?? "") : "",
            pageCursor,
            PAGE_SIZE,
            sort
          );
        }
        const { items: nextItems, next_cursor, has_more } = result;
        setItems((prev) => {
          if (append) return [...prev, ...nextItems];
          return nextItems.map((i) => ({
            ...i,
            purchased: i.purchased || (prev.find((p) => p.id === i.id)?.purchased ?? false),
          }));
        });
        setCursor(next_cursor);
        setHasMore(has_more);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load");
        if (pageCursor === 0) setItems([]);
      } finally {
        loadingRef.current = false;
        setIsLoading(false);
        setIsLoadingMore(false);
      }
    },
    [
      props.mode,
      props.mode === "tag" ? props.tag : "",
      props.mode === "search" ? props.query : "",
      props.mode === "search" ? (props.category ?? "") : "",
      sort,
    ]
  );

  useEffect(() => {
    setItems([]);
    setCursor(0);
    setHasMore(true);
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

  if (isLoading && items.length === 0) {
    return (
      <div className="min-h-[200px] flex items-center justify-center border-b border-[var(--border)] py-12">
        <div className="flex items-center gap-2 text-[var(--muted)]">
          <svg className="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden>
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          <span className="text-sm">{tFeed("loading")}</span>
        </div>
      </div>
    );
  }

  if (error && items.length === 0) {
    return (
      <div className="border-b border-[var(--border)] px-4 py-8 text-center">
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        <button
          type="button"
          onClick={() => loadPage(0, false)}
          className="mt-4 rounded-lg border border-[var(--border)] px-4 py-2 text-sm hover:bg-[var(--hover)]"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <>
      <div className="flex items-center gap-2 border-b border-[var(--border)] px-4 py-2">
        <span className="text-sm text-[var(--muted)]">{tFeed("sortBy")}</span>
        <div className="flex rounded-lg border border-[var(--border)] p-0.5">
          <button
            type="button"
            onClick={() => setSort("created_at")}
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
            onClick={() => setSort("view_count")}
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
            onClick={() => setSort("free")}
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
      {items.map((item) => (
        <ContentCard
          key={item.id}
          item={item}
          locale={locale}
          onPurchased={(purchasedItem) => {
            setItems((prev) =>
              prev.map((i) =>
                i.id === purchasedItem.id ? { ...i, purchased: true } : i
              )
            );
            loadPage(0, false);
          }}
          onDeleted={(deletedItem) => {
            setItems((prev) => prev.filter((i) => i.id !== deletedItem.id));
          }}
        />
      ))}
      <div
        ref={loadMoreRef}
        className="flex min-h-[72px] flex-col items-center justify-center border-b border-[var(--border)] py-6"
      >
        {isLoadingMore && (
          <div className="flex items-center gap-2 text-[var(--muted)]">
            <svg className="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden>
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            <span className="text-sm">{tFeed("loading")}</span>
          </div>
        )}
        {!hasMore && items.length > 0 && !isLoadingMore && (
          <p className="text-sm text-[var(--muted)]">{tFeed("allCaughtUp")}</p>
        )}
      </div>
    </>
  );
}
