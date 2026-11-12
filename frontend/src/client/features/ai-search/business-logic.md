# AI visibility: rules that protect money and trust

- **Nothing paid on load.** Tabs read saved answers. Test a question and
  Market research buy only after a submit in the same page load; an opened
  URL, reload or redirect fills the form and waits. Why: an expired cache
  plus an old bookmark used to buy a lookup with no click.
- **No automatic retries in the browser.** Paid queries set `retry: false`
  and ignore window focus and reconnect.
- **Cost before the click.** The tracker shows the estimate before saving or
  running; the run sends the approved credits and the server refuses a run
  whose cost grew. Market research asks before a large charge. Holds settle
  to the real provider cost, never above the approved ceiling.
- **Credits round trip exactly.** A ceiling of n credits is held as
  `n / billing.CreditsPerProviderUSD` dollars and converts back to n credits
  (`billing.CreditsForUSD` ignores a millionth of a credit of float error).
- **Cheapest real answers.** ChatGPT and Gemini use the LLM Scraper standard
  queue ($0.0012) instead of live calls. Share of voice is computed from
  answers already bought, not from extra LLM Mentions tasks.
- **Measured means answered.** Failed or empty answers never count as "not
  named"; a question is a gap only when an engine answered and none named
  the brand. Changes compare only runs with the same questions, engines and
  market (`scopeKey`).
- **Alerts only compare like with like.** Two runs are compared only when
  their `scopeKey` matches, and only question/engine pairs that both got a
  real answer. A collection failure never shows as a lost mention. A tracked
  competitor is never announced as a "new" business.
- **Proof for every number.** Every cell opens the saved answer text,
  recommended businesses and citations, so an agency can show a client
  exactly what AI said.
- **One run at a time per project**, guarded by the server (`409`) and an
  Idempotency-Key on the run request.
