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
  Ornament,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";

/**
 * A memory's picture, at the shape it was taken in.
 *
 * An earlier pass cropped every photo to a height chosen by its position, to
 * give the page an album's rhythm. That is the wrong way round: it crops a
 * couple's own photographs — faces included — to suit a layout. Letting each
 * one keep its aspect gives the same varied rhythm, except the variation is
 * theirs rather than imposed. The cap is only so a panorama cannot take over
 * the screen.
 *
 * The placeholder is now only the degraded case — a memory that says it has a
 * photo whose URL we could not build, which happens when Cloudinary is not
 * configured. It deliberately ignores the album height: there is nothing to
 * look at, and a tall empty block is a worse answer than a short one.
 */
/**
 * A month, as a running head rather than a floating label.
 *
 * The rule is what makes it a heading: a label alone in space is read as one
 * more line of text, and an archive that runs for years needs the eye to catch
 * where one month stops. Print has done it this way for five hundred years.
 */
function RunningHead({ children }: { children: string }) {
  return (
    <div className="flex items-center gap-3 pt-1">
      <Micro>{children}</Micro>
      <span className="h-px grow bg-line" />
    </div>
  );
}

function Photo({ label, src }: { label: string; src?: string }) {
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
        className="max-h-[440px] w-full rounded-btn bg-photo object-cover"
      />
    );
  }
  return (
    <div
      role="img"
      aria-label={label}
      className="flex h-[132px] items-center justify-center gap-2 rounded-btn bg-photo text-[13px] font-semibold text-stone"
    >
      <Icon name="image" size={18} />
      Photo
    </div>
  );
}

/**
 * The line under the end-mark. It states a fact — how much is kept, and how far
 * back it goes — because the bottom of an archive is worth a sentence and is
 * not worth a slogan.
 */
function keptLine(count: number, oldest: string) {
  const moments = count === 1 ? "One moment" : `${count} moments`;
  return `${moments} kept, back to ${monthName(oldest)} ${oldest.slice(0, 4)}.`;
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
                      {showMonth ? <RunningHead>{month}</RunningHead> : null}
                      <article className="flex flex-col gap-1">
                        {m.has_photo ? <Photo label={m.title} src={m.photo_url} /> : null}
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
                <Ornament caption={keptLine(sorted.length, sorted[sorted.length - 1].date)} />
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
