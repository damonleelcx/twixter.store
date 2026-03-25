import { SlideshowClient } from "./slideshow-client";
import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = {
  params: Promise<{ locale: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { slideshowTitle?: string; slideshowDescription?: string };
  const title = meta.slideshowTitle ?? `Slideshow | ${SITE_NAME}`;
  const description =
    meta.slideshowDescription ??
    "Watch a free 60-second preview from the middle of each video in a TikTok-style slideshow.";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/slideshow`;

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
    twitter: {
      card: "summary_large_image",
      title,
      description,
      site: SITE_NAME,
    },
    alternates: { canonical: canonicalUrl },
    robots: { index: true, follow: true },
  };
}

export default async function SlideshowPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <SlideshowClient />;
}

