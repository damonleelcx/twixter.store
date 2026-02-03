import { ExploreContent } from "@/components/ExploreContent";
import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebarWrapper } from "@/components/RightSidebarWrapper";
import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";
import { Suspense } from "react";

const SITE_NAME = "Twixter";

type Props = {
  params: Promise<{ locale: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { exploreTitle: string; exploreDescription: string };
  const title = meta.exploreTitle ?? `Explore | ${SITE_NAME}`;
  const description = meta.exploreDescription ?? "Discover and search content on Twixter";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/explore`;

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

export default async function ExplorePage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <Suspense fallback={<div className="p-4 text-[var(--muted)]">Loading…</div>}>
            <ExploreContent />
          </Suspense>
        </main>
        <RightSidebarWrapper />
      </div>
      <MobileBottomNav />
    </div>
  );
}
