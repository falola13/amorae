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

function Photo({ h, label }: { h: number; label: string }) {
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
                        {m.has_photo ? <Photo h={m.note ? 190 : 120} label={m.title} /> : null}
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
      <BottomActions>
        <Button icon="plus" onClick={() => setOpen(true)}>
          Save a moment from today
        </Button>
      </BottomActions>
      <MemoryComposer open={open} onClose={() => setOpen(false)} />
    </>
  );
}
