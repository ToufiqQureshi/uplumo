import { useMemo, useState } from "react";

// The server accepts at most this many names, each this long.
export const MAX_BRAND_NAMES = 20;
export const MAX_BRAND_NAME_LENGTH = 100;

const storageKey = (projectId: string) => `uplumo:brand-names:${projectId}`;

/** "Acme, acme shop" -> ["Acme", "acme shop"]: trimmed, unique, within limits. */
export function parseBrandNames(text: string): string[] {
  const seen = new Set<string>();
  const names: string[] = [];
  for (const part of text.split(/[,\n]/)) {
    const name = Array.from(part.trim())
      .slice(0, MAX_BRAND_NAME_LENGTH)
      .join("");
    const key = name.toLowerCase();
    if (!name || seen.has(key)) continue;
    seen.add(key);
    names.push(name);
    if (names.length === MAX_BRAND_NAMES) break;
  }
  return names;
}

function read(projectId: string): string {
  try {
    return window.localStorage.getItem(storageKey(projectId)) ?? "";
  } catch {
    return "";
  }
}

/**
 * The owner's brand names for one project, kept in this browser. They are only
 * used to split branded from non-branded searches; nothing guesses a brand.
 * Storage can be blocked, so the page works without it.
 */
export function useBrandNames(projectId: string) {
  const [state, setState] = useState(() => ({
    projectId,
    text: read(projectId),
  }));
  if (state.projectId !== projectId) {
    setState({ projectId, text: read(projectId) });
  }
  const text = state.projectId === projectId ? state.text : "";
  const names = useMemo(() => parseBrandNames(text), [text]);
  const save = (next: string) => {
    setState({ projectId, text: next });
    try {
      window.localStorage.setItem(storageKey(projectId), next);
    } catch {
      // Blocked storage: the names still apply until the page is closed.
    }
  };
  return { text, names, save };
}
