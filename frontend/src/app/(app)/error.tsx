'use client';

import { Main } from '@/components/Screen';
import { ErrorState, LinkButton } from '@/components/ui';

export default function AppError({ reset }: { error: Error; reset: () => void }) {
  return <Main><ErrorState title="Something went wrong on our side" text="Your prayers and everything you’ve marked are safe. Try again in a moment." onRetry={reset} secondary={<LinkButton href="/" variant="text">Go home</LinkButton>} /></Main>;
}
