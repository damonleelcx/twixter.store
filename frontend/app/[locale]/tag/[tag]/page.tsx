import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebarWrapper } from "@/components/RightSidebarWrapper";
import { TagPageContent } from "@/components/TagPageContent";
import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = {
  params: Promise<{ locale: string; tag: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, tag: tagEncoded } = await params;
  const tag = typeof tagEncoded === "string" ? decodeURIComponent(tagEncoded) : "";
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { tagTitle: string; tagDescription: string };
  const title = tag ? (meta.tagTitle ?? `Posts tagged ${tag} | ${SITE_NAME}`).replace(/\{tag\}/g, tag) : SITE_NAME;
  const description = tag
    ? (meta.tagDescription ?? `Browse content with tag ${tag} on ${SITE_NAME}`).replace(/\{tag\}/g, tag)
    : "Twixter";
  const canonicalUrl = tagEncoded ? `${getBaseUrlForMetadata()}/${locale}/tag/${tagEncoded}` : undefined;

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
    ...(canonicalUrl && { alternates: { canonical: canonicalUrl } }),
    robots: { index: true, follow: true },
  };
}

export default async function TagPage({ params }: Props) {
  const { locale, tag: tagEncoded } = await params;
  setRequestLocale(locale);
  const tag = typeof tagEncoded === "string" ? decodeURIComponent(tagEncoded) : "";

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <TagPageContent tag={tag} />
        </main>
        <RightSidebarWrapper />
      </div>
      <MobileBottomNav />
    </div>
  );
}
