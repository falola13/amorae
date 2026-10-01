"use client";

import { useState } from "react";

import { Button, Para, Sheet } from "@/components/ui/kit";
import { useStartChallenge } from "@/features/together/hooks";
import type { ChallengeKind, ChallengeTemplate } from "@/lib/api/types";
import { today } from "@/lib/today";
import { startConfirm, startProblem, startedOnToSend } from "../challenges";
import { KindChips, StartsChips, tomorrow, type StartWhen } from "./start-options";

/** Confirms before a challenge begins: who is taking part, and when it starts. */
export function StartChallengeSheet({
  template,
  onClose,
  onStarted,
}: {
  template: ChallengeTemplate | null;
  onClose: () => void;
  /** Called once it has started (or been saved to send when back online). */
  onStarted?: () => void;
}) {
  if (!template) return null;
  // Mounted only while a template is picked, so each opening starts from the defaults.
  return <Body template={template} onClose={onClose} onStarted={onStarted} />;
}

function Body({
  template,
  onClose,
  onStarted,
}: {
  template: ChallengeTemplate;
  onClose: () => void;
  onStarted?: () => void;
}) {
  const start = useStartChallenge();
  const [kind, setKind] = useState<ChallengeKind>("together");
  const [when, setWhen] = useState<StartWhen>("today");
  const [date, setDate] = useState(tomorrow);
  const now = today();
  const problem = when === "pick" ? startProblem(date, now) : null;
  const startedOn = when === "pick" ? startedOnToSend(date, now) : undefined;
  const done = () => {
    onClose();
    onStarted?.();
  };
  return (
    <Sheet open onClose={onClose} title={`Start ${template.title}?`} labelledBy="ch-start-h">
      <div className="flex flex-col gap-4">
        <KindChips value={kind} onChange={setKind} />
        <StartsChips
          when={when}
          onWhen={setWhen}
          date={date}
          onDate={setDate}
          error={problem ?? undefined}
        />
      </div>
      <Para>{startConfirm(kind, startedOn)}</Para>
      <div className="flex flex-col gap-1">
        <Button
          loading={start.isPending}
          disabled={problem !== null}
          onClick={() =>
            start.mutate(
              {
                template: template.key,
                ...(kind === "mine" ? { kind } : {}),
                ...(startedOn ? { started_on: startedOn } : {}),
              },
              { onSuccess: done, onQueued: done },
            )
          }
        >
          Start it
        </Button>
        <Button variant="text" onClick={onClose}>
          Not yet
        </Button>
      </div>
    </Sheet>
  );
}
