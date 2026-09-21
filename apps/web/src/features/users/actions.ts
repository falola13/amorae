"use server";

import { revalidatePath } from "next/cache";

import { toFormState, type FormState } from "@/lib/forms";
import { updateProfile } from "./api";

export async function updateProfileAction(_prevState: FormState, formData: FormData): Promise<FormState> {
  const displayName = String(formData.get("display_name") ?? "");

  try {
    await updateProfile(displayName);
  } catch (error) {
    return toFormState(error, { display_name: displayName });
  }

  revalidatePath("/dashboard");
  return { status: "success", message: "Profile updated." };
}
