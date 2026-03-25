/**
 * Returns a human-readable resolution label (e.g. HD, 4K) from width×height.
 * Uses the shorter side as the "p" value (e.g. 1920×1080 → Full HD).
 */
export function getResolutionLabel(width: number, height: number): string | null {
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    return null;
  }
  const p = Math.min(width, height);
  if (p >= 4320) return "8K";
  if (p >= 2160) return "4K";
  if (p >= 1440) return "2K";
  if (p >= 1080) return "Full HD";
  if (p >= 720) return "HD";
  if (p >= 480) return "SD";
  return null;
}
