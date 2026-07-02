export const heavyEase = [0.32, 0.72, 0, 1];

export const huds = [
  {
    className: "policy-card",
    label: "Policy",
    title: "Forbidden path denied",
    copy: "The agent asks to touch frontend files. NODE blocks it before the write lands.",
    rotateY: -22,
    rotateX: 8,
    z: 150,
  },
  {
    className: "approval-card",
    label: "Approval",
    title: "Human gate opened",
    copy: "Consequential writes pause with digest, capability, and cost context.",
    rotateY: -16,
    rotateX: 10,
    z: 190,
  },
  {
    className: "spend-card",
    label: "Spend",
    title: "$5.00 run cap",
    copy: "Token, time, child-task, and tool budgets stay outside the model.",
    rotateY: 18,
    rotateX: 8,
    z: 130,
  },
  {
    className: "audit-card",
    label: "Audit",
    title: "Replay bundle ready",
    copy: "Every approval, denial, usage tick, and artifact is exportable proof.",
    rotateY: -12,
    rotateX: 7,
    z: 210,
  },
];

export const panels = [
  {
    label: "01 / The problem",
    title: "Agents now touch real systems.",
    copy: "Prompts are not a security boundary once an agent can write files, call tools, use secrets, or open pull requests.",
  },
  {
    label: "02 / The control plane",
    title: "NODE makes the run inspectable.",
    copy: "Every run gets identity, lifecycle, policy, budget, approval gates, recovery, replay, and audit history.",
  },
  {
    label: "03 / The buyer proof",
    title: "Trust becomes an artifact.",
    copy: "The customer sees denied actions, approved work, token cost, test result, branch diff, and redacted audit bundle.",
  },
];

export const branchLights = [
  { className: "dot-a", delay: 0, frame: 0 },
  { className: "dot-b", delay: 0.45, frame: 14 },
  { className: "dot-c", delay: 0.9, frame: 27 },
  { className: "dot-d", delay: 1.35, frame: 41 },
  { className: "dot-e", delay: 1.8, frame: 54 },
  { className: "dot-f", delay: 2.25, frame: 68 },
];

export const signalRunners = [
  { className: "runner-a", duration: 4.6, frameStart: 18, frameEnd: 122 },
  { className: "runner-b", duration: 5.2, frameStart: 42, frameEnd: 158 },
  { className: "runner-c", duration: 5.8, frameStart: 70, frameEnd: 194 },
];

export const hyperframes = {
  id: "node-signal-plane",
  fps: 30,
  durationInFrames: 240,
  width: 1920,
  height: 1080,
  beats: [
    { id: "field-ignites", frame: 0, label: "Circuit field wakes" },
    { id: "node-rises", frame: 34, label: "NODE process becomes the center" },
    { id: "policy-deny", frame: 68, label: "Policy blocks forbidden action" },
    { id: "approval-gate", frame: 104, label: "Human gate opens with context" },
    { id: "audit-proof", frame: 150, label: "Replay and audit proof lock in" },
    { id: "buyer-story", frame: 190, label: "Problem, control, proof panels scroll" },
  ],
};
