const paymentLinks = window.NODE_PAYMENT_LINKS || {};
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
const allowedHosts = new Set((paymentLinks.allowedHosts || ["buy.stripe.com"]).filter((host) => reviewedPaymentHosts.has(host)));
const contactEmail = typeof paymentLinks.contactEmail === "string" ? paymentLinks.contactEmail.trim() : "";

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

function contactHref(link) {
  const pilotContactUrl = safeContactUrl(paymentLinks.pilotContactUrl);
  if (pilotContactUrl) return pilotContactUrl;
  if (!contactEmail) return null;

  const subject = encodeURIComponent(link.dataset.contactSubject || "NODE Pilot Request");
  const body = encodeURIComponent(link.dataset.contactBody || "I want to discuss NODE.");
  return `mailto:${contactEmail}?subject=${subject}&body=${body}`;
}

function safeContactUrl(value) {
  if (!value) return null;

  try {
    const url = new URL(value);
    if (url.protocol !== "https:") return null;
    if (!reviewedContactHosts.has(url.hostname)) return null;
    return url.toString();
  } catch {
    return null;
  }
}

function safeProofUrl(value) {
  if (!value) return null;

  try {
    const url = new URL(value);
    if (url.protocol !== "https:") return null;
    if (!reviewedProofHosts.has(url.hostname)) return null;
    return url.toString();
  } catch {
    return null;
  }
}

function applyExternalTarget(link, href) {
  if (href.startsWith("https://")) {
    link.target = "_blank";
    link.rel = "noopener noreferrer";
  } else {
    link.removeAttribute("target");
    link.removeAttribute("rel");
  }
}

function applyLabel(link, name) {
  const label = link.dataset[name];
  if (label) {
    link.textContent = label;
  }
}

for (const link of document.querySelectorAll("[data-checkout-plan]")) {
  const plan = link.dataset.checkoutPlan;
  const fallbackHref = link.dataset.fallbackHref || link.getAttribute("href") || "#plans";
  const liveHref = safeExternalUrl(paymentLinks[plan]);
  const emailHref = contactHref(link);

  if (liveHref) {
    link.href = liveHref;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    link.dataset.checkoutState = "live";
    applyLabel(link, "liveLabel");
  } else if (emailHref) {
    link.href = emailHref;
    applyExternalTarget(link, emailHref);
    link.dataset.checkoutState = "contact";
    applyLabel(link, "liveLabel");
  } else {
    link.href = fallbackHref;
    link.removeAttribute("target");
    link.removeAttribute("rel");
    link.dataset.checkoutState = "fallback";
    applyLabel(link, "fallbackLabel");
  }
}

for (const link of document.querySelectorAll("[data-contact-link]")) {
  const emailHref = contactHref(link);
  const fallbackHref = link.dataset.fallbackHref || link.getAttribute("href") || "#plans";
  link.href = emailHref || fallbackHref;
  if (emailHref) {
    applyExternalTarget(link, emailHref);
    applyLabel(link, "liveLabel");
  } else {
    link.removeAttribute("target");
    link.removeAttribute("rel");
    applyLabel(link, "fallbackLabel");
  }
  link.dataset.contactState = emailHref ? "contact" : "fallback";
}

for (const link of document.querySelectorAll("[data-proof-demo-link]")) {
  const proofHref = safeProofUrl(paymentLinks.proofDemoUrl);
  const fallbackHref = link.dataset.fallbackHref || link.getAttribute("href") || "#proof";
  link.href = proofHref || fallbackHref;
  if (proofHref) {
    applyExternalTarget(link, proofHref);
    applyLabel(link, "liveLabel");
  } else {
    link.removeAttribute("target");
    link.removeAttribute("rel");
    applyLabel(link, "fallbackLabel");
  }
  link.dataset.proofDemoState = proofHref ? "live" : "fallback";
}
