export function TweetCard({
  name,
  handle,
  time,
  text,
  videoUrl,
}: {
  name: string;
  handle: string;
  time: string;
  text: string;
  videoUrl?: string;
}) {
  return (
    <article className="flex gap-3 border-b border-[var(--border)] px-4 py-3 transition-colors hover:bg-[var(--hover)]">
      <div className="h-10 w-10 shrink-0 rounded-full bg-[var(--muted)]/30 flex items-center justify-center">
        <span className="text-sm font-semibold text-[var(--muted)]">
          {name.charAt(0)}
        </span>
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1">
          <span className="font-semibold text-[var(--foreground)] truncate">
            {name}
          </span>
          <span className="text-[var(--muted)] truncate">{handle}</span>
          <span className="text-[var(--muted)]">·</span>
          <time className="text-[var(--muted)] text-sm">{time}</time>
        </div>
        <p className="mt-0.5 break-words text-[15px] leading-5">{text}</p>
        {videoUrl && (
          <div className="mt-2 rounded-2xl overflow-hidden border border-[var(--border)] bg-[var(--muted)]/20 aspect-video max-h-[280px] flex items-center justify-center">
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
      </div>
    </article>
  );
}
