import { useRouterState } from "@tanstack/react-router";

declare module "@tanstack/react-router" {
  interface HistoryState {
    /** Set by a search the customer submitted in this page load. */
    aiRun?: string;
  }
}

/**
 * Changes on every page load. A submitted search marks its history entry with
 * it, so back/forward inside this load may show (and buy) that search again,
 * while a reload, bookmark or shared link never buys anything by itself.
 */
const PAGE_LOAD = crypto.randomUUID();

/** History state for a navigation the customer asked to run. */
export const userRequestedRun = { aiRun: PAGE_LOAD } as const;

/** True when the current URL's search was submitted by the customer. */
export function useUserRequestedRun(): boolean {
  return useRouterState({
    select: (s) => s.location.state.aiRun === PAGE_LOAD,
  });
}
