import type { FeedItem } from "@/components/types";
import {
  FETCH_DELAY_MS,
  MAX_PAGES,
  PAGE_SIZE,
  tweetKeys,
} from "@/components/constants";

const NAMES = [
  "Twixter",
  "Design Daily",
  "Dev Tips",
  "React",
  "Next.js",
  "Vercel",
  "Tailwind",
  "TypeScript",
];
const HANDLES = [
  "@twixter",
  "@designdaily",
  "@devtips",
  "@reactjs",
  "@nextjs",
  "@vercel",
  "@tailwindcss",
  "@typescript",
];

/** Server-side: get a single post by id for SSR post detail page. */
export function getPostById(
  id: string,
  tTweets: (key: string) => string
): FeedItem | null {
  const texts: string[] = [
    tTweets("welcome"),
    tTweets("designDaily"),
    tTweets("devTips"),
    tTweets("react"),
    "Next.js App Router and Server Components are the default. Faster and simpler.",
    "Deploy in seconds. Edge and serverless. Try Vercel for your next project.",
    "Utility-first CSS. Build modern UIs without leaving your HTML.",
    "TypeScript extends JavaScript with types. Catch errors before runtime.",
  ];
  // initial-1 .. initial-4
  const initialMatch = id.match(/^initial-(\d+)$/);
  if (initialMatch) {
    const i = parseInt(initialMatch[1], 10) - 1;
    if (i < 0 || i >= tweetKeys.length) return null;
    return {
      id,
      name: NAMES[i],
      handle: HANDLES[i],
      time: ["2h", "4h", "6h", "8h"][i],
      text: tTweets(tweetKeys[i]),
      videoUrl: i === 1 ? "/placeholder" : undefined,
    };
  }
  // page-cursor-index
  const pageMatch = id.match(/^page-(\d+)-(\d+)$/);
  if (pageMatch) {
    const cursor = parseInt(pageMatch[1], 10);
    const index = parseInt(pageMatch[2], 10);
    const base = cursor * PAGE_SIZE;
    const idx = (base + index) % NAMES.length;
    return {
      id,
      name: NAMES[idx],
      handle: HANDLES[idx],
      time: `${base + index + 1}h`,
      text: texts[idx],
      videoUrl: (base + index) % 3 === 0 ? "/placeholder" : undefined,
    };
  }
  return null;
}

export function buildInitialFeed(
  tTweets: (key: string) => string
): FeedItem[] {
  return tweetKeys.map((key, i) => ({
    id: `initial-${i + 1}`,
    name: ["Twixter", "Design Daily", "Dev Tips", "React"][i],
    handle: ["@twixter", "@designdaily", "@devtips", "@reactjs"][i],
    time: ["2h", "4h", "6h", "8h"][i],
    text: tTweets(key),
    videoUrl: i === 1 ? "/placeholder" : undefined,
  }));
}

export async function fetchFeedPage(
  cursor: number,
  tTweets: (key: string) => string
): Promise<{ items: FeedItem[]; nextCursor: number; hasMore: boolean }> {
  await new Promise((r) => setTimeout(r, FETCH_DELAY_MS));
  const base = cursor * PAGE_SIZE;
  const names = [
    "Twixter",
    "Design Daily",
    "Dev Tips",
    "React",
    "Next.js",
    "Vercel",
    "Tailwind",
    "TypeScript",
  ];
  const handles = [
    "@twixter",
    "@designdaily",
    "@devtips",
    "@reactjs",
    "@nextjs",
    "@vercel",
    "@tailwindcss",
    "@typescript",
  ];
  const texts = [
    tTweets("welcome"),
    tTweets("designDaily"),
    tTweets("devTips"),
    tTweets("react"),
    "Next.js App Router and Server Components are the default. Faster and simpler.",
    "Deploy in seconds. Edge and serverless. Try Vercel for your next project.",
    "Utility-first CSS. Build modern UIs without leaving your HTML.",
    "TypeScript extends JavaScript with types. Catch errors before runtime.",
  ];
  const items: FeedItem[] = [];
  for (let i = 0; i < PAGE_SIZE; i++) {
    const idx = (base + i) % names.length;
    const id = `page-${cursor}-${i}`;
    items.push({
      id,
      name: names[idx],
      handle: handles[idx],
      time: `${base + i + 1}h`,
      text: texts[idx],
      videoUrl: (base + i) % 3 === 0 ? "/placeholder" : undefined,
    });
  }
  const nextCursor = cursor + 1;
  const hasMore = nextCursor < MAX_PAGES;
  return { items, nextCursor, hasMore };
}
