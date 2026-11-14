"use client";

import { useState } from "react";

import { Button, Field, Para, Section, cx } from "@/components/ui/kit";
import { prayedAtLabel } from "@/features/prayers/derive";
import { useAsk, usePrayedFor, useTodayRequests, useWithdraw } from "@/features/prayers/hooks";

/**
 * "Pray for me today": one line each, once a day. Theirs comes first because
 * someone is waiting on it, and it is the one raised panel; yours sits under
 * it, in plain type. Renders nothing until the day's requests have loaded, so
 * a slow fetch never pushes the rest of Home around.
 */
export function PrayerRequestCard({ partner }: { partner: string }) {
  const requests = useTodayRequests();
  const first = partner.split(" ")[0];
  if (!requests.data) return null;
  const { mine, theirs } = requests.data;

  return (
    <Section
      label="Pray for me today"
      className="mt-8"
      trailing={theirs ? <span className="text-support text-stone">{first}</span> : undefined}
    >
      {theirs ? (
        <TheirRequest text={theirs.text} prayedAt={theirs.prayed_at} first={first} />
      ) : null}
      <MyRequest
        text={mine?.text ?? null}
        prayedAt={mine?.prayed_at ?? null}
        first={first}
        theirsAsked={Boolean(theirs)}
      />
    </Section>
  );
}

function TheirRequest({
  text,
  prayedAt,
  first,
}: {
  text: string;
  prayedAt: string | null;
  first: string;
}) {
  const pray = usePrayedFor();
  return (
    // The screen's one raised leaf: the thing someone is waiting on.
    <div className="mt-3 flex flex-col gap-4 rounded-card border border-line bg-surface p-5">
      <blockquote className="m-0 font-text text-reading text-ink" data-selectable>
        &ldquo;{text}&rdquo;
      </blockquote>
      {prayedAt ? (
        <div
          role="status"
          className="flex h-[54px] items-center justify-between rounded-btn bg-green-tint pl-[18px] pr-1.5"
        >
          <span className="text-[16px] font-semibold text-green">Prayed</span>
          <button
            type="button"
            disabled={pray.isPending}
            onClick={() => pray.mutate(false)}
            className="press h-11 px-3 text-[15px] font-semibold text-green"
          >
            Undo
          </button>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          <Button loading={pray.isPending} onClick={() => pray.mutate(true)}>
            I&rsquo;m praying for you
          </Button>
          <span className="text-support text-stone">{first} will see it</span>
        </div>
      )}
    </div>
  );
}

function MyRequest({
  text,
  prayedAt,
  first,
  theirsAsked,
}: {
  text: string | null;
  prayedAt: string | null;
  first: string;
  theirsAsked: boolean;
}) {
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

  if (editing || (text === null && !theirsAsked)) {
    return (
      <form
        className={cx("flex flex-col gap-3", theirsAsked || text !== null ? "mt-4" : "mt-3")}
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
          <Button type="submit" loading={ask.isPending}>
            Send
          </Button>
          {editing ? (
            <Button variant="quiet" onClick={() => setEditing(false)}>
              Cancel
            </Button>
          ) : null}
        </div>
      </form>
    );
  }

  if (text === null) {
    // They have asked and you haven't: a quiet way in, under their request.
    return (
      <Button variant="text" className="mt-1" onClick={open}>
        Ask {first} to pray for you too
      </Button>
    );
  }

  return (
    <div className="mt-4 flex flex-col gap-1 border-t border-line pt-4">
      <span className="text-micro uppercase text-stone">You asked</span>
      <p className="m-0 font-text text-bodylg text-ink" data-selectable>
        &ldquo;{text}&rdquo;
      </p>
      <Para size="support">
        {prayedAt ? `${first} prayed for you · ${prayedAtLabel(prayedAt)}` : `Waiting for ${first}`}
      </Para>
      {prayedAt ? null : (
        <div className="-ml-3 flex items-center">
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
