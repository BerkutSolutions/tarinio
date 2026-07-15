function renderCountryFlag(code) {
  const token = String(code || "").trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(token)) {
    return "";
  }
  const src = `/static/flags/16x12/${token.toLowerCase()}.png`;
  return `<img class="country-flag-img" src="${src}" width="16" height="12" alt="${token}" loading="lazy" onerror="this.style.display='none';this.nextSibling.style.display='inline-block'"><span class="country-flag-fallback" style="display:none">${token}</span>`;
}

export { renderCountryFlag };
