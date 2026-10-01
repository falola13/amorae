"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { BareTextarea, Button, ComposeBar, Field, Para, Sheet, cx } from "@/components/ui/kit";
import {
  CUSTOM_DAY_PICKS,
  DEFAULT_CUSTOM_DAYS,
  MAX_CUSTOM_DAYS,
  MAX_CUSTOM_LINE,
  MIN_CUSTOM_DAYS,
  customProblem,
  customPrompts,
  linesToPrompts,
  type CustomMode,
} from "@/features/together/challenges";
import { useStartCustomChallenge } from "@/features/together/hooks";
import { isApiError } from "@/lib/api/errors";
import { routes } from "@/lib/routes";

const MODES: { value: CustomMode; label: string }[] = [
  { value: "same", label: "Same every day" },
  { value: "each", label: "Different each day" },
];

/**
 * A challenge of the couple's own. Most are a habit — one line, many days —
 * so that is the default; writing thirty identical fields was the old way.
 * "Different each day" is one box, a line per day, so a list can be pasted.
 */
export default function NewChallengePage() {
  const router = useRouter();
  const start = useStartCustomChallenge();
  const [title, setTitle] = useState("");
  const [mode, setMode] = useState<CustomMode>("same");
  const [line, setLine] = useState("");
  // What is typed in the days box, which can be empty or half-typed.
  const [daysText, setDaysText] = useState(String(DEFAULT_CUSTOM_DAYS));
  const [list, setList] = useState("");
  const [leaving, setLeaving] = useState(false);
  const [errors, setErrors] = useState<{ title?: string; prompts?: string; form?: string }>({});

  const days = Number(daysText);
  const listed = linesToPrompts(list).length;
  const dirty = title.trim() !== "" || line.trim() !== "" || list.trim() !== "";
  const leave = () => router.replace(routes.challenges);
  const cancel = () => (dirty ? setLeaving(true) : leave());

  const submit = (e?: FormEvent) => {
    e?.preventDefault();
    const prompts = customPrompts(mode, line, days, list);
    const problem = customProblem(title, prompts);
    if (problem) {
      setErrors(problem.startsWith("Give") ? { title: problem } : { prompts: problem });
      return;
    }
    setErrors({});
    start.mutate(
      { custom: { title: title.trim(), prompts } },
      {
        onSuccess: leave,
        onError: (err) => {
          if (isApiError(err)) {
            const f = err.fields ?? {};
            setErrors({
              title: f["custom.title"],
              prompts: f["custom.prompts"],
              form: Object.keys(f).length ? undefined : err.message,
            });
          } else {
            setErrors({ form: "That didn’t go through. Try again in a moment." });
          }
        },
      },
    );
  };

  return (
    <>
      <ComposeBar
        onCancel={cancel}
        label="Our own challenge"
        done="Start"
        onDone={() => submit()}
        busy={start.isPending}
      />
      <form onSubmit={submit} className="flex grow flex-col gap-5 px-6 pb-8 pt-5" noValidate>
        <Field
          label="Name"
          value={title}
          maxLength={80}
          autoFocus
          placeholder="Thirty days, no soda"
          error={errors.title}
          onChange={(e) => setTitle(e.target.value)}
        />

        <div role="radiogroup" aria-label="How the days go" className="-mx-1 flex flex-wrap gap-1">
          {MODES.map((m) => (
            <button
              key={m.value}
              type="button"
              role="radio"
              aria-checked={mode === m.value}
              onClick={() => {
                setMode(m.value);
                setErrors({});
              }}
              className={cx(
                "press h-9 rounded-full px-3.5 text-[14px] font-semibold",
                mode === m.value ? "bg-plum-tint text-plum" : "text-stone",
              )}
            >
              {m.label}
            </button>
          ))}
        </div>

        {mode === "same" ? (
          <>
            <Field
              label="What to do each day"
              value={line}
              maxLength={MAX_CUSTOM_LINE}
              placeholder="No soda — fruit only"
              onChange={(e) => setLine(e.target.value)}
            />
            <div className="flex flex-col gap-2">
              <div
                role="radiogroup"
                aria-label="Number of days"
                className="-mx-1 flex flex-wrap gap-1"
              >
                {CUSTOM_DAY_PICKS.map((n) => (
                  <button
                    key={n}
                    type="button"
                    role="radio"
                    aria-checked={days === n}
                    onClick={() => setDaysText(String(n))}
                    className={cx(
                      "press h-9 min-w-11 rounded-full px-3 text-[14px] font-semibold",
                      days === n ? "bg-plum-tint text-plum" : "text-stone",
                    )}
                  >
                    {n}
                  </button>
                ))}
              </div>
              <Field
                label="Number of days"
                hint={`Or any number from ${MIN_CUSTOM_DAYS} to ${MAX_CUSTOM_DAYS}.`}
                inputMode="numeric"
                value={daysText}
                onChange={(e) => setDaysText(e.target.value.replace(/\D/g, "").slice(0, 3))}
                className="w-28"
              />
            </div>
            {line.trim() && days >= MIN_CUSTOM_DAYS && days <= MAX_CUSTOM_DAYS ? (
              <Para size="support" className="text-stone">
                {days} days of: {line.trim()}
              </Para>
            ) : null}
          </>
        ) : (
          <div className="flex flex-col gap-1.5">
            <BareTextarea
              label="One line per day"
              rows={8}
              value={list}
              placeholder={
                "Pray for each other’s week\nCook something new together\nWrite one thing you’re grateful for"
              }
              className="min-h-[180px] rounded-input border border-edge bg-surface px-4 py-3 text-body leading-[1.6]"
              onChange={(e) => setList(e.target.value)}
            />
            <Para size="support" className="text-stone">
              {listed === 0
                ? `Write or paste a line for each day — ${MIN_CUSTOM_DAYS} to ${MAX_CUSTOM_DAYS}.`
                : `${listed} ${listed === 1 ? "day" : "days"}`}
            </Para>
          </div>
        )}

        {errors.prompts ? (
          <div role="alert" className="text-[13px] text-red">
            {errors.prompts}
          </div>
        ) : null}
        {errors.form ? (
          <div role="alert" className="text-[13px] text-red">
            {errors.form}
          </div>
        ) : null}
        <button type="submit" className="sr-only">
          Start
        </button>
      </form>

      <Sheet
        open={leaving}
        onClose={() => setLeaving(false)}
        title="Leave without starting?"
        labelledBy="ch-leave-h"
      >
        <Para>What you have written here won&rsquo;t be kept.</Para>
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
