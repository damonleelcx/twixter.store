"use client";

import { Icon } from "@/components/Icon";
import type { ContentFeedItem } from "@/lib/api";
import { fetchCurrentUser, fetchSlideshowFeed, getAccessToken, getApiBase } from "@/lib/api";
import { getViewingCookie, getViewingFromSearchParams } from "@/lib/viewing";
import Hls from "hls.js";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslations } from "next-intl";

type Category = "light" | "dark";

const PAGE_SIZE = 10;
const AUTO_ADVANCE_MS = 60_000;

function getQueryCategory(): Category {
  if (typeof window === "undefined") return "light";
  const raw = new URLSearchParams(window.location.search).get("category");
  return raw === "dark" ? "dark" : "light";
}

function Slide({
  item,
  active,
  onAutoNext,
  topInsetPx,
  t,
  soundEnabled,
  requestEnableSound,
  category,
  viewingToken,
}: {
  item: ContentFeedItem;
  active: boolean;
  onAutoNext: () => void;
  topInsetPx: number;
  t: (key: string) => string;
  soundEnabled: boolean;
  requestEnableSound: () => void;
  category: Category;
  viewingToken: string | null;
}) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const hlsRef = useRef<Hls | null>(null);
  const autoTimerRef = useRef<number | null>(null);
  const didSeekMiddleRef = useRef(false);
  const overlayTimerRef = useRef<number | null>(null);
  const countdownTimerRef = useRef<number | null>(null);
  const elapsedMsRef = useRef(0);
  const lastTickAtRef = useRef<number | null>(null);
  const [isPaused, setIsPaused] = useState(true);
  const [showCenterOverlay, setShowCenterOverlay] = useState(false);
  const [remainingSec, setRemainingSec] = useState(60);
  const token = getAccessToken();
  const streamUrl = useMemo(() => {
    const base = `${getApiBase()}/content/files/${item.first_file_id}/slideshow-preview`;
    // For dark category without auth, pass viewing token so backend can set twixter_viewing cookie.
    if (category === "dark" && !token && viewingToken) {
      return `${base}?viewing=${encodeURIComponent(viewingToken)}`;
    }
    return base;
  }, [category, item.first_file_id, token, viewingToken]);

  const flashOverlay = (paused: boolean) => {
    setIsPaused(paused);
    setShowCenterOverlay(true);
    if (overlayTimerRef.current != null) window.clearTimeout(overlayTimerRef.current);
    overlayTimerRef.current = window.setTimeout(() => setShowCenterOverlay(false), 650);
  };

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    if (Hls.isSupported()) {
      const hls = new Hls({
        xhrSetup(xhr) {
          // Slideshow preview supports anonymous (light). For dark, backend will require auth.
          xhr.withCredentials = true; // send twixter_viewing cookie cross-port
          if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);
        },
      });
      hlsRef.current = hls;
      hls.loadSource(streamUrl);
      hls.attachMedia(video);
      return () => {
        hls.destroy();
        hlsRef.current = null;
      };
    }

    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = streamUrl;
      return () => {
        video.removeAttribute("src");
      };
    }
  }, [streamUrl, token]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const clearAuto = () => {
      if (autoTimerRef.current != null) {
        window.clearTimeout(autoTimerRef.current);
        autoTimerRef.current = null;
      }
    };
    const clearCountdown = () => {
      if (countdownTimerRef.current != null) {
        window.clearInterval(countdownTimerRef.current);
        countdownTimerRef.current = null;
      }
      lastTickAtRef.current = null;
    };

    const onLoadedMetadata = () => {
      if (!active) return;
      if (!Number.isFinite(video.duration) || video.duration <= 0) return;
      if (didSeekMiddleRef.current) return;
      didSeekMiddleRef.current = true;
      const mid = Math.max(0, video.duration / 2);
      try {
        video.currentTime = mid;
      } catch {
        // ignore seek errors
      }
    };

    const onEnded = () => {
      if (!active) return;
      onAutoNext();
    };

    const onPlay = () => {
      setIsPaused(false);
      // start countdown ticking while playing
      if (countdownTimerRef.current == null) {
        lastTickAtRef.current = Date.now();
        countdownTimerRef.current = window.setInterval(() => {
          const now = Date.now();
          const last = lastTickAtRef.current ?? now;
          const delta = Math.max(0, now - last);
          lastTickAtRef.current = now;
          elapsedMsRef.current = Math.min(AUTO_ADVANCE_MS, elapsedMsRef.current + delta);
          const remainMs = Math.max(0, AUTO_ADVANCE_MS - elapsedMsRef.current);
          setRemainingSec(Math.max(0, Math.ceil(remainMs / 1000)));
        }, 250);
      }
    };
    const onPause = () => {
      setIsPaused(true);
      clearCountdown();
    };

    video.addEventListener("loadedmetadata", onLoadedMetadata);
    video.addEventListener("ended", onEnded);
    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);

    if (active) {
      didSeekMiddleRef.current = false;
      clearAuto();
      clearCountdown();
      elapsedMsRef.current = 0;
      setRemainingSec(60);
      autoTimerRef.current = window.setTimeout(() => onAutoNext(), AUTO_ADVANCE_MS);
      // Autoplay policies: default to muted until user enables sound.
      video.muted = !soundEnabled;
      video.playsInline = true;
      video
        .play()
        .catch(() => {
          // ignore autoplay failures; user can tap to play
        });
    } else {
      clearAuto();
      clearCountdown();
      video.pause();
    }

    return () => {
      clearAuto();
      clearCountdown();
      video.removeEventListener("loadedmetadata", onLoadedMetadata);
      video.removeEventListener("ended", onEnded);
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
    };
  }, [active, onAutoNext, soundEnabled]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = !soundEnabled;
    if (soundEnabled) {
      video.volume = 1;
    }
  }, [soundEnabled]);

  useEffect(() => {
    return () => {
      if (overlayTimerRef.current != null) window.clearTimeout(overlayTimerRef.current);
      if (countdownTimerRef.current != null) window.clearInterval(countdownTimerRef.current);
    };
  }, []);

  return (
    <div className="relative h-screen w-full snap-start bg-black">
      <video
        ref={videoRef}
        className="h-full w-full object-contain"
        controls={false}
        playsInline
        preload="metadata"
        onClick={() => {
          const v = videoRef.current;
          if (!v) return;
          if (!soundEnabled) {
            requestEnableSound();
          }
          if (v.paused) {
            v.play().catch(() => {});
            flashOverlay(false);
          } else {
            v.pause();
            flashOverlay(true);
          }
        }}
      />
      {/* Center play/pause overlay */}
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        <div
          className={`flex items-center justify-center rounded-full bg-black/45 backdrop-blur-sm transition-opacity duration-200 ${
            showCenterOverlay ? "opacity-100" : "opacity-0"
          }`}
          style={{ width: 72, height: 72 }}
          aria-hidden
        >
          <Icon
            d={
              isPaused
                ? "M8 5v14l11-7z"
                : "M6 5h4v14H6V5zm8 0h4v14h-4V5z"
            }
            className="h-8 w-8 text-white"
          />
        </div>
      </div>
      {/* Bottom gradient + meta to avoid overlapping the fixed header */}
      <div className="pointer-events-none absolute inset-x-0 bottom-0">
        <div className="h-28 bg-linear-to-t from-black/70 to-transparent" />
        <div className="px-4 pb-[max(16px,env(safe-area-inset-bottom))]">
          <div className="mb-2 flex items-end justify-between gap-3">
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold text-white">{item.name}</div>
              {item.description ? (
                <div className="line-clamp-2 max-w-[80vw] text-xs text-white/75">
                  {item.description}
                </div>
              ) : null}
            </div>
            <div className="shrink-0 rounded-full bg-black/45 px-3 py-1 text-[11px] text-white/90">
              {remainingSec}
              {t("countdownSuffix")}
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-[11px] text-white/75">
            <span className="rounded-full bg-black/45 px-2 py-1">{t("hintStartMiddle")}</span>
            <span className="rounded-full bg-black/45 px-2 py-1">{t("hintAutoNext")}</span>
            <span className="rounded-full bg-black/25 px-2 py-1">{t("tapToPlayPause")}</span>
          </div>
        </div>
      </div>
      {/* Spacer overlay so video content doesn't sit under header area (for readability). */}
      <div className="pointer-events-none absolute inset-x-0 top-0" style={{ height: topInsetPx }} />
    </div>
  );
}

export function SlideshowClient() {
  const params = useParams<{ locale: string }>();
  const locale = typeof params?.locale === "string" ? params.locale : "en";
  const t = useTranslations("slideshow");
  const [category, setCategory] = useState<Category>(() => getQueryCategory());
  const [items, setItems] = useState<ContentFeedItem[]>([]);
  const [cursor, setCursor] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const [activeIndex, setActiveIndex] = useState(0);
  const [loading, setLoading] = useState(false);
  const [errMsg, setErrMsg] = useState<string | null>(null);
  const [canViewNsfw, setCanViewNsfw] = useState<boolean>(false);
  const [topInsetPx, setTopInsetPx] = useState(56);
  const [soundEnabled, setSoundEnabled] = useState(false);
  const [hasViewingToken, setHasViewingToken] = useState(false);
  const [viewingToken, setViewingToken] = useState<string | null>(null);

  const containerRef = useRef<HTMLDivElement | null>(null);
  const slideRefs = useRef<Array<HTMLDivElement | null>>([]);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (cancelled) return;
      setCanViewNsfw(Boolean(u?.permissions?.includes("can_view_nsfw")));
    });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const cookie = getViewingCookie();
    const param =
      typeof window !== "undefined"
        ? getViewingFromSearchParams(new URLSearchParams(window.location.search))
        : null;
    const v = cookie || param;
    setViewingToken(v);
    setHasViewingToken(Boolean(v));
  }, []);

  useEffect(() => {
    const compute = () => {
      // Fixed header: py-3 + icon size. Add a bit more to avoid overlap with video content.
      const safeTop =
        typeof window !== "undefined"
          ? Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue("--sat") || "0")
          : 0;
      // Use CSS env(safe-area-inset-top) via inline style; JS fallback just uses a constant.
      setTopInsetPx(72 + (Number.isFinite(safeTop) ? safeTop : 0));
    };
    compute();
    window.addEventListener("resize", compute);
    return () => window.removeEventListener("resize", compute);
  }, []);

  async function loadPage(reset: boolean) {
    if (loading) return;
    setLoading(true);
    setErrMsg(null);
    try {
      const nextCursor = reset ? 0 : cursor;
      const res = await fetchSlideshowFeed(category, nextCursor, PAGE_SIZE, "created_at");
      setItems((prev) => (reset ? res.items : [...prev, ...res.items]));
      setCursor(res.next_cursor);
      setHasMore(res.has_more);
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setErrMsg(msg || "加载失败");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    setItems([]);
    setCursor(0);
    setHasMore(true);
    setActiveIndex(0);
    slideRefs.current = [];
    loadPage(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [category]);

  useEffect(() => {
    const root = containerRef.current;
    if (!root) return;
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => (b.intersectionRatio ?? 0) - (a.intersectionRatio ?? 0))[0];
        if (!visible) return;
        const idxStr = (visible.target as HTMLElement).dataset.index;
        const idx = idxStr ? Number(idxStr) : NaN;
        if (!Number.isFinite(idx)) return;
        setActiveIndex(idx);
      },
      { root, threshold: [0.6] }
    );
    slideRefs.current.forEach((el) => el && observer.observe(el));
    return () => observer.disconnect();
  }, [items.length]);

  useEffect(() => {
    if (!hasMore) return;
    if (activeIndex >= items.length - 3) {
      loadPage(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeIndex, hasMore, items.length]);

  function scrollToIndex(nextIdx: number) {
    const el = slideRefs.current[nextIdx];
    if (!el) return;
    el.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  return (
    <div className="h-screen w-full bg-black">
      <div
        className="fixed left-0 right-0 top-0 z-10 flex items-center justify-between px-4 py-3 text-white"
        style={{ paddingTop: "max(12px, env(safe-area-inset-top))" }}
      >
        <div className="flex items-center gap-2">
          <a
            href={`/${locale}`}
            className="inline-flex items-center justify-center rounded-full bg-white/10 p-2 hover:bg-white/20"
            aria-label={t("back")}
          >
            <Icon
              d="M15 19l-7-7 7-7"
              className="h-5 w-5"
            />
          </a>
          <div className="text-sm font-semibold">{t("title")}</div>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setSoundEnabled((v) => !v)}
            className="inline-flex items-center gap-2 rounded-full bg-white/10 px-3 py-1 text-xs hover:bg-white/20"
            aria-label={soundEnabled ? t("soundOn") : t("soundOff")}
            title={soundEnabled ? t("soundOn") : t("soundOff")}
          >
            <Icon
              d={
                soundEnabled
                  ? "M11 5L6 9H2v6h4l5 4V5zm7.07 1.93a10 10 0 010 14.14M15.54 10.46a4 4 0 010 3.08"
                  : "M11 5L6 9H2v6h4l5 4V5zm9.19 2.81l-1.41-1.41L5.22 20.96l1.41 1.41L20.19 8.81z"
              }
              className="h-4 w-4"
            />
            <span className="hidden sm:inline">{soundEnabled ? t("soundOn") : t("soundOff")}</span>
          </button>
          <button
            type="button"
            className={`rounded-full px-3 py-1 text-xs ${category === "light" ? "bg-white/20" : "bg-white/10 hover:bg-white/20"}`}
            onClick={() => setCategory("light")}
          >
            {t("light")}
          </button>
          <button
            type="button"
            disabled={!canViewNsfw && !hasViewingToken}
            title={!canViewNsfw && !hasViewingToken ? t("needNsfwPermission") : ""}
            className={`rounded-full px-3 py-1 text-xs ${
              category === "dark"
                ? "bg-white/20"
                : canViewNsfw || hasViewingToken
                  ? "bg-white/10 hover:bg-white/20"
                  : "bg-white/5 text-white/40 cursor-not-allowed"
            }`}
            onClick={() => setCategory("dark")}
          >
            {t("dark")}
          </button>
        </div>
      </div>

      {errMsg ? (
        <div className="flex h-screen items-center justify-center p-6 text-center text-white/80">
          <div className="max-w-md">
            <div className="mb-3 text-sm font-semibold">{t("loadFailed")}</div>
            <div className="mb-4 text-xs whitespace-pre-wrap">{errMsg}</div>
            <button
              type="button"
              className="rounded-full bg-white/15 px-4 py-2 text-sm hover:bg-white/25"
              onClick={() => loadPage(items.length === 0)}
            >
              {t("retry")}
            </button>
          </div>
        </div>
      ) : null}

      <div
        ref={containerRef}
        className="h-screen w-full snap-y snap-mandatory overflow-y-scroll"
        style={{ scrollBehavior: "smooth" }}
      >
        {/* padding so first slide isn't hidden behind fixed header */}
        <div style={{ height: "max(0px, env(safe-area-inset-top))" }} />
        {items.map((it, idx) => (
          <div
            key={`${it.id}-${idx}`}
            ref={(el) => {
              slideRefs.current[idx] = el;
            }}
            data-index={idx}
          >
            <Slide
              item={it}
              active={idx === activeIndex}
              onAutoNext={() => scrollToIndex(Math.min(idx + 1, items.length - 1))}
              topInsetPx={topInsetPx}
              t={t}
              soundEnabled={soundEnabled}
              requestEnableSound={() => setSoundEnabled(true)}
              category={category}
              viewingToken={viewingToken}
            />
          </div>
        ))}

        {loading ? (
          <div className="flex h-28 items-center justify-center text-xs text-white/60">{t("loading")}</div>
        ) : null}
        {!loading && items.length > 0 && !hasMore ? (
          <div className="flex h-28 items-center justify-center text-xs text-white/60">{t("noMore")}</div>
        ) : null}
      </div>
    </div>
  );
}

