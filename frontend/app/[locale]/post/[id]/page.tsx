import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import { SaveViewingFromUrl } from "@/components/SaveViewingFromUrl";
import { getPostById } from "@/lib/feed";
import { getTranslations, setRequestLocale } from "next-intl/server";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

type Props = {
  params: Promise<{ locale: string; id: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const tTweets = await getTranslations({ locale, namespace: "tweets" });
  const post = getPostById(id, (key) => tTweets(key));
  if (!post) return { title: "Twixter" };
  const snippet = post.text.slice(0, 50).replace(/\s+/g, " ");
  return {
    title: `${post.name}: ${snippet}${post.text.length > 50 ? "…" : ""} | Twixter`,
    description: post.text,
  };
}

export default async function PostPage({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);

  const t = await getTranslations({ locale, namespace: "postDetail" });
  const tTweets = await getTranslations({ locale, namespace: "tweets" });
  const post = getPostById(id, (key) => tTweets(key));

  if (!post) notFound();

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <SaveViewingFromUrl />
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          {/* Sticky header: back + title - Twitter style */}
          <header className="sticky top-0 z-10 flex h-14 items-center gap-4 border-b border-[var(--border)] bg-[var(--background)]/80 px-4 backdrop-blur-md">
            <Link
              href={`/${locale}`}
              className="flex h-9 w-9 -ml-1 items-center justify-center rounded-full text-[var(--foreground)] transition-colors hover:bg-[var(--hover)]"
              aria-label={t("back")}
            >
              <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16l-4-4m0 0l4-4m-4 4h18" />
              </svg>
            </Link>
            <h1 className="text-xl font-bold">{t("title")}</h1>
          </header>

          {/* Main tweet - expanded Twitter style */}
          <article className="border-b border-[var(--border)] px-4 pt-3 pb-2">
            <div className="flex gap-3">
              <div className="h-12 w-12 shrink-0 rounded-full bg-[var(--muted)]/30 flex items-center justify-center">
                <span className="text-base font-semibold text-[var(--muted)]">
                  {post.name.charAt(0)}
                </span>
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-1">
                  <span className="font-bold text-[var(--foreground)] truncate text-[15px]">
                    {post.name}
                  </span>
                  <span className="text-[var(--muted)] truncate text-[15px]">{post.handle}</span>
                  <span className="text-[var(--muted)]">·</span>
                  <time className="text-[var(--muted)] text-[15px]" dateTime={post.time}>
                    {post.time}
                  </time>
                </div>
                <p className="mt-2 break-words text-[17px] leading-[1.35]">{post.text}</p>
                {post.videoUrl && (
                  <div className="mt-3 rounded-2xl overflow-hidden border border-[var(--border)] bg-[var(--muted)]/20 aspect-video max-h-[340px] flex items-center justify-center">
                    <div className="w-16 h-16 rounded-full bg-[var(--foreground)]/80 flex items-center justify-center">
                      <svg className="w-8 h-8 text-white ml-1" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M8 5v14l11-7z" />
                      </svg>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Engagement bar - Twitter style */}
            <div className="mt-1 flex items-center justify-between max-w-[425px] text-[var(--muted)]">
              <button type="button" className="group flex items-center gap-2 py-3 min-w-0" aria-label={t("reply")}>
                <span className="flex h-9 w-9 items-center justify-center rounded-full transition-colors group-hover:bg-[var(--hover)] group-hover:text-[var(--foreground)]">
                  <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                  </svg>
                </span>
                <span className="text-sm hidden sm:inline">{t("reply")}</span>
              </button>
              <button type="button" className="group flex items-center gap-2 py-3 min-w-0" aria-label={t("repost")}>
                <span className="flex h-9 w-9 items-center justify-center rounded-full transition-colors group-hover:bg-[var(--hover)] group-hover:text-[var(--foreground)]">
                  <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </span>
                <span className="text-sm hidden sm:inline">{t("repost")}</span>
              </button>
              <button type="button" className="group flex items-center gap-2 py-3 min-w-0" aria-label={t("like")}>
                <span className="flex h-9 w-9 items-center justify-center rounded-full transition-colors group-hover:bg-[var(--hover)] group-hover:text-[var(--foreground)]">
                  <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                  </svg>
                </span>
                <span className="text-sm hidden sm:inline">{t("like")}</span>
              </button>
              <button type="button" className="group flex items-center gap-2 py-3 min-w-0" aria-label={t("share")}>
                <span className="flex h-9 w-9 items-center justify-center rounded-full transition-colors group-hover:bg-[var(--hover)] group-hover:text-[var(--foreground)]">
                  <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
                  </svg>
                </span>
                <span className="text-sm hidden sm:inline">{t("share")}</span>
              </button>
              <button type="button" className="group flex items-center gap-2 py-3 min-w-0" aria-label={t("bookmark")}>
                <span className="flex h-9 w-9 items-center justify-center rounded-full transition-colors group-hover:bg-[var(--hover)] group-hover:text-[var(--foreground)]">
                  <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
                  </svg>
                </span>
                <span className="text-sm hidden sm:inline">{t("bookmark")}</span>
              </button>
            </div>
          </article>

          {/* Replies section placeholder - Twitter style */}
          <div className="border-b border-[var(--border)] px-4 py-6 text-center">
            <p className="text-[var(--muted)] text-[15px]">{t("showReplies")}</p>
          </div>
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
