'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { SafeTop } from '@/components/Screen';
import { Button, Field, Para, Title, TopBar } from '@/components/ui';
import { Bottom, ShowButton } from '@/components/OnboardingBits';

const schema = z.object({
  name: z.string().trim().min(1, 'Tell us what your partner should call you.'),
  email: z.string().trim().email('Add the part after the @, like .com'),
  password: z.string().min(10, 'Make it at least 10 characters.'),
});
type Form = z.infer<typeof schema>;

export default function SignUp() {
  const router = useRouter(); const qc = useQueryClient();
  const [show, setShow] = useState(false);
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<Form>({ resolver: zodResolver(schema) });
  const onSubmit = async (v: Form) => { await api.register(v.name, v.email); await qc.invalidateQueries(); router.push('/couple'); };
  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex grow flex-col">
      <SafeTop />
      <TopBar back="Back" backHref="/welcome" />
      <div className="flex grow flex-col gap-7 px-6 pt-3">
        <div className="flex flex-col gap-2"><Title>Create your account</Title><Para>Your partner will make their own, then you&rsquo;ll link the two.</Para></div>
        <div className="flex flex-col gap-[18px]">
          <Field label="Your first name" autoComplete="given-name" hint="This is what your partner will see." error={errors.name?.message} {...register('name')} />
          <Field label="Email" type="email" autoComplete="email" inputMode="email" error={errors.email?.message} {...register('email')} />
          <Field label="Password" type={show ? 'text' : 'password'} autoComplete="new-password" placeholder="At least 10 characters" error={errors.password?.message} trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />} {...register('password')} />
        </div>
      </div>
      <Bottom>
        <Button type="submit" loading={isSubmitting}>Continue</Button>
        <div className="pt-1.5 text-center text-[13px] leading-normal text-stone">By continuing you agree to the <a href="#terms" className="font-semibold text-plum">Terms</a> and <a href="#privacy" className="font-semibold text-plum">Privacy Policy</a>.</div>
      </Bottom>
    </form>
  );
}
