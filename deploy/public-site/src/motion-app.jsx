import React, { useEffect, useRef } from "react";
import { createRoot } from "react-dom/client";
import { motion, useMotionValue, useReducedMotion, useSpring, useTransform } from "motion/react";

const ease = [0.22, 1, 0.36, 1];

function MotionHero() {
  const reduce = useReducedMotion();
  const ref = useRef(null);
  const pointerX = useMotionValue(0);
  const pointerY = useMotionValue(0);
  const smoothX = useSpring(pointerX, { stiffness: 90, damping: 22, mass: 0.35 });
  const smoothY = useSpring(pointerY, { stiffness: 90, damping: 22, mass: 0.35 });
  const rotateY = useTransform(smoothX, [-0.5, 0.5], reduce ? [0, 0] : [-13, 13]);
  const rotateX = useTransform(smoothY, [-0.5, 0.5], reduce ? [0, 0] : [9, -9]);

  useEffect(() => {
    const element = ref.current;
    if (!element || reduce) return;
    function handlePointerMove(event) {
      const rect = element.getBoundingClientRect();
      pointerX.set((event.clientX - rect.left) / rect.width - 0.5);
      pointerY.set((event.clientY - rect.top) / rect.height - 0.5);
    }
    function resetPointer() {
      pointerX.set(0);
      pointerY.set(0);
    }
    element.addEventListener("pointermove", handlePointerMove);
    element.addEventListener("pointerleave", resetPointer);
    return () => {
      element.removeEventListener("pointermove", handlePointerMove);
      element.removeEventListener("pointerleave", resetPointer);
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
      initial={reduce ? false : { opacity: 0, y: 18, scale: 0.98 }}
      animate={reduce ? { opacity: 1 } : { opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.55, ease }}
    >
      <motion.div className="glass-orbit orbit-a" animate={reduce ? {} : { y: [-6, 12, -6], rotateZ: [-18, -13, -18] }} transition={floatTransition} />
      <motion.div className="glass-orbit orbit-b" animate={reduce ? {} : { y: [8, -10, 8], rotateZ: [17, 12, 17] }} transition={{ ...floatTransition, duration: 5.6 }} />
      <motion.article
        className="process-console"
        initial={reduce ? false : { opacity: 0, y: 24, rotateY: -24 }}
        animate={reduce ? { opacity: 1 } : { opacity: 1, y: [0, -12, 0], rotateY: -15 }}
        transition={reduce ? { duration: 0.01 } : { opacity: { duration: 0.4 }, y: { duration: 5.8, repeat: Infinity, ease: "easeInOut" }, rotateY: { duration: 0.6, ease } }}
      >
        <div className="console-topline">
          <span>process.run</span>
          <strong>approved</strong>
        </div>
        <pre><code>{`id: proc_7f2a
policy: backend-write-only
budget: $5.00 max
approval: human gate
audit: redacted bundle`}</code></pre>
        <div className="run-line"><span></span><p>Replayable evidence captured</p></div>
      </motion.article>
      <FloatingCard className="policy-card" label="Policy" title="frontend denied" copy="Forbidden paths are blocked before side effects land." delay={0.05} reduce={reduce} />
      <FloatingCard className="audit-card" label="Audit" title="support ready" copy="Health, approvals, denials, usage, and replay." delay={0.18} reduce={reduce} />
      <FloatingCard className="spend-card" label="Spend" title="BYOK" copy="No managed model credits in the first paid release." delay={0.3} reduce={reduce} />
    </motion.div>
  );
}

function FloatingCard({ className, label, title, copy, delay, reduce }) {
  return (
    <motion.article
      className={`floating-card ${className}`}
      initial={reduce ? false : { opacity: 0, y: 22, scale: 0.96 }}
      animate={reduce ? { opacity: 1 } : { opacity: 1, y: [0, -14, 0], scale: 1 }}
      whileHover={reduce ? undefined : { y: -18, scale: 1.035, transition: { duration: 0.22, ease } }}
      transition={reduce ? { duration: 0.01 } : { opacity: { delay, duration: 0.35 }, y: { delay, duration: 4.6, repeat: Infinity, ease: "easeInOut" }, scale: { delay, duration: 0.35 } }}
    >
      <span>{label}</span>
      <strong>{title}</strong>
      <p>{copy}</p>
    </motion.article>
  );
}

const hero = document.querySelector(".hero-visual");
if (hero) {
  createRoot(hero).render(<MotionHero />);
}
