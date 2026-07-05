import React, { Suspense, lazy, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { motion, useReducedMotion } from "motion/react";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

gsap.registerPlugin(ScrollTrigger);

const Scene = lazy(() => import("./reference-scene.jsx"));

const CONTACT_URL = "https://calendly.com/wesleyong2004/node-pilot-fit-check";

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
    kicker: "Choose one NODE program",
    title: "Start with one risky workflow, not a platform promise.",
    body: "The current commercial motion is founder-led: qualify the workflow, reserve a private pilot, produce the proof packet, then invoice or attach a Stripe payment link after fit.",
    plans: [
      ["1", "Founder Proof Call", "Free", "30-minute fit check"],
      ["2", "Private Pilot", "From S$500", "One controlled repo workflow"],
      ["3", "Managed Control Layer", "Custom", "Hosted later, after security review"]
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
    title: "Teams scale agents when the evidence is exportable.",
    body: "IBM-reported coverage of a 2,000-executive study said 77% believed AI adoption had outpaced governance, only 11% felt fully prepared for large-scale agents, and organizations averaged 54 AI-related incidents last year. NODE is built for the gap between adoption and control.",
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

function TopNav({ activeIndex }) {
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
        <a className="fx-cta" data-contact-link data-live-label="Start a pilot" data-fallback-label="Start a pilot" data-contact-subject="NODE Pilot Fit Check" data-contact-body="I want to review whether NODE is a fit for a paid private pilot." href={contactHref} target="_blank" rel="noopener noreferrer">
          Start a pilot
        </a>
      </nav>
    </>
  );
}

function ChapterFrame({ chapter, index, active }) {
  const id = `${chapter.id}-title`;
  const content = {
    hero: <Hero chapter={chapter} id={id} />,
    split: <Split chapter={chapter} id={id} />,
    console: <Console chapter={chapter} id={id} />,
    program: <Program chapter={chapter} id={id} />,
    ribbon: <Ribbon chapter={chapter} id={id} />,
    proof: <Proof chapter={chapter} id={id} />,
    faq: <FAQ chapter={chapter} id={id} />
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

function Hero({ chapter, id }) {
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
        <a className="fx-button primary" data-contact-link data-live-label="Start pilot fit check" data-fallback-label="Start pilot fit check" data-contact-subject="NODE Pilot Fit Check" data-contact-body="I want to review whether NODE is a fit for a paid private pilot." href={contactHref} target="_blank" rel="noopener noreferrer">Start pilot fit check</a>
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

function Program({ chapter, id }) {
  const contactHref = safeContactHref();
  return (
    <div className="program-wrap">
      <header className="center-head">
        <p className="mini-kicker">{chapter.kicker}</p>
        <h2 id={id}>{chapter.title}</h2>
        <p>{chapter.body}</p>
      </header>
      <div className="program-stack" aria-label="NODE pilot options">
        {chapter.plans.map(([number, name, price, copy], index) => (
          <article className={`program-card program-${index}`} key={name}>
            <div>
              <span>{number}</span>
              <b>{name}</b>
            </div>
            <strong>{price}</strong>
            <p>{copy}</p>
            {index === 1 ? <a href={contactHref} target="_blank" rel="noopener noreferrer">Learn more</a> : null}
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

function FAQ({ chapter, id }) {
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
      <a className="fx-button primary final-cta" href={contactHref} target="_blank" rel="noopener noreferrer">Book the pilot fit check</a>
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
      <TopNav activeIndex={activeIndex} />
      <main className="reference-story">
        {chapters.map((chapter, index) => (
          <ChapterFrame chapter={chapter} index={index} active={index === activeIndex} key={chapter.id} />
        ))}
      </main>
      <SeoBand />
    </div>
  );
}

createRoot(document.getElementById("root")).render(<App />);
