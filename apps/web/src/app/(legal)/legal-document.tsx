import type { ReactNode } from "react";

import { SafeTop } from "@/components/layout/screen";
import { Micro, Para, Title, TopBar } from "@/components/ui/kit";
import { legal } from "@/lib/legal";
import { routes } from "@/lib/routes";

/** Shared chrome for the Terms and Privacy pages: title, draft notice, sections. */
export function LegalDocument({ title, intro, children }: { title: string; intro: string; children: ReactNode }) {
  return (
    <>
      <SafeTop />
      <TopBar back="Back" backHref={routes.welcome} />
      <article className="flex flex-col gap-6 px-6 pb-16 pt-3">
        <header className="flex flex-col gap-2">
          <Micro>Last updated {legal.updated}</Micro>
          <Title>{title}</Title>
          <Para>{intro}</Para>
        </header>
        <p role="note" className="m-0 rounded-card border border-line bg-surface px-4 py-3 text-support text-stone">
          <strong className="font-semibold text-ink">Draft.</strong> This hasn&rsquo;t been reviewed by a lawyer yet. It will be
          finalised before Amorae launches, and we&rsquo;ll tell you in the app if anything important changes.
        </p>
        {children}
        <LegalSection title="Contact">
          {legal.contactEmail ? (
            <>Write to <a href={`mailto:${legal.contactEmail}`} className="font-semibold text-plum">{legal.contactEmail}</a>.</>
          ) : (
            <>Contact details will be added here before launch.</>
          )}
        </LegalSection>
      </article>
    </>
  );
}

export function LegalSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="m-0 text-[19px] font-semibold tracking-[-0.01em] text-ink">{title}</h2>
      <div className="flex flex-col gap-2 text-body text-stone [&_li]:ml-5 [&_li]:list-disc [&_ul]:m-0 [&_ul]:flex [&_ul]:flex-col [&_ul]:gap-1 [&_ul]:p-0">{children}</div>
    </section>
  );
}
