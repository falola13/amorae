"use client";

import { useState } from "react";

import { Icon } from "@/components/icons";
import { Button, LinkButton, Sheet } from "@/components/ui/kit";
import { JournalComposer } from "@/features/together/components/journal-composer";
import { MemoryComposer } from "@/features/together/components/memory-composer";
import { routes } from "@/lib/routes";

/**
 * Home's "+": one place to start any of the things you keep together, rather
 * than a jump straight to a new plan. The composers are the same ones Our
 * story uses, so a memory or journal entry is written without leaving Home.
 */
export function HomeAdd({
  first,
  canAsk,
  onAsk,
  onPhotoFailed,
}: {
  first: string;
  /** Whether there's a request to ask for: needs a partner, and not one already prayed for. */
  canAsk: boolean;
  onAsk: () => void;
  onPhotoFailed: () => void;
}) {
  const [adding, setAdding] = useState(false);
  const [memoryOpen, setMemoryOpen] = useState(false);
  const [journalOpen, setJournalOpen] = useState(false);
  return (
    <>
      <button
        type="button"
        aria-label="Add something to our space"
        onClick={() => setAdding(true)}
        className="press flex h-11 w-11 items-center justify-center rounded-full text-ink"
      >
        <Icon name="plus" size={24} />
      </button>
      <Sheet open={adding} onClose={() => setAdding(false)} title="Add…" labelledBy="home-add-h">
        <div className="flex flex-col gap-1">
          {canAsk ? (
            <Button
              variant="secondary"
              icon="book"
              onClick={() => {
                setAdding(false);
                onAsk();
              }}
            >
              Ask {first} to pray for you
            </Button>
          ) : null}
          <Button
            variant="secondary"
            icon="image"
            onClick={() => {
              setAdding(false);
              setMemoryOpen(true);
            }}
          >
            Keep a memory
          </Button>
          <Button
            variant="secondary"
            icon="pencil"
            onClick={() => {
              setAdding(false);
              setJournalOpen(true);
            }}
          >
            Write in the journal
          </Button>
          <LinkButton href={routes.eventNew()} variant="secondary" icon="calendar">
            Plan something
          </LinkButton>
        </div>
      </Sheet>
      <MemoryComposer
        open={memoryOpen}
        onClose={() => setMemoryOpen(false)}
        onPhotoFailed={onPhotoFailed}
      />
      <JournalComposer open={journalOpen} entry={null} onClose={() => setJournalOpen(false)} />
    </>
  );
}
