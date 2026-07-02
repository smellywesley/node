const paymentLinks = window.NODE_PAYMENT_LINKS || {};
const allowedHosts = new Set(paymentLinks.allowedHosts || ["buy.stripe.com"]);
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
  if (!contactEmail) return null;

  const subject = encodeURIComponent(link.dataset.contactSubject || "NODE Pilot Request");
  const body = encodeURIComponent(link.dataset.contactBody || "I want to discuss NODE.");
  return `mailto:${contactEmail}?subject=${subject}&body=${body}`;
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
    if (link.dataset.liveLabel) {
      link.textContent = link.dataset.liveLabel;
    }
  } else if (emailHref) {
    link.href = emailHref;
    link.removeAttribute("target");
    link.removeAttribute("rel");
    link.dataset.checkoutState = "contact";
  } else {
    link.href = fallbackHref;
    link.removeAttribute("target");
    link.removeAttribute("rel");
    link.dataset.checkoutState = "fallback";
  }
}

for (const link of document.querySelectorAll("[data-contact-link]")) {
  const emailHref = contactHref(link);
  const fallbackHref = link.dataset.fallbackHref || link.getAttribute("href") || "#plans";
  link.href = emailHref || fallbackHref;
  link.dataset.contactState = emailHref ? "contact" : "fallback";
}
