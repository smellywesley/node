import React from "react";
import { AbsoluteFill, Easing, interpolate, spring, useCurrentFrame, useVideoConfig } from "remotion";
import { branchLights, huds, hyperframes, panels } from "../hyperframes.js";

const cardPositions = [
  { x: 1060, y: 160, rotate: -7 },
  { x: 1390, y: 245, rotate: 6 },
  { x: 780, y: 690, rotate: -5 },
  { x: 1360, y: 680, rotate: 5 },
];

const terminalLines = [
  "agentos run task: fix-backend-only",
  "policy frontend.write -> DENIED",
  "approval gate -> opened by Alex R.",
  "audit bundle -> replay-ready",
];

function fade(frame, start, end) {
  return interpolate(frame, [start, end], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
}

export function NodeSignalPlane() {
  const frame = useCurrentFrame();
  const config = useVideoConfig();
  const nodeLift = spring({ frame: frame - 28, fps: config.fps, config: { damping: 18, stiffness: 90 } });
  const boardDrift = interpolate(frame, [0, hyperframes.durationInFrames], [-34, 38]);
  const runner = interpolate(frame % 120, [0, 92, 120], [-220, 760, 880], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const typedCount = Math.min(terminalLines.length, Math.floor(Math.max(0, frame - 60) / 22) + 1);

  return (
    <AbsoluteFill style={styles.canvas}>
      <div style={styles.vignette} />
      <div
        style={{
          ...styles.board,
          transform: `translateY(${boardDrift}px) rotateX(61deg) rotateZ(-15deg)`,
        }}
      >
        <div style={{ ...styles.runner, transform: `translateX(${runner}px)` }} />
        {branchLights.map((light, index) => {
          const pulse = fade((frame - light.frame + 80) % 80, 0, 24);
          return (
            <div
              key={light.className}
              style={{
                ...styles.dot,
                left: `${18 + index * 12}%`,
                top: `${index % 2 === 0 ? 57 - index * 3 : 28 + index * 4}%`,
                opacity: 0.28 + pulse * 0.72,
                transform: `scale(${0.8 + pulse * 0.55})`,
              }}
            />
          );
        })}
      </div>
      <div style={{ ...styles.node, transform: `translate(-50%, -50%) translateY(${(1 - nodeLift) * 38}px) scale(${0.9 + nodeLift * 0.1})` }}>
        <span style={styles.nodeLabel}>NODE</span>
        <strong style={styles.nodeTitle}>Controlled agent process</strong>
        <em style={styles.nodeMeta}>policy live</em>
      </div>
      {huds.map((hud, index) => {
        const entrance = fade(frame, 32 + index * 18, 64 + index * 18);
        const pos = cardPositions[index];
        return (
          <div
            key={hud.title}
            style={{
              ...styles.card,
              left: pos.x,
              top: pos.y,
              opacity: entrance,
              transform: `translateY(${(1 - entrance) * 32}px) rotate(${pos.rotate}deg)`,
            }}
          >
            <span style={styles.cardLabel}>{hud.label}</span>
            <strong style={styles.cardTitle}>{hud.title}</strong>
            <p style={styles.cardCopy}>{hud.copy}</p>
          </div>
        );
      })}
      <div style={styles.terminal}>
        <span style={styles.cardLabel}>Replay terminal</span>
        {terminalLines.slice(0, typedCount).map((line, index) => (
          <p key={line} style={{ ...styles.terminalLine, opacity: fade(frame, 60 + index * 22, 72 + index * 22) }}>
            {line}
          </p>
        ))}
      </div>
      <div style={styles.panelRail}>
        {panels.map((panel, index) => (
          <div key={panel.label} style={{ ...styles.panel, opacity: fade(frame, 150 + index * 16, 176 + index * 16) }}>
            <span style={styles.cardLabel}>{panel.label}</span>
            <strong style={styles.panelTitle}>{panel.title}</strong>
            <p style={styles.cardCopy}>{panel.copy}</p>
          </div>
        ))}
      </div>
    </AbsoluteFill>
  );
}

const styles = {
  canvas: {
    overflow: "hidden",
    color: "#eefcf6",
    background: "radial-gradient(circle at 70% 40%, rgba(72,246,178,.24), transparent 26%), linear-gradient(135deg, #020403, #06100d 48%, #020403)",
    fontFamily: '"Plus Jakarta Sans", "Geist", "Aptos", sans-serif',
  },
  vignette: {
    position: "absolute",
    inset: 0,
    background: "radial-gradient(circle at 50% 50%, transparent 38%, rgba(0,0,0,.62) 100%)",
  },
  board: {
    position: "absolute",
    left: 560,
    top: 140,
    width: 1120,
    height: 690,
    border: "1px solid rgba(131,255,204,.22)",
    borderRadius: 52,
    background: "linear-gradient(rgba(131,255,204,.12) 1px, transparent 1px), linear-gradient(90deg, rgba(131,255,204,.1) 1px, transparent 1px), linear-gradient(135deg, rgba(7,35,29,.92), rgba(3,10,9,.96))",
    backgroundSize: "54px 54px, 54px 54px, auto",
    boxShadow: "0 70px 150px rgba(0,0,0,.46), inset 0 0 90px rgba(72,246,178,.1)",
  },
  runner: {
    position: "absolute",
    left: 130,
    top: 338,
    width: 140,
    height: 8,
    borderRadius: 999,
    background: "linear-gradient(90deg, transparent, #48f6b2, #82f7ff, transparent)",
    boxShadow: "0 0 26px rgba(72,246,178,.85)",
  },
  dot: {
    position: "absolute",
    width: 22,
    height: 22,
    borderRadius: 999,
    background: "#48f6b2",
    boxShadow: "0 0 30px rgba(72,246,178,.95), 0 0 70px rgba(72,246,178,.42)",
  },
  node: {
    position: "absolute",
    left: "58%",
    top: "50%",
    width: 430,
    minHeight: 190,
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    border: "1px solid rgba(199,255,233,.34)",
    borderRadius: 44,
    background: "linear-gradient(145deg, rgba(8,31,27,.96), rgba(4,12,11,.94))",
    boxShadow: "0 34px 90px rgba(0,0,0,.42), 0 0 86px rgba(72,246,178,.28), inset 0 1px 0 rgba(255,255,255,.16)",
  },
  nodeLabel: { color: "#48f6b2", fontSize: 52, letterSpacing: 10, fontWeight: 500 },
  nodeTitle: { marginTop: 12, fontSize: 22 },
  nodeMeta: { marginTop: 8, color: "rgba(238,252,246,.62)", fontStyle: "normal" },
  card: {
    position: "absolute",
    width: 310,
    minHeight: 158,
    padding: 24,
    border: "1px solid rgba(199,255,233,.22)",
    borderRadius: 28,
    background: "linear-gradient(145deg, rgba(8,31,27,.86), rgba(4,12,11,.78))",
    boxShadow: "0 32px 90px rgba(0,0,0,.36), inset 0 1px 0 rgba(255,255,255,.12)",
  },
  cardLabel: { display: "block", color: "#77ffd1", fontSize: 14, fontWeight: 900, letterSpacing: 2.4, textTransform: "uppercase" },
  cardTitle: { display: "block", marginTop: 14, fontSize: 24, lineHeight: 1.15 },
  cardCopy: { marginTop: 10, color: "rgba(222,248,238,.68)", fontSize: 17, lineHeight: 1.55 },
  terminal: {
    position: "absolute",
    left: 120,
    bottom: 116,
    width: 560,
    padding: 28,
    border: "1px solid rgba(131,255,204,.2)",
    borderRadius: 30,
    background: "rgba(4,12,11,.84)",
    boxShadow: "0 34px 90px rgba(0,0,0,.38)",
  },
  terminalLine: { margin: "12px 0 0", color: "#d1fadf", fontFamily: "Consolas, monospace", fontSize: 21 },
  panelRail: { position: "absolute", right: 92, bottom: 80, width: 430, display: "grid", gap: 16 },
  panel: {
    padding: 22,
    border: "1px solid rgba(131,255,204,.18)",
    borderRadius: 26,
    background: "rgba(7,28,23,.78)",
    boxShadow: "inset 0 1px 0 rgba(255,255,255,.1)",
  },
  panelTitle: { display: "block", marginTop: 10, fontSize: 21 },
};
