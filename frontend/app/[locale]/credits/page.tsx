"use client";

import { LeftSidebar } from "@/components/LeftSidebar";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { RightSidebar } from "@/components/RightSidebar";
import {
  capturePayPalOrder,
  createCreditsCheckout,
  createPayPalCreditsOrder,
  fetchCurrentUser,
  type CurrentUser,
} from "@/lib/api";
import { PayPalButtons, PayPalScriptProvider } from "@paypal/react-paypal-js";
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

const CREDIT_PACKAGES = [
  { credits: 69, amount: 0.69 },
  { credits: 169, amount: 3.69 },
  { credits: 369, amount: 13.69 },
  { credits: 569, amount: 23.69 },
  { credits: 769, amount: 33.69 },
  { credits: 969, amount: 43.69 },
];

const stripePublishableKey = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || "";
const stripePromise = stripePublishableKey ? loadStripe(stripePublishableKey) : null;
const paypalClientId = process.env.NEXT_PUBLIC_PAYPAL_CLIENT_ID || "";

function CreditsCheckoutForm({
  locale,
  userEmail,
  selectedAmount,
  selectedCredits,
}: {
  locale: string;
  userEmail: string | null;
  selectedAmount: number;
  selectedCredits: number;
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
  // StripeCheckoutAmount: amount = display string (e.g. "$5.00"), minorUnitsAmount = cents
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
      // Stripe requires email; use current user's email or updateEmail before confirm
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
    <div className="space-y-6">
      <form onSubmit={handleSubmit} className="space-y-4">
        <h4 className="font-bold">Payment</h4>
        <PaymentElement id="payment-element" />
        <button
          type="submit"
          disabled={isSubmitting}
          className="w-full rounded-full bg-[var(--accent)] py-3 font-bold text-[var(--accent-foreground)] transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {isSubmitting ? "Processing…" : amountDisplay ? `Pay ${amountDisplay} now` : "Pay now"}
        </button>
        {message && (
          <p className="text-sm text-red-600 dark:text-red-400" id="payment-message">
            {message}
          </p>
        )}
      </form>

      {paypalClientId && (
        <div className="space-y-2">
          <p className="text-sm text-[var(--muted)]">{t("orPayWithPayPal")}</p>
          <PayPalScriptProvider
            options={{
              clientId: paypalClientId,
              currency: "USD",
              intent: "capture",
              components: "buttons",
            }}
          >
            <PayPalButtons
              style={{ layout: "vertical", color: "gold", label: "paypal" }}
              createOrder={async () => {
                const { id } = await createPayPalCreditsOrder(selectedAmount, selectedCredits);
                return id;
              }}
              onApprove={async (data) => {
                try {
                  await capturePayPalOrder(data.orderID ?? "");
                  router.push(`/${locale}/purchase/return`);
                  router.refresh();
                } catch (err) {
                  console.error("PayPal capture failed:", err);
                }
              }}
            />
          </PayPalScriptProvider>
        </div>
      )}
    </div>
  );
}

export default function CreditsPage() {
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const t = useTranslations("auth");
  const tNav = useTranslations("nav");
  const [user, setUser] = useState<CurrentUser | null | undefined>(undefined);
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [selectedPackage, setSelectedPackage] = useState<{ amount: number; credits: number } | null>(null);
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

  async function handleSelectPackage(amount: number, credits: number) {
    if (!stripePublishableKey) {
      setError("Stripe is not configured. Set NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY.");
      return;
    }
    setError("");
    setSelectedPackage({ amount, credits });
    setLoading(true);
    try {
      const data = await createCreditsCheckout(amount, credits);
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
                onClick={() => { setClientSecret(null); setSelectedPackage(null); }}
                className="text-sm text-[var(--muted)] hover:text-[var(--foreground)]"
              >
                ← Back to packages
              </button>
            </div>
            <CheckoutProvider
              stripe={stripePromise}
              options={{
                clientSecret,
                elementsOptions: { appearance },
              }}
            >
              <CreditsCheckoutForm
                locale={locale}
                userEmail={user.email ?? null}
                selectedAmount={selectedPackage?.amount ?? 0}
                selectedCredits={selectedPackage?.credits ?? 0}
              />
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
            <h1 className="text-xl font-bold">{t("buyCredits")}</h1>
          </div>
          <div className="p-6 space-y-6">
            {error && (
              <p className="rounded-lg border border-red-500/50 bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400">
                {error}
              </p>
            )}
            <p className="text-sm text-[var(--muted)]">
              {t("creditsPackageHint")}
            </p>
            <div className="grid gap-4 sm:grid-cols-3">
              {CREDIT_PACKAGES.map((pkg) => (
                <button
                  key={pkg.credits}
                  type="button"
                  disabled={loading}
                  onClick={() => handleSelectPackage(pkg.amount, pkg.credits)}
                  className="rounded-xl border border-[var(--border)] p-4 text-left transition-colors hover:bg-[var(--hover)] disabled:opacity-50"
                >
                  <p className="font-bold">{pkg.credits} {t("credits")}</p>
                  <p className="text-lg font-bold text-[var(--accent)]">${pkg.amount}</p>
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
