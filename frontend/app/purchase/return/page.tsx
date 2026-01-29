"use client";

import { routing } from "@/i18n/routing";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";

/**
 * Stripe redirects to /purchase/return?session_id=... (no locale).
 * Redirect to default-locale return page so the app layout and i18n work.
 */
export default function PurchaseReturnRedirectPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session_id");
  const locale = routing.defaultLocale;

  useEffect(() => {
    const query = sessionId
      ? `?session_id=${encodeURIComponent(sessionId)}`
      : "";
    router.replace(`/${locale}/purchase/return${query}`);
  }, [locale, sessionId, router]);

  return (
    <html>
      <body>
        <div className="flex min-h-screen items-center justify-center bg-[var(--background)] text-[var(--foreground)]">
          <p className="text-[var(--muted)]">Redirecting...</p>
        </div>
      </body>
    </html>
  );
}
