"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { Alert, Button, Para, Sheet } from "@/components/ui/kit";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { routes } from "@/lib/routes";
import { useLeaveCouple } from "../hooks";

/**
 * Leaving ends the space for both people, so the sheet says so before it says
 * anything else — the most common misreading of a "leave" button is that it
 * only removes you.
 *
 * There is no "type LEAVE to confirm" here, unlike deleting an account: this
 * is undoable for thirty days in the only sense that matters, because
 * everything written is still there to read and download.
 */
export function LeaveCoupleSheet({
  open,
  onClose,
  partner,
}: {
  open: boolean;
  onClose: () => void;
  partner?: string;
}) {
  const leave = useLeaveCouple();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);

  const close = () => {
    setError(null);
    onClose();
  };

  const confirm = () =>
    leave.mutate(undefined, {
      onSuccess: (ended) => {
        onClose();
        // Straight to what is left of it, rather than dropping someone back
        // into onboarding with no sign of where their space went.
        router.replace(ended.length > 0 ? routes.settingsPastSpace : routes.home);
      },
      onError: (e) => setError(isApiError(e) ? e.message : GENERIC_ERROR_MESSAGE),
    });

  return (
    <Sheet open={open} onClose={close} title="End this space?" labelledBy="leave-h">
      <Para>
        This ends it for {partner ?? "your partner"} too. Neither of you keeps what you wrote here
        without the other.
      </Para>
      <Para>
        For 30 days afterwards you can both still read everything and download a copy. After that it
        is deleted. Your account stays, and you can start a new space whenever you want.
      </Para>
      {error ? <Alert message={error} /> : null}
      <div className="flex flex-col gap-1">
        <Button
          variant="secondary"
          className="border-red text-red"
          loading={leave.isPending}
          onClick={confirm}
        >
          End our space
        </Button>
        <Button variant="text" onClick={close}>
          Keep it
        </Button>
      </div>
    </Sheet>
  );
}
