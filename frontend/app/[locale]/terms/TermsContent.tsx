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

export function TermsContent() {
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("terms");

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
              {SITE} is a platform that helps connect fans with creators they love. Our mission is to put creators first, and these terms attempt to do that. We have done our best to make this document easy to read. Above key sections we summarize the main points; those summaries are not legally binding—please read the full text for a complete understanding.
            </p>

            <nav className="my-8 rounded-xl border border-[var(--border)] bg-[var(--hover)] p-4">
              <h2 className="mb-3 text-sm font-semibold text-[var(--muted)]">{t("topics")}</h2>
              <ul className="space-y-1 text-sm">
                <li><a href="#welcome" className="text-[var(--accent)] hover:underline">Welcome to {SITE}</a></li>
                <li><a href="#account" className="text-[var(--accent)] hover:underline">{t("yourAccount")}</a></li>
                <li><a href="#abuse" className="text-[var(--accent)] hover:underline">{t("abusiveConduct")}</a></li>
                <li><a href="#creator" className="text-[var(--accent)] hover:underline">{t("beingCreator")}</a></li>
                <li><a href="#patron" className="text-[var(--accent)] hover:underline">{t("beingPatron")}</a></li>
                <li><a href="#role" className="text-[var(--accent)] hover:underline">{t("platformRole")}</a></li>
                <li><a href="#deletion" className="text-[var(--accent)] hover:underline">{t("accountDeletion")}</a></li>
                <li><a href="#creations" className="text-[var(--accent)] hover:underline">{t("yourCreations")}</a></li>
                <li><a href="#thirdparty" className="text-[var(--accent)] hover:underline">{t("thirdParty")}</a></li>
                <li><a href="#ourcreations" className="text-[var(--accent)] hover:underline">{t("ourCreations")}</a></li>
                <li><a href="#indemnity" className="text-[var(--accent)] hover:underline">{t("indemnity")}</a></li>
                <li><a href="#warranty" className="text-[var(--accent)] hover:underline">{t("warranty")}</a></li>
                <li><a href="#liability" className="text-[var(--accent)] hover:underline">{t("limitationLiability")}</a></li>
                <li><a href="#disputes" className="text-[var(--accent)] hover:underline">{t("disputeResolution")}</a></li>
                <li><a href="#everything" className="text-[var(--accent)] hover:underline">{t("everythingElse")}</a></li>
              </ul>
            </nav>

            <section id="welcome" className="scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">Welcome to {SITE}!</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryAgree")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                These are {SITE}&apos;s Terms of Use, and they apply to all users of the {SITE} platform. &quot;We,&quot; &quot;our,&quot; and &quot;us&quot; refer to the operator of {DOMAIN}. &quot;{SITE}&quot; refers to this platform and the services offered by us, including the website {DOMAIN} and related services. By using {SITE}, you agree to these terms and to any other policies we post, such as our Privacy Policy. Please read them carefully. If you have questions, contact us.
              </p>
            </section>

            <section id="account" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("yourAccount")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryAccount")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                When you create an account, you must provide us with accurate information, in good faith, and you agree to keep your information updated if it changes. To create an account or use {SITE}, you must be at least 13 years old and old enough to consent to the processing of your personal data in your country (in some countries we may allow your parent or guardian to do so). You must be at least 18 years old or have your parent&apos;s or legal guardian&apos;s permission to upload content for sale or to purchase content or subscriptions on {SITE}. You are responsible for anything that occurs when anyone is signed in to your account, as well as the security of the account. Please contact us immediately if you believe your account is compromised.
              </p>
            </section>

            <section id="abuse" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("abusiveConduct")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryAbuse")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                You are responsible for all activity on your account. If you violate our terms or policies, we may terminate your account. Do not do anything illegal, abusive towards others, that abuses {SITE} in a technical way, or that exploits {SITE} in an unintended manner (for example, using {SITE} as a storage platform). If you are a creator earning on {SITE}, we may be exposed to risk based on what you do with those funds, and we may look at what you do outside of {SITE}. If you find a new and creative way to harm {SITE} or our community, we may take action to prevent it.
              </p>
            </section>

            <section id="creator" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("beingCreator")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryCreator")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                A creator is someone who uploads or publishes content on {SITE} to engage with users who purchase content or subscriptions. You can use the tools we provide to showcase your content, set prices in credits, and offer subscriptions. As a creator, you may make content available for one-time purchase and may offer automatically-renewing subscriptions where supported. We typically handle payments, fraud, chargebacks, and payment disputes in accordance with our policies. Access to funds may be delayed or withheld for violations of our terms, for compliance reasons, or for tax reporting. Fees may apply to your offerings and subscriptions; the current fee information is available in our help or pricing documentation. You are responsible for reporting any income or taxes due on money you earn on {SITE}. We may collect tax information and remit transaction taxes where required by law. You must not post content that violates our terms or policies, that infringes others&apos; intellectual property, or that is illegal, abusive, or misleading. An account is tied to your creative output and cannot be sold or transferred to another creator.
              </p>
            </section>

            <section id="patron" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("beingPatron")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryPatron")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                A patron or buyer is someone who purchases content or subscribes on {SITE}, which may come with benefits from creators. You pay on a one-time or automatically-renewing basis as specified; {SITE} or its designated billing entity may appear on your charge. The timing and amount of each purchase or subscription depends on what you select and the creator. Subject to these terms and full payment, where a purchase or subscription includes access to a creator&apos;s content, you receive a limited, non-exclusive, non-transferable, revocable license to access and view that content for your personal, non-commercial use. Content you gain access to may become unavailable in certain circumstances (e.g. creator removal, policy violation). Subscriptions renew until you cancel via your account settings. We do not guarantee refunds; exceptions may be made at our sole discretion. You may not share content with others who have not purchased access.
              </p>
            </section>

            <section id="role" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("platformRole")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryRole")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                We may review pages and content on {SITE} to enforce these terms and our policies. We investigate reports of potential violations. These investigations may take time and may include activity outside of {SITE}. We will work with users to resolve issues where possible. We may terminate or remove accounts or content when we believe it is necessary to protect users, {SITE}, or the community. We comply with applicable economic sanctions and trade restrictions. Please report potential violations of our policies.
              </p>
            </section>

            <section id="deletion" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("accountDeletion")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryDeletion")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                You can request account deletion through your account or privacy settings or by contacting us. We can terminate or suspend your account at any time at our sole discretion. We can also cancel subscriptions and remove content or benefits. You may not bring a claim against us for suspending or terminating another person&apos;s account. These terms remain in effect even after your account is closed.
              </p>
            </section>

            <section id="creations" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("yourCreations")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryCreations")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                Creators keep ownership of their creations; users keep ownership of what they post. By making content available or posting on {SITE}, you grant us a royalty-free, perpetual, irrevocable, non-exclusive, sublicensable, worldwide license to use, copy, store, transmit, distribute, perform, display, and prepare derivative works of your content in connection with operating {SITE} (e.g. hosting, promoting, community features). You keep full ownership; we are not acquiring your intellectual property for our own gain. You must not post content that infringes others&apos; intellectual property. We may remove content that violates our terms or policies and respond to copyright notices in accordance with applicable law.
              </p>
            </section>

            <section id="thirdparty" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("thirdParty")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryThirdParty")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                When you connect your {SITE} account to third-party websites, apps, or services, you may grant them access to your account information or permission to act on your behalf. If you do, we will follow your instructions to the extent you have selected. You can revoke such access through your settings or the third party. See our Privacy Policy for more information.
              </p>
            </section>

            <section id="ourcreations" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("ourCreations")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryOurCreations")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                Our platform, design, text, code, logos, and other materials are protected by copyright, trademark, and other laws. You may not use, reproduce, or prepare derivative works of our creations except as we permit (e.g. promoting your page with our branding where we allow). Any feedback, suggestions, or bug reports you provide may be used by us without obligation to you, and you agree that we may treat such feedback as non-confidential and owned by us.
              </p>
            </section>

            <section id="indemnity" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("indemnity")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryIndemnity")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                You will indemnify us from all losses and liabilities, including legal fees, that arise from these terms or relate to your use of {SITE}. We reserve the right to control the defense of any claim covered by this clause, and you agree to cooperate. This obligation applies to our affiliates, officers, directors, employees, agents, and service providers.
              </p>
            </section>

            <section id="warranty" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("warranty")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryWarranty")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                {SITE} is provided &quot;as is&quot; and without warranty of any kind. To the greatest extent permitted by law, we disclaim all warranties, whether express or implied, including merchantability, fitness for a particular purpose, and non-infringement. These disclaimers also apply to our affiliates and service providers.
              </p>
            </section>

            <section id="liability" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("limitationLiability")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryLiability")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                To the greatest extent permitted by law, we are not liable to you for any incidental, consequential, or punitive damages arising from these terms or your use of {SITE}. Our liability for damages is limited to the amount we have earned through your use of {SITE}. We are not liable for loss associated with unfulfilled offerings or benefits or with conflicting contractual agreements. &quot;We&quot; and &quot;our&quot; in this section include our affiliates, officers, directors, employees, agents, and service providers.
              </p>
            </section>

            <section id="disputes" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("disputeResolution")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryDisputes")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                We encourage you to contact us if you have an issue. If a dispute arises out of these terms or your use of {SITE}, it will be resolved in accordance with the laws and courts applicable to the operator of {DOMAIN}, as stated in our policies or on the site. You agree to the jurisdiction and venue chosen for resolving such disputes.
              </p>
            </section>

            <section id="everything" className="mt-8 scroll-mt-6">
              <h2 className="text-xl font-bold text-[var(--foreground)]">{t("everythingElse")}</h2>
              <p className="mt-1 text-sm font-medium text-[var(--muted)]">{t("summaryEverything")}</p>
              <p className="mt-3 text-[var(--foreground)]">
                These terms and any referenced policies are the entire agreement between you and us and supersede prior agreements. They do not create a partnership, employment, or franchise relationship. If any provision is held unenforceable, it will be modified to the extent necessary to enforce it, or severed if it cannot be. Our failure to enforce a right does not waive future enforcement. We may change these terms; if we make material changes, we will notify you. Continued use of {SITE} after changes means you accept the new terms.
              </p>
            </section>

            <footer className="mt-12 border-t border-[var(--border)] pt-6">
              <p className="text-sm text-[var(--muted)]">
                © {new Date().getFullYear()} {SITE}. {DOMAIN}.
              </p>
              <p className="mt-2">
                <Link
                  href={`/${locale}`}
                  className="text-[var(--accent)] hover:underline"
                >
                  {t("backToHome")}
                </Link>
              </p>
            </footer>
          </article>
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
