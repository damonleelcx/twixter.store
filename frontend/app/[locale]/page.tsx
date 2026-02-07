import { LeftSidebar } from "@/components/LeftSidebar";
import { MainFeedWithTabs } from "@/components/MainFeedWithTabs";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebarWrapper } from "@/components/RightSidebarWrapper";
import { SaveViewingFromUrl } from "@/components/SaveViewingFromUrl";
import { AUTH_TOKEN_COOKIE, fetchContentFeedServer } from "@/lib/api";
import { setRequestLocale } from "next-intl/server";
import { cookies } from "next/headers";

type Props = {
  params: Promise<{ locale: string }>;
  searchParams?: Promise<{ [key: string]: string | string[] | undefined }>;
};

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
