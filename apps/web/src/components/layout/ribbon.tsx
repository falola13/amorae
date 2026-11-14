import { cx } from "@/components/ui/kit";

/**
 * The bookmark ribbon that marks the current tab. Plum is one of three cues
 * (with the ink label at 750 and the heavier icon stroke), so the current
 * place never rests on colour alone.
 *
 * `down`: hangs from the tab bar's hairline, swallowtail at the foot.
 * `in`: the same ribbon turned sideways, pointing in from the spine (rail and
 * side nav).
 */
export function Ribbon({ dir, className }: { dir: "down" | "in"; className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={cx(
        "pointer-events-none absolute bg-plum",
        dir === "down"
          ? "-top-px left-1/2 -ml-1.5 h-[15px] w-3 [clip-path:polygon(0_0,100%_0,100%_100%,50%_74%,0_100%)]"
          : "left-0 top-1/2 h-[14px] w-[13px] -translate-y-1/2 [clip-path:polygon(0_0,100%_0,70%_50%,100%_100%,0_100%)] lg:h-5 lg:w-3",
        className,
      )}
    />
  );
}
