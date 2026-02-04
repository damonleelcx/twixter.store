"use client";

import type { ContentFeedItem, ContentFeedResponse } from "@/lib/api";
import { fetchContentFeed } from "@/lib/api";
import { useLocale, useTranslations } from "next-intl";
import { useCallback, useEffect, useRef, useState } from "react";
import { ContentCard } from "./ContentCard";
import type { FeedTab } from "./MainFeedHeader";

type FeedListProps = {
  activeTab: FeedTab;
  /** Server-fetched first page for "For you" tab (SSR). */
  initialForYouFeed?: ContentFeedResponse;
};

const PAGE_SIZE = 20;

export function FeedList({ activeTab, initialForYouFeed }: FeedListProps) {
  const locale = useLocale();
  const tFeed = useTranslations("feed");
  const [items, setItems] = useState<ContentFeedItem[]>(() =>
    initialForYouFeed ? initialForYouFeed.items : []
  );
  const [isLoading, setIsLoading] = useState(() => !initialForYouFeed);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(() => initialForYouFeed?.has_more ?? true);
  const [cursor, setCursor] = useState(() => initialForYouFeed?.next_cursor ?? 0);
  const [error, setError] = useState<string | null>(null);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  const category = activeTab === "forYou" ? "light" : "dark";

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
        const { items: nextItems, next_cursor, has_more } = await fetchContentFeed(
          category,
          pageCursor,
          PAGE_SIZE
        );
        setItems((prev) => {
          if (append) return [...prev, ...nextItems];
          // 替换时保留之前已标记为已购买的状态，避免 refetch 覆盖刚购买后的 optimistic 更新
          return nextItems.map((i) => ({
            ...i,
            purchased: i.purchased || (prev.find((p) => p.id === i.id)?.purchased ?? false),
          }));
        });
        setCursor(next_cursor);
        setHasMore(has_more);
      } catch (err) {
        const raw = err instanceof Error ? err.message : "Failed to load";
        const msg =
          activeTab === "premium" &&
          (raw.toLowerCase().includes("nsfw") ||
            raw.toLowerCase().includes("can_view") ||
            raw.toLowerCase().includes("unauthorized") ||
            raw.toLowerCase().includes("permission"))
            ? "Unable to load premium content."
            : raw;
        setError(msg);
        if (pageCursor === 0) setItems([]);
      } finally {
        loadingRef.current = false;
        setIsLoading(false);
        setIsLoadingMore(false);
      }
    },
    [category]
  );

  // 切换 For you / Premium 时都重新拉取当前 tab 的列表
  useEffect(() => {
    setItems([]);
    setCursor(0);
    setHasMore(true);
    setError(null);
    loadPage(0, false);
  }, [activeTab, loadPage]);

  // After returning from post edit, refetch current tab so list shows updated content
  useEffect(() => {
    if (typeof window === "undefined") return;
    if (sessionStorage.getItem("twixter_feed_refetch") === "1") {
      sessionStorage.removeItem("twixter_feed_refetch");
      loadPage(0, false);
    }
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
            // Refetch current page so GIF preview and server state are correct
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
        {!hasMore && items.length > 0 && !isLoadingMore && (
          <p className="text-sm text-[var(--muted)]">{tFeed("allCaughtUp")}</p>
        )}
      </div>
    </>
  );
}
