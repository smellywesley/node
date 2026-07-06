import React, { Suspense, lazy, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { motion, useReducedMotion } from "motion/react";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

const Scene = lazy(() => import("./reference-scene.jsx"));

const CONTACT_URL = "https://calendly.com/wesleyong2004/node-pilot-fit-check";
const DEFAULT_INTAKE_EMAIL = "";

const intakeDefaults = {
  name: "",
  email: "",
  company: "",
  agent: "",
  riskyAction: "",
  proofNeed: "",
  localProof: "yes",
  budget: "yes"
};

const localProofLabels = {
  yes: "Can run a local/private Docker + BYOK proof",
  maybe: "May need help running local/private proof",
  no: "Requires hosted/SaaS before evaluation"
};

const budgetLabels = {
  yes: "Can pay for a focused pilot if fit is clear",
  maybe: "Budget is possible but sponsor is not confirmed",
  no: "Not ready to pay for a pilot yet"
};

const proofRun = {
  request: "Fix auth timeout. Do not touch frontend.",
  repo: "acme/backend",
  policy: "backend/** allowed, frontend/** denied",
  status: "frontend write blocked",
  spend: "$2.17 / $5.00",
  audit: "replay bundle ready"
};

const controlStages = [
  {
    id: "request",
    label: "Request",
    title: "Intent enters",
    copy: "Fix auth timeout. Do not touch frontend.",
    value: "plain English"
  },
  {
    id: "policy",
    label: "Policy",
    title: "Gate decides",
    copy: "backend/** allowed. frontend/** denied.",
    value: "blocked path"
  },
  {
    id: "sandbox",
    label: "Sandbox",
    title: "Work is contained",
    copy: "Agent writes only inside the controlled worker.",
    value: "scoped run"
  },
  {
    id: "cost",
    label: "Cost",
    title: "Meter stays armed",
    copy: "$2.17 spent against a $5.00 run cap.",
    value: "$5 cap"
  },
  {
    id: "audit",
    label: "Audit",
    title: "Proof exports",
    copy: "Prompt, policy, denial, spend, and replay bundle.",
    value: "bundle"
  }
];

const chapters = [
  {
    id: "home",
    nav: "Home",
    layout: "hero",
    scene: "hero",
    kicker: "Agentic OS / controlled workers",
    title: ["Turn AI agents", "into controlled workers."],
    subtitle: "Policy gates, approvals, cost limits, runtime state, replay, and audit proof for AI coding agents working in real repositories.",
    micro: ["private pilot", "BYOK ready", "audit bundle", "human approval"],
    camera: [0, 6.2, 16],
    target: [0, .2, 0]
  },
  {
    id: "market",
    nav: "Market",
    layout: "split",
    scene: "wire",
    oversize: "Every file, token, tool and approval is now part of the risk surface.",
    kicker: "The market moved first",
    title: "Agents now want production access.",
    body: "Gartner expects one-third of enterprise software to include agentic capabilities by 2028, with 15% of day-to-day work decisions made autonomously. The moment agents can write files, call tools, and spawn subtasks, security teams need a control layer around the run.",
    sideTitle: "NODE Control",
    sideBody: "Request in. Policy compiled. Sandbox started. Cost meter armed. Audit trail written.",
    camera: [-5.4, 4.7, 10.2],
    target: [-1.2, .2, 0]
  },
  {
    id: "control",
    nav: "Control",
    layout: "console",
    scene: "console",
    kicker: "Best control in every run",
    title: "The request becomes a controlled process.",
    body: "NODE does not ask the model to behave. It places the model inside an operating path where each action is checked before it lands.",
    tabs: ["request", "policy", "sandbox", "cost", "audit"],
    stages: controlStages,
    run: proofRun,
    camera: [-1.5, 3.8, 7.2],
    target: [0, .45, 0]
  },
  {
    id: "pilot",
    nav: "Pilot",
    layout: "program",
    scene: "program",
    kicker: "Pricing / pilot path",
    title: "Choose how much control you need.",
    body: "Start small if you need proof. Move to Pro when one team is running agents against real repos. Go Enterprise when security, roles, and audit evidence become board-level concerns.",
    plans: [
      {
        number: "1",
        name: "Proof",
        price: "Free",
        copy: "30-minute founder fit check",
        details: ["One risky workflow", "Control gap review", "Pilot go/no-go"]
      },
      {
        number: "2",
        name: "Pro",
        price: "From S$500",
        copy: "Private pilot for one controlled repo workflow",
        details: ["Policy gate", "Sandbox run", "Cost cap", "Audit bundle"],
        featured: true
      },
      {
        number: "3",
        name: "Enterprise",
        price: "Custom",
        copy: "Security-led rollout for teams scaling agent access",
        details: ["Multi-repo controls", "Approvals", "Roles", "Security review"]
      }
    ],
    camera: [2.8, 4.4, 8.8],
    target: [1.2, .35, 0]
  },
  {
    id: "path",
    nav: "Path",
    layout: "ribbon",
    scene: "ribbon",
    kicker: "And how every run moves",
    title: "And how every run is controlled?",
    chain: ["Request", "Policy Gate", "NODE Control", "Sandbox", "Cost Meter", "Audit Bundle"],
    body: "The scroll journey is the product story: the work travels through controlled zones, and every zone leaves proof behind.",
    camera: [0, 3.7, 6.3],
    target: [0, .2, 0]
  },
  {
    id: "proof",
    nav: "Proof",
    layout: "proof",
    scene: "chart",
    kicker: "Proof beats trust",
    title: "No proof. No scale. NODE turns every run into evidence.",
    body: "Agent adoption is already outrunning governance. IBM-reported coverage of a 2,000-executive study said 77% believed AI adoption had outpaced governance, only 11% felt fully prepared for large-scale agents, and organizations averaged 54 AI-related incidents last year. NODE makes control visible enough for security, platform, and leadership to say yes.",
    impact: ["54", "AI-related incidents per organization last year", "Turn incidents into controlled, replayable proof."],
    facts: [
      ["33%", "enterprise software with agentic capabilities by 2028"],
      ["15%", "day-to-day work decisions made autonomously by 2028"],
      ["77%", "leaders saying governance is already behind AI adoption"]
    ],
    camera: [5.6, 4.8, 8.8],
    target: [2.6, .25, -1.1]
  },
  {
    id: "faq",
    nav: "FAQ",
    layout: "faq",
    scene: "faq",
    kicker: "Buyer questions",
    title: "Bring one risky agent workflow. We prove control in 30 minutes.",
    body: "NODE is for founders, platform teams, DevOps, and security teams already testing AI coding agents against real repositories.",
    faqs: [
      ["Does NODE replace my coding agent?", "No. It controls the boundary, approval flow, spend, runtime state, and audit trail around the agent."],
      ["What happens in the pilot?", "One repo workflow, file policy, approval gate, cost cap, sandbox run, and audit bundle."],
      ["Is hosted SaaS ready?", "Not yet. Hosted backend comes after tenant isolation, roles, billing ledger, load evidence, and external security review."],
      ["How does payment work?", "Book the fit check. If there is fit, reserve a pilot slot through invoice or a live payment link."]
    ],
    camera: [0, 5.4, 13],
    target: [0, .2, 0]
  }
];

function safeContactHref() {
  const config = window.NODE_PAYMENT_LINKS || {};
  const value = config.pilotContactUrl || CONTACT_URL;
  try {
    const url = new URL(value);
    if (url.protocol !== "https:") return CONTACT_URL;
    if (!["calendly.com", "www.calendly.com"].includes(url.hostname)) return CONTACT_URL;
    return url.toString();
  } catch {
    return CONTACT_URL;
  }
}

function safeIntakeEmail() {
  const config = window.NODE_PAYMENT_LINKS || {};
  const email = typeof config.contactEmail === "string" ? config.contactEmail.trim() : DEFAULT_INTAKE_EMAIL;
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) ? email : "";
}

function scorePilotFit(values) {
  let score = 0;
  if (values.agent.trim().length > 8) score += 2;
  if (values.riskyAction.trim().length > 12) score += 2;
  if (values.proofNeed.trim().length > 10) score += 2;
  if (values.localProof === "yes") score += 2;
  if (values.localProof === "maybe") score += 1;
  if (values.budget === "yes") score += 2;
  if (values.budget === "maybe") score += 1;

  const label = score >= 9 ? "Strong pilot candidate" : score >= 6 ? "Needs founder review" : "Not ready yet";
  return { score, label };
}

function buildIntakeSummary(values) {
  const fit = scorePilotFit(values);
  return [
    `Pilot fit: ${fit.label} (${fit.score}/10)`,
    "",
    `Name: ${values.name || "Not provided"}`,
    `Email: ${values.email || "Not provided"}`,
    `Company/team: ${values.company || "Not provided"}`,
    "",
    `Coding agents in use: ${values.agent || "Not provided"}`,
    `Riskiest agent action: ${values.riskyAction || "Not provided"}`,
    `Proof needed: ${values.proofNeed || "Not provided"}`,
    `Local/private proof readiness: ${localProofLabels[values.localProof]}`,
    `Pilot budget readiness: ${budgetLabels[values.budget]}`,
    "",
    "Requested next step: NODE pilot fit check"
  ].join("\n");
}

function buildMailtoHref(values) {
  const to = safeIntakeEmail();
  if (!to) return "";
  const subject = `NODE pilot intake: ${values.company || values.name || "new fit check"}`;
  return `mailto:${encodeURIComponent(to)}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(buildIntakeSummary(values))}`;
}

function useJourneyProgress(reduce) {
  const [progress, setProgress] = useState(0);

  useEffect(() => {
    if (reduce) {
      setProgress(0);
      return undefined;
    }

    const trigger = ScrollTrigger.create({
      trigger: ".reference-story",
      start: "top top",
      end: "bottom bottom",
      scrub: .95,
      onUpdate: (self) => setProgress(self.progress)
    });

    requestAnimationFrame(() => {
      ScrollTrigger.refresh();
      ScrollTrigger.update();
    });

    return () => trigger.kill();
  }, [reduce]);

  return progress;
}

function useCursorAura() {
  useEffect(() => {
    function onPointerMove(event) {
      document.documentElement.style.setProperty("--cursor-x", `${event.clientX}px`);
      document.documentElement.style.setProperty("--cursor-y", `${event.clientY}px`);
    }

    window.addEventListener("pointermove", onPointerMove, { passive: true });
    return () => window.removeEventListener("pointermove", onPointerMove);
  }, []);
}

function useActiveChapter() {
  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    let frame = 0;
    function updateActive() {
      frame = 0;
      const center = window.innerHeight * .5;
      const sections = [...document.querySelectorAll(".fx-section")];
      const closest = sections.reduce((winner, section, index) => {
        const rect = section.getBoundingClientRect();
        const sectionCenter = rect.top + rect.height * .5;
        const distance = Math.abs(sectionCenter - center);
        return distance < winner.distance ? { index, distance } : winner;
      }, { index: 0, distance: Number.POSITIVE_INFINITY });
      setActiveIndex(closest.index);
    }

    function onScroll() {
      if (frame) return;
      frame = requestAnimationFrame(updateActive);
    }

    updateActive();
    window.addEventListener("scroll", onScroll, { passive: true });
    window.addEventListener("resize", onScroll);
    return () => {
      if (frame) cancelAnimationFrame(frame);
      window.removeEventListener("scroll", onScroll);
      window.removeEventListener("resize", onScroll);
    };
  }, []);

  return activeIndex;
}

function scrollToChapter(event, id) {
  event?.preventDefault?.();
  const target = document.getElementById(id);
  if (!target) return;
  const reduce = window.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches;
  target.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
  window.history.pushState(null, "", `#${id}`);
}

function TopNav({ activeIndex, onPilotClick }) {
  const contactHref = safeContactHref();
  return (
    <>
      <div className="nav-reveal-zone" aria-hidden="true" />
      <nav className={`fx-nav ${activeIndex > 0 ? "is-submerged" : ""}`} aria-label="Primary">
        <a className="fx-brand" href="#home" onClick={(event) => scrollToChapter(event, "home")}>
          <span>NODE</span>
        </a>
        <div className="fx-links">
          {chapters.slice(0, 6).map((chapter, index) => (
            <a
              key={chapter.id}
              className={index === activeIndex ? "is-active" : ""}
              href={`#${chapter.id}`}
              onClick={(event) => scrollToChapter(event, chapter.id)}
            >
              {chapter.nav}
            </a>
          ))}
        </div>
        <a className="fx-login" href="#faq" onClick={(event) => scrollToChapter(event, "faq")}>FAQ</a>
        <a className="fx-cta" data-contact-link data-live-label="Start a pilot" data-fallback-label="Start a pilot" data-contact-subject="NODE Pilot Fit Check" data-contact-body="I want to review whether NODE is a fit for a paid private pilot." href={contactHref} target="_blank" rel="noopener noreferrer" onClick={onPilotClick}>
          Start a pilot
        </a>
      </nav>
    </>
  );
}

function ChapterFrame({ chapter, index, active, onPilotClick }) {
  const id = `${chapter.id}-title`;
  const content = {
    hero: <Hero chapter={chapter} id={id} onPilotClick={onPilotClick} />,
    split: <Split chapter={chapter} id={id} />,
    console: <Console chapter={chapter} id={id} />,
    program: <Program chapter={chapter} id={id} onPilotClick={onPilotClick} />,
    ribbon: <Ribbon chapter={chapter} id={id} />,
    proof: <Proof chapter={chapter} id={id} />,
    faq: <FAQ chapter={chapter} id={id} onPilotClick={onPilotClick} />
  }[chapter.layout];

  return (
    <section className={`fx-section fx-${chapter.layout} ${active ? "is-active" : ""}`} id={chapter.id} aria-labelledby={id}>
      <motion.div
        className="section-inner"
        initial={false}
        animate={{ opacity: active ? 1 : .78, y: active ? 0 : 26, scale: active ? 1 : .985 }}
        transition={{ duration: .72, ease: [.16, 1, .3, 1] }}
      >
        {content}
      </motion.div>
      {index > 0 ? <p className="section-step">0{index}</p> : null}
    </section>
  );
}

function GlowTitle({ lines, id }) {
  return (
    <h1 id={id} className="glow-title">
      {lines.map((line, index) => (
        <span key={line} className={index === lines.length - 1 ? "accent-line" : ""}>{line}</span>
      ))}
    </h1>
  );
}

function Hero({ chapter, id, onPilotClick }) {
  const contactHref = safeContactHref();
  return (
    <div className="hero-copy">
      <p className="mini-kicker">{chapter.kicker}</p>
      <GlowTitle lines={chapter.title} id={id} />
      <p className="hero-subtitle">{chapter.subtitle}</p>
      <div className="micro-row" aria-label="NODE pilot features">
        {chapter.micro.map((item) => <span key={item}>{item}</span>)}
      </div>
      <div className="hero-actions">
        <a className="fx-button primary" data-contact-link data-live-label="Start pilot fit check" data-fallback-label="Start pilot fit check" data-contact-subject="NODE Pilot Fit Check" data-contact-body="I want to review whether NODE is a fit for a paid private pilot." href={contactHref} target="_blank" rel="noopener noreferrer" onClick={onPilotClick}>Start pilot fit check</a>
        <a className="fx-button ghost" href="#control" onClick={(event) => scrollToChapter(event, "control")}>See the control path</a>
      </div>
    </div>
  );
}

function Split({ chapter, id }) {
  return (
    <div className="split-wrap">
      <p className="mega-mask" aria-hidden="true">{chapter.oversize}</p>
      <div className="wire-copy">
        <p className="mini-kicker">{chapter.kicker}</p>
        <h2 id={id}>{chapter.title}</h2>
      </div>
      <article className="copy-block">
        <span>{chapter.sideTitle}</span>
        <p>{chapter.body}</p>
        <small>{chapter.sideBody}</small>
      </article>
    </div>
  );
}

function Console({ chapter, id }) {
  return (
    <div className="console-wrap">
      <header className="center-head">
        <p className="mini-kicker">{chapter.kicker}</p>
        <h2 id={id}>{chapter.title}</h2>
        <p>{chapter.body}</p>
      </header>
      <div className="tab-row" aria-label="Control sequence">
        {chapter.tabs.map((tab, index) => <span className={index === 1 ? "is-selected" : ""} key={tab}>{tab}</span>)}
      </div>
      <div className="execution-map" aria-label="NODE execution path">
        <div className="execution-beam" aria-hidden="true">
          {chapter.stages.map((stage) => <i className={`beam-node node-${stage.id}`} key={stage.id} />)}
        </div>
        {chapter.stages.map((stage, index) => (
          <article className={`stage-lens stage-${stage.id}`} key={stage.id}>
            <span>{String(index + 1).padStart(2, "0")} / {stage.label}</span>
            <b>{stage.title}</b>
            <p>{stage.copy}</p>
            <small>{stage.value}</small>
          </article>
        ))}
      </div>
      <article className="run-terminal">
        <div className="terminal-top">
          <span>NODE RUN</span>
          <b>{chapter.run.status}</b>
        </div>
        <code><span>request</span>"{chapter.run.request}"</code>
        <code><span>repo</span>{chapter.run.repo}</code>
        <code className="decision-deny"><span>policy</span>{chapter.run.policy}</code>
        <code><span>cost</span>{chapter.run.spend}</code>
        <code><span>audit</span>{chapter.run.audit}</code>
      </article>
    </div>
  );
}

function Program({ chapter, id, onPilotClick }) {
  const contactHref = safeContactHref();
  return (
    <div className="program-wrap">
      <header className="center-head">
        <p className="mini-kicker">{chapter.kicker}</p>
        <h2 id={id}>{chapter.title}</h2>
        <p>{chapter.body}</p>
      </header>
      <div className="program-stack" aria-label="NODE pilot options">
        {chapter.plans.map((plan, index) => (
          <article className={`program-card program-${index} ${plan.featured ? "is-featured" : ""}`} key={plan.name}>
            <div>
              <span>{plan.number}</span>
              <b>{plan.name}</b>
            </div>
            <strong>{plan.price}</strong>
            <p>{plan.copy}</p>
            <ul>
              {plan.details.map((detail) => <li key={detail}>{detail}</li>)}
            </ul>
            {index === 1 ? <a href={contactHref} target="_blank" rel="noopener noreferrer" onClick={onPilotClick}>Start Pro pilot</a> : null}
          </article>
        ))}
      </div>
    </div>
  );
}

function Ribbon({ chapter, id }) {
  return (
    <div className="ribbon-wrap">
      <p className="mini-kicker">{chapter.kicker}</p>
      <h2 id={id}>{chapter.title}</h2>
      <div className="path-chain" aria-label="NODE controlled execution sequence">
        {chapter.chain.map((item) => <span key={item}>{item}</span>)}
      </div>
      <p>{chapter.body}</p>
    </div>
  );
}

function Proof({ chapter, id }) {
  return (
    <div className="proof-wrap">
      <header className="proof-head">
        <p className="mini-kicker">{chapter.kicker}</p>
        <h2 id={id}>{chapter.title}</h2>
      </header>
      <article className="proof-copy">
        <div className="impact-lockup" aria-label="AI governance incident proof point">
          <strong>{chapter.impact[0]}</strong>
          <span>{chapter.impact[1]}</span>
          <small>{chapter.impact[2]}</small>
        </div>
        <p>{chapter.body}</p>
        <div className="source-line">
          <a href="https://www.itpro.com/technology/artificial-intelligence/practical-ai-the-age-of-agentic-ai" target="_blank" rel="noreferrer">Gartner forecast coverage</a>
          <a href="https://www.itpro.com/technology/artificial-intelligence/cios-and-ctos-are-making-high-stakes-decisions-with-incomplete-information-ibm-survey-reveals" target="_blank" rel="noreferrer">IBM governance survey coverage</a>
        </div>
      </article>
      <div className="fact-rail" aria-label="Market proof numbers">
        {chapter.facts.map(([number, copy]) => (
          <div key={number}>
            <strong>{number}</strong>
            <span>{copy}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function FAQ({ chapter, id, onPilotClick }) {
  const contactHref = safeContactHref();
  return (
    <div className="faq-wrap">
      <header className="center-head">
        <p className="mini-kicker">{chapter.kicker}</p>
        <h2 id={id}>{chapter.title}</h2>
        <p>{chapter.body}</p>
      </header>
      <div className="faq-list">
        {chapter.faqs.map(([question, answer]) => (
          <details key={question}>
            <summary>{question}</summary>
            <p>{answer}</p>
          </details>
        ))}
      </div>
      <a className="fx-button primary final-cta" href={contactHref} target="_blank" rel="noopener noreferrer" onClick={onPilotClick}>Book the pilot fit check</a>
    </div>
  );
}

function PilotIntake({ open, onClose, contactHref }) {
  const [values, setValues] = useState(intakeDefaults);
  const [submitted, setSubmitted] = useState(false);
  const [copied, setCopied] = useState(false);
  const mailtoHref = buildMailtoHref(values);
  const fit = scorePilotFit(values);

  useEffect(() => {
    if (!open) return undefined;
    setSubmitted(false);
    setCopied(false);
    document.body.classList.add("intake-open");
    return () => document.body.classList.remove("intake-open");
  }, [open]);

  if (!open) return null;

  function updateField(event) {
    const { name, value } = event.target;
    setValues((current) => ({ ...current, [name]: value }));
  }

  function submitIntake(event) {
    event.preventDefault();
    setSubmitted(true);
    window.setTimeout(() => document.querySelector(".intake-result")?.scrollIntoView({ block: "nearest" }), 0);
  }

  async function copySummary() {
    const summary = buildIntakeSummary(values);
    try {
      await navigator.clipboard?.writeText(summary);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="intake-shell" role="dialog" aria-modal="true" aria-labelledby="intake-title">
      <button className="intake-backdrop" type="button" aria-label="Close pilot intake" onClick={onClose} />
      <article className="intake-panel">
        <button className="intake-close" type="button" aria-label="Close pilot intake" onClick={onClose}>x</button>
        <div className="intake-head">
          <p className="mini-kicker">Before Calendly</p>
          <h2 id="intake-title">Tell us what the pilot must prove.</h2>
          <p>These answers score fit before the call and prepare the exact proof you need.</p>
        </div>

        <form className="intake-form" onSubmit={submitIntake}>
          <div className="field-grid">
            <label>
              <span>Your name</span>
              <input name="name" value={values.name} onChange={updateField} autoComplete="name" required />
            </label>
            <label>
              <span>Email</span>
              <input name="email" value={values.email} onChange={updateField} type="email" autoComplete="email" required />
            </label>
            <label>
              <span>Company / team</span>
              <input name="company" value={values.company} onChange={updateField} autoComplete="organization" required />
            </label>
            <label>
              <span>Which coding agents are touching repos?</span>
              <textarea name="agent" value={values.agent} onChange={updateField} placeholder="Cursor, Claude Code, Copilot agents, Devin, internal repo agent..." required />
            </label>
            <label>
              <span>What action would be dangerous without approval?</span>
              <textarea name="riskyAction" value={values.riskyAction} onChange={updateField} placeholder="Write outside backend, touch secrets, change deployment config..." required />
            </label>
            <label>
              <span>What proof would make this worth a pilot?</span>
              <textarea name="proofNeed" value={values.proofNeed} onChange={updateField} placeholder="Denied write, approval gate, cost cap, audit bundle, security review artifact..." required />
            </label>
          </div>

          <fieldset>
            <legend>Can you run a local/private Docker + BYOK proof?</legend>
            {Object.entries(localProofLabels).map(([value, label]) => (
              <label className="radio-row" key={value}>
                <input type="radio" name="localProof" value={value} checked={values.localProof === value} onChange={updateField} />
                <span>{label}</span>
              </label>
            ))}
          </fieldset>

          <fieldset>
            <legend>If fit is clear, can this become a paid pilot?</legend>
            {Object.entries(budgetLabels).map(([value, label]) => (
              <label className="radio-row" key={value}>
                <input type="radio" name="budget" value={value} checked={values.budget === value} onChange={updateField} />
                <span>{label}</span>
              </label>
            ))}
          </fieldset>

          <div className="fit-readout" aria-live="polite">
            <strong>{fit.label}</strong>
            <span>{fit.score}/10 fit score</span>
          </div>

          <button className="fx-button primary intake-submit" type="submit">Review answers</button>
        </form>

        {submitted ? (
          <div className="intake-result">
            <p>Send this intake first, then book Calendly. That keeps the call focused on your exact control proof.</p>
            <div className="intake-actions">
              {mailtoHref ? (
                <a className="fx-button primary" href={mailtoHref} onClick={copySummary}>Email answers to NODE</a>
              ) : (
                <button className="fx-button primary" type="button" onClick={copySummary}>{copied ? "Copied answers" : "Copy answers"}</button>
              )}
              <a className="fx-button ghost" href={contactHref} target="_blank" rel="noopener noreferrer" onClick={onClose}>Continue to Calendly</a>
            </div>
            {!mailtoHref ? <small>Set NODE_PUBLIC_CONTACT_EMAIL on Render to enable the prefilled email step automatically.</small> : null}
          </div>
        ) : null}
      </article>
    </div>
  );
}

function SeoBand() {
  return (
    <section className="seo-band" aria-labelledby="seo-title">
      <div>
        <p className="mini-kicker">AI-readable summary</p>
        <h2 id="seo-title">NODE is an agentic operating layer for governed coding-agent work.</h2>
        <p>
          NODE controls autonomous coding agents before they touch production code. It provides policy gates, approval workflows, cost limits, runtime monitoring, replay, and audit bundles for private/local pilots with real repositories.
        </p>
      </div>
    </section>
  );
}

function App() {
  const reduce = useReducedMotion();
  const progress = useJourneyProgress(reduce);
  const activeIndex = useActiveChapter();
  const [intakeOpen, setIntakeOpen] = useState(false);
  const contactHref = safeContactHref();
  useCursorAura();

  useEffect(() => {
    if (!window.location.hash) return undefined;
    const id = window.location.hash.slice(1);
    const timer = window.setTimeout(() => document.getElementById(id)?.scrollIntoView({ block: "start" }), 160);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <div className="reference-site">
      <a className="skip-link" href="#home">Skip to content</a>
      <Suspense fallback={<div className="reference-scene" aria-hidden="true" />}>
        <Scene chapters={chapters} progress={progress} activeIndex={activeIndex} reduce={reduce} />
      </Suspense>
      <div className="fx-noise" aria-hidden="true" />
      <div className="fx-vignette" aria-hidden="true" />
      <div className="fx-cursor" aria-hidden="true" />
      <TopNav activeIndex={activeIndex} onPilotClick={(event) => {
        event.preventDefault();
        setIntakeOpen(true);
      }} />
      <main className="reference-story">
        {chapters.map((chapter, index) => (
          <ChapterFrame chapter={chapter} index={index} active={index === activeIndex} key={chapter.id} onPilotClick={(event) => {
            event.preventDefault();
            setIntakeOpen(true);
          }} />
        ))}
      </main>
      <PilotIntake open={intakeOpen} onClose={() => setIntakeOpen(false)} contactHref={contactHref} />
      <SeoBand />
    </div>
  );
}

createRoot(document.getElementById("root")).render(<App />);
