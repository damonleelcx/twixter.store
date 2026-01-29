import { setRequestLocale } from "next-intl/server";

type Props = { children: React.ReactNode; params: Promise<{ locale: string }> };

export default async function AuthLayout({ children, params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-[var(--background)] px-4 py-8">
      <div className="w-full max-w-[400px]">{children}</div>
    </div>
  );
}
