import { LeftSidebar } from "@/components/LeftSidebar";
import { MainFeedWithTabs } from "@/components/MainFeedWithTabs";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebarWrapper } from "@/components/RightSidebarWrapper";
import { SaveViewingFromUrl } from "@/components/SaveViewingFromUrl";
import { AUTH_TOKEN_COOKIE, fetchContentFeedServer } from "@/lib/api";
import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";
import { cookies } from "next/headers";
import Link from "next/link";

const SITE_NAME = "Twixter";

type Props = {
  params: Promise<{ locale: string }>;
  searchParams?: Promise<{ [key: string]: string | string[] | undefined }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { title: string; description: string };
  const title = meta?.title ?? `${SITE_NAME} - Home`;
  const description = meta?.description ?? "A fresh take on social";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}`;
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

export default async function Home({ params, searchParams }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);

  const sp = await searchParams;
  const hasViewingParam =
    typeof sp?.viewing === "string" && sp.viewing.trim() !== "";

  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_TOKEN_COOKIE)?.value;
  const initialForYouFeed = await fetchContentFeedServer("light", 0, 20, token).catch(
    () => ({ items: [], next_cursor: 0, has_more: false })
  );

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <SaveViewingFromUrl />
      {!token && (
        <div className="border-b border-[var(--border)] bg-[var(--accent)] px-4 py-3 text-center text-sm text-[var(--accent-foreground)]">
          <span className="font-medium">Limited Time Offer: Sign up now to earn free credits</span>
          {" · "}
          <Link
            href={`/${locale}/auth/signup?promo=freecredits`}
            className="font-bold underline hover:no-underline"
          >
            Sign up now
          </Link>
        </div>
      )}
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <MainFeedWithTabs
            initialForYouFeed={initialForYouFeed}
            initialTab={hasViewingParam ? "premium" : undefined}
          />
        </main>
        <RightSidebarWrapper />
      </div>
      <MobileBottomNav />
    </div>
  );
}
