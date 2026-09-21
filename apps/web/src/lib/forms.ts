import { isApiError } from "@/lib/api/errors";

// Shared shape for every useActionState-driven form in the app (login,
// register, profile). Keeping it in one place means every form renders
// errors the same way instead of each feature inventing its own.
//
// `values` echoes what the user typed back into the form. React resets an
// uncontrolled form after its action returns, so without this a failed
// submit would wipe every field. Never put a password in it: this state is
// serialised into the page.
export type FormState = {
  status: "idle" | "error" | "success";
  message?: string;
  fields?: Record<string, string>;
  values?: Record<string, string>;
};

export const idleFormState: FormState = { status: "idle" };

export function toFormState(error: unknown, values?: Record<string, string>): FormState {
  if (isApiError(error)) {
    return { status: "error", message: error.message, fields: error.fields, values };
  }
  return { status: "error", message: "Something went wrong. Please try again.", values };
}
