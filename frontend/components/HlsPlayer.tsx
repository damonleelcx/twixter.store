"use client";

import { getAccessToken, getApiBase } from "@/lib/api";
import Hls from "hls.js";
import { useCallback, useEffect, useRef, useState } from "react";

type HlsPlayerProps = {
  fileId: number;
  className?: string;
  poster?: string;
  /** Optional "Play video" label for overlay (theme-aware). */
  playLabel?: string;
  /** For vertical video: displayed aspect ratio = (width * factor) / height. Default 1/1.5 (taller). Use 1.5 in cards to shorten vertical videos. */
  verticalAspectFactor?: number;
};

const DEFAULT_VERTICAL_ASPECT_FACTOR = 1 / 1.5;

export function HlsPlayer({ fileId, className, poster, playLabel = "Play video", verticalAspectFactor }: HlsPlayerProps) {
  const factor = verticalAspectFactor ?? DEFAULT_VERTICAL_ASPECT_FACTOR;
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
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

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const onPlay = () => setShowPlayOverlay(false);
    const onPause = () => setShowPlayOverlay(true);
    const onEnded = () => setShowPlayOverlay(true);

    video.addEventListener("play", onPlay);
    video.addEventListener("pause", onPause);
    video.addEventListener("ended", onEnded);
    return () => {
      video.removeEventListener("play", onPlay);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("ended", onEnded);
    };
  }, []);

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
