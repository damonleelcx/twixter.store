import { LeftSidebar } from "@/components/LeftSidebar";
import { ChatContent } from "@/components/ChatContent";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebarWrapper } from "@/components/RightSidebarWrapper";
import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { getTranslations, setRequestLocale } from "next-intl/server";

const SITE_NAME = "Twixter";

type Props = {
  params: Promise<{ locale: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { chatTitle?: string; chatDescription?: string };
  const title = meta?.chatTitle ?? "Chat | Twixter";
  const description = meta?.chatDescription ?? "Agent chat.";
  const canonicalUrl = `${getBaseUrlForMetadata()}/${locale}/chat`;
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
    robots: { index: false, follow: true },
  };
}

export default async function ChatPage({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("nav");

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="flex min-h-[calc(100vh-4rem)] min-w-0 flex-1 flex-col border-x border-[var(--border)] md:max-w-[600px]">
          <header className="sticky top-0 z-10 border-b border-[var(--border)] bg-[var(--background)] px-4 py-3">
            <h1 className="text-xl font-semibold">{t("chat")}</h1>
          </header>
          <ChatContent />
        </main>
        <RightSidebarWrapper />
      </div>
      <MobileBottomNav />
    </div>
  );
}
