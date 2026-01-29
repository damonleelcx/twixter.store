import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { PostContent } from "@/components/PostContent";
import { RightSidebar } from "@/components/RightSidebar";
import { SaveViewingFromUrl } from "@/components/SaveViewingFromUrl";
import { AUTH_TOKEN_COOKIE, getContentByIdServer } from "@/lib/api";
import type { Metadata } from "next";
import { setRequestLocale } from "next-intl/server";
import { cookies } from "next/headers";
import { notFound } from "next/navigation";

type Props = {
  params: Promise<{ locale: string; id: string }>;
  searchParams?: Promise<{ [key: string]: string | string[] | undefined }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale, id } = await params;
  const contentId = parseInt(id, 10);
  if (Number.isNaN(contentId)) return { title: "Twixter" };
  return {
    title: `Post ${id} | Twixter`,
    description: "View content",
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
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
