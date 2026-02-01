import { SignUpContent } from "./SignUpContent";
import { Suspense } from "react";

export default function SignUpPage() {
  return (
    <Suspense fallback={<div className="text-center text-[var(--muted)]">Loading…</div>}>
      <SignUpContent />
    </Suspense>
  );
}
