'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import type { z } from 'zod';

import { useAddJournal } from '@/features/together/hooks';
import { journalSchema } from '@/lib/api/schemas';
import type { JournalEntry } from '@/lib/api/types';
import { BareTextarea, Button, Sheet, cx } from '@/components/ui/kit';

const TAGS: JournalEntry['tag'][] = ['Gratitude', 'Reflection', 'Memory', 'Appreciation', 'Plans'];

type JournalFormInput = z.infer<typeof journalSchema>;

export function JournalComposer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const add = useAddJournal();
  const { register, handleSubmit, control, reset, formState: { errors } } = useForm<JournalFormInput>({
    resolver: zodResolver(journalSchema),
    defaultValues: { tag: 'Gratitude', text: '' },
  });

  const close = () => { reset(); onClose(); };
  const onSubmit = (v: JournalFormInput) => add.mutate(v, { onSuccess: close });

  return (
    <Sheet open={open} onClose={close} title="Write something for us." labelledBy="j-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <Controller
          name="tag"
          control={control}
          render={({ field }) => (
            <div role="radiogroup" aria-label="Kind of entry" className="-mx-1 flex flex-wrap gap-1">
              {TAGS.map((t) => (
                <button key={t} type="button" role="radio" aria-checked={field.value === t} onClick={() => field.onChange(t)} className={cx('press h-9 rounded-full px-3.5 text-[14px] font-semibold', field.value === t ? 'bg-plum-tint text-plum' : 'text-stone')}>{t}</button>
              ))}
            </div>
          )}
        />
        <BareTextarea label="Entry" hideLabel rows={4} autoFocus placeholder="A line or two, in your own words." className="min-h-[110px] text-[19px] leading-[1.6]" error={errors.text?.message} {...register('text')} />
        <div className="flex flex-col gap-1"><Button type="submit" loading={add.isPending}>Save to our journal</Button><Button type="button" variant="text" onClick={close}>Not now</Button></div>
      </form>
    </Sheet>
  );
}
