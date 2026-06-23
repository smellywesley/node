"use strict";

const state = {
  operatorToken: "",
  selectedProcessID: "",
  processes: [],
  refreshTimer: null,
};

const terminalStates = new Set(["succeeded", "failed", "cancelled"]);

const elements = {
  authPanel: document.querySelector("#auth-panel"),
  authForm: document.querySelector("#auth-form"),
  operatorToken: document.querySelector("#operator-token"),
  controlPlane: document.querySelector("#control-plane"),
  healthDot: document.querySelector("#health-dot"),
  healthLabel: document.querySelector("#health-label"),
  refreshButton: document.querySelector("#refresh-button"),
  processRows: document.querySelector("#process-rows"),
  processEmpty: document.querySelector("#process-empty"),
  approvalList: document.querySelector("#approval-list"),
  approvalEmpty: document.querySelector("#approval-empty"),
  approvalCount: document.querySelector("#approval-count"),
  launchToggle: document.querySelector("#launch-toggle"),
  launchPanel: document.querySelector("#launch-panel"),
  launchForm: document.querySelector("#launch-form"),
  launchCancel: document.querySelector("#launch-cancel"),
  manifestInput: document.querySelector("#manifest-input"),
  detailEmpty: document.querySelector("#detail-empty"),
  detailContent: document.querySelector("#detail-content"),
  detailName: document.querySelector("#detail-name"),
  detailID: document.querySelector("#detail-id"),
  detailState: document.querySelector("#detail-state"),
  detailTask: document.querySelector("#detail-task"),
  budgetGrid: document.querySelector("#budget-grid"),
  capabilityList: document.querySelector("#capability-list"),
  eventList: document.querySelector("#event-list"),
  eventCount: document.querySelector("#event-count"),
  suspendButton: document.querySelector("#suspend-button"),
  resumeButton: document.querySelector("#resume-button"),
  cancelButton: document.querySelector("#cancel-button"),
  replayButton: document.querySelector("#replay-button"),
  auditButton: document.querySelector("#audit-button"),
  toast: document.querySelector("#toast"),
  statTotal: document.querySelector("#stat-total"),
  statActive: document.querySelector("#stat-active"),
  statApproval: document.querySelector("#stat-approval"),
  statTokens: document.querySelector("#stat-tokens"),
  statCost: document.querySelector("#stat-cost"),
};

function element(tag, className = "", text = "") {
  const node = document.createElement(tag);
  if (className) {
    node.className = className;
  }
  if (text !== "") {
    node.textContent = text;
  }
  return node;
}

function clear(node) {
  while (node.firstChild) {
    node.removeChild(node.firstChild);
  }
}

function setHidden(node, hidden) {
  node.classList.toggle("hidden", hidden);
}

function showToast(message, isError = false) {
  elements.toast.textContent = message;
  elements.toast.classList.toggle("error", isError);
  setHidden(elements.toast, false);
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => setHidden(elements.toast, true), 5000);
}

function readLaunchToken() {
  const params = new URLSearchParams(window.location.hash.slice(1));
  const token = params.get("token");
  const approverToken = params.get("approver_token");
  if (token) {
    window.sessionStorage.setItem("agentos.operatorToken", token);
  }
  if (approverToken) {
    window.sessionStorage.setItem("agentos.approverToken", approverToken);
  }
  if (token || approverToken) {
    window.history.replaceState(null, "", window.location.pathname + window.location.search);
  }
}

function loadToken() {
  readLaunchToken();
  state.operatorToken = window.sessionStorage.getItem("agentos.operatorToken") || "";
  setHidden(elements.authPanel, Boolean(state.operatorToken));
  setHidden(elements.controlPlane, !state.operatorToken);
}

async function api(path, options = {}, credential = "operator") {
  const token = credential === "operator"
    ? state.operatorToken
    : window.sessionStorage.getItem("agentos.approverToken") || "";
  const headers = new Headers(options.headers || {});
  headers.set("Authorization", `Bearer ${token}`);
  if (options.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const response = await window.fetch(path, {...options, headers});
  const raw = await response.text();
  let body = null;
  if (raw) {
    try {
      body = JSON.parse(raw);
    } catch {
      body = raw;
    }
  }
  if (!response.ok) {
    const message = body && body.error ? body.error : `${response.status} ${response.statusText}`;
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return body;
}

function formatNumber(value) {
  return new Intl.NumberFormat().format(value || 0);
}

function formatCost(value) {
  return `$${Number(value || 0).toFixed(4)}`;
}

function formatTime(value) {
  if (!value) {
    return "n/a";
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(new Date(value));
}

function statePill(processState) {
  return element("span", `state-pill state-${processState}`, processState.replace("_", " "));
}

function renderStats(processes, approvals) {
  const active = processes.filter((process) => !terminalStates.has(process.state)).length;
  const pending = approvals.filter((approval) => approval.status === "pending").length;
  const tokens = processes.reduce((total, process) => total + Number(process.usage.tokens || 0), 0);
  const cost = processes.reduce((total, process) => total + Number(process.usage.cost_usd || 0), 0);
  elements.statTotal.textContent = formatNumber(processes.length);
  elements.statActive.textContent = formatNumber(active);
  elements.statApproval.textContent = formatNumber(pending);
  elements.statTokens.textContent = formatNumber(tokens);
  elements.statCost.textContent = formatCost(cost);
}

function renderProcesses(processes) {
  clear(elements.processRows);
  setHidden(elements.processEmpty, processes.length > 0);
  for (const process of processes) {
    const row = element("tr", process.id === state.selectedProcessID ? "selected" : "");
    const identity = element("td");
    const select = element("button", "process-button");
    select.type = "button";
    select.append(element("strong", "", process.name), element("code", "", process.id));
    select.addEventListener("click", () => selectProcess(process.id));
    identity.append(select);
    row.append(identity);

    const stateCell = element("td");
    stateCell.append(statePill(process.state));
    row.append(
      stateCell,
      element("td", "", String(process.attempt || 0)),
      element("td", "", formatNumber(process.usage.tokens)),
      element("td", "", formatCost(process.usage.cost_usd)),
      element("td", "", formatTime(process.updated_at)),
    );
    elements.processRows.append(row);
  }
}

function approvalToken() {
  let token = window.sessionStorage.getItem("agentos.approverToken") || "";
  if (token) {
    return token;
  }
  token = window.prompt("Enter AGENTOS_APPROVER_TOKEN. It stays in this browser tab only.") || "";
  if (token) {
    window.sessionStorage.setItem("agentos.approverToken", token);
  }
  return token;
}

async function decideApproval(id, decision) {
  if (!approvalToken()) {
    return;
  }
  const reason = window.prompt(`Reason for ${decision === "approved" ? "approval" : "denial"} (optional):`) || "";
  try {
    await api(`/v1/approvals/${encodeURIComponent(id)}/${decision}`, {
      method: "POST",
      body: JSON.stringify({reason}),
    }, "approver");
    showToast(`Action ${decision}.`);
    await refreshAll();
  } catch (error) {
    if (error.status === 401) {
      window.sessionStorage.removeItem("agentos.approverToken");
    }
    showToast(error.message, true);
  }
}

function renderApprovals(approvals) {
  const pending = approvals.filter((approval) => approval.status === "pending");
  clear(elements.approvalList);
  setHidden(elements.approvalEmpty, pending.length > 0);
  elements.approvalCount.textContent = `${pending.length} pending`;
  for (const approval of pending) {
    const card = element("article", "approval-card");
    const copy = element("div");
    copy.append(
      element("strong", "", approval.action),
      element("p", "", approval.resource || "No resource"),
      element("code", "", `process ${approval.process_id} · digest ${approval.action_digest.slice(0, 12)}`),
    );
    const actions = element("div", "approval-actions");
    const approve = element("button", "button button-primary", "Approve");
    approve.type = "button";
    approve.addEventListener("click", () => decideApproval(approval.id, "approved"));
    const deny = element("button", "button button-danger", "Deny");
    deny.type = "button";
    deny.addEventListener("click", () => decideApproval(approval.id, "denied"));
    actions.append(approve, deny);
    card.append(copy, actions);
    elements.approvalList.append(card);
  }
}

function meter(label, used, limit, formatter = formatNumber) {
  const item = element("div", "budget-item");
  item.append(
    element("span", "", label),
    element("strong", "", limit > 0 ? `${formatter(used)} / ${formatter(limit)}` : `${formatter(used)} / unlimited`),
  );
  const track = element("div", "meter");
  const fill = element("progress");
  const percentage = limit > 0 ? Math.min(100, (Number(used) / Number(limit)) * 100) : 0;
  fill.max = 100;
  fill.value = percentage;
  fill.setAttribute("aria-label", `${label} budget used`);
  track.append(fill);
  item.append(track);
  return item;
}

function renderBudget(process) {
  clear(elements.budgetGrid);
  const budget = process.manifest.budget || {};
  elements.budgetGrid.append(
    meter("Tokens", process.usage.tokens, budget.max_tokens),
    meter("Cost", process.usage.cost_usd, budget.max_cost_usd, formatCost),
    meter("Duration limit", 0, budget.max_duration_seconds, (value) => `${value || 0}s`),
    meter("Child ceiling", 0, budget.max_children),
  );
}

function capabilityRow(label, values) {
  const row = element("div", "capability-row");
  row.append(element("strong", "", label));
  const valueList = element("div", "capability-values");
  const entries = Array.isArray(values) ? values : [];
  if (entries.length === 0) {
    valueList.append(element("code", "", "none"));
  } else {
    for (const value of entries) {
      valueList.append(element("code", "", String(value)));
    }
  }
  row.append(valueList);
  return row;
}

function renderCapabilities(process) {
  clear(elements.capabilityList);
  const capabilities = process.manifest.capabilities || {};
  elements.capabilityList.append(
    capabilityRow("Tools", capabilities.tools),
    capabilityRow("Read paths", capabilities.filesystem_read),
    capabilityRow("Write paths", capabilities.filesystem_write),
    capabilityRow("Network", capabilities.network_destinations),
    capabilityRow("Secrets", capabilities.secrets),
  );
}

function renderEvents(events) {
  clear(elements.eventList);
  elements.eventCount.textContent = `${events.length} events`;
  const newestFirst = [...events].reverse();
  for (const event of newestFirst) {
    const item = element("li", "event-item");
    item.append(element("span", "event-marker"));
    const copy = element("div", "event-copy");
    copy.append(element("strong", "", event.type));
    if (event.data && Object.keys(event.data).length > 0) {
      const details = element("pre");
      details.textContent = JSON.stringify(event.data, null, 2);
      copy.append(details);
    }
    item.append(copy, element("time", "event-time", formatTime(event.created_at)));
    elements.eventList.append(item);
  }
}

function setProcessActions(process) {
  elements.suspendButton.disabled = !["queued", "running", "waiting_approval"].includes(process.state);
  elements.resumeButton.disabled = process.state !== "suspended";
  elements.cancelButton.disabled = terminalStates.has(process.state);
}

async function selectProcess(id) {
  state.selectedProcessID = id;
  renderProcesses(state.processes);
  try {
    const [process, events] = await Promise.all([
      api(`/v1/processes/${encodeURIComponent(id)}`),
      api(`/v1/processes/${encodeURIComponent(id)}/events`),
    ]);
    setHidden(elements.detailEmpty, true);
    setHidden(elements.detailContent, false);
    elements.detailName.textContent = process.name;
    elements.detailID.textContent = process.id;
    elements.detailTask.textContent = process.manifest.task;
    elements.detailState.className = `state-pill state-${process.state}`;
    elements.detailState.textContent = process.state.replace("_", " ");
    renderBudget(process);
    renderCapabilities(process);
    renderEvents(events);
    setProcessActions(process);
  } catch (error) {
    showToast(error.message, true);
  }
}

async function transition(action) {
  if (!state.selectedProcessID) {
    return;
  }
  if (action === "cancel" && !window.confirm("Cancel this process and all of its descendants?")) {
    return;
  }
  try {
    await api(`/v1/processes/${encodeURIComponent(state.selectedProcessID)}/${action}`, {
      method: "POST",
      body: "{}",
    });
    showToast(`Process ${action} request accepted.`);
    await refreshAll();
  } catch (error) {
    showToast(error.message, true);
  }
}

async function replaySelected() {
  if (!state.selectedProcessID) {
    return;
  }
  try {
    const result = await api(`/v1/processes/${encodeURIComponent(state.selectedProcessID)}/replay`);
    showToast(`Replay rebuilt state "${result.state}" with side effects disabled.`);
  } catch (error) {
    showToast(error.message, true);
  }
}

async function exportAudit() {
  if (!state.selectedProcessID) {
    return;
  }
  try {
    const audit = await api(`/v1/processes/${encodeURIComponent(state.selectedProcessID)}/audit`);
    const blob = new Blob([JSON.stringify(audit, null, 2) + "\n"], {type: "application/json"});
    const url = URL.createObjectURL(blob);
    const link = element("a");
    link.href = url;
    link.download = `agentos-audit-${state.selectedProcessID}.json`;
    document.body.append(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    showToast("Redacted audit bundle exported.");
  } catch (error) {
    showToast(error.message, true);
  }
}

async function refreshAll() {
  if (!state.operatorToken) {
    return;
  }
  try {
    const [health, processes, approvals] = await Promise.all([
      window.fetch("/v1/health").then((response) => response.json()),
      api("/v1/processes"),
      api("/v1/approvals"),
    ]);
    state.processes = processes;
    elements.healthDot.className = "health-dot online";
    elements.healthLabel.textContent = health.status === "ok" ? "Daemon online" : "Daemon degraded";
    renderStats(processes, approvals);
    renderProcesses(processes);
    renderApprovals(approvals);
    if (state.selectedProcessID && processes.some((process) => process.id === state.selectedProcessID)) {
      await selectProcess(state.selectedProcessID);
    } else if (processes.length > 0 && !state.selectedProcessID) {
      await selectProcess(processes[0].id);
    }
    setHidden(elements.authPanel, true);
    setHidden(elements.controlPlane, false);
  } catch (error) {
    elements.healthDot.className = "health-dot offline";
    elements.healthLabel.textContent = "Connection failed";
    if (error.status === 401) {
      state.operatorToken = "";
      window.sessionStorage.removeItem("agentos.operatorToken");
      setHidden(elements.authPanel, false);
      setHidden(elements.controlPlane, true);
    }
    showToast(error.message, true);
  }
}

elements.authForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  state.operatorToken = elements.operatorToken.value.trim();
  window.sessionStorage.setItem("agentos.operatorToken", state.operatorToken);
  await refreshAll();
});

elements.refreshButton.addEventListener("click", refreshAll);
elements.launchToggle.addEventListener("click", () => setHidden(elements.launchPanel, false));
elements.launchCancel.addEventListener("click", () => setHidden(elements.launchPanel, true));
elements.suspendButton.addEventListener("click", () => transition("suspend"));
elements.resumeButton.addEventListener("click", () => transition("resume"));
elements.cancelButton.addEventListener("click", () => transition("cancel"));
elements.replayButton.addEventListener("click", replaySelected);
elements.auditButton.addEventListener("click", exportAudit);

elements.launchForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const manifest = elements.manifestInput.value.trim();
  if (!manifest) {
    showToast("Enter a YAML manifest first.", true);
    return;
  }
  try {
    const process = await api("/v1/processes", {
      method: "POST",
      headers: {"Content-Type": "application/yaml"},
      body: manifest,
    });
    state.selectedProcessID = process.id;
    elements.manifestInput.value = "";
    setHidden(elements.launchPanel, true);
    showToast(`Process ${process.name} created.`);
    await refreshAll();
  } catch (error) {
    showToast(error.message, true);
  }
});

loadToken();
refreshAll();
state.refreshTimer = window.setInterval(refreshAll, 2500);
