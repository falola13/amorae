import Link from 'next/link';
import { Icon } from '@/components/Icon';

export default function NotFound() {
  return (
    <div className="mx-auto flex min-h-[100dvh] max-w-[520px] flex-col justify-center gap-3.5 bg-bg px-6">
      <Icon name="alert" size={28} strokeWidth={1.4} className="text-red" />
      <h1 className="m-0 text-[24px] font-semibold leading-tight tracking-[-0.02em]">That page isn&rsquo;t here</h1>
      <p className="m-0 text-body text-stone">Nothing of yours is lost. Head back to your space.</p>
      <Link href="/" className="press mt-2.5 flex h-[54px] items-center justify-center rounded-btn bg-plum text-[16px] font-semibold text-surface no-underline">Go home</Link>
    </div>
  );
}
