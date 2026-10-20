// A draft stays in this tab, scoped to the project, rather than exposing report
// data in a URL. Opening the analyst never sends a paid message automatically.
export function saveAnalystDraft(project: string, prompt: string) {
  sessionStorage.setItem(
    `growth-analyst:${project}`,
    JSON.stringify({ text: prompt.slice(0, 16000), created: Date.now() }),
  );
}
export function takeAnalystDraft(project: string): string {
  const key = `growth-analyst:${project}`;
  try {
    const raw = sessionStorage.getItem(key);
    if (!raw) return "";
    const draft: unknown = JSON.parse(raw);
    if (
      typeof draft !== "object" ||
      draft === null ||
      !("text" in draft) ||
      typeof draft.text !== "string" ||
      !("created" in draft) ||
      typeof draft.created !== "number" ||
      Date.now() - draft.created > 30 * 60 * 1000
    )
      return "";
    return draft.text.slice(0, 16000);
  } catch {
    return "";
  }
}
export function clearAnalystDraft(project: string) {
  try {
    sessionStorage.removeItem(`growth-analyst:${project}`);
  } catch {
    /* Storage can be disabled. */
  }
}

export function isAnalystSession(project: string, session: string) {
  try {
    return sessionStorage.getItem(`growth-mode:${project}:${session}`) === "1";
  } catch {
    return false;
  }
}
export function setAnalystSession(
  project: string,
  session: string,
  enabled: boolean,
) {
  try {
    const key = `growth-mode:${project}:${session}`;
    if (enabled) sessionStorage.setItem(key, "1");
    else sessionStorage.removeItem(key);
  } catch {
    /* The active mounted chat still enforces its selected mode. */
  }
}
