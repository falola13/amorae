'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { SafeTop } from '@/components/Screen';
import { Button, Field, LinkButton, Para, Title, TopBar } from '@/components/ui';
import { Bottom, ShowButton } from '@/components/OnboardingBits';

const schema = z.object({ email: z.string().trim().email('Add the part after the @, like .com'), password: z.string().min(1, 'Enter your password.') });
type Form = z.infer<typeof schema>;

export default function LogIn() {
  const router = useRouter(); const qc = useQueryClient();
  const [show, setShow] = useState(false);
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema), defaultValues: { email: 'femi@example.com' } });
  const onSubmit = async () => { await api.login(); await qc.invalidateQueries(); router.replace('/'); };
  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex grow flex-col">
      <SafeTop />
      <TopBar back="Back" backHref="/welcome" />
      <div className="flex grow flex-col gap-7 px-6 pt-3">
        <div className="flex flex-col gap-2"><Title>Welcome back</Title><Para>Everything is where the two of you left it.</Para></div>
        <div className="flex flex-col gap-[18px]">
          <Field label="Email" type="email" autoComplete="email" inputMode="email" error={errors.email?.message} {...register('email')} />
          <Field label="Password" type={show ? 'text' : 'password'} autoComplete="current-password" error={errors.password?.message} trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />} {...register('password')} />
          <Link href="#reset" className="press -mt-2.5 flex h-11 items-center self-start text-[15px] font-semibold text-plum no-underline">Forgot your password?</Link>
        </div>
      </div>
      <Bottom>
        <Button type="submit" loading={isSubmitting}>Log in</Button>
        <LinkButton href="/sign-up" variant="text">Create an account instead</LinkButton>
      </Bottom>
    </form>
  );
}
