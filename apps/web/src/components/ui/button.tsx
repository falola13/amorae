import type { ButtonHTMLAttributes } from "react";

export type ButtonVariant = "primary" | "secondary";

const base =
  "inline-flex items-center justify-center rounded-md px-4 py-2.5 text-sm font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-50 disabled:pointer-events-none";

const variants: Record<ButtonVariant, string> = {
  primary: "bg-accent text-accent-fg hover:bg-accent-hover",
  secondary: "border border-border bg-bg-elevated text-fg hover:bg-bg",
};

// Exported so non-<button> elements that need the same look (e.g. the
// landing page's <Link> CTAs) can share it without wrapping next/link in a
// "polymorphic" abstraction.
export function buttonClassName(variant: ButtonVariant = "primary", className = ""): string {
  return `${base} ${variants[variant]} ${className}`.trim();
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
}

export function Button({ variant = "primary", className = "", ...props }: ButtonProps) {
  return <button className={buttonClassName(variant, className)} {...props} />;
}
