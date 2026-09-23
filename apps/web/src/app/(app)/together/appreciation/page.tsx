"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import type { z } from "zod";

import { useCouple } from "@/features/couple/hooks";
import {
  useAppreciations,
  useSendAppreciation,
  useUndoAppreciation,
} from "@/features/together/hooks";
import { appreciationSchema } from "@/lib/api/schemas";
import { iso, relativeDay } from "@/lib/dates";
import { useOnline } from "@/lib/query/offline";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";
import { Icon } from "@/components/icons";
import { BareTextarea, ComposeBar, Micro, Toast, cx } from "@/components/ui/kit";

type AppreciationFormInput = z.infer<typeof appreciationSchema>;

// How long "Undo" stays on screen after sending. The API accepts the undo a
// little longer (docs/API.md) so a slow connection doesn't lose it.
const UNDO_VISIBLE_MS = 5000;

export default function Appreciation() {
  const router = useRouter();
  const couple = useCouple();
  const list = useAppreciations();
  const send = useSendAppreciation();
  const undo = useUndoAppreciation();
  const online = useOnline();
  const [sentId, setSentId] = useState<string | null>(null);
  const me = couple.data?.me;
  const partner = couple.data?.partner?.display_name ?? "your partner";
  const todayIso = iso(today());
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<AppreciationFormInput>({
    resolver: zodResolver(appreciationSchema),
    defaultValues: { text: "" },
  });
  const onSubmit = (v: AppreciationFormInput) =>
    send.mutate(v.text, {
      onSuccess: (a) => {
        setSentId(a.id);
        reset();
        setTimeout(() => setSentId(null), UNDO_VISIBLE_MS);
      },
    });
  return (
    <>
      <ComposeBar
        cancelHref={routes.together}
        label={`To ${partner}`}
        done="Send"
        onDone={() => void handleSubmit(onSubmit)()}
        busy={send.isPending}
      />
      <form
        method="post"
        onSubmit={handleSubmit(onSubmit)}
        className="flex flex-col gap-2.5 px-6 pt-6"
        noValidate
      >
        <p aria-hidden="true" className="text-title leading-[1.2] text-stone">
          Something I appreciate about you&hellip;
        </p>
        <BareTextarea
          id="ap"
          label="Something I appreciate about you"
          hideLabel
          rows={4}
          autoFocus
          placeholder="One true sentence is plenty."
          className="min-h-[128px] text-reading"
          error={errors.text?.message}
          {...register("text")}
        />
        <div className="flex items-center gap-2 text-support text-stone">
          <Icon name="bell" size={16} />
          {partner} will get one quiet notification.
        </div>
        <button type="submit" className="sr-only">
          Send
        </button>
      </form>
      <section className="mt-7 flex grow flex-col px-6">
        <Micro className="pb-0.5">Between you, lately</Micro>
        {(list.data ?? []).map((a, i, arr) => (
          <div
            key={a.id}
            className={cx(
              "flex flex-col gap-1 py-3.5",
              i < arr.length - 1 && "border-b border-line",
            )}
          >
            <div className="text-[13px] font-semibold text-stone">
              From {a.from_id === me?.id ? "you" : partner} &middot; {relativeDay(a.date, todayIso)}
            </div>
            <div className="text-body leading-[1.55]" data-selectable>
              {a.text}
            </div>
          </div>
        ))}
        {list.data && list.data.length === 0 ? (
          <div className="py-3.5 text-support text-stone">Nothing yet. Say the first one.</div>
        ) : null}
      </section>
      {/* Undo is online-only (see features/together/writes.ts), so it's hidden while offline rather than offered and failed. */}
      {sentId ? (
        <Toast
          message={`Sent to ${partner}`}
          action={online ? "Undo" : undefined}
          onAction={() => {
            undo.mutate(sentId);
            setSentId(null);
          }}
        />
      ) : null}
      <button type="button" className="sr-only" onClick={() => router.push(routes.together)}>
        Back to our space
      </button>
    </>
  );
}
