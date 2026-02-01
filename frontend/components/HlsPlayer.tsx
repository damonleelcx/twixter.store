"use client";

import { getAccessToken, getApiBase, recordWatchProgress } from "@/lib/api";
import Hls from "hls.js";
import { useCallback, useEffect, useRef, useState } from "react";

const WATCH_PROGRESS_INTERVAL_SEC = 30;

type HlsPlayerProps = {
  fileId: number;
  /** When set with durationSeconds, watch progress is reported for analytics (TotalWatchTime, AverageWatchTime, CompletionRate). */
  contentId?: number;
  /** Video duration in seconds (from content file). Used for completion rate when reporting watch progress. */
  durationSeconds?: number;
  className?: string;
  poster?: string;
  /** Optional "Play video" label for overlay (theme-aware). */
  playLabel?: string;
  /** For vertical video: displayed aspect ratio = (width * factor) / height. Default 1/1.5 (taller). Use 1.5 in cards to shorten vertical videos. */
  verticalAspectFactor?: number;
};

const DEFAULT_VERTICAL_ASPECT_FACTOR = 1 / 1.5;

export function HlsPlayer({
  fileId,
  contentId,
  durationSeconds,
  className,
  poster,
  playLabel = "Play video",
  verticalAspectFactor,
}: HlsPlayerProps) {
  const factor = verticalAspectFactor ?? DEFAULT_VERTICAL_ASPECT_FACTOR;
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const lastReportedTimeRef = useRef(0);
  const [showPlayOverlay, setShowPlayOverlay] = useState(true);
  const [wrapperStyle, setWrapperStyle] = useState<React.CSSProperties>({});

  const streamUrl = `${getApiBase()}/content/files/${fileId}/stream`;
  const token = getAccessToken();

  const handlePlay = useCallback(() => {
    const video = videoRef.current;
    if (video) {
      video.play();
      setShowPlayOverlay(false);
    }
  }, []);

  const reportProgress = useCallback(
    (watchTimeSeconds: number, durationSec?: number) => {
      if (!contentId || watchTimeSeconds <= 0) return;
      const duration = durationSec ?? durationSeconds;
      recordWatchProgress(contentId, watchTimeSeconds, duration).catch(() => {});
    },
    [contentId, durationSeconds]
  );

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => {
      setShowPlayOverlay(false);
      lastReportedTimeRef.current = video.currentTime;
    };
    const onPause = () => {
      setShowPlayOverlay(true);
      if (contentId) {
        const delta = Math.max(0, video.currentTime - lastReportedTimeRef.current);
        if (delta > 0) {
          reportProgress(delta, Number.isFinite(video.duration) ? video.duration : undefined);
          lastReportedTimeRef.current = video.currentTime;
        }
      }
    };
    const onEnded = () => {
      setShowPlayOverlay(true);
      if (contentId) {
        const delta = Math.max(0, video.currentTime - lastReportedTimeRef.current);
        if (delta > 0) {
          const dur = Number.isFinite(video.duration) ? video.duration : durationSeconds;
          reportProgress(delta, dur);
        }
        lastReportedTimeRef.current = video.currentTime;
      }
    };

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("ended", onEnded);
    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("ended", onEnded);
    };
  }, [contentId, durationSeconds, reportProgress]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !contentId) return;

    const onTimeUpdate = () => {
      const delta = video.currentTime - lastReportedTimeRef.current;
      if (delta >= WATCH_PROGRESS_INTERVAL_SEC) {
        reportProgress(delta, Number.isFinite(video.duration) ? video.duration : durationSeconds);
        lastReportedTimeRef.current = video.currentTime;
      }
    };

    video.addEventListener("timeupdate", onTimeUpdate);
    return () => video.removeEventListener("timeupdate", onTimeUpdate);
  }, [contentId, durationSeconds, reportProgress]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onLoadedMetadata = () => {
      const w = video.videoWidth;
      const h = video.videoHeight;
      if (h > w && h > 0) {
        setWrapperStyle({ aspectRatio: (w * factor) / h });
      } else {
        setWrapperStyle({});
      }
    };

    video.addEventListener("loadedmetadata", onLoadedMetadata);
    if (video.videoWidth > 0) onLoadedMetadata();
    return () => video.removeEventListener("loadedmetadata", onLoadedMetadata);
  }, [factor]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    if (Hls.isSupported()) {
      const hls = new Hls({
        xhrSetup(xhr) {
          if (token) {
            xhr.setRequestHeader("Authorization", `Bearer ${token}`);
          }
        },
      });
      hlsRef.current = hls;
      hls.loadSource(streamUrl);
      hls.attachMedia(video);
      hls.on(Hls.Events.MANIFEST_PARSED, (_event, data) => {
        if (data.levels.length) {
          // Optional: select quality
        }
      });
      return () => {
        hls.destroy();
        hlsRef.current = null;
      };
    }
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      // Safari native HLS: can't add Authorization header; use same URL (backend may allow cookie)
      video.src = streamUrl;
      return () => {
        video.removeAttribute("src");
      };
    }
  }, [streamUrl, token]);

  const isVerticalBox = "aspectRatio" in wrapperStyle && wrapperStyle.aspectRatio != null;

  return (
    <div className="relative w-full" style={wrapperStyle}>
      <video
        ref={videoRef}
        className={isVerticalBox ? "block w-full h-full object-contain" : className}
        poster={poster}
        controls
        playsInline
        preload="metadata"
      />
      {showPlayOverlay && (
        <button
          type="button"
          onClick={handlePlay}
          className="absolute inset-0 flex flex-col items-center justify-center bg-black/30 hover:bg-black/40 transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--accent)] focus:ring-offset-2 focus:ring-offset-[var(--background)]"
          aria-label={playLabel}
        >
          <span className="rounded-full bg-[var(--accent)] flex items-center justify-center w-14 h-14 text-[var(--accent-foreground)] shadow-lg hover:opacity-90">
            <svg className="w-7 h-7 ml-1" fill="currentColor" viewBox="0 0 24 24" aria-hidden>
              <path d="M8 5v14l11-7z" />
            </svg>
          </span>
          <span className="mt-2 text-sm font-medium text-[var(--accent-foreground)] drop-shadow-[0_1px_2px_rgba(0,0,0,0.5)]">
            {playLabel}
          </span>
        </button>
      )}
    </div>
  );
}
