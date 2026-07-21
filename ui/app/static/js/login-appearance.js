const themes = new Set(["command-center", "incident-console", "command-center-classic", "security-card", "incident-console-classic"]);
const defaultTheme = "command-center";

function updateLoginThemeTime() {
  const target = document.getElementById("login-theme-current-time");
  if (!target) return;
  const render = () => {
    target.textContent = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(new Date());
  };
  render();
  window.setInterval(render, 1000);
}

export async function applyLoginAppearance() {
  document.body.classList.remove("login-appearance-ready");
  let appearance = defaultTheme;
  try {
    const response = await fetch("/api/public/login-appearance", {
      credentials: "same-origin",
      cache: "no-store",
      headers: { Accept: "application/json" }
    });
    if (response.ok) {
      const payload = await response.json();
      const selected = String(payload?.login_appearance || "").trim();
      if (themes.has(selected)) appearance = selected;
    }
  } catch {
    // Keep the default theme when the control plane is unavailable.
  } finally {
    document.body.dataset.loginAppearance = appearance;
    document.body.classList.add("login-appearance-ready");
    updateLoginThemeTime();
  }
}
