"use client";

import {
  addBookmark,
  fetchCurrentUser,
  getAccessToken,
  getApiBase,
  isInsufficientCreditsError,
  purchaseContent,
  removeBookmark,
  type ContentFeedItem,
} from "@/lib/api";
import { buildShareUrl } from "@/lib/viewing";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { HlsPlayer } from "./HlsPlayer";
import { VerticalAspectImage } from "./VerticalAspectImage";

/** Format created_at to Twitter-style relative time (e.g. "2h", "3d", "Jan 29") */
function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffM = Math.floor(diffMs / 60000);
  const diffH = Math.floor(diffMs / 3600000);
  const diffD = Math.floor(diffMs / 86400000);
  if (diffM < 1) return "now";
  if (diffM < 60) return `${diffM}m`;
  if (diffH < 24) return `${diffH}h`;
  if (diffD < 7) return `${diffD}d`;
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

type ContentCardProps = {
  item: ContentFeedItem;
  locale: string;
  /** Called after successful purchase so the list can update the item. */
  onPurchased?: (item: ContentFeedItem) => void;
  /** Called when bookmark is toggled (e.g. to remove from bookmarks list when unbookmarked). */
  onBookmarkedChange?: (item: ContentFeedItem, bookmarked: boolean) => void;
};

export function ContentCard({ item, locale, onPurchased, onBookmarkedChange }: ContentCardProps) {
  const router = useRouter();
  const tFeed = useTranslations("feed");
  const tPost = useTranslations("postDetail");
  const tAuth = useTranslations("auth");
  const [blurredPreviewUrl, setBlurredPreviewUrl] = useState<string | null>(null);
  const [purchasing, setPurchasing] = useState(false);
  const [bookmarked, setBookmarked] = useState(() => item.bookmarked ?? false);
  const [bookmarking, setBookmarking] = useState(false);
  // Avoid hydration mismatch: server has no localStorage so getAccessToken() is null; only use token after mount
  const [hasAuth, setHasAuth] = useState(false);
  const [canEditContent, setCanEditContent] = useState(false);
  const blobUrlRef = useRef<string | null>(null);
  const timeAgo = useMemo(() => formatTimestamp(item.created_at), [item.created_at]);
  const displayName = item.author_username || item.name;
  const avatarInitial = (displayName.charAt(0) || "?").toUpperCase();

  useEffect(() => {
    setHasAuth(getAccessToken() !== null);
  }, []);
  useEffect(() => {
    fetchCurrentUser().then((user) => {
      setCanEditContent((user?.permissions ?? []).includes("can_edit_content"));
    });
  }, []);

  const handlePurchase = useCallback(
    async (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (!getAccessToken() || item.purchased || purchasing || item.price <= 0) return;
      setPurchasing(true);
      try {
        await purchaseContent(item.id);
        onPurchased?.(item);
      } catch (err) {
        if (isInsufficientCreditsError(err)) {
          router.push(`/${locale}/credits`);
          return;
        }
      } finally {
        setPurchasing(false);
      }
    },
    [item, locale, onPurchased, purchasing, router]
  );

  // Fetch preview from API for both purchased (real GIF) and not purchased (blurred) so image always loads
  useEffect(() => {
    if (!item.first_file_id) return;
    const base = getApiBase();
    const token = getAccessToken();
    const url = `${base}/content/files/${item.first_file_id}/preview`;
    const headers: HeadersInit = {};
    if (token) (headers as Record<string, string>)["Authorization"] = `Bearer ${token}`;
    let cancelled = false;
    fetch(url, { headers })
      .then((r) => (r.ok ? r.blob() : null))
      .then((blob) => {
        if (cancelled || !blob) return;
        const objectUrl = URL.createObjectURL(blob);
        blobUrlRef.current = objectUrl;
        setBlurredPreviewUrl(objectUrl);
      })
      .catch(() => { });
    return () => {
      cancelled = true;
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current);
        blobUrlRef.current = null;
      }
      setBlurredPreviewUrl(null);
    };
  }, [item.purchased, item.first_file_id]);

  // Sync bookmarked from server when item.bookmarked is present
  useEffect(() => {
    if (item.bookmarked !== undefined) setBookmarked(item.bookmarked);
  }, [item.bookmarked]);

  const previewSrc = blurredPreviewUrl ?? item.preview_gif_url;

  return (
    <article className="block border-b border-[var(--border)] transition-colors hover:bg-[var(--hover)]">
      <div className="flex gap-3 px-4 pt-3">
        <Link
          href={`/${locale}/post/${item.id}`}
          className="shrink-0 outline-none rounded-full focus:ring-2 focus:ring-[var(--accent)]"
          aria-label={displayName}
        >
          {item.author_avatar ? (
            <img
              src={item.author_avatar}
              alt=""
              className="h-10 w-10 rounded-full object-cover"
            />
          ) : (
            <div className="h-10 w-10 rounded-full bg-[var(--muted)]/30 flex items-center justify-center">
              <span className="text-sm font-semibold text-[var(--muted)]">
                {avatarInitial}
              </span>
            </div>
          )}
        </Link>
        <div className="min-w-0 flex-1">
          <Link href={`/${locale}/post/${item.id}`} className="outline-none block">
            {/* 1) Content upload user (author + time + credits) */}
            <div className="flex flex-wrap items-center gap-1 text-[15px]">
              <span className="font-semibold text-[var(--foreground)] truncate">
                {displayName}
              </span>
              <span className="text-[var(--muted)]">
                · {timeAgo}
              </span>
              {item.price > 0 && (
                <span className="text-[var(--muted)]">
                  · {item.price} {tFeed("credits")}
                </span>
              )}
            </div>
            {/* 2) Title */}
            <p className="mt-0.5 font-medium text-[var(--foreground)] text-[15px] leading-5 truncate">
              {item.name}
            </p>
            {/* 3) Description */}
            {item.description && (
              <p className="mt-0.5 break-words text-[15px] leading-5 text-[var(--muted)] line-clamp-2">
                {item.description}
              </p>
            )}
            {/* 4) Tags (clickable → tag feed; use span + router to avoid <a> inside <a>) */}
            {Array.isArray(item.tags) && item.tags.length > 0 && (
              <div className="mt-1 flex flex-wrap gap-1" onClick={(e) => e.stopPropagation()}>
                {item.tags.slice(0, 5).map((tag) => (
                  <span
                    key={tag}
                    role="button"
                    tabIndex={0}
                    className="inline-flex cursor-pointer items-center rounded-full bg-[var(--muted)]/30 px-2 py-0.5 text-xs text-[var(--muted)] hover:bg-[var(--muted)]/50 hover:text-[var(--accent)]"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      router.push(`/${locale}/tag/${encodeURIComponent(tag)}`);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        e.stopPropagation();
                        router.push(`/${locale}/tag/${encodeURIComponent(tag)}`);
                      }
                    }}
                  >
                    #{tag}
                  </span>
                ))}
                {item.tags.length > 5 && (
                  <span className="text-xs text-[var(--muted)]">+{item.tags.length - 5}</span>
                )}
              </div>
            )}
          </Link>
        </div>
        <button
          type="button"
          disabled={!hasAuth || bookmarking}
          onClick={async (e) => {
            e.preventDefault();
            e.stopPropagation();
            if (!hasAuth || bookmarking) return;
            setBookmarking(true);
            try {
              if (bookmarked) {
                await removeBookmark(item.id);
                setBookmarked(false);
                onBookmarkedChange?.(item, false);
              } else {
                await addBookmark(item.id);
                setBookmarked(true);
                onBookmarkedChange?.(item, true);
              }
            } finally {
              setBookmarking(false);
            }
          }}
          className="shrink-0 p-1.5 rounded-full text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--accent)] transition-colors disabled:opacity-50"
          aria-label={tPost("bookmark")}
        >
          {bookmarked ? (
            <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
              <path d="M4 4.5C4 3.12 5.119 2 6.5 2h11C18.881 2 20 3.12 20 4.5v18.44l-8-5.71-8 5.71V4.5z" />
            </svg>
          ) : (
            <svg className="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
              <path d="M4 4.5C4 3.12 5.119 2 6.5 2h11C18.881 2 20 3.12 20 4.5v18.44l-8-5.71-8 5.71V4.5z" />
            </svg>
          )}
        </button>
        <button
          type="button"
          onClick={async (e) => {
            e.preventDefault();
            e.stopPropagation();
            const url = await buildShareUrl(
              typeof window !== "undefined" ? window.location.origin : "",
              locale,
              item.id,
              item.category === "dark" ? "dark" : "light"
            );
            if (typeof navigator !== "undefined" && navigator.share) {
              navigator.share({ url, title: item.name }).catch(() => { });
            } else {
              void navigator.clipboard?.writeText(url);
            }
          }}
          className="shrink-0 p-1.5 rounded-full text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--accent)] transition-colors"
          aria-label={tPost("share")}
        >
          <svg className="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
            <path d="M4 12v8a2 2 0 002 2h12a2 2 0 002-2v-8M16 6l-4-4-4 4M12 2v13" />
          </svg>
        </button>
      </div>
      <div className="px-4 pb-3">
        <div className="mt-2 relative w-full rounded-2xl overflow-hidden border border-[var(--border)] bg-[var(--muted)]/20 min-h-[200px]">
          {item.purchased && item.first_file_id ? (
            <HlsPlayer
              fileId={item.first_file_id}
              contentId={item.id}
              className="block w-full h-auto object-contain"
              poster={previewSrc ?? undefined}
              playLabel={tPost("playVideo")}
              verticalAspectFactor={1.5}
            />
          ) : previewSrc ? (
            <VerticalAspectImage
              src={previewSrc}
              className="block w-full h-auto object-contain"
              factor={1.5}
            />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center min-h-[200px]">
              <div className="w-14 h-14 rounded-full bg-[var(--foreground)]/80 flex items-center justify-center">
                <svg
                  className="w-6 h-6 text-white ml-1"
                  fill="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path d="M8 5v14l11-7z" />
                </svg>
              </div>
            </div>
          )}

          {!item.purchased && (
            <div className="absolute inset-0 flex flex-col items-center justify-center bg-black/40 backdrop-blur-[2px] text-left">
              <span className="text-white font-semibold text-base drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)] [text-shadow:0_0_12px_rgba(0,0,0,0.9),0_1px_3px_rgba(0,0,0,1)]">
                {hasAuth ? tFeed("unlockToView") : tPost("signUpToView")}
              </span>
              {item.price > 0 && hasAuth && (
                <span className="text-white/95 text-sm mt-1.5 font-medium drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)] [text-shadow:0_0_8px_rgba(0,0,0,0.8),0_1px_2px_rgba(0,0,0,1)]">
                  {item.price} {tFeed("credits")}
                </span>
              )}
              {hasAuth ? (
                <button
                  type="button"
                  onClick={handlePurchase}
                  disabled={purchasing || item.price <= 0}
                  className="mt-4 rounded-lg bg-[var(--accent)] px-6 py-2 text-sm font-medium text-[var(--accent-foreground)] disabled:opacity-50 hover:opacity-90 cursor-pointer disabled:cursor-wait"
                >
                  {purchasing ? tFeed("loading") + "…" : tPost("purchase")}
                </button>
              ) : (
                <Link
                  href={`/${locale}/auth/signup`}
                  className="mt-4 rounded-lg bg-[var(--accent)] px-6 py-2 text-sm font-medium text-[var(--accent-foreground)] hover:opacity-90"
                >
                  {tAuth("signUp")}
                </Link>
              )}
            </div>
          )}
        </div>
        {canEditContent && (
          <div className="mt-2 w-full px-1 self-start">
            <Link
              href={`/${locale}/post/${item.id}/edit`}
              onClick={(e) => e.stopPropagation()}
              className="inline-flex items-center gap-1.5 text-sm text-[var(--muted)] hover:text-[var(--accent)] transition-colors"
              aria-label={tPost("edit")}
            >
              <svg className="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
                <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" />
                <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" />
              </svg>
              {tPost("edit")}
            </Link>
          </div>
        )}
      </div>
    </article>
  );
}
