"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Main } from "@/components/layout/screen";
import { BareTextarea, Button, ComposeBar, Para, Sheet, Skeleton } from "@/components/ui/kit";
import { useAnniversary, useWriteLetter } from "@/features/anniversary/hooks";
import { useCouple } from "@/features/couple/hooks";
import { isApiError } from "@/lib/api/errors";
import { weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";

const MAX = 6000;

/**
 * Writing the letter. Paper, a salutation, a signature — and a plain
 * statement of when the other one can open it, since that is the thing a
 * person writing it most wants to be sure of.
 */
export default function LetterPage() {
  const router = useRouter();
  const anniversary = useAnniversary();
  const couple = useCouple();
  const write = useWriteLetter();
  const a = anniversary.data;
  // null until the person types: until then the draft is whatever is kept.
  const [text, setText] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [leaving, setLeaving] = useState(false);

  const me = couple.data?.me.display_name.split(" ")[0] ?? "";
  const partner = couple.data?.partner?.display_name.split(" ")[0] ?? "them";
  const kept = a?.mine?.text ?? "";
  const draft = text ?? kept;
  const dirty = draft.trim() !== kept.trim();
  const leave = () => router.replace(routes.ourDay());

  const keep = () => {
    if (!draft.trim()) {
      setError("Write a little before you keep it.");
      return;
    }
    setError(null);
    write.mutate(draft, {
      onSuccess: leave,
      onQueued: leave,
      onError: (err) =>
        setError(
          isApiError(err)
            ? (err.fields?.text ?? err.message)
            : "That didn’t go through. Try again in a moment.",
        ),
    });
  };

  if (!a) {
    return (
      <>
        <ComposeBar onCancel={leave} label="Your letter" done="Keep" />
        <Main>
          <div className="pt-6">
            {anniversary.isError ? (
              <Para>There&rsquo;s no letter to write just now.</Para>
            ) : (
              <Skeleton lines={5} />
            )}
          </div>
        </Main>
      </>
    );
  }

  if (!a.writable) {
    return (
      <>
        <ComposeBar onCancel={leave} label="Your letter" done="" />
        <Main>
          <Para size="lg" className="pt-6">
            Letters for this anniversary open for writing on {weekdayDate(a.writable_from)}.
          </Para>
        </Main>
      </>
    );
  }

  const theDay = a.days_until === 0;
  return (
    <>
      <ComposeBar
        onCancel={() => (dirty ? setLeaving(true) : leave())}
        label={a.mine ? "Your letter" : `A letter to ${partner}`}
        done={theDay ? "Give it" : "Seal it"}
        onDone={keep}
        busy={write.isPending}
      />
      <Main className="pb-8">
        <Para size="support" className="pt-3 text-center">
          {theDay
            ? `${partner} can read it as soon as you keep it.`
            : `Sealed until ${weekdayDate(a.on)}. ${partner} will only know something is waiting.`}
        </Para>
        <div className="mt-4 rounded-card border border-line bg-letter bg-position-[0_20px] px-5 pb-6 pt-5">
          <p className="m-0 font-serif text-[26px] font-medium italic leading-[32px] text-plum-dark">
            Dear {partner},
          </p>
          <BareTextarea
            label={`Your letter to ${partner}`}
            hideLabel
            autoFocus
            value={draft}
            maxLength={MAX}
            placeholder="What this year was, what you saw in them, what you hope for the next…"
            onChange={(e) => setText(e.target.value)}
            error={error ?? undefined}
            className="min-h-[50dvh] text-[17px] leading-[32px]"
          />
          <p className="m-0 mt-2 text-right font-serif text-[24px] italic leading-[32px] text-plum">
            with love, {me}
          </p>
        </div>
        {draft.length > MAX - 400 ? (
          <Para size="support" className="mt-2 text-right">
            {MAX - draft.length} characters left
          </Para>
        ) : null}
      </Main>

      <Sheet
        open={leaving}
        onClose={() => setLeaving(false)}
        title="Leave without keeping it?"
        labelledBy="letter-leave-h"
      >
        <Para>What you&rsquo;ve written since you last kept it won&rsquo;t be saved.</Para>
        <div className="flex flex-col gap-1">
          <Button variant="secondary" onClick={leave}>
            Leave
          </Button>
          <Button variant="text" onClick={() => setLeaving(false)}>
            Keep writing
          </Button>
        </div>
      </Sheet>
    </>
  );
}
