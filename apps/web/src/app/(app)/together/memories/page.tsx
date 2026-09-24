"use client";

import { useState } from "react";
import { useMemories } from "@/features/together/hooks";
import { MemoryComposer } from "@/features/together/components/memory-composer";
import { longDate, monthName } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import {
  BottomActions,
  Button,
  EmptyState,
  Micro,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";

/**
 * A memory's picture, or the space where one would be.
 *
 * The placeholder is still here because a memory saved without a photo is the
 * common case and an archive of bare text needs some rhythm. What changed is
 * that a memory with a photo now shows the photo.
 */
function Photo({ h, label, src }: { h: number; label: string; src?: string }) {
  if (src) {
    return (
      // Cloudinary already delivers this with f_auto,q_auto from its own CDN.
      // Sending it through Vercel's optimiser too would add a hop and, on
      // Hobby, a bill, to re-do work that is already done.
      // eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN
      <img
        src={src}
        alt={label}
        loading="lazy"
        className="w-full rounded-btn bg-photo object-cover"
        style={{ height: h }}
      />
    );
  }
  return (
    <div
      role="img"
      aria-label={label}
      className="flex items-center justify-center gap-2 rounded-btn bg-photo text-[13px] font-semibold text-stone"
      style={{ height: h }}
    >
      <Icon name="image" size={18} />
      Photo
    </div>
  );
}

export default function Memories() {
  const memories = useMemories();
  // The empty state already offers this, centred, with a line saying what
  // it is for. Showing the bar as well put two buttons for the same thing
  // on one screen, one under the other. The bar is for when there is a
  // list to add to.
  const hasAny = (memories.data?.length ?? 0) > 0;
  const [open, setOpen] = useState(false);
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2">
          <Title>Memories</Title>
        </div>
        <QueryState queries={[memories]} loading={<Skeleton />}>
          {(memoriesData) => {
            if (memoriesData.length === 0) {
              return (
                <EmptyState
                  ghost="frames"
                  title="Your story starts here."
                  text="Save your first shared moment."
                  cta={
                    <Button icon="plus" onClick={() => setOpen(true)}>
                      Save a moment
                    </Button>
                  }
                />
              );
            }
            const sorted = [...memoriesData].sort((a, b) => b.date.localeCompare(a.date));
            return (
              <div className="mt-3.5 flex flex-col gap-[26px] pb-4">
                {sorted.map((m, idx) => {
                  const month = `${monthName(m.date)} ${m.date.slice(0, 4)}`;
                  const prev =
                    idx > 0
                      ? `${monthName(sorted[idx - 1].date)} ${sorted[idx - 1].date.slice(0, 4)}`
                      : null;
                  const showMonth = month !== prev;
                  return (
                    <div key={m.id} className="flex flex-col gap-3.5">
                      {showMonth ? <Micro>{month}</Micro> : null}
                      <article className="flex flex-col gap-1">
                        {m.has_photo ? (
                          <Photo h={m.note ? 190 : 120} label={m.title} src={m.photo_url} />
                        ) : null}
                        <div className="mt-2 text-bodylg font-semibold tracking-[-0.01em]">
                          {m.title}
                        </div>
                        <div className="text-support text-stone">
                          {longDate(m.date)} {m.date.slice(0, 4)}
                          {m.location ? ` · ${m.location}` : ""}
                        </div>
                        {m.note ? (
                          <p
                            className="m-0 mt-0.5 text-[15px] leading-[1.55] text-stone"
                            data-selectable
                          >
                            {m.note}
                          </p>
                        ) : null}
                      </article>
                    </div>
                  );
                })}
              </div>
            );
          }}
        </QueryState>
      </Main>
      {hasAny ? (
        <BottomActions>
          <Button icon="plus" onClick={() => setOpen(true)}>
            Save a moment from today
          </Button>
        </BottomActions>
      ) : null}
      <MemoryComposer open={open} onClose={() => setOpen(false)} />
    </>
  );
}
