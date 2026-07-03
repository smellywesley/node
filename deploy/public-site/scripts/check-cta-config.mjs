import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";
import {fileURLToPath} from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const html = fs.readFileSync(path.join(root, "index.html"), "utf8");
const configSource = fs.readFileSync(path.join(root, "public", "payment-links.js"), "utf8");
const checkoutSource = fs.readFileSync(path.join(root, "src", "checkout-links.js"), "utf8");

const sandbox = {window: {}};
vm.createContext(sandbox);
vm.runInContext(configSource, sandbox, {filename: "payment-links.js"});

const config = sandbox.window.NODE_PAYMENT_LINKS || {};
const allowedHosts = new Set(config.allowedHosts || ["buy.stripe.com"]);
const contactEmail = typeof config.contactEmail === "string" ? config.contactEmail.trim() : "";
const configuredContact = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(contactEmail);
const checkoutPlanPattern = /<a\b[^>]*\bdata-checkout-plan="([^"]+)"[^>]*>/g;
const contactLinkPattern = /<a\b[^>]*\bdata-contact-link\b[^>]*>/g;
const fallbackPattern = /\bdata-fallback-href="([^"]*)"/;
const anchorPattern = /<a\b[^>]*>/g;
const attributePattern = /([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:="([^"]*)")?/g;

function safeExternalUrl(value) {
  if (!value) return null;

  try {
    const url = new URL(value);
    if (url.protocol !== "https:") return null;
    if (!allowedHosts.has(url.hostname)) return null;
    return url.toString();
  } catch {
    return null;
  }
}

function attr(tag, pattern) {
  const match = tag.match(pattern);
  return match ? match[1] : "";
}

function datasetKey(name) {
  return name
    .slice("data-".length)
    .replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
}

class LinkElement {
  constructor(tag) {
    this.attributes = new Map();
    this.dataset = {};
    this.textContent = "";

    for (const match of tag.matchAll(attributePattern)) {
      const name = match[1];
      const value = match[2] || "";
      if (name === "a") continue;
      this.attributes.set(name, value);
      if (name.startsWith("data-")) {
        this.dataset[datasetKey(name)] = value;
      }
    }
  }

  getAttribute(name) {
    return this.attributes.get(name) || null;
  }

  setAttribute(name, value) {
    this.attributes.set(name, value);
    if (name.startsWith("data-")) {
      this.dataset[datasetKey(name)] = value;
    }
  }

  removeAttribute(name) {
    this.attributes.delete(name);
  }

  get href() {
    return this.attributes.get("href") || "";
  }

  set href(value) {
    this.attributes.set("href", value);
  }

  get target() {
    return this.attributes.get("target") || "";
  }

  set target(value) {
    this.attributes.set("target", value);
  }

  get rel() {
    return this.attributes.get("rel") || "";
  }

  set rel(value) {
    this.attributes.set("rel", value);
  }
}

const links = [...html.matchAll(anchorPattern)].map((match) => new LinkElement(match[0]));
const runtimeSandbox = {
  URL,
  window: {NODE_PAYMENT_LINKS: config},
  document: {
    querySelectorAll(selector) {
      if (selector === "[data-checkout-plan]") {
        return links.filter((link) => link.dataset.checkoutPlan !== undefined);
      }
      if (selector === "[data-contact-link]") {
        return links.filter((link) => link.dataset.contactLink !== undefined);
      }
      return [];
    }
  }
};

vm.createContext(runtimeSandbox);
vm.runInContext(checkoutSource, runtimeSandbox, {filename: "checkout-links.js"});

const failures = [];
const seenPlans = new Set();

for (const match of html.matchAll(checkoutPlanPattern)) {
  const tag = match[0];
  const plan = match[1];
  seenPlans.add(plan);
  const liveUrl = safeExternalUrl(config[plan]);
  const fallback = attr(tag, fallbackPattern);

  if (!liveUrl && !configuredContact) {
    failures.push(`${plan}: missing allowed Stripe/payment URL and missing real contactEmail`);
  }

  if (fallback === "#plans") {
    failures.push(`${plan}: fallback points back to #plans, which makes the commercial CTA feel dead`);
  }
}

if (seenPlans.size === 0) {
  failures.push("no data-checkout-plan CTAs found");
}

if (html.match(contactLinkPattern) && !configuredContact) {
  failures.push("data-contact-link exists but contactEmail is empty or invalid");
}

for (const link of links.filter((item) => item.dataset.checkoutPlan !== undefined)) {
  if (link.dataset.checkoutState === "fallback") {
    failures.push(`${link.dataset.checkoutPlan}: runtime checkout state is fallback instead of live/contact`);
  }
}

for (const link of links.filter((item) => item.dataset.contactLink !== undefined)) {
  if (link.dataset.contactState === "fallback") {
    failures.push("runtime contact CTA state is fallback instead of contact");
  }
}

if (failures.length > 0) {
  console.error("CTA configuration check failed:");
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  console.error("Fix: add live allowed payment links or a real pilot contact email in public/payment-links.js.");
  process.exit(1);
}

console.log("CTA configuration check passed");
