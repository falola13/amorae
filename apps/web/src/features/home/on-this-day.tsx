"use client";

import Link from "next/link";

import { Micro } from "@/components/ui/kit";
import { useMemories } from "@/features/together/hooks";
import { routes } from "@/lib/routes";
import { onThisDay, yearsAgoLabel } from "./derive";

/** A memory from this day in an earlier year, ruled rather than boxed. Nothing
 *  at all when there isn't one, or while it loads. */
export function OnThisDay({ todayIso }: { todayIso: string }) {
  const memories = useMemories();
  const found = onThisDay(memories.data ?? [], todayIso);
  if (!found) return null;
  const { memory: m, yearsAgo } = found;
  return (
    <Link
      href={routes.memories}
      className="press mt-8 flex flex-col border-y border-line py-4 text-ink no-underline"
    >
      <Micro>{yearsAgoLabel(yearsAgo)}</Micro>
      <span className="mt-2 font-text text-section">{m.title}</span>
      {m.photo_url ? (
        // eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN
        <img
          src={m.photo_url}
          alt=""
          loading="lazy"
          decoding="async"
          className="mt-3 aspect-[4/3] w-full rounded-xs bg-photo object-cover"
        />
      ) : null}
      {m.location ? <span className="mt-2 text-support text-stone">{m.location}</span> : null}
      {m.note ? (
        <span className="mt-1 line-clamp-2 font-text text-bodylg leading-[1.45]">{m.note}</span>
      ) : null}
    </Link>
  );
}
