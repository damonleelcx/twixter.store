import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = { children: React.ReactNode; params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { adminRevenueAnalyticsTitle?: string; adminRevenueAnalyticsDescription?: string };
  const title = meta?.adminRevenueAnalyticsTitle ?? `Revenue Analytics | ${SITE_NAME}`;
  const description = meta?.adminRevenueAnalyticsDescription ?? "View content and credits revenue.";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/admin/analytics/revenue`;
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

export default async function AdminRevenueAnalyticsLayout({ children, params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <>{children}</>;
}
