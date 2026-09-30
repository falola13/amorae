"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Icon } from "@/components/icons";
import { Button, ComposeBar, Field, Para, Sheet } from "@/components/ui/kit";
import {
  DEFAULT_CUSTOM_DAYS,
  MAX_CUSTOM_DAYS,
  MIN_CUSTOM_DAYS,
  customProblem,
  resizePrompts,
} from "@/features/together/challenges";
import { useStartCustomChallenge } from "@/features/together/hooks";
import { isApiError } from "@/lib/api/errors";
import { routes } from "@/lib/routes";

export default function NewChallengePage() {
  const router = useRouter();
  const start = useStartCustomChallenge();
  const [title, setTitle] = useState("");
  const [prompts, setPrompts] = useState<string[]>(() => resizePrompts([], DEFAULT_CUSTOM_DAYS));
  // What is typed in the days box, which can be empty or half-typed while the list keeps its size.
  const [daysText, setDaysText] = useState(String(DEFAULT_CUSTOM_DAYS));
  const [leaving, setLeaving] = useState(false);
  const [errors, setErrors] = useState<{ title?: string; prompts?: string; form?: string }>({});

  const dirty = title.trim() !== "" || prompts.some((p) => p.trim() !== "");
  const leave = () => router.replace(routes.challenges);
  const cancel = () => (dirty ? setLeaving(true) : leave());

  const setCount = (n: number) => {
    const next = resizePrompts(prompts, n);
    setPrompts(next);
    setDaysText(String(next.length));
  };
  const removeAt = (i: number) => {
    const next = prompts.filter((_, j) => j !== i);
    setPrompts(next);
    setDaysText(String(next.length));
  };
  const onDays = (raw: string) => {
    const digits = raw.replace(/\D/g, "").slice(0, 2);
    setDaysText(digits);
    const n = Number(digits);
    if (n >= MIN_CUSTOM_DAYS && n <= MAX_CUSTOM_DAYS) setPrompts(resizePrompts(prompts, n));
  };

  const submit = (e?: FormEvent) => {
    e?.preventDefault();
    const problem = customProblem(title, prompts);
    if (problem) {
      setErrors(problem.startsWith("Give") ? { title: problem } : { prompts: problem });
      return;
    }
    setErrors({});
    start.mutate(
      { custom: { title: title.trim(), prompts: prompts.map((p) => p.trim()) } },
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
          placeholder="Seven evenings, no phones"
          error={errors.title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Field
          label="Number of days"
          hint={`${MIN_CUSTOM_DAYS} to ${MAX_CUSTOM_DAYS}. One line for each.`}
          inputMode="numeric"
          value={daysText}
          onChange={(e) => onDays(e.target.value)}
          onBlur={() => setCount(Number(daysText) || DEFAULT_CUSTOM_DAYS)}
          className="w-28"
        />
        <ol className="m-0 flex list-none flex-col gap-3 p-0">
          {prompts.map((p, i) => (
            <li key={i} className="flex items-end gap-2">
              <div className="min-w-0 grow">
                <Field
                  label={`Day ${i + 1}`}
                  value={p}
                  maxLength={200}
                  placeholder="One small thing to do together"
                  onChange={(e) =>
                    setPrompts((all) => all.map((x, j) => (j === i ? e.target.value : x)))
                  }
                />
              </div>
              <button
                type="button"
                aria-label={`Remove day ${i + 1}`}
                disabled={prompts.length <= MIN_CUSTOM_DAYS}
                onClick={() => removeAt(i)}
                className="press flex h-[52px] w-11 shrink-0 items-center justify-center text-stone disabled:opacity-30"
              >
                <Icon name="x" size={18} />
              </button>
            </li>
          ))}
        </ol>
        {errors.prompts ? (
          <div role="alert" className="text-[13px] text-red">
            {errors.prompts}
          </div>
        ) : null}
        <div>
          <Button
            variant="text"
            icon="plus"
            disabled={prompts.length >= MAX_CUSTOM_DAYS}
            onClick={() => setCount(prompts.length + 1)}
          >
            Add a day
          </Button>
        </div>
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
