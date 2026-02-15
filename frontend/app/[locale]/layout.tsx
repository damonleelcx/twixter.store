import { ChatFab } from "@/components/ChatFab";
import { routing } from "@/i18n/routing";
import { getBaseUrlForMetadata } from "@/lib/metadata";
import type { Metadata } from "next";
import { NextIntlClientProvider } from "next-intl";
import { getMessages, setRequestLocale } from "next-intl/server";
import { Geist, Geist_Mono } from "next/font/google";
import Script from "next/script";
import { Suspense } from "react";
import "../globals.css";
import { SetLocaleLang } from "./SetLocaleLang";

const GA_MEASUREMENT_ID = "G-KS7MBWYBZ8";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

type Props = {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params;
  const messages = (await import(`@/messages/${locale}.json`)).default;
  const meta = messages.meta as { title: string; description: string };
  return {
    metadataBase: new URL(getBaseUrlForMetadata()),
    title: meta.title,
    description: meta.description,
  };
}

export default async function LocaleLayout({ children, params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const messages = await getMessages();

  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){var t=localStorage.getItem("twixter-theme");if(t==="dark")document.documentElement.classList.add("dark");else if(t==="light")document.documentElement.classList.remove("dark");})();`,
          }}
        />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <Script
          src={`https://www.googletagmanager.com/gtag/js?id=${GA_MEASUREMENT_ID}`}
          strategy="afterInteractive"
        />
        <Script id="google-analytics" strategy="afterInteractive">
          {`
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', '${GA_MEASUREMENT_ID}');
          `}
        </Script>
        <Script id="microsoft-clarity" strategy="afterInteractive">
          {`
            (function(c,l,a,r,i,t,y){
              c[a]=c[a]||function(){(c[a].q=c[a].q||[]).push(arguments)};
              t=l.createElement(r);t.async=1;t.src="https://www.clarity.ms/tag/"+i;
              y=l.getElementsByTagName(r)[0];y.parentNode.insertBefore(t,y);
            })(window, document, "clarity", "script", "vadzxgbtzn");
          `}
        </Script>
        {/* AdSense: 使用普通 script 避免 Next.js Script 注入的 data-nscript，否则会报错 */}
        <script
          src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-9330067151515952"
          async
          crossOrigin="anonymous"
        />
        <NextIntlClientProvider messages={messages} locale={locale}>
          <SetLocaleLang locale={locale} />
          {children}
          <Suspense fallback={null}>
            <ChatFab />
          </Suspense>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
