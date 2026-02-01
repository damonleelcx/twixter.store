"use client";

import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams } from "next/navigation";

const SITE = "Twixter";
const DOMAIN = "twixter.store";
const EFFECTIVE_DATE = "January 30, 2025";

export function PrivacyContent() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("privacy");

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <article className="prose prose-neutral dark:prose-invert mx-auto max-w-none px-4 py-6">
            <header className="mb-8">
              <h1 className="text-2xl font-bold text-[var(--foreground)] md:text-3xl">
                {t("title")}
              </h1>
              <p className="mt-2 text-sm text-[var(--muted)]">
                {t("effective", { date: EFFECTIVE_DATE })}
              </p>
            </header>

            <p className="text-[var(--foreground)]">
              {SITE} is a platform that helps connect fans with creators they love. &quot;We,&quot; &quot;our,&quot; and &quot;us&quot; refer to the operator of {DOMAIN}. This Privacy Policy is part of our Terms of Use and describes how we collect, use, and share information when you use {SITE}.
            </p>

            <nav className="my-8 rounded-xl border border-[var(--border)] bg-[var(--hover)] p-4">
              <h2 className="mb-3 text-sm font-semibold text-[var(--muted)]">{t("topics")}</h2>
              <ul className="space-y-1 text-sm">
                <li><a href="#welcome" className="text-[var(--accent)] hover:underline">{t("welcome")}</a></li>
                <li><a href="#you-provide" className="text-[var(--accent)] hover:underline">{t("youProvide")}</a></li>
                <li><a href="#automatic" className="text-[var(--accent)] hover:underline">{t("automatic")}</a></li>
                <li><a href="#how-we-use" className="text-[var(--accent)] hover:underline">{t("howWeUse")}</a></li>
                <li><a href="#shared-creators" className="text-[var(--accent)] hover:underline">{t("sharedCreators")}</a></li>
                <li><a href="#shared-public" className="text-[var(--accent)] hover:underline">{t("sharedPublic")}</a></li>
                <li><a href="#shared-third" className="text-[var(--accent)] hover:underline">{t("sharedThird")}</a></li>
                <li><a href="#your-control" className="text-[var(--accent)] hover:underline">{t("yourControl")}</a></li>
                <li><a href="#cookies" className="text-[var(--accent)] hover:underline">{t("cookies")}</a></li>
                <li><a href="#children" className="text-[var(--accent)] hover:underline">{t("children")}</a></li>
                <li><a href="#changes" className="text-[var(--accent)] hover:underline">{t("changes")}</a></li>
                <li><a href="#contact" className="text-[var(--accent)] hover:underline">{t("contact")}</a></li>
              </ul>
            </nav>

            <section id="welcome" className="scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("welcome")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                Welcome to {SITE}. We take your privacy seriously. This policy explains what data we collect, why we collect it, and how you can control it. By using {SITE}, you agree to this Privacy Policy and our Terms of Use.
              </p>
            </section>

            <section id="you-provide" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("youProvide")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                We collect information you provide when you create an account, such as your email address, username (if you set one), and password. Depending on whether you are a creator or a buyer, we may also collect:
              </p>
              <ul className="mt-2 list-disc pl-6 space-y-1 text-[var(--foreground)]">
                <li>Name and contact information</li>
                <li>Payment information (processed by our payment partners; we do not store full card numbers)</li>
                <li>For creators: tax identification and payout information (e.g. for PayPal or other payout methods)</li>
                <li>Profile information you choose to add (e.g. bio, location)</li>
              </ul>
              <p className="mt-3 text-[var(--foreground)]">
                If you sign up or pay through a third party (e.g. social login or PayPal), that provider may share with us information such as your name, email, or profile picture in accordance with their policies. You can revoke that access in the third party&apos;s settings.
              </p>
              <p className="mt-3 text-[var(--foreground)]">
                <strong>Patrons.</strong> As a buyer or subscriber, we collect information about the content and subscriptions you purchase, the creators you support, and benefits you receive. Payment details are handled by our payment partners (e.g. Stripe, PayPal); we may store limited payment-related data such as the last digits of a card or billing region where needed for support or fraud prevention.
              </p>
              <p className="mt-3 text-[var(--foreground)]">
                <strong>Creators.</strong> As a creator, we collect information needed to run your page and pay you, including identity and tax information where required by law, payout account details, and content you upload (titles, descriptions, tags, and files).
              </p>
            </section>

            <section id="automatic" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("automatic")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                When you use {SITE}, we automatically collect information such as your IP address, approximate location (e.g. from IP), browser and device type, operating system, language settings, referring page, pages you visit, links you click, and how long you spend on the site. We may use cookies and similar technologies; see the Cookies section below.
              </p>
            </section>

            <section id="how-we-use" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("howWeUse")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                We use the information we collect to: provide and operate {SITE}; process payments and payouts; comply with legal and tax obligations; verify identity; send you service-related messages and, where you have agreed, marketing; provide customer support; personalize your experience (e.g. recommendations); improve our services and conduct analytics; prevent fraud and abuse; and enforce our Terms of Use and policies.
              </p>
            </section>

            <section id="shared-creators" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("sharedCreators")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                When you purchase content or subscribe to a creator, we share with that creator information needed to fulfill the relationship, such as your name (or username), email address, what you purchased or subscribed to, and payment tier. We do not share your full payment card details with creators. Creators may use third-party services to deliver benefits; those services have their own privacy policies.
              </p>
            </section>

            <section id="shared-public" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("sharedPublic")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                Some information may be public or visible to other users: for example, your public profile (username, avatar, bio if you add one), content you publish as a creator (titles, descriptions, thumbnails), and comments or reactions you make on public or shared content. Visibility depends on the settings you and creators choose.
              </p>
            </section>

            <section id="shared-third" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("sharedThird")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                We do not sell your personal information. We share data with: (1) service providers who help us operate {SITE} (hosting, payments, analytics, security, etc.) under contracts that limit their use of your data; (2) payment partners to process payments and payouts; (3) government or law enforcement when required by law or to protect rights and safety; (4) tax authorities as required for reporting; (5) a successor entity in the event of a merger or sale of assets. When you connect third-party apps or accounts to {SITE}, we may share information you authorize with those services.
              </p>
            </section>

            <section id="your-control" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("yourControl")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                You can access, update, or delete much of your data through your account settings. You can request a copy of your data or request deletion by contacting us. You can opt out of marketing emails via the link in those emails or in your settings. If you are in the European Union, United Kingdom, or other regions with similar rights, you may have additional rights (e.g. access, rectification, restriction, portability, objection, withdrawal of consent). Contact us to exercise these rights. We will verify your identity before fulfilling requests.
              </p>
            </section>

            <section id="cookies" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("cookies")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                We and our service providers use cookies and similar technologies to operate the site, remember your preferences, analyze usage, and improve security. You can adjust your browser settings to limit or block cookies; some features may not work fully if you disable cookies. Third-party features (e.g. embedded content, social buttons) may set their own cookies; see their privacy policies for details.
              </p>
            </section>

            <section id="children" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("children")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                {SITE} is not directed to children under 13. You must be at least 13 to use {SITE} and old enough in your country to consent to processing of your personal data (parents may consent on your behalf where allowed). You must be at least 18 or have parental permission to upload content for sale or to purchase content or subscriptions. We do not knowingly collect personal data from children under 13; if we learn we have done so, we will delete it.
              </p>
            </section>

            <section id="changes" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("changes")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                We may update this Privacy Policy from time to time. If we make material changes, we will notify you (e.g. by email or a notice on the site) before they take effect. Continued use of {SITE} after changes means you accept the updated policy.
              </p>
            </section>

            <section id="contact" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("contact")}</h2>
              <p className="mt-3 text-[var(--foreground)]">
                For privacy questions, to exercise your rights, or to contact our data protection contact, please use the contact information provided on {DOMAIN} or in our Terms of Use. If you are in the EU/UK and have a complaint, you may also have the right to lodge a complaint with your local data protection authority.
              </p>
            </section>

            <footer className="mt-12 border-t border-[var(--border)] pt-6">
              <p className="text-sm text-[var(--muted)]">
                © {new Date().getFullYear()} {SITE}. {DOMAIN}.
              </p>
              <div className="mt-2 flex flex-wrap gap-4">
                <Link href={`/${locale}/terms`} className="text-[var(--accent)] hover:underline">
                  {t("termsOfUse")}
                </Link>
                <Link href={`/${locale}`} className="text-[var(--accent)] hover:underline">
                  {t("backToHome")}
                </Link>
              </div>
            </footer>
          </article>
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
