/**
 * Reports the customer asked to buy in this browser session, by clicking
 * Search or Refresh. Every other load (a page opened from a saved link, a
 * reload, a recent-search tab) asks the server for cached data only, so an
 * expired cache shows a Refresh button instead of buying again unasked.
 * Kept in memory on purpose: a new tab or reload starts with nothing allowed.
 */
const allowed = new Set<string>();

export function allowPurchase(key: string) {
  allowed.add(key);
}

export function purchaseAllowed(key: string) {
  return allowed.has(key);
}
