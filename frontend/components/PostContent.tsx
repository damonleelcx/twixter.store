"use client";

import {
  addBookmark,
  deleteContent,
  fetchCurrentUser,
  getAccessToken,
  getContentById,
  getContentFilePreviewUrlSameOrigin,
  isInsufficientCreditsError,
  purchaseContent,
  removeBookmark,
  type ContentDetail,
} from "@/lib/api";
import { buildShareUrl } from "@/lib/viewing";
import { useLocale, useTranslations } from "next-intl";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { HlsPlayer } from "./HlsPlayer";
import { VerticalAspectImage } from "./VerticalAspectImage";

/** Format created_at to Twitter-style relative time */
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

type PostContentProps = {
  contentId: number;
  /** Server-fetched content (SSR). If null, show "Content not found". If undefined, client will fetch. */
  initialData?: ContentDetail | null;
};

export function PostContent({ contentId, initialData }: PostContentProps) {
  const locale = useLocale();
  const router = useRouter();
  const searchParams = useSearchParams();
  const t = useTranslations("postDetail");
  const viewingToken = searchParams.get("viewing");
  const [data, setData] = useState<ContentDetail | null>(() =>
    initialData !== undefined ? initialData : null
  );
  const [loading, setLoading] = useState(() => initialData === undefined);
  const [error, setError] = useState<string | null>(() =>
    initialData === null ? "Content not found" : null
  );
  const [purchasing, setPurchasing] = useState(false);
  const [justPurchased, setJustPurchased] = useState(false);
  const [bookmarked, setBookmarked] = useState(() => initialData?.content?.bookmarked ?? false);
  const [bookmarking, setBookmarking] = useState(false);
  const [canEditContent, setCanEditContent] = useState(false);
  const [canDeleteContent, setCanDeleteContent] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [lockedPreviewFailed, setLockedPreviewFailed] = useState(false);

  useEffect(() => {
    fetchCurrentUser().then((user) => {
      const perms = user?.permissions ?? [];
      setCanEditContent(perms.includes("can_edit_content"));
      setCanDeleteContent(perms.includes("can_delete_content"));
    });
  }, []);

  const load = useCallback(async (bypassCache?: boolean) => {
    setLoading(true);
    setError(null);
    try {
      const res = await getContentById(contentId, viewingToken, bypassCache ? { cache: "no-store" } : undefined);
      setData(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [contentId, viewingToken]);

  useEffect(() => {
    if (initialData === undefined) {
      load();
    } else if (initialData === null) {
      // Server had no cookie or backend error — retry once with client token (localStorage)
      load();
    }
  }, [initialData, load]);

  // After redirect from edit page (?updated=1), refetch with no-store so the detail shows latest data
  useEffect(() => {
    if (searchParams.get("updated") === "1") {
      load(true);
      router.replace(`/${locale}/post/${contentId}`, { scroll: false });
    }
  }, [contentId, locale, load, router, searchParams]);

  // Sync bookmarked from loaded content
  useEffect(() => {
    if (data?.content?.bookmarked !== undefined) setBookmarked(data.content.bookmarked);
  }, [data?.content?.bookmarked]);

  // Reset locked preview error when content/files change
  useEffect(() => {
    setLockedPreviewFailed(false);
  }, [data?.content?.id, data?.files]);

  // 后端按权限返回 gif_file_url（已购买=正常 GIF，未购买=模糊 GIF），直接使用
  const handlePurchase = useCallback(async () => {
    if (!getAccessToken()) return;
    setPurchasing(true);
    try {
      await purchaseContent(contentId);
      setJustPurchased(true);
      await load();
    } catch (err) {
      if (isInsufficientCreditsError(err)) {
        router.push(`/${locale}/credits`);
        return;
      }
      setError(err instanceof Error ? err.message : "Purchase failed");
    } finally {
      setPurchasing(false);
    }
  }, [contentId, load, locale, router]);

  const timeAgo = useMemo(
    () => (data?.content?.created_at ? formatTimestamp(data.content.created_at) : ""),
    [data?.content?.created_at]
  );

  if (loading && !data) {
    return (
      <div className="flex min-h-[200px] items-center justify-center py-12">
        <p className="text-[var(--muted)]">Loading...</p>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="px-4 py-8 text-center">
        <p className="text-red-600 dark:text-red-400">{error}</p>
        <Link
          href={`/${locale}`}
          className="mt-4 inline-block rounded-lg border border-[var(--border)] px-4 py-2 text-sm hover:bg-[var(--hover)]"
        >
          {t("back")}
        </Link>
      </div>
    );
  }

  if (!data) return null;

  const { content, files } = data;
  const firstFile = files[0];
  const purchased = content.purchased || justPurchased;
  const displayName = content.author_username || content.name;
  const avatarInitial = (displayName.charAt(0) || "?").toUpperCase();

  // 未购买时预览：必须用预览流式 API（同源 URL 带 cookie，后端按权限返回模糊/清晰 GIF）
  const lockedPreviewSrc = firstFile ? getContentFilePreviewUrlSameOrigin(firstFile.id) : undefined;
  // 视频时长或分辨率文案（未购买遮罩用）
  const videoMetaLabel =
    firstFile?.duration != null && firstFile.duration > 0
      ? `${Math.floor(firstFile.duration / 60)}:${String(
          Math.floor(firstFile.duration % 60)
        ).padStart(2, "0")}`
      : firstFile?.width != null &&
          firstFile?.height != null &&
          firstFile.width > 0 &&
          firstFile.height > 0
        ? `${firstFile.width}×${firstFile.height}`
        : null;

  // 未购买时 wrapper 宽高比需与内容一致（竖屏时按 1.5 倍压缩）；无尺寸时不强制，避免横屏被误成竖屏
  const lockedAspectRatio =
    firstFile?.width != null &&
    firstFile?.height != null &&
    firstFile.width > 0 &&
    firstFile.height > 0
      ? firstFile.height > firstFile.width
        ? (firstFile.width * 1.5) / firstFile.height
        : firstFile.width / firstFile.height
      : undefined;

  return (
    <>
      <header className="sticky top-0 z-10 flex h-14 items-center gap-4 border-b border-[var(--border)] bg-[var(--background)]/80 px-4 backdrop-blur-md">
        <Link
          href={`/${locale}`}
          className="-ml-1 flex h-9 w-9 items-center justify-center rounded-full text-[var(--foreground)] transition-colors hover:bg-[var(--hover)]"
          aria-label={t("back")}
        >
          <svg
            className="h-5 w-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M7 16l-4-4m0 0l4-4m-4 4h18"
            />
          </svg>
        </Link>
        <h1 className="flex-1 text-xl font-bold">{t("title")}</h1>
        <button
          type="button"
          disabled={!getAccessToken() || bookmarking}
          onClick={async () => {
            if (!getAccessToken() || bookmarking || !data?.content) return;
            setBookmarking(true);
            try {
              if (bookmarked) {
                await removeBookmark(data.content.id);
                setBookmarked(false);
              } else {
                await addBookmark(data.content.id);
                setBookmarked(true);
              }
            } finally {
              setBookmarking(false);
            }
          }}
          className="flex h-9 w-9 items-center justify-center rounded-full text-[var(--muted)] transition-colors hover:bg-[var(--hover)] hover:text-[var(--accent)] disabled:opacity-50"
          title={t("bookmark")}
          aria-label={t("bookmark")}
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
          onClick={async () => {
            const url = await buildShareUrl(
              typeof window !== "undefined" ? window.location.origin : "",
              locale,
              contentId,
              content.category === "dark" ? "dark" : "light"
            );
            if (typeof navigator !== "undefined" && navigator.share) {
              navigator.share({ url, title: content.name }).catch(() => {});
            } else {
              void navigator.clipboard?.writeText(url);
            }
          }}
          className="flex h-9 w-9 items-center justify-center rounded-full text-[var(--muted)] transition-colors hover:bg-[var(--hover)] hover:text-[var(--accent)]"
          title={t("share")}
          aria-label={t("share")}
        >
          <svg className="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
            <path d="M4 12v8a2 2 0 002 2h12a2 2 0 002-2v-8M16 6l-4-4-4 4M12 2v13" />
          </svg>
        </button>
      </header>

      <article className="border-b border-[var(--border)] px-4 pt-3 pb-2">
        <div className="flex gap-3">
          {content.author_avatar ? (
            <img
              src={content.author_avatar}
              alt=""
              className="h-12 w-12 shrink-0 rounded-full object-cover"
            />
          ) : (
            <div className="h-12 w-12 shrink-0 rounded-full bg-[var(--muted)]/30 flex items-center justify-center">
              <span className="text-base font-semibold text-[var(--muted)]">
                {avatarInitial}
              </span>
            </div>
          )}
          <div className="min-w-0 flex-1">
            {/* 1) Upload user (author + time + duration/dimension) */}
            <div className="flex flex-wrap items-center gap-1 text-[15px]">
              <span className="font-bold text-[var(--foreground)]">
                {displayName}
              </span>
              <span className="text-[var(--muted)]">
                · {timeAgo}
              </span>
              {videoMetaLabel && (
                <span className="text-[var(--muted)]">
                  · {videoMetaLabel}
                </span>
              )}
            </div>
            {/* 2) Title */}
            <p className="mt-0.5 font-medium text-[var(--foreground)] text-[17px] leading-5">
              {content.name}
            </p>
            {/* 3) Description */}
            {content.description && (
              <p className="mt-1 break-words text-[17px] leading-[1.35] text-[var(--foreground)]">
                {content.description}
              </p>
            )}
            {/* 4) Tags (clickable → tag feed) */}
            {Array.isArray(content.tags) && content.tags.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-2">
                {content.tags.map((tag) => (
                  <Link
                    key={tag}
                    href={`/${locale}/tag/${encodeURIComponent(tag)}`}
                    className="inline-flex items-center rounded-full bg-[var(--muted)]/30 px-3 py-1 text-sm text-[var(--foreground)] hover:bg-[var(--muted)]/50 hover:text-[var(--accent)]"
                  >
                    #{tag}
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>

        <div
          className="mt-3 relative w-full rounded-2xl overflow-hidden border border-[var(--border)] bg-[var(--muted)]/20 min-h-[200px]"
          style={
            !purchased && lockedAspectRatio != null
              ? { aspectRatio: lockedAspectRatio }
              : undefined
          }
        >
          {purchased && firstFile ? (
            <HlsPlayer
              fileId={firstFile.id}
              contentId={contentId}
              durationSeconds={firstFile.duration}
              className="block w-full h-auto object-contain"
              poster={getContentFilePreviewUrlSameOrigin(firstFile.id)}
              playLabel={t("playVideo")}
              verticalAspectFactor={1.5}
            />
          ) : (
            <>
              {lockedPreviewSrc && !lockedPreviewFailed ? (
                <VerticalAspectImage
                  src={lockedPreviewSrc}
                  className="block w-full h-auto object-contain"
                  factor={1.5}
                  onError={() => setLockedPreviewFailed(true)}
                />
              ) : (
                <div className="absolute inset-0 min-h-[200px] flex items-center justify-center">
                  <div className="w-16 h-16 rounded-full bg-[var(--foreground)]/80 flex items-center justify-center">
                    <svg
                      className="w-8 h-8 text-white ml-1"
                      fill="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path d="M8 5v14l11-7z" />
                    </svg>
                  </div>
                </div>
              )}
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-white/10 backdrop-blur-[2px]">
                <span className="text-white font-medium">{t("unlockToView")}</span>
                {content.price > 0 && (
                  <span className="text-white/90 text-sm mt-1">
                    {content.price} {t("credits")}
                  </span>
                )}
                {getAccessToken() ? (
                  <button
                    type="button"
                    onClick={handlePurchase}
                    disabled={purchasing}
                    className="mt-4 rounded-lg bg-[var(--accent)] px-6 py-2 text-sm font-medium text-[var(--accent-foreground)] disabled:opacity-50 hover:opacity-90"
                  >
                    {purchasing ? t("purchasing") : t("purchase")}
                  </button>
                ) : (
                  <Link
                    href={`/${locale}/auth/signup`}
                    className="mt-4 rounded-lg bg-[var(--accent)] px-6 py-2 text-sm font-medium text-[var(--accent-foreground)] hover:opacity-90"
                  >
                    {t("signUpToView")}
                  </Link>
                )}
              </div>
            </>
          )}
        </div>
        {(canEditContent || canDeleteContent) && (
          <div className="mt-3 px-0 flex items-center gap-4">
            {canEditContent && (
              <Link
                href={`/${locale}/post/${contentId}/edit`}
                className="inline-flex items-center gap-1.5 text-sm text-[var(--muted)] hover:text-[var(--accent)] transition-colors"
                aria-label={t("edit")}
              >
                <svg className="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
                  <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" />
                  <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" />
                </svg>
                {t("edit")}
              </Link>
            )}
            {canDeleteContent && (
              <button
                type="button"
                onClick={async () => {
                  if (!confirm(`${t("deleteConfirm")}\n${t("deleteConfirmMessage")}`)) return;
                  setDeleting(true);
                  try {
                    await deleteContent(contentId);
                    router.push(`/${locale}`);
                  } catch {
                    setDeleting(false);
                  }
                }}
                disabled={deleting}
                className="inline-flex items-center gap-1.5 text-sm text-[var(--muted)] hover:text-red-600 transition-colors disabled:opacity-50"
                aria-label={t("delete")}
              >
                <svg className="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
                  <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2M10 11v6M14 11v6" strokeLinecap="round" strokeLinejoin="round" />
                </svg>
                {t("delete")}
              </button>
            )}
          </div>
        )}
      </article>
    </>
  );
}
