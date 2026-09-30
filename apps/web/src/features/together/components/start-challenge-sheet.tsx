"use client";

import { Button, Para, Sheet } from "@/components/ui/kit";
import { useStartChallenge } from "@/features/together/hooks";
import type { ChallengeTemplate } from "@/lib/api/types";

/** Confirms before a challenge begins, since it starts for both of you at once. */
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
  const start = useStartChallenge();
  if (!template) return null;
  const done = () => {
    onClose();
    onStarted?.();
  };
  return (
    <Sheet open onClose={onClose} title={`Start ${template.title}?`} labelledBy="ch-start-h">
      <Para>Day 1 opens today for both of you.</Para>
      <div className="flex flex-col gap-1">
        <Button
          loading={start.isPending}
          onClick={() =>
            start.mutate({ template: template.key }, { onSuccess: done, onQueued: done })
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
