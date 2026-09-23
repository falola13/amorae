"use client";

import { Icon } from "@/components/icons";

export default function GlobalError({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col justify-center gap-3.5 bg-bg px-6">
      <Icon name="alert" size={28} strokeWidth={1.4} className="text-red" />
      <h1 className="m-0 text-[24px] font-semibold leading-tight tracking-[-0.02em]">
        Something went wrong on our side
      </h1>
      <p className="m-0 text-body text-stone">Nothing of yours is lost. Try again in a moment.</p>
      <button
        type="button"
        onClick={reset}
        className="press mt-2.5 flex h-[54px] items-center justify-center rounded-btn bg-plum text-[16px] font-semibold text-surface"
      >
        Try again
      </button>
    </div>
  );
}
