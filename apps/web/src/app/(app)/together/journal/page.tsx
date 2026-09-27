"use client";

import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import { useJournal } from "@/features/together/hooks";
import { JournalComposer } from "@/features/together/components/journal-composer";
import { iso, relativeDay } from "@/lib/dates";
import type { JournalEntry } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";
import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import {
  BottomActions,
  Button,
  EmptyState,
  Initial,
  Para,
  Skeleton,
  Title,
  TopBar,
  cx,
} from "@/components/ui/kit";

export default function Journal() {
  const journal = useJournal();
  // Hides the bottom add-button when the empty state already shows one.
  const hasAny = (journal.data?.length ?? 0) > 0;
  const couple = useCouple();
  const [open, setOpen] = useState(false);
  // Set when editing one of your own; null means writing a new one.
  const [editing, setEditing] = useState<JournalEntry | null>(null);
  const me = couple.data?.me;
  const partner = couple.data?.partner?.display_name ?? "Partner";
  const partnerPhoto = couple.data?.partner?.photo_url;
  const todayIso = iso(today());
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2">
          <Title>Journal</Title>
        </div>
        <QueryState queries={[journal]} loading={<Skeleton />}>
          {(entries) => {
            if (entries.length === 0) {
              return (
                <EmptyState
                  ghost="lines"
                  title="A notebook for the two of you."
                  text="Gratitude, a memory, something you noticed. Nothing here is public."
                  cta={
                    <Button icon="pencil" onClick={() => setOpen(true)}>
                      Write something for us
                    </Button>
                  }
                />
              );
            }
            return (
              <>
                <Para className="mt-1.5">A shared notebook, only for the two of you.</Para>
                <div className="mt-2 flex flex-col pb-4">
                  {entries.map((j, i) => (
                    <article
                      key={j.id}
                      className={cx(
                        "flex flex-col gap-2 py-[18px]",
                        i < entries.length - 1 && "border-b border-line",
                      )}
                    >
                      <div className="flex items-center gap-2.5">
                        <Initial
                          letter={(j.author_id === me?.id ? (me?.display_name ?? "Y") : partner)[0]}
                          photoUrl={j.author_id === me?.id ? me?.photo_url : partnerPhoto}
                          size={26}
                        />
                        <span className="grow text-support text-stone">
                          <span className="font-semibold text-ink">
                            {j.author_id === me?.id ? "You" : partner}
                          </span>{" "}
                          &middot; {relativeDay(j.date, todayIso)}
                        </span>
                        <span className="text-[12px] font-bold uppercase tracking-[0.06em] text-stone">
                          {j.tag}
                        </span>
                        {/* Only your own: the server refuses edits to your partner's. */}
                        {j.author_id === me?.id ? (
                          <button
                            type="button"
                            aria-label="Edit this entry"
                            onClick={() => {
                              setEditing(j);
                              setOpen(true);
                            }}
                            className="press -mr-2 flex h-11 w-11 items-center justify-center text-stone"
                          >
                            <Icon name="pencil" size={18} />
                          </button>
                        ) : null}
                      </div>
                      <p className="m-0 text-bodylg leading-[1.6]" data-selectable>
                        {j.text}
                      </p>
                    </article>
                  ))}
                </div>
              </>
            );
          }}
        </QueryState>
      </Main>
      {hasAny ? (
        <BottomActions>
          <Button
            icon="pencil"
            onClick={() => {
              setEditing(null);
              setOpen(true);
            }}
          >
            Write something for us
          </Button>
        </BottomActions>
      ) : null}
      <JournalComposer
        key={editing?.id ?? "new"}
        open={open}
        entry={editing}
        onClose={() => {
          setOpen(false);
          setEditing(null);
        }}
      />
    </>
  );
}
