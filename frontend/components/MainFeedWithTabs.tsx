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
};

export function MainFeedWithTabs({ initialForYouFeed }: MainFeedWithTabsProps) {
  const [activeTab, setActiveTab] = useState<FeedTab>("forYou");
  const defaultTabApplied = useRef(false);

  useEffect(() => {
    if (defaultTabApplied.current) return;
    let cancelled = false;
    fetchCurrentUser().then((user) => {
      if (cancelled || defaultTabApplied.current) return;
      if (user?.permissions?.includes("can_view_nsfw")) {
        defaultTabApplied.current = true;
        setActiveTab("premium");
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <>
      <MainFeedHeader activeTab={activeTab} onTabChange={setActiveTab} />
      <MobileWalletBar />
      <ComposeBox />
      <FeedList activeTab={activeTab} initialForYouFeed={initialForYouFeed} />
    </>
  );
}
