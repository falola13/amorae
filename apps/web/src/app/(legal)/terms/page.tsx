import type { Metadata } from "next";

import Link from "next/link";

import { routes } from "@/lib/routes";
import { LegalDocument, LegalSection } from "../legal-document";

export const metadata: Metadata = { title: "Terms" };

export default function TermsPage() {
  return (
    <LegalDocument
      title="Terms"
      intro="The agreement between you and Amorae when you use the app. Plain words, kept short."
    >
      <LegalSection title="Your account">
        <ul>
          <li>You need to be at least 18 to use Amorae.</li>
          <li>
            Keep your password to yourself. You&rsquo;re responsible for what happens under your
            account.
          </li>
          <li>Tell us if you think someone else has got into your account.</li>
        </ul>
      </LegalSection>

      <LegalSection title="Your shared space">
        <p className="m-0">
          Amorae is for two people who have both agreed to share a space. Only invite someone who
          wants to join you, and remember that what you add is visible to your partner.
        </p>
      </LegalSection>

      <LegalSection title="Your content">
        <p className="m-0">
          What you write and add stays yours. You give us permission to store it and show it to you
          and your partner, only so the app can work. You can delete it, or your whole account,
          whenever you like. How we handle it is set out in the{" "}
          <Link href={routes.privacy} className="font-semibold text-plum">
            Privacy Policy
          </Link>
          .
        </p>
      </LegalSection>

      <LegalSection title="Using Amorae fairly">
        <ul>
          <li>Don&rsquo;t try to get into anyone else&rsquo;s account or space.</li>
          <li>Don&rsquo;t use Amorae to harass anyone or to share anything illegal.</li>
          <li>Don&rsquo;t overload, copy or interfere with the service.</li>
        </ul>
        <p className="m-0">We may suspend an account that breaks these rules.</p>
      </LegalSection>

      <LegalSection title="The service">
        <p className="m-0">
          We&rsquo;re building Amorae carefully, but it&rsquo;s provided as it is, and features may
          change. As far as the law allows, we aren&rsquo;t responsible for losses from using it or
          from it being unavailable.
        </p>
      </LegalSection>

      <LegalSection title="Changes">
        <p className="m-0">
          If we change these terms in a way that matters, we&rsquo;ll tell you in the app before the
          change takes effect.
        </p>
      </LegalSection>
    </LegalDocument>
  );
}
