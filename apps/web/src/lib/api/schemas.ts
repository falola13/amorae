import { z } from "zod";

// Zod schemas shared by React Hook Form (client validation) and the Server
// Actions (they re-parse, so a tampered form still meets the same rules).
// Messages say how to fix the problem, per the design's error copy.
export const emailSchema = z.string().trim().email("Add the part after the @, like .com");
// The password rule, identical to the Go API's (auth.passwordProblem):
// - at least 10 *characters*, counted as a person would (Array.from counts
//   code points, so an emoji is one; .min() would count UTF-16 units and let
//   5 emoji through as "10");
// - at most 72 *bytes*, because bcrypt ignores everything after that and a
//   password must never be silently truncated. Accents and emoji take 2-4 bytes.
const MIN_PASSWORD_CHARS = 10;
const MAX_PASSWORD_BYTES = 72;
export const passwordSchema = z
  .string()
  .refine(
    (s) => Array.from(s).length >= MIN_PASSWORD_CHARS,
    `Make it at least ${MIN_PASSWORD_CHARS} characters.`,
  )
  .refine(
    (s) => new TextEncoder().encode(s).length <= MAX_PASSWORD_BYTES,
    "That’s a little long. Try a shorter password.",
  );

export const loginSchema = z.object({
  email: emailSchema,
  password: z.string().min(1, "Enter your password."),
  next: z.string().optional(),
});
export const registerSchema = z.object({
  display_name: z
    .string()
    .trim()
    .min(1, "Tell us what your partner should call you.")
    .max(50, "Keep it under 50 characters."),
  email: emailSchema,
  password: passwordSchema,
  // FR-AUTH-010 and 011, checked again by the API. The faith box is separate
  // and optional (FR-AUTH-012): agreeing to the Terms never implies it.
  age_confirmed: z.boolean().refine((v) => v, "You need to be 18 or older to use Amorae."),
  accepted_terms: z
    .boolean()
    .refine((v) => v, "Agree to the Terms and Privacy Policy to continue."),
  faith_consent: z.boolean(),
});
export const forgotSchema = z.object({ email: emailSchema });
export type ForgotInput = z.infer<typeof forgotSchema>;
export const resetSchema = z.object({ new_password: passwordSchema });
export type ResetInput = z.infer<typeof resetSchema>;
export const joinSchema = z.object({
  code: z
    .string()
    .trim()
    .regex(/^[A-Z0-9]{3}-[A-Z0-9]{3}$/i, "Codes look like XXX-XXX."),
});
// Email isn't part of the profile form: changing it needs the current
// password, so it has its own form and endpoint (changeEmailSchema).
export const profileSchema = z.object({
  display_name: registerSchema.shape.display_name,
  timezone: z.string().min(1),
});
export const changeEmailSchema = z.object({
  email: emailSchema,
  current_password: z.string().min(1, "Enter your current password."),
});
export type ChangeEmailInput = z.infer<typeof changeEmailSchema>;
export const changePasswordSchema = z.object({
  current_password: z.string().min(1, "Enter your current password."),
  new_password: passwordSchema,
});
export type ChangePasswordInput = z.infer<typeof changePasswordSchema>;

// The word a person types to confirm deleting their account. The Go API
// applies the same rule (auth.isDeleteConfirmation): case and surrounding
// spaces don't matter, the word does.
export const DELETE_CONFIRMATION = "delete";
export const isDeleteConfirmation = (s: string) => s.trim().toLowerCase() === DELETE_CONFIRMATION;
export const deleteAccountSchema = z.object({
  confirm: z.string().refine(isDeleteConfirmation, `Type “${DELETE_CONFIRMATION}” to confirm.`),
  current_password: z.string().min(1, "Enter your current password."),
});
export type DeleteAccountInput = z.infer<typeof deleteAccountSchema>;

// The role rule matches couples.ValidateRole in Go: 1–32 characters.
export const coupleSchema = z.object({
  name: z.string().trim().min(1, "Give your space a name.").max(60, "Keep it under 60 characters."),
  relationship_start_date: z
    .string()
    .regex(/^(\d{4}-\d{2}-\d{2})?$/, "Pick a date.")
    .optional(),
  role: z.string().trim().max(32, "Keep it under 32 characters.").optional(),
});
export type CoupleInput = z.infer<typeof coupleSchema>;

export const prayerPointSchema = z.object({
  title: z.string().trim().min(1, "Give this prayer a title.").max(80, "Keep the title short."),
  text: z.string().trim().max(1000, "Keep it under 1000 characters."),
  scripture: z.string().trim().max(60).optional(),
});

export const eventSchema = z.object({
  title: z.string().trim().min(1, "What would you like to do together?").max(80),
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Pick a date."),
  start_time: z.string().optional(),
  end_time: z.string().optional(),
  location: z.string().trim().max(120).optional(),
  reminder: z.string().trim().max(40).optional(),
  notes: z.string().trim().max(1000).optional(),
});

export const goalSchema = z.object({
  title: z.string().trim().min(1, "What would you like to build together?").max(80),
  why: z.string().trim().max(200).optional(),
  target: z.number().positive("Set a target above zero."),
  unit: z.enum(["naira", "count"]),
  start_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  end_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
});

export const journalSchema = z.object({
  tag: z.enum(["Gratitude", "Reflection", "Memory", "Appreciation", "Plans"]),
  text: z.string().trim().min(1, "Write a line first.").max(2000),
});
export const appreciationSchema = z.object({
  text: z.string().trim().min(1, "One true sentence is plenty.").max(500),
});
export const memorySchema = z.object({
  title: z.string().trim().min(1, "What happened?").max(80),
  location: z.string().trim().max(80).optional(),
  note: z.string().trim().max(500).optional(),
});
export const milestoneSchema = z.object({
  title: z.string().trim().min(1, "What is it?").max(80),
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Pick a date."),
  sub: z.string().trim().max(80).optional(),
});
export const progressSchema = z.object({
  amount: z.number().positive("Add an amount above zero."),
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
export type ProfileInput = z.infer<typeof profileSchema>;
export type EventInput = z.infer<typeof eventSchema>;
export type GoalInput = z.infer<typeof goalSchema>;
