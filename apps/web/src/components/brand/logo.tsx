/* eslint-disable @next/next/no-img-element -- static SVG lockups: next/image
   would add nothing but an optimiser that refuses SVG anyway. */

// The mark is drawn inline with currentColor so it follows the theme
// (text-accent is plum in light mode and a soft plum-white in dark).
// Brand rule: below 32px use the heavier drawing, not a scaled-down master.
export function Mark({ size = 32, className = "" }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={size < 32 ? 2.4 : 1.7}
      strokeLinecap="round"
      aria-hidden="true"
      className={className}
    >
      <path d="M4.64 16.25A8.5 8.5 0 0 1 12 3.5a3.5 3.5 0 0 1 0 7" />
      <path d="M19.36 7.75A8.5 8.5 0 0 1 12 20.5a3.5 3.5 0 0 1 0-7" />
    </svg>
  );
}

// The wordmark is outlined in the SVG (no font needed), so the lockups ship
// as files: plum-on-light for the light theme, soft white for the dark one.
const lockups = {
  horizontal: { width: 244, height: 96 },
  stacked: { width: 239, height: 220 },
} as const;

export function Lockup({
  variant = "horizontal",
  height,
  className = "",
}: {
  variant?: keyof typeof lockups;
  height: number;
  className?: string;
}) {
  const { width: w, height: h } = lockups[variant];
  const width = Math.round((w / h) * height);
  const common = { width, height, decoding: "async" as const };

  // Only one image is ever displayed; the other is display:none, which also
  // hides it from screen readers, so "Amorae" is announced once.
  return (
    <span className={className}>
      <img src={`/brand/lockup-${variant}-plum.svg`} alt="Amorae" {...common} className="dark:hidden" />
      <img src={`/brand/lockup-${variant}-white.svg`} alt="Amorae" {...common} className="hidden dark:block" />
    </span>
  );
}
