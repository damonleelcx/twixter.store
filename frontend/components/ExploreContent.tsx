"use client";

import { SearchOrTagFeedList } from "@/components/SearchOrTagFeedList";
import { useTranslations } from "next-intl";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

export function ExploreContent() {
  const params = useParams();
  const router = useRouter();
  const searchParams = useSearchParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("nav");
  const tFeed = useTranslations("feed");

  const qFromUrl = searchParams.get("q") ?? "";
  const [query, setQuery] = useState(qFromUrl);
  const [submittedQuery, setSubmittedQuery] = useState(qFromUrl);

  useEffect(() => {
    const q = searchParams.get("q") ?? "";
    setQuery(q);
    setSubmittedQuery(q);
  }, [searchParams]);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      setSubmittedQuery(query.trim());
      const params = new URLSearchParams();
      if (query.trim()) params.set("q", query.trim());
      const path = params.toString() ? `/${locale}/explore?${params}` : `/${locale}/explore`;
      router.replace(path, { scroll: false });
    },
    [query, locale, router]
  );

  return (
    <>
      <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[var(--background)] px-4 py-3">
        <h1 className="text-xl font-semibold">{t("explore")}</h1>
        <form onSubmit={handleSubmit} className="mt-3">
          <div className="flex gap-2">
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={tFeed("searchPlaceholder")}
              className="min-w-0 flex-1 rounded-full border border-[var(--border)] bg-[var(--background)] px-4 py-2.5 text-[var(--foreground)] placeholder:text-[var(--muted)] focus:border-[var(--accent)] focus:outline-none focus:ring-2 focus:ring-[var(--accent)]/20"
              aria-label={tFeed("searchPlaceholder")}
            />
            <button
              type="submit"
              className="shrink-0 rounded-full bg-[var(--accent)] px-5 py-2.5 text-sm font-medium text-[var(--accent-foreground)] hover:opacity-90"
            >
              {tFeed("search")}
            </button>
          </div>
        </form>
      </header>
      <SearchOrTagFeedList mode="search" query={submittedQuery} category="" />
    </>
  );
}
