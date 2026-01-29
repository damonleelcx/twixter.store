"use client";

import type { ContentFeedResponse } from "@/lib/api";
import { useState } from "react";
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

  return (
    <>
      <MainFeedHeader activeTab={activeTab} onTabChange={setActiveTab} />
      <MobileWalletBar />
      <ComposeBox />
      <FeedList activeTab={activeTab} initialForYouFeed={initialForYouFeed} />
    </>
  );
}
