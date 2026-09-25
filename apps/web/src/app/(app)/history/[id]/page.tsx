"use client";

import { useParams } from "next/navigation";
import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import { setterSentence, prayerCount } from "@/features/prayers/derive";
import { useSaveReflection, useWeekById } from "@/features/prayers/hooks";
import { range } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { BareTextarea, Button, Micro, Para, Skeleton, Title, TopBar } from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";

export default function HistoryDetail() {
  const { id } = useParams<{ id: string }>();
  const week = useWeekById(id);
  const couple = useCouple();
  const save = useSaveReflection();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const partner = couple.data?.partner?.display_name ?? "your partner";

  return (
    <>
      <TopBar back="History" backHref={routes.history} />
      <QueryState
        queries={[week]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(w) => {
          const who = (pid: string) => {
            const me = w.my_completed.includes(pid),
              them = w.partner_completed.includes(pid);
            return me && them
              ? "Both of you prayed"
              : me
                ? "You prayed"
                : them
                  ? `${partner} prayed`
                  : "Not prayed yet";
          };
          return (
            <Main>
              <div className="pt-3">
                <Micro>{range(w.week_start, w.week_end)}</Micro>
              </div>
              <Title className="mt-2">{setterSentence(w, couple.data?.me.id, partner)}</Title>
              <Para className="mt-1.5">
                {prayerCount(w.points.length)}. You prayed {w.my_completed.length}, {partner} prayed{" "}
                {w.partner_completed.length}.
              </Para>
              <ol className="m-0 mt-[18px] list-none border-t border-line p-0">
                {w.points.map((p, i) => (
                  <li key={p.id} className="flex min-h-[64px] gap-4 border-b border-line py-3">
                    <span className="tabular w-[22px] pt-[3px] text-[13px] font-semibold text-stone">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <span className="flex flex-col gap-px">
                      <span className="text-bodylg font-semibold tracking-[-0.01em]">
                        {p.title}
                      </span>
                      <span className="text-support text-stone">{who(p.id)}</span>
                    </span>
                  </li>
                ))}
              </ol>
              <section className="mb-4 mt-5 flex flex-col gap-1.5 rounded-card border border-line bg-surface px-[18px] pb-2 pt-4">
                <Micro>Your reflection</Micro>
                {editing ? (
                  <>
                    <BareTextarea
                      label="Reflection"
                      hideLabel
                      value={draft}
                      onChange={(e) => setDraft(e.target.value)}
                      rows={3}
                      autoFocus
                      placeholder="A line or two about this week."
                      className="text-body"
                    />
                    <div className="flex gap-2">
                      <Button
                        variant="text"
                        className="w-auto px-0"
                        onClick={() => {
                          // Closes at once so an offline save can queue, but a failed
                          // one reopens with the draft — it used to vanish with the box.
                          save.mutate(
                            { weekId: w.id, text: draft.trim() },
                            { onError: () => setEditing(true) },
                          );
                          setEditing(false);
                        }}
                      >
                        Save
                      </Button>
                      <Button
                        variant="text"
                        className="w-auto px-3 text-stone"
                        onClick={() => setEditing(false)}
                      >
                        Cancel
                      </Button>
                    </div>
                  </>
                ) : (
                  <>
                    {w.reflection ? (
                      <p className="m-0 text-body" data-selectable>
                        {w.reflection}
                      </p>
                    ) : (
                      <Para>Nothing written yet.</Para>
                    )}
                    <Button
                      variant="text"
                      icon="pencil"
                      className="w-auto self-start px-0"
                      onClick={() => {
                        setDraft(w.reflection ?? "");
                        setEditing(true);
                      }}
                    >
                      {w.reflection ? "Edit reflection" : "Add a reflection"}
                    </Button>
                  </>
                )}
              </section>
            </Main>
          );
        }}
      </QueryState>
    </>
  );
}
