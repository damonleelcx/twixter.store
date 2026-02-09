import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = { children: React.ReactNode; params: Promise<{ locale: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { profileTitle?: string; profileDescription?: string };
  const title = meta?.profileTitle ?? `Profile | ${SITE_NAME}`;
  const description = meta?.profileDescription ?? "Manage your Twixter profile and referral link.";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/profile`;
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

export default async function ProfileLayout({ children, params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <>{children}</>;
}
