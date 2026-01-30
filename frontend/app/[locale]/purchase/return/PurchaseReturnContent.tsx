"use client";

import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import { verifyCheckout } from "@/lib/api";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

type VerifyState =
  | { status: "idle" }
  | { status: "verifying" }
  | { status: "success"; payment_status: string }
  | { status: "error"; message: string };

export function PurchaseReturnContent() {
  const params = useParams();
  const searchParams = useSearchParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const tNav = useTranslations("nav");
  const tReturn = useTranslations("purchaseReturn");
  const sessionId = searchParams.get("session_id");
  const [verifyState, setVerifyState] = useState<VerifyState>({ status: "idle" });

  useEffect(() => {
    if (!sessionId) {
      const t = setTimeout(
        () => setVerifyState({ status: "success", payment_status: "paid" }),
        0
      );
      return () => clearTimeout(t);
    }
    let cancelled = false;
    verifyCheckout(sessionId)
      .then((data) => {
        if (!cancelled) {
          setVerifyState({
            status: "success",
            payment_status: data.payment_status ?? "",
          });
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setVerifyState({
            status: "error",
            message: err instanceof Error ? err.message : t("errorGeneric"),
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [sessionId, t]);

  const isPaid =
    verifyState.status === "success" &&
    (verifyState.payment_status === "paid" || !sessionId);

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px] p-6">
          <div className="rounded-xl border border-[var(--border)] p-6 text-center">
            {(verifyState.status === "verifying" ||
              (verifyState.status === "idle" && sessionId)) && (
              <>
                <h1 className="text-xl font-bold text-[var(--foreground)]">
                  {tReturn("verifyingTitle")}
                </h1>
                <p className="mt-2 text-sm text-[var(--muted)]">
                  {tReturn("verifyingHint")}
                </p>
              </>
            )}
            {verifyState.status === "error" && (
              <>
                <h1 className="text-xl font-bold text-red-600 dark:text-red-400">
                  {tReturn("verificationFailed")}
                </h1>
                <p className="mt-2 text-sm text-[var(--muted)]">
                  {verifyState.message}
                </p>
                <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-center">
                  <Link
                    href={`/${locale}/credits`}
                    className="rounded-full bg-[var(--accent)] px-6 py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
                  >
                    {t("buyCredits")}
                  </Link>
                  <Link
                    href={`/${locale}`}
                    className="rounded-full border border-[var(--border)] px-6 py-3 font-bold transition-colors hover:bg-[var(--hover)]"
                  >
                    {tReturn("home")}
                  </Link>
                </div>
              </>
            )}
            {(verifyState.status === "idle" && !sessionId) ||
            (verifyState.status === "success" && isPaid) ? (
              <>
                <h1 className="text-xl font-bold text-green-600 dark:text-green-400">
                  {tReturn("paymentComplete")}
                </h1>
                <p className="mt-2 text-sm text-[var(--muted)]">
                  {tReturn("creditsAdded")}
                </p>
                <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-center">
                  <Link
                    href={`/${locale}/profile`}
                    className="rounded-full bg-[var(--accent)] px-6 py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
                  >
                    {tNav("profile")}
                  </Link>
                  <Link
                    href={`/${locale}`}
                    className="rounded-full border border-[var(--border)] px-6 py-3 font-bold transition-colors hover:bg-[var(--hover)]"
                  >
                    {tReturn("home")}
                  </Link>
                </div>
              </>
            ) : verifyState.status === "success" && !isPaid ? (
              <>
                <h1 className="text-xl font-bold text-amber-600 dark:text-amber-400">
                  {tReturn("paymentNotComplete")}
                </h1>
                <p className="mt-2 text-sm text-[var(--muted)]">
                  {tReturn("statusHint", { status: verifyState.payment_status })}
                </p>
                <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-center">
                  <Link
                    href={`/${locale}/credits`}
                    className="rounded-full bg-[var(--accent)] px-6 py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90"
                  >
                    {t("buyCredits")}
                  </Link>
                  <Link
                    href={`/${locale}`}
                    className="rounded-full border border-[var(--border)] px-6 py-3 font-bold transition-colors hover:bg-[var(--hover)]"
                  >
                    {tReturn("home")}
                  </Link>
                </div>
              </>
            ) : null}
          </div>
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
