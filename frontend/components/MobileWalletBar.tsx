"use client";

import { fetchCurrentUser } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

/** Wallet balance + Buy credits bar; visible only when RightSidebar is hidden (mobile/tablet). */
export function MobileWalletBar() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const tSlide = useTranslations("slideshow");
  const [walletBalance, setWalletBalance] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled) {
        if (u != null) {
          setWalletBalance(typeof u.wallet_balance === "number" ? u.wallet_balance : 0);
        } else {
          setWalletBalance(null);
        }
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  if (walletBalance === null) return null;

  return (
    <div className="xl:hidden border-b border-[var(--border)] bg-[var(--hover)]/50 px-4 py-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-xs text-[var(--muted)]">{t("walletBalance")}</p>
          <p className="font-bold">{walletBalance} {t("credits")}</p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Link
            href={`/${locale}/slideshow`}
            className="rounded-full border border-[var(--border)] bg-[var(--background)] px-4 py-2 text-sm font-semibold text-[var(--foreground)] hover:bg-[var(--hover)]"
          >
            {tSlide("freeSlideshow")}
          </Link>
          <Link
            href={`/${locale}/credits`}
            className="rounded-full bg-[var(--accent)] px-4 py-2 text-sm font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
          >
            {t("buyCredits")}
          </Link>
        </div>
      </div>
    </div>
  );
}
