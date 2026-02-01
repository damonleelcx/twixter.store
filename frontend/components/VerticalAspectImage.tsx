"use client";

import { useState } from "react";

type VerticalAspectImageProps = {
  src: string;
  className?: string;
  /** For vertical images: displayed aspect ratio = (width * factor) / height. 1/1.5 = decrease ratio by 1.5 (less compressed). */
  factor?: number;
};

export function VerticalAspectImage({ src, className, factor = 1 / 1.5 }: VerticalAspectImageProps) {
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
        style={wrapperStyle.aspectRatio ? { width: "100%", height: "100%", objectFit: "contain" } : undefined}
      />
    </div>
  );
}
