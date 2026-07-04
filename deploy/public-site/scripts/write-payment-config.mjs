import fs from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);
const outputArgIndex = args.indexOf("--output");
const outputPath = outputArgIndex >= 0 && args[outputArgIndex + 1]
  ? path.resolve(root, args[outputArgIndex + 1])
  : path.join(root, "public", "payment-links.js");
const reviewedPaymentHosts = new Set(["buy.stripe.com"]);
const reviewedContactHosts = new Set([
  "calendly.com",
  "www.calendly.com",
  "tally.so",
  "www.tally.so",
  "form.typeform.com",
  "forms.gle",
  "docs.google.com"
]);
const reviewedProofHosts = new Set([
  "youtube.com",
  "www.youtube.com",
  "youtu.be",
  "loom.com",
  "www.loom.com",
  "vimeo.com",
  "www.vimeo.com"
]);

const env = process.env;
const allowedHosts = (env.NODE_PUBLIC_PAYMENT_ALLOWED_HOSTS || "buy.stripe.com")
  .split(",")
  .map((host) => host.trim())
  .filter(Boolean);

for (const host of allowedHosts) {
  if (!reviewedPaymentHosts.has(host)) {
    throw new Error(`${host} is not a reviewed public payment host`);
  }
}

function cleanEmail(value) {
  const email = (value || "").trim();
  if (!email) return "";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    throw new Error("NODE_PUBLIC_CONTACT_EMAIL must be a valid public contact email");
  }
  return email;
}

function cleanPaymentUrl(name, value) {
  const raw = (value || "").trim();
  if (!raw) return "";

  const url = new URL(raw);
  if (url.protocol !== "https:") {
    throw new Error(`${name} must be an https URL`);
  }
  if (!allowedHosts.includes(url.hostname)) {
    throw new Error(`${name} host ${url.hostname} is not listed in NODE_PUBLIC_PAYMENT_ALLOWED_HOSTS`);
  }
  return url.toString();
}

function cleanContactUrl(name, value) {
  const raw = (value || "").trim();
  if (!raw) return "";

  const url = new URL(raw);
  if (url.protocol !== "https:") {
    throw new Error(`${name} must be an https URL`);
  }
  if (!reviewedContactHosts.has(url.hostname)) {
    throw new Error(`${name} host ${url.hostname} is not a reviewed public pilot-intake host`);
  }
  return url.toString();
}

function cleanProofUrl(name, value) {
  const raw = (value || "").trim();
  if (!raw) return "";

  const url = new URL(raw);
  if (url.protocol !== "https:") {
    throw new Error(`${name} must be an https URL`);
  }
  if (!reviewedProofHosts.has(url.hostname)) {
    throw new Error(`${name} host ${url.hostname} is not a reviewed public proof-video host`);
  }
  return url.toString();
}

const config = {
  pro: cleanPaymentUrl("NODE_PUBLIC_PRO_PAYMENT_LINK", env.NODE_PUBLIC_PRO_PAYMENT_LINK),
  pilot: cleanPaymentUrl("NODE_PUBLIC_PILOT_PAYMENT_LINK", env.NODE_PUBLIC_PILOT_PAYMENT_LINK),
  enterprise: cleanPaymentUrl("NODE_PUBLIC_ENTERPRISE_PAYMENT_LINK", env.NODE_PUBLIC_ENTERPRISE_PAYMENT_LINK),
  pilotContactUrl: cleanContactUrl("NODE_PUBLIC_PILOT_CONTACT_URL", env.NODE_PUBLIC_PILOT_CONTACT_URL),
  proofDemoUrl: cleanProofUrl("NODE_PUBLIC_PROOF_DEMO_URL", env.NODE_PUBLIC_PROOF_DEMO_URL),
  contactEmail: cleanEmail(env.NODE_PUBLIC_CONTACT_EMAIL),
  allowedHosts
};

const file = `window.NODE_PAYMENT_LINKS = ${JSON.stringify(config, null, 2)};\n`;
fs.mkdirSync(path.dirname(outputPath), {recursive: true});
fs.writeFileSync(outputPath, file);

const liveLinks = ["pro", "pilot", "enterprise"].filter((key) => config[key]);
const contactState = config.contactEmail || config.pilotContactUrl ? "configured" : "empty";
console.log(`wrote ${path.relative(root, outputPath)} (${liveLinks.length} payment link(s), contact ${contactState})`);
