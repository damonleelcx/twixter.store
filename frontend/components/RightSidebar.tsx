"use client";

import { fetchCurrentUser } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { trendCounts, trendTags } from "./constants";

export function RightSidebar() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const tSidebar = useTranslations("sidebar");
  const tAuth = useTranslations("auth");
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

  return (
    <aside className="sticky top-0 hidden h-screen w-[350px] shrink-0 overflow-y-auto p-4 xl:block">
      {walletBalance !== null && (
        <div className="rounded-2xl bg-[var(--hover)] p-4 mb-4">
          <h2 className="text-sm font-medium text-[var(--muted)]">{tAuth("walletBalance")}</h2>
          <p className="text-xl font-bold mt-0.5">{walletBalance} {tAuth("credits")}</p>
          <Link
            href={`/${locale}/credits`}
            className="mt-2 block w-full rounded-full bg-[var(--accent)] py-2 text-center text-sm font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
          >
            {tAuth("buyCredits")}
          </Link>
        </div>
      )}
      <div className="rounded-2xl bg-[var(--hover)] p-4">
        <h2 className="text-xl font-bold">{tSidebar("premiumTitle")}</h2>
        <p className="mt-1 text-sm text-[var(--muted)]">
          {tSidebar("premiumDescription")}
        </p>
        <Link
          href={`/${locale}/subscribe`}
          className="mt-3 block w-full rounded-full bg-[var(--accent)] py-2 text-center font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
        >
          {tSidebar("subscribe")}
        </Link>
      </div>
      <div className="mt-4 rounded-2xl bg-[var(--hover)] p-4">
        <h2 className="text-xl font-bold">{tSidebar("whatsHappening")}</h2>
        <div className="mt-3 space-y-3">
          {trendTags.map((tag, i) => (
            <a
              key={tag}
              href="#"
              className="block rounded-lg p-2 hover:bg-[var(--border)]/50"
            >
              <span className="text-sm text-[var(--muted)]">
                {tSidebar("trending")}
              </span>
              <p className="font-bold">{tag}</p>
              <span className="text-sm text-[var(--muted)]">
                {tSidebar("posts", { count: trendCounts[i] })}
              </span>
            </a>
          ))}
        </div>
      </div>
    </aside>
  );
}
