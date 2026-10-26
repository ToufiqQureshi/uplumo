// Public pages share the app's saved preference; no account or API is needed.
(() => {
  const key = "theme-preference";
  const media = window.matchMedia("(prefers-color-scheme: light)");
  let unsaved = null;
  function preference() {
    if (unsaved) return unsaved;
    try {
      const value = window.localStorage.getItem(key);
      return value === "light" || value === "dark" ? value : "system";
    } catch {
      return "system";
    }
  }
  function apply() {
    const value = preference();
    const light = value === "light" || (value === "system" && media.matches);
    document.documentElement.setAttribute(
      "data-theme",
      light ? "uplumo" : "uplumo-dark",
    );
    const meta = document.querySelector('meta[name="theme-color"]');
    if (meta) meta.content = light ? "#fafcf9" : "#07090a";
    const control = document.getElementById("site-theme");
    if (control) control.value = value;
  }
  // Runs before CSS to avoid a wrong-theme flash; CSP permits self-hosted JS.
  apply();
  media.addEventListener("change", apply);
  window.addEventListener("storage", (event) => {
    if (event.key === null || event.key === key) {
      unsaved = null;
      apply();
    }
  });
  document.addEventListener("DOMContentLoaded", () => {
    const control = document.getElementById("site-theme");
    control.disabled = false;
    control.addEventListener("change", () => {
      const value = control.value;
      try {
        if (value === "system") window.localStorage.removeItem(key);
        else window.localStorage.setItem(key, value);
        unsaved = null;
      } catch {
        unsaved = value;
      }
      apply();
      window.dispatchEvent(new Event("theme-preference-change"));
    });
    apply();
  });
})();
