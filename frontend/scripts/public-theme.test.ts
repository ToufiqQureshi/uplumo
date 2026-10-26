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
    value: "system",
    disabled: true,
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
    pick(value: string) {
      control.value = value;
      listeners.change();
    },
  };
}

describe("public page appearance", () => {
  it("shares the app preference across page loads and restores system mode", () => {
    const b = browser("dark", true);
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
    expect(b.control.disabled).toBe(false);
    b.pick("light");
    expect(b.attributes["data-theme"]).toBe("uplumo");
    expect(b.meta.content).toBe("#fafcf9");
    expect(b.stored()).toBe("light");
    expect(browser(b.stored()).control.value).toBe("light");
    b.pick("system");
    expect(b.stored()).toBeNull();
    b.media.matches = false;
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
  });
  it("honors OS changes only in system mode and syncs other tabs", () => {
    const b = browser();
    b.media.matches = true;
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo");
    b.pick("dark");
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
    b.externalPick("light");
    b.listeners.storage({ key: "theme-preference" });
    expect(b.control.value).toBe("light");
    expect(b.attributes["data-theme"]).toBe("uplumo");
  });
  it("keeps an in-page selection working when storage is blocked", () => {
    const b = browser(null, false, true);
    b.pick("light");
    b.listeners.media();
    expect(b.attributes["data-theme"]).toBe("uplumo");
    b.pick("system");
    expect(b.attributes["data-theme"]).toBe("uplumo-dark");
  });
});
