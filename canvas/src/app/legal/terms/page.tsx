import type { Metadata } from "next";

const LAST_UPDATED = "April 21, 2026";

export const metadata: Metadata = {
  title: "Terms of Service — Molecule AI",
  description:
    "Terms and conditions governing use of the Molecule AI platform.",
};

export default function TermsOfServicePage() {
  return (
    <>
      <h1 className="text-4xl font-bold tracking-tight text-white">
        Terms of Service
      </h1>
      <p className="mt-2 text-sm text-zinc-500">
        Last updated: {LAST_UPDATED}
      </p>

      <p className="mt-6 text-zinc-300">
        These Terms of Service (&ldquo;Terms&rdquo;) govern your access to and
        use of the Molecule AI platform, including the canvas application, API,
        workspace runtime, and all related services (collectively, the
        &ldquo;Service&rdquo;) operated by Molecule AI, Inc.
        (&ldquo;Molecule AI,&rdquo; &ldquo;we,&rdquo; &ldquo;us,&rdquo; or
        &ldquo;our&rdquo;). By accessing or using the Service, you agree to be
        bound by these Terms. If you do not agree, do not use the Service.
      </p>

      {/* ------------------------------------------------------------ */}
      <Section title="1. Eligibility and Accounts">
        <p>
          You must be at least 16 years old and legally capable of entering
          into a binding agreement to use the Service. When you create an
          account, you agree to provide accurate, current, and complete
          information and to keep it updated.
        </p>
        <p className="mt-3">
          You are responsible for safeguarding your account credentials and
          for all activity that occurs under your account. Notify us
          immediately at{" "}
          <a
            href="mailto:contact@moleculesai.app"
            className="text-blue-400 underline hover:text-blue-300"
          >
            contact@moleculesai.app
          </a>{" "}
          if you suspect unauthorized access.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="2. The Service">
        <p>
          Molecule AI is an AI agent orchestration platform. The Service
          allows you to create organizations, configure workspaces, deploy AI
          agent teams, connect third-party integrations (such as GitHub and
          Slack), and manage agent workflows through our canvas interface and
          API.
        </p>
        <p className="mt-3">
          We reserve the right to modify, suspend, or discontinue any part of
          the Service at any time, with reasonable notice where practicable.
          We will not be liable for any modification, suspension, or
          discontinuation.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="3. Acceptable Use">
        <p>You agree not to use the Service to:</p>
        <ul className="mt-3 list-disc space-y-2 pl-5 text-zinc-300">
          <li>
            Violate any applicable law, regulation, or third-party rights.
          </li>
          <li>
            Generate, distribute, or facilitate content that is illegal,
            harmful, abusive, hateful, defamatory, or that promotes violence.
          </li>
          <li>
            Attempt to gain unauthorized access to other users&rsquo;
            accounts, workspaces, or data.
          </li>
          <li>
            Circumvent rate limits, usage quotas, or security controls.
          </li>
          <li>
            Use the Service to build a competing product by systematically
            scraping or reverse-engineering our platform.
          </li>
          <li>
            Transmit malware, viruses, or other malicious code through
            workspaces or integrations.
          </li>
          <li>
            Use AI agents deployed through the Service to impersonate real
            individuals without their consent or to conduct fraud.
          </li>
          <li>
            Resell access to the Service without our prior written consent.
          </li>
        </ul>
        <p className="mt-3">
          We reserve the right to suspend or terminate accounts that violate
          these acceptable-use requirements, with or without prior notice
          depending on the severity of the violation.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="4. Your Data and Content">
        <p>
          You retain all rights to the data, prompts, configurations, and
          content you submit to the Service (&ldquo;Your Content&rdquo;). By
          using the Service, you grant us a limited, non-exclusive license to
          host, process, and transmit Your Content solely as necessary to
          operate and provide the Service to you.
        </p>
        <p className="mt-3">
          You are responsible for ensuring you have the right to submit Your
          Content and that it does not infringe on any third party&rsquo;s
          intellectual property or other rights.
        </p>
        <p className="mt-3">
          We do not use Your Content to train machine learning models. Agent
          prompts and responses are processed by third-party LLM providers
          under their respective terms; see our{" "}
          <a
            href="/legal/privacy"
            className="text-blue-400 underline hover:text-blue-300"
          >
            Privacy Policy
          </a>{" "}
          for details on third-party data sharing.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="5. Intellectual Property">
        <p>
          The Service, including its source code, design, documentation, and
          branding, is owned by Molecule AI and protected by intellectual
          property laws. The Molecule AI source code is available under the
          license specified in our{" "}
          <a
            href="https://github.com/Molecule-AI/molecule-monorepo"
            className="text-blue-400 underline hover:text-blue-300"
          >
            GitHub repository
          </a>
          . Nothing in these Terms grants you rights to our trademarks, logos,
          or branding beyond what is necessary to use the Service.
        </p>
        <p className="mt-3">
          AI-generated output produced by workspaces you configure belongs to
          you, subject to the terms of the underlying LLM provider(s) and any
          applicable law.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="6. Billing and Payment">
        <p>
          Certain features of the Service require a paid subscription. Pricing
          details are available on our{" "}
          <a
            href="/pricing"
            className="text-blue-400 underline hover:text-blue-300"
          >
            pricing page
          </a>
          . By subscribing, you agree to pay the applicable fees and any
          usage-based overage charges.
        </p>
        <p className="mt-3">
          Fees are billed in advance on a monthly or annual basis (depending
          on your plan) and are non-refundable except where required by law.
          We may change pricing with at least 30 days&rsquo; notice; continued
          use after a price change constitutes acceptance.
        </p>
        <p className="mt-3">
          Payments are processed by Stripe. Your payment information is
          handled directly by Stripe under their terms of service; we do not
          store your full payment card details.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="7. Third-Party Services">
        <p>
          The Service integrates with third-party services including LLM
          providers (OpenAI, Anthropic, Google, and others), GitHub, Slack,
          and cloud infrastructure providers. Your use of these integrations
          is subject to the respective third party&rsquo;s terms and privacy
          policies. We are not responsible for the availability, accuracy, or
          practices of third-party services.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="8. Disclaimer of Warranties">
        <p>
          THE SERVICE IS PROVIDED &ldquo;AS IS&rdquo; AND &ldquo;AS
          AVAILABLE&rdquo; WITHOUT WARRANTIES OF ANY KIND, WHETHER EXPRESS,
          IMPLIED, OR STATUTORY, INCLUDING BUT NOT LIMITED TO IMPLIED
          WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND
          NON-INFRINGEMENT.
        </p>
        <p className="mt-3">
          We do not warrant that the Service will be uninterrupted,
          error-free, or secure, or that AI-generated output will be accurate,
          complete, or suitable for any particular purpose. You are solely
          responsible for evaluating and verifying any output produced by AI
          agents deployed through the Service.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="9. Limitation of Liability">
        <p>
          TO THE MAXIMUM EXTENT PERMITTED BY LAW, MOLECULE AI AND ITS
          OFFICERS, DIRECTORS, EMPLOYEES, AND AGENTS SHALL NOT BE LIABLE FOR
          ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE
          DAMAGES, OR ANY LOSS OF PROFITS, REVENUE, DATA, OR GOODWILL,
          WHETHER ARISING FROM CONTRACT, TORT, OR OTHERWISE, EVEN IF ADVISED
          OF THE POSSIBILITY OF SUCH DAMAGES.
        </p>
        <p className="mt-3">
          OUR TOTAL AGGREGATE LIABILITY FOR ANY CLAIMS ARISING OUT OF OR
          RELATING TO THE SERVICE SHALL NOT EXCEED THE GREATER OF (A) THE
          AMOUNT YOU PAID US IN THE 12 MONTHS PRECEDING THE CLAIM, OR (B)
          ONE HUNDRED U.S. DOLLARS ($100).
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="10. Indemnification">
        <p>
          You agree to indemnify and hold harmless Molecule AI, its officers,
          directors, employees, and agents from any claims, damages, losses,
          liabilities, costs, and expenses (including reasonable
          attorneys&rsquo; fees) arising from your use of the Service, your
          violation of these Terms, or your infringement of any third
          party&rsquo;s rights.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="11. Termination">
        <p>
          You may close your account at any time by contacting us. We may
          suspend or terminate your account if you violate these Terms or if
          we reasonably believe continued access poses a risk to the Service
          or other users.
        </p>
        <p className="mt-3">
          Upon termination, your right to use the Service ceases immediately.
          We will retain your data for 30 days after termination to allow for
          data export, after which it will be scheduled for deletion in
          accordance with our{" "}
          <a
            href="/legal/privacy"
            className="text-blue-400 underline hover:text-blue-300"
          >
            Privacy Policy
          </a>
          .
        </p>
        <p className="mt-3">
          Sections that by their nature should survive termination (including
          liability limitations, indemnification, and intellectual property
          provisions) will remain in effect.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="12. Governing Law and Disputes">
        <p>
          These Terms are governed by the laws of the State of Delaware,
          United States, without regard to conflict-of-law principles. Any
          dispute arising from these Terms or the Service shall be resolved
          exclusively in the state or federal courts located in Delaware,
          and you consent to the personal jurisdiction of those courts.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="13. Changes to These Terms">
        <p>
          We may update these Terms from time to time. When we make material
          changes, we will notify you by posting a notice on the Service or
          sending you an email at least 30 days before the changes take
          effect. Your continued use of the Service after the updated Terms
          take effect constitutes acceptance.
        </p>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="14. Miscellaneous">
        <ul className="mt-3 list-disc space-y-2 pl-5 text-zinc-300">
          <li>
            <strong className="text-zinc-100">Entire Agreement.</strong> These
            Terms, together with our Privacy Policy, constitute the entire
            agreement between you and Molecule AI regarding the Service.
          </li>
          <li>
            <strong className="text-zinc-100">Severability.</strong> If any
            provision of these Terms is found unenforceable, the remaining
            provisions remain in full force and effect.
          </li>
          <li>
            <strong className="text-zinc-100">Waiver.</strong> Our failure to
            enforce any provision is not a waiver of our right to enforce it
            later.
          </li>
          <li>
            <strong className="text-zinc-100">Assignment.</strong> You may not
            assign your rights under these Terms without our consent. We may
            assign our rights without restriction.
          </li>
        </ul>
      </Section>

      {/* ------------------------------------------------------------ */}
      <Section title="15. Contact Us">
        <p>
          If you have questions about these Terms, contact us at:
        </p>
        <address className="mt-3 not-italic text-zinc-300">
          Molecule AI, Inc.
          <br />
          Email:{" "}
          <a
            href="mailto:contact@moleculesai.app"
            className="text-blue-400 underline hover:text-blue-300"
          >
            contact@moleculesai.app
          </a>
        </address>
      </Section>
    </>
  );
}

/* ------------------------------------------------------------------ */
/*  Helper components                                                  */
/* ------------------------------------------------------------------ */

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold text-white">{title}</h2>
      <div className="mt-3 text-zinc-300 leading-relaxed">{children}</div>
    </section>
  );
}
