import Link from "next/link";
import type { PrayerPoint } from "@/lib/api/types";
import { StatusMark } from "@/components/ui/kit";

export function PrayerRow({
  p,
  index,
  done,
  href,
  note,
}: {
  p: PrayerPoint;
  index: number;
  done: boolean;
  href: string;
  note?: string;
}) {
  return (
    <Link
      href={href}
      className="press flex min-h-[76px] items-center gap-4 border-b border-line py-3.5 text-ink no-underline"
    >
      <span className="lining w-7 shrink-0 self-start pt-px font-text text-[17px] tabular-nums text-stone">
        {String(index + 1).padStart(2, "0")}
      </span>
      <span className="flex grow flex-col gap-0.5">
        <span className="font-text text-bodylg">{p.title}</span>
        {p.text ? <span className="text-support text-stone">{p.text}</span> : null}
        {note ? <span className="text-support font-semibold text-amber">{note}</span> : null}
      </span>
      <StatusMark done={done} />
    </Link>
  );
}
