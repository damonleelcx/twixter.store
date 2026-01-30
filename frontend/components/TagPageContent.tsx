"use client";

import { SearchOrTagFeedList } from "@/components/SearchOrTagFeedList";
import { useTranslations } from "next-intl";

type TagPageContentProps = {
  tag: string;
};

export function TagPageContent({ tag }: TagPageContentProps) {
  const tFeed = useTranslations("feed");

  return (
    <>
      <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[var(--background)] px-4 py-3">
        <h1 className="text-xl font-semibold">{tag ? `#${tag}` : tFeed("loading")}</h1>
        <p className="mt-1 text-sm text-[var(--muted)]">{tag ? tFeed("tagFeedHint") : ""}</p>
      </header>
      {tag ? (
        <SearchOrTagFeedList mode="tag" tag={tag} />
      ) : (
        <div className="flex min-h-[200px] items-center justify-center py-12 text-[var(--muted)]">
          {tFeed("loading")}
        </div>
      )}
    </>
  );
}
