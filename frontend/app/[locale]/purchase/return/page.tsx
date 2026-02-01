import { Suspense } from "react";
import { PurchaseReturnContent } from "./PurchaseReturnContent";

export default function PurchaseReturnPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center text-[var(--muted)]">Loading…</div>}>
      <PurchaseReturnContent />
    </Suspense>
  );
}
