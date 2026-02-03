import { getBaseUrlForMetadata } from "@/lib/metadata";
import { TermsContent } from "./TermsContent";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = {
  params: Promise<{ locale: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.terms as { title: string; description: string } | undefined;
  const title = meta?.title ?? `Terms of Use | ${SITE_NAME}`;
  const description = meta?.description ?? "Terms of Use for Twixter (twixter.store)";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/terms`;

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
    robots: { index: true, follow: true },
  };
}

export default async function TermsPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);

  return <TermsContent />;
}
