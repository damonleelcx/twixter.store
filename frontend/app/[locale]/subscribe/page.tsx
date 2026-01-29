"use client";

import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import { createMembershipCheckout, fetchCurrentUser, type CurrentUser } from "@/lib/api";
import {
  CheckoutProvider,
  PaymentElement,
  useCheckout,
} from "@stripe/react-stripe-js/checkout";
import { loadStripe } from "@stripe/stripe-js";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

const MONTHLY_PRICE = 6.99;
const MEMBERSHIP_PLANS: { months: 1 | 3 | 9; price: number; labelKey: string }[] = [
  { months: 1, price: MONTHLY_PRICE * 1, labelKey: "plan1Month" },
  { months: 3, price: 16.99, labelKey: "plan3Months" },
  { months: 9, price: 56.99, labelKey: "plan9Months" },
];

const stripePublishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || "";
const stripePromise = stripePublishableKey ? loadStripe(stripePublishableKey) : null;

function MembershipCheckoutForm({
  locale,
  userEmail,
}: {
  locale: string;
  userEmail: string | null;
}) {
  const router = useRouter();
  const t = useTranslations("auth");
  const checkoutState = useCheckout();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  if (checkoutState.type === "loading") {
    return <div className="text-[var(--muted)]">Loading checkout...</div>;
  }
  if (checkoutState.type === "error") {
    return (
      <div className="rounded-lg border border-red-500/50 bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400">
        {checkoutState.error.message}
      </div>
    );
  }

  const { checkout } = checkoutState;
  const totalAmount = checkout.total?.total;
  const amountDisplay =
    totalAmount?.amount ??
    (typeof totalAmount?.minorUnitsAmount === "number"
      ? `$${(totalAmount.minorUnitsAmount / 100).toFixed(2)}`
      : "");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setMessage(null);
    setIsSubmitting(true);
    try {
      if (userEmail) {
        const updateResult = await checkout.updateEmail(userEmail);
        if (updateResult.type === "error") {
          setMessage(updateResult.error.message);
          setIsSubmitting(false);
          return;
        }
      }
      const result = await checkout.confirm(userEmail ? { email: userEmail } : undefined);
      if (result.type === "error") {
        setMessage(result.error.message);
      } else {
        router.push(`/${locale}/purchase/return`);
        router.refresh();
      }
    } catch (err) {
      setMessage(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <h4 className="font-bold">{t("payment")}</h4>
      <PaymentElement id="payment-element" />
      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 disabled:opacity-50"
      >
        {isSubmitting ? t("processing") : amountDisplay ? `${t("payNow")} ${amountDisplay}` : t("payNow")}
      </button>
      {message && (
        <p className="text-sm text-red-600 dark:text-red-400" id="payment-message">
          {message}
        </p>
      )}
    </form>
  );
}

export default function SubscribePage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const tNav = useTranslations("nav");
  const tSidebar = useTranslations("sidebar");
  const [user, setUser] = useState<CurrentUser | null | undefined>(undefined);
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    fetchCurrentUser().then((u) => {
      if (!cancelled) setUser(u ?? null);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleSelectPlan(months: 1 | 3 | 9) {
    if (!stripePublishableKey) {
      setError("Stripe is not configured. Set NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY.");
      return;
    }
    setError("");
    setLoading(true);
    try {
      const data = await createMembershipCheckout(months);
      if (data.clientSecret) {
        setClientSecret(data.clientSecret);
      } else if (data.url) {
        window.location.href = data.url;
      } else {
        setError("Invalid checkout response.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setLoading(false);
    }
  }

  if (user === undefined) {
    return (
      <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
        <div className="mx-auto flex max-w-[1280px]">
          <LeftSidebar locale={locale} />
          <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px] p-6">
            <p className="text-[var(--muted)]">Loading...</p>
          </main>
          <RightSidebar />
        </div>
        <MobileBottomNav />
      </div>
    );
  }

  if (user === null) {
    router.replace(`/${locale}/auth/signin`);
    return null;
  }

  if (clientSecret && stripePromise) {
    const appearance = { theme: "stripe" as const };
    return (
      <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
        <div className="mx-auto flex max-w-[1280px]">
          <LeftSidebar locale={locale} />
          <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px] p-6">
            <div className="mb-4">
              <button
                type="button"
                onClick={() => setClientSecret(null)}
                className="text-sm text-[var(--muted)] hover:text-[var(--foreground)]"
              >
                ← {t("backToPlans")}
              </button>
            </div>
            <CheckoutProvider
              stripe={stripePromise}
              options={{
                clientSecret,
                elementsOptions: { appearance },
              }}
            >
              <MembershipCheckoutForm locale={locale} userEmail={user.email ?? null} />
            </CheckoutProvider>
          </main>
          <RightSidebar />
        </div>
        <MobileBottomNav />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--foreground)]">
      <div className="mx-auto flex max-w-[1280px]">
        <LeftSidebar locale={locale} />
        <main className="min-w-0 flex-1 border-x border-[var(--border)] md:max-w-[600px]">
          <div className="border-b border-[var(--border)] p-4">
            <h1 className="text-xl font-bold">{tSidebar("premiumTitle")}</h1>
          </div>
          <div className="p-6 space-y-6">
            <p className="text-sm text-[var(--muted)]">{tSidebar("premiumDescription")}</p>
            {error && (
              <p className="rounded-lg border border-red-500/50 bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400">
                {error}
              </p>
            )}
            <div className="grid gap-4 sm:grid-cols-3">
              {MEMBERSHIP_PLANS.map((plan) => (
                <button
                  key={plan.months}
                  type="button"
                  disabled={loading}
                  onClick={() => handleSelectPlan(plan.months)}
                  className="rounded-xl border border-[var(--border)] p-4 text-left transition-colors hover:bg-[var(--hover)] disabled:opacity-50"
                >
                  <p className="font-bold">{t(plan.labelKey)}</p>
                  <p className="text-lg font-bold text-[var(--accent)]">${plan.price.toFixed(2)}</p>
                </button>
              ))}
            </div>
            <Link
              href={`/${locale}/profile`}
              className="inline-block text-sm text-[var(--muted)] hover:text-[var(--foreground)]"
            >
              ← {tNav("profile")}
            </Link>
          </div>
        </main>
        <RightSidebar />
      </div>
      <MobileBottomNav />
    </div>
  );
}
