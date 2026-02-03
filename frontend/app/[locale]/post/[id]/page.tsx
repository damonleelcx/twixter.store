import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { PostContent } from "@/components/PostContent";
import { RightSidebarWrapper } from "@/components/RightSidebarWrapper";
import { SaveViewingFromUrl } from "@/components/SaveViewingFromUrl";
import { AUTH_TOKEN_COOKIE, getContentByIdServer } from "@/lib/api";
import { getBaseUrlForMetadata, isSecureUrlForMeta } from "@/lib/metadata";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";

const SITE_NAME = "Twixter";
const DEFAULT_DESCRIPTION = "View content on Twixter";

type Props = {
  params: Promise<{ locale: string; id: string }>;
  searchParams?: Promise<{ [key: string]: string | string[] | undefined }>;
};

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const contentId = parseInt(id, 10);
  if (Number.isNaN(contentId) || contentId <= 0) {
    return { title: SITE_NAME };
  }
  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_TOKEN_COOKIE)?.value;
  const sp = await searchParams;
  const viewingToken = typeof sp?.viewing === "string" ? sp.viewing : undefined;
  const data = await getContentByIdServer(contentId, token, viewingToken);
  if (!data?.content) {
    return {
      title: `${id} | ${SITE_NAME}`,
      description: DEFAULT_DESCRIPTION,
    };
  }
  const content = data.content;
  const title = content.name ? `${content.name} | ${SITE_NAME}` : `Post ${id} | ${SITE_NAME}`;
  const description =
    (typeof content.description === "string" && content.description.trim()) ||
    (content.author_username ? `Content by @${content.author_username} on ${SITE_NAME}` : DEFAULT_DESCRIPTION);
  const descTruncated = description.length > 160 ? description.slice(0, 157) + "..." : description;
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/post/${id}`;
  const rawOgImage =
    data.files?.[0]?.gif_file_url ||
    (data.files?.[0] as { original_file_url?: string } | undefined)?.original_file_url;
  const ogImage = isSecureUrlForMeta(rawOgImage) ? rawOgImage : undefined;
  const keywords =
    Array.isArray(content.tags) && content.tags.length > 0
      ? content.tags.join(", ")
      : undefined;

  const openGraph: Metadata["openGraph"] = {
    type: "website",
    locale: locale === "zh" ? "zh_CN" : "en_US",
    url: canonicalUrl,
    siteName: SITE_NAME,
    title,
    description: descTruncated,
  };
  if (ogImage) {
    openGraph.images = [
      { url: ogImage, width: 1200, height: 630, alt: content.name || "Post preview" },
    ];
  }

  const twitter: NonNullable<Metadata["twitter"]> = {
    card: ogImage ? "summary_large_image" : "summary",
    title,
    description: descTruncated,
    creator: content.author_username ? `@${content.author_username}` : undefined,
    site: SITE_NAME,
  };
  if (ogImage) {
    twitter.images = [ogImage];
  }

  const other: Record<string, string> = {};
  if (content.created_at) other["article:published_time"] = content.created_at;
  if (content.author_username) other["article:author"] = content.author_username;

  return {
    title,
    description: descTruncated,
    keywords: keywords ? [keywords, SITE_NAME] : undefined,
    authors: content.author_username ? [{ name: content.author_username }] : undefined,
    creator: content.author_username ?? SITE_NAME,
    publisher: SITE_NAME,
    robots: {
      index: true,
      follow: true,
      googleBot: { index: true, follow: true },
    },
    alternates: {
      canonical: canonicalUrl,
    },
    openGraph,
    twitter,
    other: Object.keys(other).length > 0 ? other : undefined,
  };
}

export default async function PostPage({ params, searchParams }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);

  const contentId = parseInt(id, 10);
  if (Number.isNaN(contentId) || contentId <= 0) notFound();

  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_TOKEN_COOKIE)?.value;
  const sp = await searchParams;
  const viewingToken = typeof sp?.viewing === "string" ? sp.viewing : undefined;
  const initialData = await getContentByIdServer(contentId, token, viewingToken);

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <SaveViewingFromUrl />
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <PostContent contentId={contentId} initialData={initialData} />
        </main>
        <RightSidebarWrapper />
      </div>
      <MobileBottomNav />
    </div>
  );
}
