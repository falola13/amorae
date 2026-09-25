"use client";

import { useRef, useState } from "react";
import {
  useDeleteMemory,
  useMemories,
  useRemovePhoto,
  useUploadPhoto,
} from "@/features/together/hooks";
import { MAX_PHOTO_BYTES } from "@/features/together/writes";
import { MemoryComposer } from "@/features/together/components/memory-composer";
import type { Memory } from "@/lib/api/types";
import { longDate, monthName } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import {
  Alert,
  BottomActions,
  Button,
  EmptyState,
  Micro,
  Ornament,
  Para,
  Sheet,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";

/**
 * A memory's picture, at the shape it was taken in.
 *
 * An earlier pass cropped every photo to a height chosen by its position, to
 * give the page an album's rhythm. That is the wrong way round: it crops a
 * couple's own photographs — faces included — to suit a layout. Letting each
 * one keep its aspect gives the same varied rhythm, except the variation is
 * theirs rather than imposed. The cap is only so a panorama cannot take over
 * the screen.
 *
 * The placeholder is now only the degraded case — a memory that says it has a
 * photo whose URL we could not build, which happens when Cloudinary is not
 * configured. It deliberately ignores the album height: there is nothing to
 * look at, and a tall empty block is a worse answer than a short one.
 */
/**
 * A month, as a running head rather than a floating label.
 *
 * The rule is what makes it a heading: a label alone in space is read as one
 * more line of text, and an archive that runs for years needs the eye to catch
 * where one month stops. Print has done it this way for five hundred years.
 */
function RunningHead({ children }: { children: string }) {
  return (
    <div className="flex items-center gap-3 pt-1">
      <Micro>{children}</Micro>
      <span className="h-px grow bg-line" />
    </div>
  );
}

function Photo({ label, src }: { label: string; src?: string }) {
  if (src) {
    return (
      // Cloudinary already delivers this with f_auto,q_auto from its own CDN.
      // Sending it through Vercel's optimiser too would add a hop and, on
      // Hobby, a bill, to re-do work that is already done.
      // eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN
      <img
        src={src}
        alt={label}
        loading="lazy"
        className="max-h-[440px] w-full rounded-btn bg-photo object-cover"
      />
    );
  }
  return (
    <div
      role="img"
      aria-label={label}
      className="flex h-[132px] items-center justify-center gap-2 rounded-btn bg-photo text-[13px] font-semibold text-stone"
    >
      <Icon name="image" size={18} />
      Photo
    </div>
  );
}

/**
 * What can be done to a moment once it is kept.
 *
 * None of this existed: a picture could be added at the instant a memory was
 * written and never afterwards, and nothing could be taken back. An upload
 * that failed — or a photo somebody would rather not keep — had no way out
 * of the app at all, which is not a reasonable thing to do with somebody
 * else's pictures.
 *
 * It is a sheet rather than a row of buttons under each entry, because the
 * page is an album. Controls on every photograph would be the loudest thing
 * on a screen whose whole job is to be quiet.
 */
function MomentActions({ memory, onClose }: { memory: Memory | null; onClose: () => void }) {
  const upload = useUploadPhoto();
  const removePhoto = useRemovePhoto();
  const remove = useDeleteMemory();
  const fileInput = useRef<HTMLInputElement>(null);
  const [confirming, setConfirming] = useState(false);
  const [tooBig, setTooBig] = useState(false);

  const close = () => {
    setConfirming(false);
    setTooBig(false);
    onClose();
  };
  if (!memory) return null;

  const choose = (file: File | null) => {
    if (!file) return;
    if (file.size > MAX_PHOTO_BYTES) {
      setTooBig(true);
      return;
    }
    setTooBig(false);
    upload.mutate({ id: memory.id, photo: file }, { onSuccess: close });
  };
  const busy = upload.isPending || removePhoto.isPending;

  if (confirming) {
    return (
      <Sheet open onClose={close} title="Delete this moment?" labelledBy="del-mem-h">
        <Para>
          “{memory.title}” goes for both of you
          {memory.has_photo ? ", and the photo with it" : ""}. There is no undo.
        </Para>
        <div className="flex flex-col gap-1">
          <Button
            variant="secondary"
            className="border-red text-red"
            loading={remove.isPending}
            onClick={() => remove.mutate(memory.id, { onSuccess: close })}
          >
            Delete it
          </Button>
          <Button variant="text" onClick={() => setConfirming(false)}>
            Keep it
          </Button>
        </div>
      </Sheet>
    );
  }

  return (
    <Sheet open onClose={close} title={memory.title} labelledBy="mem-actions-h">
      <input
        ref={fileInput}
        type="file"
        accept="image/*"
        className="sr-only"
        onChange={(e) => choose(e.target.files?.[0] ?? null)}
      />
      {tooBig ? <Alert message="That photo is over 10MB. Pick a smaller one." /> : null}
      <div className="flex flex-col gap-1">
        <Button
          variant="secondary"
          icon="image"
          loading={upload.isPending}
          onClick={() => fileInput.current?.click()}
        >
          {memory.has_photo ? "Change the photo" : "Add a photo"}
        </Button>
        {memory.has_photo ? (
          <Button
            variant="text"
            disabled={busy}
            loading={removePhoto.isPending}
            onClick={() => removePhoto.mutate(memory.id, { onSuccess: close })}
          >
            Remove the photo
          </Button>
        ) : null}
        <Button
          variant="text"
          className="text-red"
          disabled={busy}
          onClick={() => setConfirming(true)}
        >
          Delete this moment
        </Button>
      </div>
    </Sheet>
  );
}

/**
 * The line under the end-mark. It states a fact — how much is kept, and how far
 * back it goes — because the bottom of an archive is worth a sentence and is
 * not worth a slogan.
 */
function keptLine(count: number, oldest: string) {
  const moments = count === 1 ? "One moment" : `${count} moments`;
  return `${moments} kept, back to ${monthName(oldest)} ${oldest.slice(0, 4)}.`;
}

export default function Memories() {
  const memories = useMemories();
  // The empty state already offers this, centred, with a line saying what
  // it is for. Showing the bar as well put two buttons for the same thing
  // on one screen, one under the other. The bar is for when there is a
  // list to add to.
  const hasAny = (memories.data?.length ?? 0) > 0;
  const [open, setOpen] = useState(false);
  const [acting, setActing] = useState<Memory | null>(null);
  // Said once, at the top, after a save where the moment was kept and the
  // picture was not. It stays until it is dismissed: it names something still
  // to be done, and a toast that disappears would be exactly the wrong shape
  // for that.
  const [photoFailed, setPhotoFailed] = useState(false);
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2">
          <Title>Memories</Title>
        </div>
        {photoFailed ? (
          <div className="mt-3 flex flex-col gap-1.5">
            <Alert message="The moment was saved, but the photo didn’t upload." />
            <button
              type="button"
              onClick={() => setPhotoFailed(false)}
              className="press self-start px-1 text-[15px] font-semibold text-plum"
            >
              Got it
            </button>
          </div>
        ) : null}
        <QueryState queries={[memories]} loading={<Skeleton />}>
          {(memoriesData) => {
            if (memoriesData.length === 0) {
              return (
                <EmptyState
                  ghost="frames"
                  title="Your story starts here."
                  text="Save your first shared moment."
                  cta={
                    <Button icon="plus" onClick={() => setOpen(true)}>
                      Save a moment
                    </Button>
                  }
                />
              );
            }
            const sorted = [...memoriesData].sort((a, b) => b.date.localeCompare(a.date));
            return (
              <div className="mt-3.5 flex flex-col gap-[26px] pb-4">
                {sorted.map((m, idx) => {
                  const month = `${monthName(m.date)} ${m.date.slice(0, 4)}`;
                  const prev =
                    idx > 0
                      ? `${monthName(sorted[idx - 1].date)} ${sorted[idx - 1].date.slice(0, 4)}`
                      : null;
                  const showMonth = month !== prev;
                  return (
                    <div key={m.id} className="flex flex-col gap-3.5">
                      {showMonth ? <RunningHead>{month}</RunningHead> : null}
                      <article className="flex flex-col gap-1">
                        {m.has_photo ? <Photo label={m.title} src={m.photo_url} /> : null}
                        <div className="mt-2 flex items-start gap-2">
                          <div className="grow text-bodylg font-semibold tracking-[-0.01em]">
                            {m.title}
                          </div>
                          <button
                            type="button"
                            onClick={() => setActing(m)}
                            aria-label={`What to do with ${m.title}`}
                            className="press -mr-2.5 -mt-1.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-btn text-stone"
                          >
                            <Icon name="more" size={20} />
                          </button>
                        </div>
                        <div className="text-support text-stone">
                          {longDate(m.date)} {m.date.slice(0, 4)}
                          {m.location ? ` · ${m.location}` : ""}
                        </div>
                        {m.note ? (
                          <p
                            className="m-0 mt-0.5 text-[15px] leading-[1.55] text-stone"
                            data-selectable
                          >
                            {m.note}
                          </p>
                        ) : null}
                      </article>
                    </div>
                  );
                })}
                <Ornament caption={keptLine(sorted.length, sorted[sorted.length - 1].date)} />
              </div>
            );
          }}
        </QueryState>
      </Main>
      {hasAny ? (
        <BottomActions>
          <Button icon="plus" onClick={() => setOpen(true)}>
            Save a moment from today
          </Button>
        </BottomActions>
      ) : null}
      <MemoryComposer
        open={open}
        onClose={() => setOpen(false)}
        onPhotoFailed={() => setPhotoFailed(true)}
      />
      <MomentActions memory={acting} onClose={() => setActing(null)} />
    </>
  );
}
