import { LeftSidebar } from "@/components/LeftSidebar";
import { BookmarksList } from "@/components/BookmarksList";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import { getTranslations, setRequestLocale } from "next-intl/server";

type Props = {
  params: Promise<{ locale: string }>;
};

export default async function BookmarksPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("nav");

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[var(--background)] px-4 py-3">
            <h1 className="text-xl font-semibold">{t("bookmarks")}</h1>
          </header>
          <BookmarksList />
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
