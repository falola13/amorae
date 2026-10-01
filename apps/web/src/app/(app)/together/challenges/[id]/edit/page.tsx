"use client";

import { useParams, useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import {
  BareTextarea,
  Button,
  ComposeBar,
  Field,
  Micro,
  Para,
  Sheet,
  Skeleton,
  TopBar,
} from "@/components/ui/kit";
import { useCouple } from "@/features/couple/hooks";
import {
  MAX_CUSTOM_LINE,
  applyLine,
  isOver,
  isScheduled,
  planChanged,
  planFromText,
  planProblem,
  planToText,
  startBounds,
  startProblem,
} from "@/features/together/challenges";
import { KindChips } from "@/features/together/components/start-options";
import { useChallengeById, useChallengePlan, useUpdateChallenge } from "@/features/together/hooks";
import { isApiError } from "@/lib/api/errors";
import type { Challenge, ChallengeKind, ChallengePatch } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

/** Change a challenge: its name, when it starts, who is in it, and the days themselves. */
export default function EditChallengePage() {
  const { id } = useParams<{ id: string }>();
  const ch = useChallengeById(id);
  return (
    <QueryState
      queries={[ch]}
      frame={inPage}
      loading={
        <>
          <TopBar back="Back" backHref={routes.challenge(id)} />
          <Main>
            <Skeleton />
          </Main>
        </>
      }
    >
      {(c) =>
        c.can_edit && !isOver(c.status) ? (
          <EditForm key={c.id} c={c} />
        ) : (
          <>
            <TopBar back="Back" backHref={routes.challenge(c.id)} />
            <Main>
              <Para className="pt-2">This one can’t be changed.</Para>
            </Main>
          </>
        )
      }
    </QueryState>
  );
}

type Errors = { title?: string; date?: string; plan?: string; form?: string };

const GENERIC = "That didn’t go through. Try again in a moment.";

function EditForm({ c }: { c: Challenge }) {
  const router = useRouter();
  const couple = useCouple();
  const update = useUpdateChallenge();
  const savePlan = useChallengePlan();
  const meId = couple.data?.me.id;
  const isCreator = !c.created_by || c.created_by === meId;
  const scheduled = isScheduled(c);
  const initialPlan = c.days.map((d) => d.text);
  // "Today" in the plan: days before it keep their words when you rewrite from today on.
  const fromToday = Math.max(1, c.today_n);

  const [title, setTitle] = useState(c.title);
  const [startedOn, setStartedOn] = useState(c.started_on);
  const [kind, setKind] = useState<ChallengeKind>(c.kind);
  const [kindLocked, setKindLocked] = useState<string | null>(null);
  const [text, setText] = useState(planToText(initialPlan));
  const [line, setLine] = useState("");
  const [saving, setSaving] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const [errors, setErrors] = useState<Errors>({});

  const plan = planFromText(text);
  const { min, max } = startBounds(today());
  const startMoved = scheduled && startedOn !== c.started_on;
  const planDirty = planChanged(initialPlan, plan);
  const dirty = title.trim() !== c.title || startMoved || kind !== c.kind || planDirty;
  const back = () => router.replace(routes.challenge(c.id));
  const cancel = () => (dirty ? setLeaving(true) : back());

  const apply = (from: number) => {
    setText(planToText(applyLine(plan, line, from)));
    setErrors((e) => ({ ...e, plan: undefined }));
  };

  const submit = async (e?: FormEvent) => {
    e?.preventDefault();
    if (saving) return;
    const next: Errors = {};
    if (!title.trim()) next.title = "Give it a name.";
    if (startMoved) {
      const p = startProblem(startedOn, today());
      if (p) next.date = p;
    }
    if (planDirty) {
      const p = planProblem(plan);
      if (p) next.plan = p;
    }
    setErrors(next);
    if (Object.keys(next).length > 0) return;
    if (!dirty) return back();

    const patch: ChallengePatch = {};
    if (title.trim() !== c.title) patch.title = title.trim();
    if (startMoved) patch.started_on = startedOn;
    if (kind !== c.kind) patch.kind = kind;

    setSaving(true);
    try {
      if (Object.keys(patch).length > 0) await update.mutateAsync({ id: c.id, patch });
    } catch (err) {
      setSaving(false);
      if (!isApiError(err)) return setErrors({ form: GENERIC });
      const f = err.fields ?? {};
      if (err.code === "challenge_kind_locked") {
        setKind(c.kind);
        setKindLocked(err.message);
        return setErrors({});
      }
      const date = f["started_on"] ?? (err.code === "challenge_begun" ? err.message : undefined);
      return setErrors({
        title: f["title"],
        date,
        form: date || Object.keys(f).length ? undefined : err.message,
      });
    }
    try {
      if (planDirty) await savePlan.mutateAsync({ id: c.id, prompts: plan });
    } catch (err) {
      setSaving(false);
      if (!isApiError(err)) return setErrors({ form: GENERIC });
      const inline = err.fields?.["prompts"] ?? (err.code === "day_has_marks" ? err.message : "");
      return setErrors(inline ? { plan: inline } : { form: err.message });
    }
    setSaving(false);
    back();
  };

  return (
    <>
      <ComposeBar
        onCancel={cancel}
        label="Edit challenge"
        done="Save"
        onDone={() => void submit()}
        busy={saving}
      />
      <form
        onSubmit={(e) => void submit(e)}
        className="flex grow flex-col gap-5 px-6 pb-8 pt-5"
        noValidate
      >
        <Field
          label="Title"
          value={title}
          maxLength={80}
          error={errors.title}
          onChange={(e) => setTitle(e.target.value)}
        />

        {scheduled ? (
          <Field
            label="Start date"
            type="date"
            min={min}
            max={max}
            value={startedOn}
            error={errors.date}
            onChange={(e) => setStartedOn(e.target.value)}
            className="w-52"
          />
        ) : (
          <div className="flex flex-col gap-1">
            <Micro>Start date</Micro>
            <Para size="support" className="text-stone">
              It’s begun — the start can’t move now.
            </Para>
          </div>
        )}

        {isCreator ? (
          <div className="flex flex-col gap-1.5">
            <KindChips value={kind} onChange={setKind} disabled={kindLocked !== null} />
            {kindLocked ? (
              <div role="alert" className="text-[13px] text-red">
                {kindLocked}
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="flex flex-col gap-1.5">
          <BareTextarea
            label="Different each day"
            rows={10}
            value={text}
            className="min-h-[220px] rounded-input border border-edge bg-surface px-4 py-3 text-body leading-[1.6]"
            onChange={(e) => {
              setText(e.target.value);
              setErrors((x) => ({ ...x, plan: undefined }));
            }}
          />
          <Para size="support" className="text-stone">
            {plan.length} {plan.length === 1 ? "day" : "days"} · one line per day
          </Para>
          {errors.plan ? (
            <div role="alert" className="text-[13px] text-red">
              {errors.plan}
            </div>
          ) : null}
        </div>

        <div className="flex flex-col gap-2 rounded-card border border-line bg-surface px-4 py-4">
          <Field
            label="Make every day the same"
            value={line}
            maxLength={MAX_CUSTOM_LINE}
            placeholder="No soda — fruit only"
            onChange={(e) => setLine(e.target.value)}
          />
          <div className="flex flex-wrap gap-1">
            <Button
              variant="secondary"
              type="button"
              disabled={!line.trim() || plan.length === 0}
              onClick={() => apply(1)}
            >
              Apply to every day
            </Button>
            {fromToday > 1 ? (
              <Button
                variant="text"
                type="button"
                disabled={!line.trim() || plan.length === 0}
                onClick={() => apply(fromToday)}
              >
                Apply from today on
              </Button>
            ) : null}
          </div>
        </div>

        {errors.form ? (
          <div role="alert" className="text-[13px] text-red">
            {errors.form}
          </div>
        ) : null}
        <button type="submit" className="sr-only">
          Save
        </button>
      </form>

      <Sheet
        open={leaving}
        onClose={() => setLeaving(false)}
        title="Leave without saving?"
        labelledBy="ch-edit-leave-h"
      >
        <Para>Your changes here won&rsquo;t be kept.</Para>
        <div className="flex flex-col gap-1">
          <Button variant="secondary" onClick={back}>
            Leave
          </Button>
          <Button variant="text" onClick={() => setLeaving(false)}>
            Keep editing
          </Button>
        </div>
      </Sheet>
    </>
  );
}
