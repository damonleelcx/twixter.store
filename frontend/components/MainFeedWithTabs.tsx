"use client";

import type { ContentFeedResponse } from "@/lib/api";
import { fetchCurrentUser } from "@/lib/api";
import { useEffect, useRef, useState } from "react";
import { ComposeBox } from "./ComposeBox";
import { FeedList } from "./FeedList";
import type { FeedTab } from "./MainFeedHeader";
import { MainFeedHeader } from "./MainFeedHeader";
import { MobileWalletBar } from "./MobileWalletBar";

type MainFeedWithTabsProps = {
  initialForYouFeed?: ContentFeedResponse;
  /** When user lands with ?viewing= (e.g. encrypted viewing_dark share link), open premium tab and allow can_view_nsfw via cookie; sign up required for purchase/membership. */
  initialTab?: FeedTab;
};

export function MainFeedWithTabs({ initialForYouFeed, initialTab }: MainFeedWithTabsProps) {
  const [activeTab, setActiveTab] = useState<FeedTab>(initialTab ?? "forYou");
  const [canViewNsfw, setCanViewNsfw] = useState(false);
  const defaultTabApplied = useRef(false);

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((user) => {
      if (cancelled) return;
      const hasNsfw = user?.permissions?.includes("can_view_nsfw") ?? false;
      setCanViewNsfw(hasNsfw);
      if (defaultTabApplied.current) return;
      if (initialTab === "premium") {
        defaultTabApplied.current = true;
        return;
      }
      if (hasNsfw) {
        defaultTabApplied.current = true;
        setActiveTab("premium");
      }
    });
    return () => {
      cancelled = true;
    };
  }, [initialTab]);

  return (
    <>
      <MainFeedHeader
        activeTab={activeTab}
        onTabChange={setActiveTab}
        canViewNsfw={canViewNsfw}
      />
      <MobileWalletBar />
      <ComposeBox />
      <FeedList activeTab={activeTab} initialForYouFeed={initialForYouFeed} />
    </>
  );
}
