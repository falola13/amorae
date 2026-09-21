interface AlertProps {
  message: string;
  variant?: "error" | "success";
}

// aria-live="polite" so screen reader users hear form errors/success as
// they appear, without the alert stealing focus the way role="alert" would.
export function Alert({ message, variant = "error" }: AlertProps) {
  const styles =
    variant === "error"
      ? "border-danger/30 bg-danger-bg text-danger"
      : "border-success/30 bg-success-bg text-success";

  return (
    <div role="status" aria-live="polite" className={`rounded-md border px-3 py-2 text-sm ${styles}`}>
      {message}
    </div>
  );
}
