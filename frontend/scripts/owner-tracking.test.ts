import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import { describe, expect, it } from "vitest";

const loader = readFileSync(
  new URL("../../backend/internal/site/static/analytics.js", import.meta.url),
  "utf8",
);
function browser(path = "/pricing", hostname = "uplumo.com", optedOut = false) {
  const scripts: Record<string, unknown>[] = [];
  const listeners: Record<string, (event: unknown) => void> = {};
  const window: { dataLayer: unknown[][]; [key: string]: unknown } = {
    dataLayer: [],
    addEventListener: (name: string, fn: (event: unknown) => void) => {
      listeners[name] = fn;
    },
  };
  const context = {
    location: { hostname, pathname: path },
    navigator: { doNotTrack: optedOut ? "1" : "0" },
    window,
    document: {
      createElement: () => ({
        setAttribute(name: string, value: string) {
          Object.assign(this, { [name]: value });
        },
      }),
      head: {
        appendChild: (script: Record<string, unknown>) => scripts.push(script),
      },
      addEventListener: (name: string, fn: (event: unknown) => void) => {
        listeners[name] = fn;
      },
    },
  };
  return { context, scripts, listeners, window };
}
describe("owner tracking boundaries", () => {
  it("initializes after client navigation leaves the login page, only once", () => {
    const b = browser("/sign-in");
    runInNewContext(loader, b.context);
    expect(b.scripts).toHaveLength(0);
    b.context.location.pathname = "/p/example/billing/plan";
    b.listeners["uplumo:navigation"]({});
    b.listeners["uplumo:navigation"]({});
    expect(b.scripts).toHaveLength(2);
  });
  it("initializes both trackers once and records named intent without form data", () => {
    const b = browser();
    runInNewContext(loader, b.context);
    runInNewContext(loader, b.context);
    expect(b.scripts).toHaveLength(2);
    expect(b.scripts[0].src).toBe(
      "https://www.googletagmanager.com/gtag/js?id=G-NTP32DNMN2",
    );
    expect(b.scripts[1]["data-site"]).toBe("DDVAYTG2IADAYZQSEM4GMU6HFA");
    expect(b.window.dataLayer).toHaveLength(2);
    b.listeners.click({
      target: { closest: () => ({ getAttribute: () => "plan_select_pro" }) },
    });
    expect(Array.from(b.window.dataLayer[2])).toEqual([
      "event",
      "plan_select_pro",
    ]);
  });
  it("does not record disabled checkout controls", () => {
    const b = browser();
    runInNewContext(loader, b.context);
    b.listeners.click({
      target: {
        closest: () => ({ disabled: true, getAttribute: () => "checkout" }),
      },
    });
    expect(b.window.dataLayer).toHaveLength(2);
  });
  it.each([
    "/sign-in",
    "/verify-email",
    "/reset-password",
    "/accept-invitation/id",
    "/api/auth/callback/google",
  ])("excludes authentication route %s", (path) => {
    const b = browser(path);
    runInNewContext(loader, b.context);
    expect(b.scripts).toHaveLength(0);
  });
  it("excludes preview hosts and privacy opt-outs", () => {
    for (const b of [
      browser("/", "localhost"),
      browser("/", "preview.up.railway.app"),
      browser("/", "uplumo.com", true),
    ]) {
      runInNewContext(loader, b.context);
      expect(b.scripts).toHaveLength(0);
    }
    const b = browser();
    Object.assign(b.context.navigator, { globalPrivacyControl: true });
    runInNewContext(loader, b.context);
    expect(b.scripts).toHaveLength(0);
  });
});
