"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm, useWatch } from "react-hook-form";

import { Icon } from "@/components/icons";
import { SafeTop } from "@/components/layout/screen";
import {
  Button,
  LinkButton,
  Micro,
  Para,
  Title,
  TopBar,
  cx,
} from "@/components/ui/kit";
import { Bottom } from "@/components/ui/onboarding-bits";
import { useJoinCouple } from "@/features/couple/hooks";
import { isApiError } from "@/lib/api/errors";
import { joinSchema } from "@/lib/api/schemas";
import { formatInviteCode, inviteChars } from "@/lib/invite-code";
import { routes } from "@/lib/routes";

type Form = { code: string };

export default function JoinCouplePage() {
  const router = useRouter();
  const join = useJoinCouple();
  const {
    register,
    handleSubmit,
    setError,
    setValue,
    control,
    formState: { errors, isValid },
  } = useForm<Form>({
    resolver: zodResolver(joinSchema),
    mode: "onChange",
    defaultValues: { code: "" },
  });
  const code = useWatch({ control, name: "code" });

  const onSubmit = (v: Form) =>
    join.mutate(inviteChars(v.code), {
      onSuccess: () => router.push(routes.install),
      onError: (e) =>
        setError("code", {
          message: isApiError(e)
            ? (e.fields?.code ?? e.message)
            : "Something went wrong. Try again.",
        }),
    });

  return (
    <form
      method="post"
      onSubmit={handleSubmit(onSubmit)}
      className="flex grow flex-col"
      noValidate
    >
      <SafeTop />
      <TopBar back="Back" backHref={routes.couple()} />
      <div className="flex grow flex-col gap-7 px-6 pt-3">
        <div className="flex flex-col gap-2">
          <Micro>Step 2 of 3</Micro>
          <Title>Join your partner</Title>
          <Para>Enter the code from the invitation they sent you.</Para>
        </div>
        <div className="flex flex-col gap-2.5">
          <label
            htmlFor="code"
            className="text-[13px] font-semibold text-stone"
          >
            Invitation code
          </label>
          <input
            id="code"
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            inputMode="text"
            maxLength={7}
            placeholder="XXX-XXX"
            aria-invalid={!!errors.code}
            className={cx(
              "tabular h-16 w-full rounded-input border-[1.5px] bg-surface px-4 text-center text-[24px] font-semibold uppercase tracking-[0.14em] text-ink",
              errors.code
                ? "border-red"
                : isValid
                  ? "border-plum"
                  : "border-edge",
            )}
            {...register("code")}
            value={code ?? ""}
            onChange={(e) => {
              setValue("code", formatInviteCode(e.target.value), { shouldValidate: true, shouldDirty: true });
            }}
          />
          {errors.code ? (
            <div
              role="alert"
              className="flex items-center gap-2 text-support text-red"
            >
              <Icon name="alert" size={18} />
              {errors.code.message}
            </div>
          ) : isValid ? (
            <div
              role="status"
              className="flex items-center gap-2 text-support font-medium text-green"
            >
              <Icon name="check" size={18} strokeWidth={2} />
              Looks right. Join when you&rsquo;re ready.
            </div>
          ) : null}
        </div>
      </div>
      <Bottom>
        <Button
          type="submit"
          disabled={!isValid || !code}
          loading={join.isPending}
        >
          Join
        </Button>
        <LinkButton href={routes.couple()} variant="text">
          Start a new space instead
        </LinkButton>
      </Bottom>
    </form>
  );
}
