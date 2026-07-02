import React, { useEffect, useRef } from "react";
import { createRoot } from "react-dom/client";
import { motion, useMotionValue, useReducedMotion, useSpring, useTransform } from "motion/react";

const heavyEase = [0.32, 0.72, 0, 1];
const huds = [
  { className: "policy-card", label: "Policy", title: "Forbidden path denied", copy: "The agent asks to touch frontend files. NODE blocks it before the write lands.", rotateY: -22, rotateX: 8, z: 150 },
  { className: "approval-card", label: "Approval", title: "Human gate opened", copy: "Consequential writes pause with digest, capability, and cost context.", rotateY: -16, rotateX: 10, z: 190 },
  { className: "spend-card", label: "Spend", title: "$5.00 run cap", copy: "Token, time, child-task, and tool budgets stay outside the model.", rotateY: 18, rotateX: 8, z: 130 },
  { className: "audit-card", label: "Audit", title: "Replay bundle ready", copy: "Every approval, denial, usage tick, and artifact is exportable proof.", rotateY: -12, rotateX: 7, z: 210 },
];
const panels = [
  { label: "01 / The problem", title: "Agents now touch real systems.", copy: "Prompts are not a security boundary once an agent can write files, call tools, use secrets, or open pull requests." },
  { label: "02 / The control plane", title: "NODE makes the run inspectable.", copy: "Every run gets identity, lifecycle, policy, budget, approval gates, recovery, replay, and audit history." },
  { label: "03 / The buyer proof", title: "Trust becomes an artifact.", copy: "The customer sees denied actions, approved work, token cost, test result, branch diff, and redacted audit bundle." },
];
const branchLights = [
  { className: "dot-a", delay: 0 },
  { className: "dot-b", delay: 0.45 },
  { className: "dot-c", delay: 0.9 },
  { className: "dot-d", delay: 1.35 },
  { className: "dot-e", delay: 1.8 },
  { className: "dot-f", delay: 2.25 },
];

function MotionHero() {
  const reduce = useReducedMotion();
  const ref = useRef(null);
  const rectRef = useRef(null);
  const pointerX = useMotionValue(0);
  const pointerY = useMotionValue(0);
  const smoothX = useSpring(pointerX, { stiffness: 90, damping: 22, mass: 0.35 });
  const smoothY = useSpring(pointerY, { stiffness: 90, damping: 22, mass: 0.35 });
  const rotateY = useTransform(smoothX, [-0.5, 0.5], reduce ? [0, 0] : [-13, 13]);
  const rotateX = useTransform(smoothY, [-0.5, 0.5], reduce ? [0, 0] : [9, -9]);
  const glowX = useTransform(smoothX, [-0.5, 0.5], ["22%", "78%"]);
  const glowY = useTransform(smoothY, [-0.5, 0.5], ["20%", "72%"]);

  useEffect(() => {
    const element = ref.current;
    if (!element || reduce) return;
    function updateRect() {
      rectRef.current = element.getBoundingClientRect();
    }
    function handlePointerMove(event) {
      const rect = rectRef.current;
      if (!rect) return;
      pointerX.set((event.clientX - rect.left) / rect.width - 0.5);
      pointerY.set((event.clientY - rect.top) / rect.height - 0.5);
    }
    function resetPointer() {
      pointerX.set(0);
      pointerY.set(0);
    }
    updateRect();
    element.addEventListener("pointerenter", updateRect);
    element.addEventListener("pointermove", handlePointerMove);
    element.addEventListener("pointerleave", resetPointer);
    window.addEventListener("resize", updateRect);
    return () => {
      element.removeEventListener("pointerenter", updateRect);
      element.removeEventListener("pointermove", handlePointerMove);
      element.removeEventListener("pointerleave", resetPointer);
      window.removeEventListener("resize", updateRect);
    };
  }, [pointerX, pointerY, reduce]);

  const floatTransition = reduce
    ? { duration: 0.01 }
    : { duration: 4.8, repeat: Infinity, repeatType: "mirror", ease: "easeInOut" };

  return (
    <motion.div
      ref={ref}
      className="motion-hero"
      style={{ rotateX, rotateY }}
      initial={reduce ? false : { opacity: 0, y: 28, scale: 0.98 }}
      animate={reduce ? { opacity: 1 } : { opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.72, ease: heavyEase }}
    >
      <motion.div className="cursor-sensor-glow" style={{ left: glowX, top: glowY }} />
      <div className="signal-plane">
        <motion.div className="signal-line line-a" animate={reduce ? {} : { opacity: [0.38, 0.88, 0.38], x: [-6, 8, -6] }} transition={{ ...floatTransition, duration: 4.2 }} />
        <motion.div className="signal-line line-b" animate={reduce ? {} : { opacity: [0.22, 0.74, 0.22], x: [8, -10, 8] }} transition={{ ...floatTransition, duration: 5.4 }} />
        <motion.div className="signal-line line-c" animate={reduce ? {} : { opacity: [0.28, 0.8, 0.28], y: [8, -8, 8] }} transition={{ ...floatTransition, duration: 6.2 }} />
        <span className="signal-runner runner-a" />
        <span className="signal-runner runner-b" />
        <span className="signal-runner runner-c" />
        {branchLights.map((light) => (
          <motion.span
            className={`branch-light ${light.className}`}
            key={light.className}
            animate={reduce ? {} : { opacity: [0.38, 1, 0.38], scale: [0.82, 1.35, 0.82] }}
            transition={reduce ? { duration: 0.01 } : { duration: 2.6, delay: light.delay, repeat: Infinity, ease: "easeInOut" }}
          />
        ))}
        <motion.div
          className="node-core"
          style={{ rotateX: 62, rotateZ: -14, z: 90 }}
          animate={reduce ? {} : { y: [0, -12, 0], scale: [1, 1.025, 1] }}
          transition={{ ...floatTransition, duration: 5.8 }}
        >
          <span>NODE</span>
          <strong>Agent process</strong>
          <em>policy live</em>
        </motion.div>
      </div>
      {huds.map((hud, index) => (
        <FloatingCard key={hud.title} {...hud} delay={index * 0.08} reduce={reduce} />
      ))}
      <motion.div
        className="scroll-hud-strip"
        initial={reduce ? false : { opacity: 0, y: 22 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.28, duration: 0.7, ease: heavyEase }}
      >
        {panels.map((panel) => (
          <article className="hud-panel" key={panel.label}>
            <span>{panel.label}</span>
            <strong>{panel.title}</strong>
            <p>{panel.copy}</p>
          </article>
        ))}
      </motion.div>
    </motion.div>
  );
}

function FloatingCard({ className, label, title, copy, delay, reduce, rotateY, rotateX, z }) {
  return (
    <motion.article
      className={`floating-card ${className}`}
      style={{ rotateY, rotateX, z }}
      initial={reduce ? false : { opacity: 0, y: 22, scale: 0.96 }}
      animate={reduce ? { opacity: 1 } : { opacity: 1, y: [0, -14, 0], scale: 1 }}
      whileHover={reduce ? undefined : { y: -20, scale: 1.035, transition: { duration: 0.24, ease: heavyEase } }}
      transition={reduce ? { duration: 0.01 } : { opacity: { delay, duration: 0.4 }, y: { delay, duration: 4.6, repeat: Infinity, ease: "easeInOut" }, scale: { delay, duration: 0.35 } }}
    >
      <span>{label}</span>
      <strong>{title}</strong>
      <p>{copy}</p>
    </motion.article>
  );
}

function installSiteMotion() {
  const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const links = document.querySelectorAll('a[href^="#"]');
  const sweep = document.createElement("div");
  sweep.className = "route-sweep";
  document.body.appendChild(sweep);

  for (const link of links) {
    link.addEventListener("click", (event) => {
      const href = link.getAttribute("href");
      if (!href || href === "#") return;
      const target = document.querySelector(href);
      if (!target) return;
      event.preventDefault();
      document.body.classList.remove("route-transition");
      if (!reduce) {
        requestAnimationFrame(() => document.body.classList.add("route-transition"));
      }
      target.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
      window.history.replaceState(null, "", href);
    });
  }

  const revealTargets = document.querySelectorAll(".section, .cards article, .plans article, .market-cards article, .proof-steps article");
  for (const target of revealTargets) target.classList.add("reveal-ready");
  const observer = new IntersectionObserver((entries) => {
    for (const entry of entries) {
      if (entry.isIntersecting) {
        entry.target.classList.add("is-visible");
        observer.unobserve(entry.target);
      }
    }
  }, { threshold: 0.18 });
  for (const target of revealTargets) observer.observe(target);
}

const hero = document.querySelector(".hero-visual");
if (hero) {
  createRoot(hero).render(<MotionHero />);
}
installSiteMotion();
