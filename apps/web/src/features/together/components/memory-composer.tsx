'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import type { z } from 'zod';

import { useAddMemory } from '@/features/together/hooks';
import { memorySchema } from '@/lib/api/schemas';
import { iso } from '@/lib/dates';
import { today } from '@/lib/today';
import { BareInput, BareTextarea, Button, Sheet } from '@/components/ui/kit';

type MemoryFormInput = z.infer<typeof memorySchema>;

export function MemoryComposer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const add = useAddMemory();
  const { register, handleSubmit, reset, formState: { errors } } = useForm<MemoryFormInput>({
    resolver: zodResolver(memorySchema),
    defaultValues: { title: '', location: '', note: '' },
  });

  const close = () => { reset(); onClose(); };
  const onSubmit = (v: MemoryFormInput) =>
    add.mutate({ title: v.title, date: iso(today()), location: v.location || undefined, note: v.note || undefined, has_photo: false }, { onSuccess: close });

  return (
    <Sheet open={open} onClose={close} title="Save a moment from today." labelledBy="m-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <BareInput label="What happened" autoFocus placeholder="Our first Amorae date night" className="h-11 text-[22px] font-semibold tracking-[-0.02em]" error={errors.title?.message} {...register('title')} />
        <BareInput label="Where, optional" placeholder="Lekki" className="h-10 text-body" error={errors.location?.message} {...register('location')} />
        <BareTextarea label="A short note, optional" rows={2} placeholder="We stayed until they stacked the chairs." className="min-h-[56px] text-body" error={errors.note?.message} {...register('note')} />
        <div className="flex flex-col gap-1"><Button type="submit" loading={add.isPending}>Save this moment</Button><Button type="button" variant="text" icon="image">Add a photo</Button></div>
      </form>
    </Sheet>
  );
}
