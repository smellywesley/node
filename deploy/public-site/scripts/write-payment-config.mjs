import fs from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const outputPath = path.join(root, "public", "payment-links.js");

const env = process.env;
const allowedHosts = (env.NODE_PUBLIC_PAYMENT_ALLOWED_HOSTS || "buy.stripe.com")
  .split(",")
  .map((host) => host.trim())
  .filter(Boolean);

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

const config = {
  pro: cleanPaymentUrl("NODE_PUBLIC_PRO_PAYMENT_LINK", env.NODE_PUBLIC_PRO_PAYMENT_LINK),
  pilot: cleanPaymentUrl("NODE_PUBLIC_PILOT_PAYMENT_LINK", env.NODE_PUBLIC_PILOT_PAYMENT_LINK),
  enterprise: cleanPaymentUrl("NODE_PUBLIC_ENTERPRISE_PAYMENT_LINK", env.NODE_PUBLIC_ENTERPRISE_PAYMENT_LINK),
  contactEmail: cleanEmail(env.NODE_PUBLIC_CONTACT_EMAIL),
  allowedHosts
};

const file = `window.NODE_PAYMENT_LINKS = ${JSON.stringify(config, null, 2)};\n`;
fs.writeFileSync(outputPath, file);

const liveLinks = ["pro", "pilot", "enterprise"].filter((key) => config[key]);
const contactState = config.contactEmail ? "configured" : "empty";
console.log(`wrote public/payment-links.js (${liveLinks.length} payment link(s), contact ${contactState})`);
