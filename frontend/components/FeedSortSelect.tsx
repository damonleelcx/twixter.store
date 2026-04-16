"use client";

import type { FeedSort } from "@/lib/api";
import { useTranslations } from "next-intl";

const FEED_SORT_VALUES: FeedSort[] = [
  "created_at",
  "created_at_asc",
  "view_count",
  "free",
  "free_asc",
];

type FeedSortSelectProps = {
  value: FeedSort;
  onChange: (sort: FeedSort) => void;
  /** Extra classes on the `<select>` (e.g. width). */
  className?: string;
};

export function FeedSortSelect({ value, onChange, className = "" }: FeedSortSelectProps) {
  const tFeed = useTranslations("feed");

  return (
    <select
      aria-label={tFeed("sortBy")}
      value={value}
      onChange={(e) => onChange(e.target.value as FeedSort)}
      className={`min-w-0 max-w-full flex-1 rounded-lg border border-[var(--border)] bg-[var(--background)] py-2 pl-3 pr-8 text-sm text-[var(--foreground)] shadow-sm outline-none focus:ring-2 focus:ring-[var(--accent)] sm:max-w-[min(100%,20rem)] ${className}`}
    >
      {FEED_SORT_VALUES.map((v) => (
        <option key={v} value={v}>
          {sortOptionLabel(tFeed, v)}
        </option>
      ))}
    </select>
  );
}

function sortOptionLabel(
  tFeed: ReturnType<typeof useTranslations<"feed">>,
  sort: FeedSort
): string {
  switch (sort) {
    case "created_at":
      return tFeed("sortOptionUploadNewest");
    case "created_at_asc":
      return tFeed("sortOptionUploadOldest");
    case "view_count":
      return tFeed("sortByViewCount");
    case "free":
      return tFeed("sortOptionFreeNewest");
    case "free_asc":
      return tFeed("sortOptionFreeOldest");
    default: {
      const _x: never = sort;
      return _x;
    }
  }
}
