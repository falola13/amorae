import type { Metadata } from "next";

import { legal } from "@/lib/legal";
import { LegalDocument, LegalSection } from "../legal-document";

export const metadata: Metadata = { title: "Privacy Policy" };

// Every statement here describes what the code actually does. When the code
// changes (a new kind of data, a new provider), change this page in the same PR.
export default function PrivacyPage() {
  return (
    <LegalDocument
      title="Privacy Policy"
      intro="Amorae is a private space for two people. This explains what we keep, why, who can see it, and how to remove it."
    >
      <LegalSection title="What we keep">
        <ul>
          <li>
            <strong className="text-ink">Your account:</strong> your first name, email, timezone,
            when you joined and when you last logged in.
          </li>
          <li>
            <strong className="text-ink">Your password:</strong> only as a one-way scrambled form (a
            bcrypt hash). We can&rsquo;t read it, and nobody at Amorae can tell you what it is.
          </li>
          <li>
            <strong className="text-ink">What the two of you add:</strong> prayers and which ones
            you&rsquo;ve prayed, reflections, events and checklists, goals and progress, challenges,
            journal entries, appreciations, memories and milestones, and your notification settings.
          </li>
          <li>
            <strong className="text-ink">Request logs:</strong> our servers record each request (the
            time, what was asked for, the result, and your IP address). We use these to keep the
            service secure and to slow down abuse, such as repeated wrong passwords.
          </li>
        </ul>
      </LegalSection>

      <LegalSection title="Who can see it">
        <p className="m-0">
          Your partner sees what either of you adds to your shared space: that&rsquo;s what Amorae
          is for. They see your first name, but never your email or password.
        </p>
        <p className="m-0">
          We don&rsquo;t sell your data, show ads, or use third-party analytics or tracking. Amorae
          runs on hosting providers who store data on our behalf; we&rsquo;ll name them here before
          launch.
        </p>
      </LegalSection>

      <LegalSection title="Signing in and cookies">
        <p className="m-0">
          When you log in, your browser keeps a random sign-in token in one cookie. Scripts on the
          page can&rsquo;t read it, and we store only a scrambled copy of it. A sign-in lasts up to{" "}
          {legal.sessionDays} days, and logging out ends it immediately. We don&rsquo;t use any
          other cookies.
        </p>
      </LegalSection>

      <LegalSection title="On your device">
        <p className="m-0">
          The app remembers a few preferences in your browser, such as whether you&rsquo;ve
          dismissed the install guide. Changes you make while offline wait on your device until they
          can be sent, and they&rsquo;re removed when you log out. The app saves its own files so it
          opens quickly, but it never saves your prayers, entries or other content to your device.
        </p>
      </LegalSection>

      <LegalSection title="Keeping it safe">
        <p className="m-0">
          Connections to Amorae are encrypted. Passwords and sign-in tokens are stored only in
          scrambled form, and changing your email needs your current password. Repeated wrong
          passwords are slowed down.
        </p>
      </LegalSection>

      <LegalSection title="Deleting your data">
        <p className="m-0">
          You can change your name, email and timezone in Settings. Deleting your account (Settings,
          then Delete account) removes your account and the content you added.
        </p>
        <p className="m-0">
          Depending on where you live, data protection law may give you further rights, such as
          getting a copy of your data. We&rsquo;ll set these out here before launch.
        </p>
      </LegalSection>
    </LegalDocument>
  );
}
