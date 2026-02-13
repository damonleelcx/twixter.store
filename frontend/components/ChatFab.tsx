"use client";

import { fetchCurrentUser } from "@/lib/api";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { Icon } from "./Icon";

/** Floating action button to open chat on mobile. Only visible when user has can_use_bot. */
export function ChatFab() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const [show, setShow] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled && u?.permissions?.includes("can_use_bot")) setShow(true);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  if (!show) return null;

  return (
    <Link
      href={`/${locale}/chat`}
      className="fixed bottom-20 right-4 z-30 flex h-14 w-14 items-center justify-center rounded-full bg-[var(--accent)] text-[var(--accent-foreground)] shadow-lg transition-opacity hover:opacity-90 md:hidden safe-bottom"
      aria-label="Open chat"
    >
      <Icon
        d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
        className="h-7 w-7"
      />
    </Link>
  );
}
