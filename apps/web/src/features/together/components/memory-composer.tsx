"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useEffect, useMemo, useRef, useState } from "react";
import type { z } from "zod";

import { useAddMemoryWithPhoto } from "@/features/together/hooks";
import { MAX_PHOTO_BYTES } from "@/features/together/writes";
import { memorySchema } from "@/lib/api/schemas";
import { iso } from "@/lib/dates";
import { today } from "@/lib/today";
import { BareInput, BareTextarea, Button, Sheet } from "@/components/ui/kit";

type MemoryFormInput = z.infer<typeof memorySchema>;

export function MemoryComposer({
  open,
  onClose,
  onPhotoFailed,
}: {
  open: boolean;
  onClose: () => void;
  /** The moment was kept and the picture was not. */
  onPhotoFailed: () => void;
}) {
  const add = useAddMemoryWithPhoto();
  // The file itself, not a form field: it never goes to our API, only to
  // Cloudinary, and only after the memory it belongs to exists.
  const [photo, setPhoto] = useState<File | null>(null);
  const [tooBig, setTooBig] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  // createObjectURL in the markup would mint a new URL on every render and
  // never release one, which is a leak that grows while somebody types.
  const preview = useMemo(() => (photo ? URL.createObjectURL(photo) : null), [photo]);
  useEffect(
    () => () => {
      if (preview) URL.revokeObjectURL(preview);
    },
    [preview],
  );
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<MemoryFormInput>({
    resolver: zodResolver(memorySchema),
    defaultValues: { title: "", location: "", note: "" },
  });

  const close = () => {
    reset();
    setPhoto(null);
    setTooBig(false);
    onClose();
  };

  const choose = (file: File | null) => {
    setTooBig(!!file && file.size > MAX_PHOTO_BYTES);
    setPhoto(file && file.size <= MAX_PHOTO_BYTES ? file : null);
  };
  const onSubmit = (v: MemoryFormInput) =>
    add.mutate(
      {
        memory: {
          title: v.title,
          date: iso(today()),
          location: v.location || undefined,
          note: v.note || undefined,
          has_photo: false,
        },
        photo,
      },
      {
        // Closes either way: the moment is saved by now, so leaving the
        // sheet open would invite a second copy of it.
        onSuccess: (result) => {
          close();
          if (result.photoFailed) onPhotoFailed();
        },
      },
    );

  return (
    <Sheet open={open} onClose={close} title="Save a moment from today." labelledBy="m-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <BareInput
          label="What happened"
          autoFocus
          placeholder="Our first Amorae date night"
          className="h-11 text-[22px] font-semibold tracking-[-0.02em]"
          error={errors.title?.message}
          {...register("title")}
        />
        <BareInput
          label="Where, optional"
          placeholder="Lekki"
          className="h-10 text-body"
          error={errors.location?.message}
          {...register("location")}
        />
        <BareTextarea
          label="A short note, optional"
          rows={2}
          placeholder="We stayed until they stacked the chairs."
          className="min-h-[56px] text-body"
          error={errors.note?.message}
          {...register("note")}
        />
        {/* This button sat here doing nothing for a long time, because there
            was nowhere for a file to go. There is now: the browser uploads
            straight to Cloudinary with a signature from our API, so a 10MB
            photo never travels through a free container host. */}
        <input
          ref={fileInput}
          type="file"
          accept="image/*"
          className="sr-only"
          onChange={(e) => choose(e.target.files?.[0] ?? null)}
        />
        {photo ? (
          <div className="flex items-center gap-3 rounded-input border border-line px-4 py-2.5">
            {/* A blob: URL for the file being chosen — next/image cannot
                optimise one, and would not want to. */}
            {/* eslint-disable-next-line @next/next/no-img-element -- local blob: preview */}
            <img
              src={preview ?? ""}
              alt=""
              className="h-11 w-11 shrink-0 rounded-btn object-cover"
            />
            <span className="grow truncate text-support text-stone">{photo.name}</span>
            <button
              type="button"
              onClick={() => choose(null)}
              className="press h-9 px-2 text-[15px] font-semibold text-plum"
            >
              Remove
            </button>
          </div>
        ) : null}
        {tooBig ? (
          <div role="alert" className="text-[13px] text-red">
            That photo is over 10MB. Pick a smaller one.
          </div>
        ) : null}
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={add.isPending}>
            {add.isPending && photo ? "Saving and uploading" : "Save this moment"}
          </Button>
          {photo ? null : (
            <Button
              type="button"
              variant="text"
              icon="image"
              onClick={() => fileInput.current?.click()}
            >
              Add a photo
            </Button>
          )}
        </div>
      </form>
    </Sheet>
  );
}
