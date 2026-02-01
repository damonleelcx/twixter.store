import Link from "next/link";
import { Icon } from "./Icon";

export function NavItem({
  label,
  icon,
  active,
  href,
}: {
  label: string;
  icon: string;
  active?: boolean;
  href?: string;
}) {
  const className = `flex items-center gap-3 rounded-full px-4 py-3 text-[15px] font-medium transition-colors md:px-3 md:py-2.5 xl:px-4 xl:py-3 ${
    active
      ? "text-[var(--foreground)]"
      : "text-[var(--muted)] hover:bg-[var(--hover)] hover:text-[var(--foreground)]"
  }`;
  const content = (
    <>
      <Icon d={icon} className="w-7 h-7 shrink-0 md:w-6 md:h-6 xl:w-7 xl:h-7" />
      <span className="hidden xl:inline">{label}</span>
    </>
  );
  if (href) {
    return (
      <Link href={href} className={className}>
        {content}
      </Link>
    );
  }
  return <a href="#" className={className}>{content}</a>;
}
