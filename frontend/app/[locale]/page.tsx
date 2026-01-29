import { ComposeBox } from "@/components/ComposeBox";
import { FeedList } from "@/components/FeedList";
import { LeftSidebar } from "@/components/LeftSidebar";
import { MainFeedHeader } from "@/components/MainFeedHeader";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { MobileWalletBar } from "@/components/MobileWalletBar";
import { RightSidebar } from "@/components/RightSidebar";
import { buildInitialFeed } from "@/lib/feed";
import { getTranslations, setRequestLocale } from "next-intl/server";

type Props = {
  params: Promise<{ locale: string }>;
};

export default async function Home({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);

  const tTweets = await getTranslations({ locale, namespace: "tweets" });
  const initialFeed = buildInitialFeed((key) => tTweets(key));

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <MainFeedHeader />
          <MobileWalletBar />
          <ComposeBox />
          <FeedList initialFeed={initialFeed} />
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
