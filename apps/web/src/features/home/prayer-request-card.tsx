"use client";

import { useState } from "react";

import { Button, Field } from "@/components/ui/kit";
import { prayedAtLabel } from "@/features/prayers/derive";
import { useAsk, usePrayedFor, useWithdraw } from "@/features/prayers/hooks";
import type { PrayerRequest } from "@/lib/api/types";

/**
 * Their request, for the "Today's prayer" section. Unprayed, it is the
 * screen's one raised leaf: the thing someone is waiting on. Prayed, it
 * collapses to a single ruled line.
 */
export function TheirRequest({ request, first }: { request: PrayerRequest; first: string }) {
  const pray = usePrayedFor();
  if (request.prayed_at) {
    return (
      <div
        role="status"
        className="mt-3 flex min-h-11 items-center justify-between gap-3 border-y border-line"
      >
        <span className="text-support text-stone">Prayed for {first}</span>
        <button
          type="button"
          disabled={pray.isPending}
          onClick={() => pray.mutate(false)}
          className="press h-11 px-3 text-[15px] font-semibold text-plum"
        >
          Undo
        </button>
      </div>
    );
  }
  return (
    <div className="mt-3 flex flex-col gap-4 rounded-card border border-line bg-surface p-5">
      <div className="flex flex-col gap-2">
        <span className="lining text-micro uppercase text-stone">
          {first} · {prayedAtLabel(request.asked_at)}
        </span>
        <blockquote className="m-0 font-text text-reading text-ink" data-selectable>
          &ldquo;{request.text}&rdquo;
        </blockquote>
      </div>
      <div className="flex flex-col gap-2">
        <Button variant="secondary" loading={pray.isPending} onClick={() => pray.mutate(true)}>
          I&rsquo;m praying for you
        </Button>
        <span className="text-support text-stone">{first} will see it</span>
      </div>
    </div>
  );
}

/** Your own request: a quiet line under the rest of the prayer section. */
export function MyRequest({ request, first }: { request: PrayerRequest | null; first: string }) {
  const text = request?.text ?? null;
  const prayedAt = request?.prayed_at ?? null;
  const ask = useAsk();
  const withdraw = useWithdraw();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | undefined>();

  const open = () => {
    setDraft(text ?? "");
    setError(undefined);
    setEditing(true);
  };
  const send = () => {
    const line = draft.trim();
    if (!line) {
      setError(`Write what you’d like ${first} to pray for.`);
      return;
    }
    setError(undefined);
    const done = () => setEditing(false);
    ask.mutate(line, { onSuccess: done, onQueued: done });
  };

  if (editing) {
    return (
      <form
        className="mt-4 flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          send();
        }}
      >
        <Field
          label={`What should ${first} pray for?`}
          value={draft}
          maxLength={200}
          placeholder={`What should ${first} pray for?`}
          error={error}
          onChange={(e) => setDraft(e.target.value)}
        />
        <div className="flex items-center gap-2">
          <Button type="submit" variant="secondary" loading={ask.isPending}>
            Send
          </Button>
          <Button variant="quiet" onClick={() => setEditing(false)}>
            Cancel
          </Button>
        </div>
      </form>
    );
  }

  if (text === null) {
    return (
      <Button variant="text" className="mt-1 w-auto self-start px-0" onClick={open}>
        Ask {first} to pray for you
      </Button>
    );
  }

  return (
    <div className="mt-3 flex flex-col">
      <p className="m-0 text-support text-stone" data-selectable>
        You asked: <span className="font-text text-ink">&ldquo;{text}&rdquo;</span> &middot;{" "}
        {prayedAt ? `${first} prayed for you ${prayedAtLabel(prayedAt)}` : "Waiting"}
      </p>
      {prayedAt ? null : (
        <div className="-ml-2 flex items-center">
          <Button variant="quiet" onClick={open}>
            Change
          </Button>
          <Button variant="quiet" loading={withdraw.isPending} onClick={() => withdraw.mutate()}>
            Withdraw
          </Button>
        </div>
      )}
    </div>
  );
}
