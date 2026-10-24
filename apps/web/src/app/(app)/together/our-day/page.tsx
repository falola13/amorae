"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useState } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { Button, LinkButton, LoadProblem, Para, Skeleton, TopBar } from "@/components/ui/kit";
import { CheersMoment, Confetti, OnTop } from "@/features/anniversary/celebrate";
import { yearsWord } from "@/features/anniversary/derive";
import { useAnniversary, useCheers } from "@/features/anniversary/hooks";
import { Opening, useOccasionTheme } from "@/features/anniversary/occasion";
import {
  Ideas,
  InYourWords,
  Keepsake,
  Letters,
  Moments,
  PastLetters,
  Prayer,
  Questions,
  WhereItBegan,
  YearInNumbers,
  YearSymbol,
} from "@/features/anniversary/sections";
import { useCouple } from "@/features/couple/hooks";
import { useAppreciations, useMemories } from "@/features/together/hooks";
import { isAbsence } from "@/lib/api/envelope";
import type { Anniversary, Couple } from "@/lib/api/types";
import { weekdayDate } from "@/lib/dates";
import { readableMessage } from "@/lib/query/client";
import { routes } from "@/lib/routes";

export default function OurDayPage() {
  return (
    <Suspense fallback={null}>
      <OurDay />
    </Suspense>
  );
}

/**
 * The anniversary's own page. In the days before, it counts down and is
 * where the letters are written; on the day, it is the day — a keepsake, a
 * glass to raise, the letters opened, the year they had, and things to do
 * with the rest of it. ?preview=1 shows the day ahead of time, everything but
 * the other one's letter, which the server keeps sealed regardless.
 */
function OurDay() {
  const preview = useSearchParams().get("preview") === "1";
  const anniversary = useAnniversary();
  const couple = useCouple();
  const a = anniversary.data;
  const theDay = a?.days_until === 0;
  const showDay = theDay || preview;
  useOccasionTheme(showDay && !!a);

  return (
    <>
      <TopBar backHref={routes.home} back="Home" />
      <Main className="pb-12">
        {anniversary.isPending || !couple.data ? (
          <div className="pt-6">
            <Skeleton lines={4} />
          </div>
        ) : anniversary.isError && isAbsence(anniversary.error) ? (
          <NoAnniversary />
        ) : anniversary.isError || !a ? (
          <LoadProblem
            icon="alert"
            tone="error"
            title="Your day didn’t load"
            text="Nothing is lost."
            onRetry={() => void anniversary.refetch()}
          />
        ) : (
          <Day a={a} couple={couple.data} showDay={showDay} preview={preview && !theDay} />
        )}
      </Main>
    </>
  );
}

function NoAnniversary() {
  return (
    <div className="flex grow flex-col justify-center gap-3.5 pb-12">
      <Icon name="heart" size={28} strokeWidth={1.4} className="text-plum" />
      <h1 className="m-0 text-[26px] font-semibold leading-tight tracking-[-0.02em]">
        Your anniversary will live here
      </h1>
      <Para size="lg">
        Once you’ve been together a year, this page becomes your day: letters to each other, and the
        year you had. Add the day you started if it isn’t set.
      </Para>
      <div className="mt-3">
        <LinkButton href={routes.settingsCouple} variant="secondary">
          Set the day you started
        </LinkButton>
      </div>
    </div>
  );
}

const arrivedKey = (on: string) => `amorae.anniversary.arrived.${on}`;

function Day({
  a,
  couple,
  showDay,
  preview,
}: {
  a: Anniversary;
  couple: Couple;
  showDay: boolean;
  preview: boolean;
}) {
  const memories = useMemories();
  const notes = useAppreciations();
  const cheers = useCheers();
  const [replay, setReplay] = useState(false);
  const [fire, setFire] = useState(0);
  const [toast, setToast] = useState<{ said: string | null } | null>(null);
  const me = couple.me;
  const partner = couple.partner?.display_name ?? "your partner";
  const first = partner.split(" ")[0];
  const nameOf = (id: string) => (id === me.id ? me.display_name : partner).split(" ")[0];

  // Arriving on the day: one handful of confetti, once per phone.
  useEffect(() => {
    if (!showDay || preview) return;
    let seen = false;
    try {
      seen = window.localStorage.getItem(arrivedKey(a.on)) === "1";
      window.localStorage.setItem(arrivedKey(a.on), "1");
    } catch {
      // Without storage it throws again next time, which is no harm today.
    }
    if (seen) return;
    const t = setTimeout(() => setFire((n) => n + 1), 450);
    return () => clearTimeout(t);
  }, [showDay, preview, a.on]);

  const raiseGlass = () => {
    setFire((n) => n + 1);
    if (preview) {
      setToast({ said: `In the preview, nothing is sent to ${first}.` });
      return;
    }
    setToast({ said: null });
    cheers.mutate(undefined, {
      onSuccess: () => setToast({ said: `${first} will see your glass raised.` }),
      onError: (e) => setToast({ said: readableMessage(e) }),
    });
  };
  const closeToast = useCallback(() => setToast(null), []);

  const mems = memories.data ?? [];
  const words = notes.data ?? [];

  return (
    <>
      {preview ? (
        <div className="mb-4 mt-1 flex items-start gap-3 rounded-[18px] border border-dashed border-edge px-4 py-3">
          <Icon name="moon" size={20} className="mt-0.5 shrink-0 text-stone" />
          <div className="flex grow flex-col">
            <span className="text-[15px] font-semibold">A preview of {weekdayDate(a.on)}</span>
            <span className="text-support text-stone">
              Only you are seeing this. {first}&rsquo;s letter stays sealed until the day.
            </span>
          </div>
        </div>
      ) : null}

      <Keepsake
        a={a}
        couple={couple}
        showDay={showDay}
        onCheers={raiseGlass}
        cheersBusy={cheers.isPending}
      />

      <Letters a={a} couple={couple} showDay={showDay} />

      {showDay ? (
        <>
          <YearInNumbers a={a} />
          <Moments a={a} memories={mems} />
          <InYourWords a={a} notes={words} nameOf={nameOf} />
          <YearSymbol years={a.years} />
          <Ideas on={a.on} />
          <Questions on={a.on} />
          <Prayer a={a} couple={couple} />
          <WhereItBegan started={couple.started_on} memories={mems} notes={words} nameOf={nameOf} />
        </>
      ) : (
        <>
          <YearSymbol years={a.years} />
          <Ideas on={a.on} />
        </>
      )}

      <PastLetters letters={a.past} nameOf={nameOf} />

      {showDay ? (
        <div className="mt-16 flex flex-col items-center gap-1 text-center">
          <p className="m-0 font-serif text-[30px] italic leading-tight text-plum-dark">
            {yearsWord(a.years)}, and counting.
          </p>
          <p className="m-0 mt-1 text-support text-stone">
            A threefold cord is not quickly broken. — Ecclesiastes 4:12
          </p>
          <div className="mt-6 flex w-full flex-col gap-1">
            <LinkButton href={routes.memories} icon="image">
              Keep a memory of today
            </LinkButton>
            <Button variant="text" onClick={() => setReplay(true)}>
              {preview ? "See the opening" : "Play the opening again"}
            </Button>
          </div>
        </div>
      ) : null}

      <Confetti fire={fire} />
      {toast ? <CheersMoment years={a.years} said={toast.said} onDone={closeToast} /> : null}
      {replay ? (
        <OnTop>
          <Opening on={a.on} years={a.years} onClose={() => setReplay(false)} />
        </OnTop>
      ) : null}
    </>
  );
}
