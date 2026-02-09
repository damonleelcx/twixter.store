import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = { children: React.ReactNode; params: Promise<{ locale: string; id: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { editPostTitle?: string; editPostDescription?: string };
  const baseTitle = meta?.editPostTitle ?? `Edit Post | ${SITE_NAME}`;
  const title = id ? `${baseTitle.replace(` | ${SITE_NAME}`, "")} ${id} | ${SITE_NAME}` : baseTitle;
  const description = meta?.editPostDescription ?? "Edit your content on Twixter.";
  const canonicalUrl = id ? `${getBaseUrlForMetadata()}/${locale}/post/${id}/edit` : undefined;
  return {
    title,
    description,
    openGraph: canonicalUrl
      ? {
          type: "website",
          locale: locale === "zh" ? "zh_CN" : "en_US",
          url: canonicalUrl,
          siteName: SITE_NAME,
          title,
          description,
        }
      : undefined,
    alternates: canonicalUrl ? { canonical: canonicalUrl } : undefined,
    robots: { index: false, follow: true },
  };
}

export default async function EditPostLayout({ children, params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return <>{children}</>;
}
