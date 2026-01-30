import { PurchaseReturnRedirectContent } from "./PurchaseReturnRedirectContent";
import { Suspense } from "react";

export default function PurchaseReturnRedirectPage() {
  return (
    <html>
      <body>
        <Suspense
          fallback={
            <div className="flex min-h-screen items-center justify-center bg-[var(--background)] text-[var(--foreground)]">
              <p className="text-[var(--muted)]">Loading…</p>
            </div>
          }
        >
          <PurchaseReturnRedirectContent />
        </Suspense>
      </body>
    </html>
  );
}
