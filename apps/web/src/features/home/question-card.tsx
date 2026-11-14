"use client";

import Link from "next/link";
import { useState } from "react";

import { BareTextarea, Button, Para, StatusMark } from "@/components/ui/kit";
import { useAnswerQuestion, useTodayQuestion } from "@/features/questions/hooks";
import { routes } from "@/lib/routes";

const MAX = 1000;

/**
 * Today's question, sealed until both of you have written. A change of paper
 * with an envelope flap, not a second raised card. The seal is the server's:
 * `theirs` simply is not in the response until you have answered.
 */
export function QuestionCard({ partner }: { partner: string }) {
  const today = useTodayQuestion();
  const first = partner.split(" ")[0];
  if (!today.data) return null;
  const { question, mine, theirs, theirs_answered: theyAnswered } = today.data;

  return (
    <section
      aria-label="Today’s question"
      className="relative -mx-6 mt-8 bg-paper px-6 pb-6 md:-mx-8 md:px-8"
    >
      <svg
        className="-mx-6 block h-10 w-[calc(100%+3rem)] md:-mx-8 md:w-[calc(100%+4rem)]"
        viewBox="0 0 390 40"
        preserveAspectRatio="none"
        aria-hidden="true"
      >
        <path
          d="M0 0 L195 38 L390 0 Z"
          className="fill-bg stroke-edge"
          strokeWidth={1}
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      <svg
        className="absolute left-1/2 top-[18px] -ml-[22px] h-11 w-11"
        viewBox="0 0 44 44"
        aria-hidden="true"
      >
        <path
          className="fill-plum"
          d="M22 2c4 0 6 2.5 9.5 3.6 3.6 1.2 7.2 2.6 8.4 6.4 1.1 3.4-.8 6.6-.2 10 .7 3.7 2.6 7-.1 10.3-2.6 3.1-6.4 3-9.8 4.6-3 1.5-5 4.8-8.5 4.9-3.6.1-5.9-3-9-4.6-3.3-1.6-7.1-1.9-9.2-5.1-2-3.2-.4-6.6-.7-10.1-.3-3.5-2.5-6.8-1-10.1 1.6-3.5 5.4-4.2 8.7-5.6C13.5 4.6 18 2 22 2z"
        />
        <text
          x="22"
          y="29"
          textAnchor="middle"
          className="fill-surface font-text text-[22px] font-medium italic"
        >
          &amp;
        </text>
      </svg>

      <div className="mt-5 text-micro uppercase text-stone">Today’s question</div>
      <h2 className="m-0 mt-3 font-text text-section" data-selectable>
        {question}
      </h2>

      {mine ? (
        <Answered
          question={question}
          mine={mine.text}
          theirs={theirs?.text ?? null}
          first={first}
          theyAnswered={theyAnswered}
        />
      ) : (
        <Unanswered first={first} theyAnswered={theyAnswered} />
      )}

      <Link
        href={routes.questions}
        className="press mt-1 inline-flex h-11 items-center text-[15px] font-semibold text-plum no-underline"
      >
        Past questions
      </Link>
    </section>
  );
}

function Who({
  first,
  theyAnswered,
  iAnswered,
}: {
  first: string;
  theyAnswered: boolean;
  iAnswered: boolean;
}) {
  return (
    <div className="mb-1 mt-4 flex flex-wrap gap-x-5 gap-y-1 text-support">
      <span className="inline-flex items-center gap-2">
        <StatusMark done={theyAnswered} size={20} label={theyAnswered ? "Answered" : "Not yet"} />
        {first} {theyAnswered ? "answered" : "hasn’t yet"}
      </span>
      <span className="inline-flex items-center gap-2">
        <StatusMark done={iAnswered} size={20} label={iAnswered ? "Answered" : "Not yet"} />
        You{iAnswered ? " answered" : ", not yet"}
      </span>
    </div>
  );
}

function Unanswered({ first, theyAnswered }: { first: string; theyAnswered: boolean }) {
  const [writing, setWriting] = useState(false);
  return (
    <>
      <Who first={first} theyAnswered={theyAnswered} iAnswered={false} />
      <Para size="support" className="mb-4">
        {theyAnswered
          ? "Both answers stay sealed until yours is written."
          : "Neither answer opens until you have both written yours."}
      </Para>
      {writing ? (
        <Composer initial="" onDone={() => setWriting(false)} onCancel={() => setWriting(false)} />
      ) : (
        <Button variant="secondary" onClick={() => setWriting(true)}>
          Write your answer
        </Button>
      )}
    </>
  );
}

function Answered({
  question,
  mine,
  theirs,
  first,
  theyAnswered,
}: {
  question: string;
  mine: string;
  theirs: string | null;
  first: string;
  theyAnswered: boolean;
}) {
  const [editing, setEditing] = useState(false);
  return (
    <>
      <Who first={first} theyAnswered={theyAnswered} iAnswered />
      {theirs !== null ? (
        <div className="mt-4 flex flex-col gap-1 border-t border-line pt-4">
          <span className="text-micro uppercase text-stone">{first}</span>
          <p className="m-0 font-text text-bodylg text-ink" data-selectable>
            {theirs}
          </p>
        </div>
      ) : (
        <Para size="support" className="mt-3">
          {first} hasn’t answered yet. Theirs opens here when they do.
        </Para>
      )}
      <div className="mt-4 flex flex-col gap-1 border-t border-line pt-4">
        <span className="text-micro uppercase text-stone">You</span>
        {editing ? (
          <Composer
            question={question}
            initial={mine}
            onDone={() => setEditing(false)}
            onCancel={() => setEditing(false)}
          />
        ) : (
          <>
            <p className="m-0 font-text text-bodylg text-ink" data-selectable>
              {mine}
            </p>
            <div className="-ml-2">
              <Button variant="quiet" onClick={() => setEditing(true)}>
                Change
              </Button>
            </div>
          </>
        )}
      </div>
    </>
  );
}

function Composer({
  initial,
  onDone,
  onCancel,
  question,
}: {
  initial: string;
  onDone: () => void;
  onCancel: () => void;
  question?: string;
}) {
  const answer = useAnswerQuestion();
  const [text, setText] = useState(initial);
  const [error, setError] = useState<string | undefined>();

  const send = () => {
    if (!text.trim()) {
      setError("Write a little before you send it.");
      return;
    }
    setError(undefined);
    answer.mutate(text, { onSuccess: onDone, onQueued: onDone });
  };

  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault();
        send();
      }}
    >
      <div className="rounded-input border border-edge bg-surface p-3 focus-within:border-[1.5px] focus-within:border-plum">
        <BareTextarea
          label={question ? `Your answer to: ${question}` : "Your answer"}
          hideLabel
          autoFocus
          value={text}
          maxLength={MAX}
          rows={4}
          placeholder="Take your time."
          onChange={(e) => setText(e.target.value)}
          error={error}
          className="min-h-28 text-bodylg"
        />
      </div>
      <div className="flex items-center gap-2">
        <Button type="submit" loading={answer.isPending}>
          Send
        </Button>
        <Button variant="quiet" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
