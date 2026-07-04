import React, { useEffect, useRef } from "react";
import { createRoot } from "react-dom/client";
import { motion, useMotionValue, useReducedMotion, useScroll, useSpring, useTransform, useVelocity } from "motion/react";
import { branchLights, heavyEase, huds, panels, signalRunners } from "./hyperframes.js";

const ghostLogs = [
  "{ pid: proc_7f2a, policy: backend-only }",
  "deny fs.write /frontend/App.jsx",
  "budget.usd.remaining = 4.62",
  "approval.wait human_gate",
  "audit.bundle redacted=true",
  "replay.event policy.denied",
];

function MotionBackdrop() {
  const reduce = useReducedMotion();
  const { scrollYProgress } = useScroll();
  const xA = useTransform(scrollYProgress, [0, 1], reduce ? ["0%", "0%"] : ["-8%", "18%"]);
  const yA = useTransform(scrollYProgress, [0, 1], reduce ? ["0%", "0%"] : ["-6%", "14%"]);
  const xB = useTransform(scrollYProgress, [0, 1], reduce ? ["0%", "0%"] : ["12%", "-14%"]);
  const yB = useTransform(scrollYProgress, [0, 1], reduce ? ["0%", "0%"] : ["10%", "-10%"]);

  return (
    <div className="volumetric-backdrop" aria-hidden="true">
      <motion.div className="spotlight spotlight-a" style={{ x: xA, y: yA }} />
      <motion.div className="spotlight spotlight-b" style={{ x: xB, y: yB }} />
      <div className="ghost-terminal">
        {[0, 1, 2].map((column) => (
          <div className="ghost-column" key={column}>
            {ghostLogs.map((line) => <span key={`${column}-${line}`}>{line}</span>)}
            {ghostLogs.map((line) => <span key={`${column}-repeat-${line}`}>{line}</span>)}
          </div>
        ))}
      </div>
    </div>
  );
}

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
  const { scrollYProgress } = useScroll({ target: ref, offset: ["start start", "end start"] });
  const scrollSpeed = useVelocity(scrollYProgress);
  const beamPath = useTransform(scrollYProgress, [0.05, 0.94], [0, 1]);
  const beamOpacity = useTransform(scrollYProgress, [0, 0.12, 0.88, 1], [0, 1, 0.92, 0.18]);
  const beamPower = useTransform(scrollSpeed, [-1, 0, 1], [0.68, 0.86, 1]);

  useEffect(() => {
    const element = ref.current;
    const canHover = window.matchMedia("(hover: hover) and (pointer: fine)").matches;
    if (!element || reduce || !canHover) return;
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
      <svg className="scroll-guide-beam" viewBox="0 0 100 120" aria-hidden="true">
        <motion.path
          d="M52 46 C52 62 50 78 50 120"
          pathLength={beamPath}
          style={{ opacity: beamOpacity }}
        />
        <motion.path
          className="scroll-guide-core"
          d="M52 46 C52 62 50 78 50 120"
          pathLength={beamPath}
          style={{ opacity: beamPower }}
        />
      </svg>
      <div className="signal-plane">
        <motion.div className="signal-line line-a" animate={reduce ? {} : { opacity: [0.38, 0.88, 0.38], x: [-6, 8, -6] }} transition={{ ...floatTransition, duration: 4.2 }} />
        <motion.div className="signal-line line-b" animate={reduce ? {} : { opacity: [0.22, 0.74, 0.22], x: [8, -10, 8] }} transition={{ ...floatTransition, duration: 5.4 }} />
        <motion.div className="signal-line line-c" animate={reduce ? {} : { opacity: [0.28, 0.8, 0.28], y: [8, -8, 8] }} transition={{ ...floatTransition, duration: 6.2 }} />
        {signalRunners.map((runner) => (
          <span className={`signal-runner ${runner.className}`} key={runner.className} />
        ))}
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

  const revealTargets = document.querySelectorAll(".section, .cards-section, .cards article, .plans article, .market-cards article, .proof-steps article");
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

  const intentSection = document.querySelector("#intent");
  if (intentSection) {
    const intentObserver = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        document.body.classList.toggle("contract-processing", entry.isIntersecting);
      }
    }, { threshold: 0.32 });
    intentObserver.observe(intentSection);
  }

  const canHoverPrecisely = window.matchMedia("(hover: hover) and (pointer: fine)").matches;
  const magneticTargets = document.querySelectorAll(".button, .market-cards article, .cards article, .request-card, .contract-card");
  for (const target of magneticTargets) {
    target.classList.add("magnetic-ready");
    if (reduce || !canHoverPrecisely) continue;
    let scheduled = false;
    let lastEvent = null;
    target.addEventListener("pointermove", (event) => {
      lastEvent = event;
      if (scheduled) return;
      scheduled = true;
      requestAnimationFrame(() => {
        scheduled = false;
        const rect = target.getBoundingClientRect();
        const x = (lastEvent.clientX - rect.left) / rect.width - 0.5;
        const y = (lastEvent.clientY - rect.top) / rect.height - 0.5;
        target.style.setProperty("--mx", (x * 10).toFixed(2));
        target.style.setProperty("--my", (y * 10).toFixed(2));
      });
    });
    target.addEventListener("pointerleave", () => {
      target.style.setProperty("--mx", "0");
      target.style.setProperty("--my", "0");
    });
  }
}

const backdropRoot = document.createElement("div");
backdropRoot.className = "motion-backdrop-root";
document.body.prepend(backdropRoot);
createRoot(backdropRoot).render(<MotionBackdrop />);

const hero = document.querySelector(".hero-visual");
if (hero) {
  createRoot(hero).render(<MotionHero />);
}
installSiteMotion();
