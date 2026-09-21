import { Mark } from './Icon';

export function SplashView({ label = 'Opening your space' }: { label?: string }) {
  return (
    <div className="mx-auto flex min-h-[100dvh] w-full max-w-[520px] flex-col bg-bg">
      <div className="flex grow flex-col items-center justify-center gap-3.5 pb-10">
        <Mark size={56} className="text-plum" />
        <div className="text-[30px] font-semibold tracking-[-0.03em]">Amorae</div>
        <div className="text-support text-stone">Two hearts, one faith.</div>
      </div>
      <div role="status" className="flex shrink-0 flex-col items-center gap-3.5 pb-[74px]">
        <div className="h-0.5 w-10 animate-breathe rounded-full bg-plum" />
        <div className="text-[13px] text-stone">{label}</div>
      </div>
    </div>
  );
}
