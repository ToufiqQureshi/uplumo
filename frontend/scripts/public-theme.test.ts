import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";
import { describe, expect, it } from "vitest";

const source = readFileSync(
  new URL("../../backend/internal/site/static/theme.js", import.meta.url),
  "utf8",
);
function browser(stored: string | null = null, light = false, blocked = false) {
  const listeners: Record<string, (event?: { key: string | null }) => void> =
    {};
  const attributes: Record<string, string> = {};
  const control = {
    hidden: true,
    label: "",
    setAttribute: (name: string, value: string) => {
      expect(name).toBe("aria-label");
      control.label = value;
    },
    addEventListener: (name: string, fn: () => void) => {
      listeners[name] = fn;
    },
  };
  const media = {
    matches: light,
    addEventListener: (_name: string, fn: () => void) => {
      listeners.media = fn;
    },
  };
  const localStorage = {
    getItem: () => {
      if (blocked) throw new Error("Storage denied");
      return stored;
    },
    setItem: (key: string, value: string) => {
      expect(key).toBe("theme-preference");
      if (blocked) throw new Error("Storage denied");
      stored = value;
    },
    removeItem: (key: string) => {
      expect(key).toBe("theme-preference");
      if (blocked) throw new Error("Storage denied");
      stored = null;
    },
  };
  const meta = { content: "" };
  runInNewContext(source, {
    Event: class {},
    window: {
      matchMedia: () => media,
      localStorage,
      dispatchEvent: () => {},
      addEventListener: (name: string, fn: () => void) => {
        listeners[name] = fn;
      },
    },
    document: {
      documentElement: {
        setAttribute: (key: string, value: string) => {
          attributes[key] = value;
        },
      },
      querySelector: () => meta,
      getElementById: () => control,
      addEventListener: (name: string, fn: () => void) => {
        listeners[name] = fn;
      },
    },
  });
  listeners.DOMContentLoaded();
  return {
    attributes,
    control,
    media,
    meta,
    listeners,
    stored: () => stored,
    externalPick(value: string) {
      stored = value;
    },
    toggle() {
      listeners.click();
    },
  };
}

describe("public page appearance", () => {
  it("shares the app preference across page loads", () => {
    const b = browser("dark", true);
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
    expect(b.control.hidden).toBe(false);
    expect(b.control.label).toBe("Switch to light theme");
    b.toggle();
    expect(b.attributes["data-theme"]).toBe("uplumo");
    expect(b.meta.content).toBe("#fafcf9");
    expect(b.control.label).toBe("Switch to dark theme");
    expect(b.stored()).toBe("light");
    expect(browser(b.stored()).attributes["data-theme"]).toBe("uplumo");
    b.toggle();
    expect(b.stored()).toBe("dark");
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
  });
  it("follows the OS until the first click, then flips what is shown", () => {
    const b = browser(null, true);
    expect(b.attributes["data-theme"]).toBe("uplumo");
    b.media.matches = false;
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
    b.toggle();
    expect(b.stored()).toBe("light");
    b.media.matches = true;
    b.listeners.media();
    b.media.matches = false;
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo");
  });
  it("syncs a choice made in another tab", () => {
    const b = browser();
    b.externalPick("light");
    b.listeners.storage({ key: "theme-preference" });
    expect(b.attributes["data-theme"]).toBe("uplumo");
    expect(b.control.label).toBe("Switch to dark theme");
  });
  it("keeps an in-page choice working when storage is blocked", () => {
    const b = browser(null, false, true);
    b.toggle();
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo");
    b.toggle();
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
  });
});
