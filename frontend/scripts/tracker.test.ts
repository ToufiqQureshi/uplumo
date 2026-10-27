// @vitest-environment happy-dom
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The tracker is served by the Go server (backend/internal/httpapi), but it
// runs in the visitor's browser, so it is tested here with a browser DOM.
const source = readFileSync(
  join(__dirname, "../../backend/internal/httpapi/tracker.js"),
  "utf8",
);
const siteKey = "site-key";
const sessionKey = `uplumo-session:${siteKey}`;

let visibility: DocumentVisibilityState;
let clock: number;
let beacons: Record<string, unknown>[];
const originalPushState = history.pushState;
let cleanup: Array<() => void>;

function runTracker(options: {
  opener: boolean;
  historyLength: number;
  navigation: string;
  recognition?: boolean;
}) {
  const script = document.createElement("script");
  script.setAttribute("data-site", siteKey);
  if (options.recognition) script.setAttribute("data-recognition", "true");
  script.src = "https://uplumo.test/t.js";
  Object.defineProperty(document, "currentScript", {
    configurable: true,
    value: script,
  });
  Object.defineProperty(window, "opener", {
    configurable: true,
    value: options.opener ? {} : null,
  });
  Object.defineProperty(history, "length", {
    configurable: true,
    value: options.historyLength,
  });
  vi.spyOn(performance, "getEntriesByType").mockReturnValue([
    { type: options.navigation } as unknown as PerformanceEntry,
  ]);
  const before = beacons.length;
  new Function(source)();
  // The first beacon of this run is its pageview.
  return beacons[before];
}

beforeEach(() => {
  visibility = "visible";
  clock = 1_000;
  beacons = [];
  cleanup = [];
  vi.useFakeTimers();
  sessionStorage.clear();
  localStorage.clear();
  window.location.href = "https://acme.test/start";
  Object.defineProperty(navigator, "doNotTrack", {
    configurable: true,
    value: "0",
  });
  Object.defineProperty(navigator, "globalPrivacyControl", {
    configurable: true,
    value: false,
  });
  Object.defineProperty(navigator, "locks", {
    configurable: true,
    value: { request: vi.fn((_key, callback) => Promise.resolve(callback())) },
  });
  for (const owner of [window, document]) {
    const add = owner.addEventListener.bind(owner);
    vi.spyOn(owner, "addEventListener").mockImplementation(
      (type, listener, options) => {
        add(type, listener, options);
        cleanup.push(() => owner.removeEventListener(type, listener, options));
      },
    );
  }
  document.addEventListener("click", (event) => event.preventDefault());
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    get: () => visibility,
  });
  vi.spyOn(performance, "now").mockImplementation(() => clock);
  Object.defineProperty(navigator, "sendBeacon", {
    configurable: true,
    value: (_url: string, body: string) => {
      beacons.push(JSON.parse(body) as Record<string, unknown>);
      return true;
    },
  });
});

afterEach(() => {
  cleanup.forEach((remove) => remove());
  history.pushState = originalPushState;
  document.body.replaceChildren();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("tracker", () => {
  const options = { opener: false, historyLength: 1, navigation: "navigate" };

  it("keeps recognition off by default", () => {
    const view = runTracker(options);
    expect(view?.b).toBeUndefined();
    expect(localStorage.getItem(`uplumo-browser:${siteKey}`)).toBeNull();
  });

  it("keeps one recognized browser across reloads and expires after 180 days", () => {
    const first = runTracker({ ...options, recognition: true });
    const second = runTracker({ ...options, recognition: true });
    expect(first?.b).toMatch(/^[0-9a-f]{32}$/);
    expect(second?.b).toBe(first?.b);
    vi.setSystemTime(Date.now() + 180 * 86400_000);
    const expired = runTracker({ ...options, recognition: true });
    expect(expired?.b).not.toBe(first?.b);
    expect(expired?.b).toMatch(/^[0-9a-f]{32}$/);
  });

  it("falls back to unrecognized visits when storage cannot persist an ID", () => {
    vi.spyOn(localStorage, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    const view = runTracker({ ...options, recognition: true });
    expect(view?.p).toBeTruthy();
    expect(view?.b).toBeUndefined();
  });

  it("does not create recognition IDs when secure randomness fails", () => {
    vi.spyOn(crypto, "getRandomValues").mockImplementation(() => {
      throw new Error("unavailable");
    });
    const view = runTracker({ ...options, recognition: true });
    expect(view?.b).toBeUndefined();
    expect(localStorage.getItem(`uplumo-browser:${siteKey}`)).toBeNull();
  });

  it("does not enable recognition without cross-tab initialization support", () => {
    Object.defineProperty(navigator, "locks", {
      configurable: true,
      value: undefined,
    });
    const view = runTracker({ ...options, recognition: true });
    expect(view?.b).toBeUndefined();
    expect(localStorage.getItem(`uplumo-browser:${siteKey}`)).toBeNull();
  });

  it.each(["doNotTrack", "globalPrivacyControl"])(
    "does not store or collect under %s",
    (signal) => {
      Object.defineProperty(navigator, signal, {
        configurable: true,
        value: signal === "doNotTrack" ? "1" : true,
      });
      const storage = vi.spyOn(localStorage, "setItem");
      runTracker({ ...options, recognition: true });
      expect(beacons).toEqual([]);
      expect(storage).not.toHaveBeenCalled();
    },
  );

  it("records one configured outbound click and strips URL secrets before sending", () => {
    runTracker(options);
    document.body.innerHTML =
      '<a data-uplumo-event="Book" href="https://partner.test/booking?email=private@example.test#token"><span>Book</span></a>';
    document
      .querySelector("span")
      ?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    const clicks = beacons.filter((beacon) => beacon.c);
    expect(clicks).toHaveLength(1);
    expect(clicks[0]).toMatchObject({
      n: "Book",
      x: "https://partner.test/booking",
      c: true,
    });
    expect(clicks[0]?.i).toMatch(/^[0-9a-f]{32}$/);
  });

  it("automatically tracks outbound links and leaves untargeted buttons unknown", () => {
    runTracker(options);
    document.body.innerHTML =
      '<a href="https://partner.test/">Partner</a><button data-uplumo-event="Open modal">Modal</button>';
    document
      .querySelector("a")
      ?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    document.querySelector("button")?.click();
    expect(beacons.filter((beacon) => beacon.c)).toEqual([
      expect.objectContaining({
        n: "outbound_click",
        x: "https://partner.test/",
      }),
      expect.objectContaining({ n: "Open modal" }),
    ]);
    expect(beacons.at(-1)?.x).toBeUndefined();
  });

  it("uses an explicit HTTP target for JavaScript buttons, ignoring unsupported schemes", () => {
    runTracker(options);
    document.body.innerHTML =
      '<button data-uplumo-event="Booking" data-uplumo-destination="https://partner.test/booking?secret=1">Book</button><a data-uplumo-event="Email" href="mailto:private@example.test">Email</a>';
    document.querySelector("button")?.click();
    document
      .querySelector("a")
      ?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    expect(beacons.find((beacon) => beacon.n === "Booking")?.x).toBe(
      "https://partner.test/booking",
    );
    expect(beacons.find((beacon) => beacon.n === "Email")?.x).toBeUndefined();
  });

  it("records phone, SMS, email and WhatsApp taps without the number or address", () => {
    runTracker(options);
    document.body.innerHTML = [
      '<a href="tel:+919800000000">Call</a>',
      '<a href="sms:+919800000000">SMS</a>',
      '<a href="MAILTO:owner@example.test">Mail</a>',
      '<a href="https://wa.me/919800000000?text=hi">Chat</a>',
      '<a href="https://api.whatsapp.com/send?phone=919800000000">Chat</a>',
      '<a href="whatsapp://send?phone=919800000000">Chat</a>',
      '<a href="https://www.whatsapp.com/download">Get WhatsApp</a>',
    ].join("");
    for (const link of document.querySelectorAll("a")) {
      link.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    }
    const clicks = beacons.filter((beacon) => beacon.c);
    expect(clicks.map((beacon) => beacon.n)).toEqual([
      "call_click",
      "sms_click",
      "email_click",
      "whatsapp_click",
      "whatsapp_click",
      "whatsapp_click",
      "outbound_click",
    ]);
    for (const click of clicks.slice(0, 6)) expect(click.x).toBeUndefined();
    expect(JSON.stringify(beacons)).not.toContain("9800000000");
    expect(JSON.stringify(beacons)).not.toContain("owner@example.test");
  });

  it("lets an owner's tag name a contact link", () => {
    runTracker(options);
    document.body.innerHTML =
      '<a data-uplumo-event="Header WhatsApp" href="https://wa.me/919800000000">Chat</a>';
    document
      .querySelector("a")
      ?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    const clicks = beacons.filter((beacon) => beacon.c);
    expect(clicks.map((beacon) => beacon.n)).toEqual(["Header WhatsApp"]);
  });

  it("records file downloads on the site and elsewhere", () => {
    runTracker(options);
    document.body.innerHTML =
      '<a href="/menu.PDF?v=2">Menu</a><a href="https://cdn.test/brochure.zip">Zip</a><a href="/about">About</a>';
    for (const link of document.querySelectorAll("a")) {
      link.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    }
    const clicks = beacons.filter((beacon) => beacon.c);
    expect(clicks).toEqual([
      expect.objectContaining({
        n: "file_download",
        x: "https://acme.test/menu.PDF",
      }),
      expect.objectContaining({
        n: "file_download",
        x: "https://cdn.test/brochure.zip",
      }),
    ]);
  });

  it("records a submitted form once, never search forms or field values", () => {
    runTracker(options);
    document.body.innerHTML = [
      '<form id="lead" method="post"><input name="phone" value="9800000000"><button>Send</button></form>',
      '<form id="search"><input name="q" value="rooms"></form>',
      '<form id="search-role" role="search" method="post"><input name="term"></form>',
      '<form id="tagged" method="post"><button data-uplumo-event="Quote">Get quote</button></form>',
      '<form id="ignored" method="post" data-uplumo-ignore><input name="x"></form>',
    ].join("");
    for (const form of document.querySelectorAll("form")) {
      form.dispatchEvent(
        new Event("submit", { bubbles: true, cancelable: true }),
      );
    }
    const submits = beacons.filter((beacon) => beacon.n === "form_submit");
    expect(submits).toHaveLength(1);
    expect(submits[0]?.c).toBeUndefined();
    expect(JSON.stringify(beacons)).not.toContain("9800000000");
  });

  it("does not count a browser that opted out with uplumo_ignore, until undone", () => {
    window.location.href = "https://acme.test/?uplumo_ignore=1";
    expect(runTracker(options)).toBeUndefined();
    window.location.href = "https://acme.test/pricing";
    expect(runTracker(options)).toBeUndefined();
    window.location.href = "https://acme.test/?uplumo_ignore=0";
    expect(runTracker(options)?.u).toBe("https://acme.test/?uplumo_ignore=0");
  });

  it("stops collection and clears identifiers on consent withdrawal", () => {
    runTracker({ ...options, recognition: true });
    const api = window as unknown as {
      uplumoConsent: (value: boolean) => void;
      uplumo: (name: string) => void;
    };
    api.uplumoConsent(false);
    const count = beacons.length;
    api.uplumo("after-withdrawal");
    clock += 15_000;
    window.dispatchEvent(new Event("pagehide"));
    vi.advanceTimersByTime(15_000);
    expect(beacons).toHaveLength(count);
    expect(localStorage.getItem(`uplumo-browser:${siteKey}`)).toBeNull();
    expect(sessionStorage.getItem(sessionKey)).toBeNull();
  });
  it("counts the visible time before the page is hidden", () => {
    const pageview = runTracker({
      opener: false,
      historyLength: 1,
      navigation: "navigate",
    });
    clock += 12_000;
    visibility = "hidden";
    document.dispatchEvent(new Event("visibilitychange"));
    const engagement = beacons.find(
      (b) => b.p === pageview?.p && typeof b.d === "number",
    );
    expect(engagement?.d).toBe(12_000);
  });

  it("does not count time while the page was hidden", () => {
    const pageview = runTracker({
      opener: false,
      historyLength: 1,
      navigation: "navigate",
    });
    clock += 5_000;
    visibility = "hidden";
    document.dispatchEvent(new Event("visibilitychange"));
    clock += 60_000;
    window.dispatchEvent(new Event("pagehide"));
    const reports = beacons.filter(
      (b) => b.p === pageview?.p && typeof b.d === "number",
    );
    expect(reports.map((b) => b.d)).toEqual([5_000, 5_000]);
  });

  it("keeps the session across reloads of a tab that has an opener", () => {
    const stored = {
      id: "a".repeat(32),
      started: Date.now(),
      last: Date.now(),
    };
    sessionStorage.setItem(sessionKey, JSON.stringify(stored));
    const reload = runTracker({
      opener: true,
      historyLength: 1,
      navigation: "reload",
    });
    expect(reload?.s).toBe(stored.id);
    const nextPage = runTracker({
      opener: true,
      historyLength: 2,
      navigation: "navigate",
    });
    expect(nextPage?.s).toBe(stored.id);
  });

  it("starts a new session on the first page of a tab opened by another page", () => {
    const copied = {
      id: "b".repeat(32),
      started: Date.now(),
      last: Date.now(),
    };
    sessionStorage.setItem(sessionKey, JSON.stringify(copied));
    const first = runTracker({
      opener: true,
      historyLength: 1,
      navigation: "navigate",
    });
    expect(first?.s).toMatch(/^[0-9a-f]{32}$/);
    expect(first?.s).not.toBe(copied.id);
  });
});
