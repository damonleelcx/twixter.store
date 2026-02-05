"use client";

import { useState } from "react";

type VerticalAspectImageProps = {
  src: string;
  className?: string;
  /** For vertical images: displayed aspect ratio = (width * factor) / height. 1/1.5 = decrease ratio by 1.5 (less compressed). */
  factor?: number;
  /** Called when the image fails to load (e.g. 401 for preview when not signed in). */
  onError?: () => void;
};

export function VerticalAspectImage({ src, className, factor = 1 / 1.5, onError }: VerticalAspectImageProps) {
  const [wrapperStyle, setWrapperStyle] = useState<React.CSSProperties>({});

  const onLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget;
    const w = img.naturalWidth;
    const h = img.naturalHeight;
    if (h > w && h > 0) {
      setWrapperStyle({ aspectRatio: (w * factor) / h });
    } else {
      setWrapperStyle({});
    }
  };

  return (
    <div className="w-full overflow-hidden" style={wrapperStyle}>
      <img
        src={src}
        alt=""
        className={className}
        onLoad={onLoad}
        onError={onError}
        style={wrapperStyle.aspectRatio ? { width: "100%", height: "100%", objectFit: "contain" } : undefined}
      />
    </div>
  );
}
