import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { Suspense } from "react";
import { PurchaseReturnContent } from "./PurchaseReturnContent";

const SITE_NAME = "Twixter";

type Props = { params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { purchaseReturnTitle?: string; purchaseReturnDescription?: string };
  const title = meta?.purchaseReturnTitle ?? `Purchase Complete | ${SITE_NAME}`;
  const description = meta?.purchaseReturnDescription ?? "Your purchase was successful. Thank you for using Twixter.";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/purchase/return`;
  return {
    title,
    description,
    openGraph: {
      type: "website",
      locale: locale === "zh" ? "zh_CN" : "en_US",
      url: canonicalUrl,
      siteName: SITE_NAME,
      title,
      description,
    },
    alternates: { canonical: canonicalUrl },
    robots: { index: false, follow: true },
  };
}

export default function PurchaseReturnPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center text-[var(--muted)]">Loading…</div>}>
      <PurchaseReturnContent />
    </Suspense>
  );
}
