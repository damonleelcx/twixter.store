"use client";

import { useLocale, useTranslations } from "next-intl";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { fetchFeedPage } from "@/lib/feed";
import type { FeedItem } from "./types";
import { TweetCard } from "./TweetCard";

type FeedListProps = {
  initialFeed: FeedItem[];
};

export function FeedList({ initialFeed }: FeedListProps) {
  const locale = useLocale();
  const tFeed = useTranslations("feed");
  const tTweets = useTranslations("tweets");
  const [feedItems, setFeedItems] = useState<FeedItem[]>(initialFeed);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [cursor, setCursor] = useState(0);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const initialLoadDone = useRef(false);
  const loadingRef = useRef(false);

  useEffect(() => {
    setFeedItems(initialFeed);
    initialLoadDone.current = true;
  }, [initialFeed]);

  const loadMore = useCallback(async () => {
    if (!initialLoadDone.current || loadingRef.current || !hasMore) return;
    loadingRef.current = true;
    setIsLoadingMore(true);
    try {
      const { items, nextCursor, hasMore: nextHasMore } = await fetchFeedPage(
        cursor,
        (key) => tTweets(key as Parameters<typeof tTweets>[0])
      );
      setFeedItems((prev) => [...prev, ...items]);
      setCursor(nextCursor);
      setHasMore(nextHasMore);
    } finally {
      loadingRef.current = false;
      setIsLoadingMore(false);
    }
  }, [cursor, hasMore, tTweets]);

  useEffect(() => {
    const el = loadMoreRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (
          !entries[0]?.isIntersecting ||
          !initialLoadDone.current ||
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
  }, [loadMore, isLoadingMore, hasMore]);

  return (
    <>
      {feedItems.map((item) => (
        <Link
          key={item.id}
          href={`/${locale}/post/${item.id}`}
          className="block"
        >
          <TweetCard
            name={item.name}
            handle={item.handle}
            time={item.time}
            text={item.text}
            videoUrl={item.videoUrl}
          />
        </Link>
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
        {!hasMore && feedItems.length > 0 && !isLoadingMore && (
          <p className="text-sm text-[var(--muted)]">{tFeed("allCaughtUp")}</p>
        )}
      </div>
    </>
  );
}
