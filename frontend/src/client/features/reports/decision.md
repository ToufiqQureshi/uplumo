# Reports and share links: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Creates report templates, lists/renders project reports and manages optional public share links.

## Decisions to preserve

- Public sharing is opt-in; never expose a report through an unguessable-but-unchecked ID alone. _Why:_ Reports may contain client data, so only an explicit share state may grant public access.
- Keep organization/project authorization on private routes and revocation effective for shared links. _Why:_ A report ID is not proof of membership, and revoked public access must not remain usable.
- Report generation should use stored data unless a clearly labeled refresh action is requested. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- Manual report content is plain text escaped into HTML; preserve character and URL limits to avoid stored markup injection and oversized report bodies. _Why:_ Report fields are user-controlled and later rendered or shared, so escaping and limits protect recipients and storage.
- Work-log sharing is opt-in and happens after the report save; a share failure must not roll back or hide a successfully saved report. _Why:_ Saving customer work and publishing a public link are distinct actions with different privacy consequences.

## Verification pointers

- `reportApi.test.ts`

## Visual builder

- Use an editable snapshot definition plus a separately rendered HTML snapshot. Raw source rows belong only to the private editor column; never embed them in HTML or share metadata. This permits editing without exposing filtered-out rows/columns to recipients.
- Reuse existing project-scoped report save/get/share/export and existing analytics/Google reads. Source import is explicit, bounded and cached; switching charts/fields/filter values performs no network request. No paid model/data provider is introduced.
- Use the same static HTML/SVG chart renderer in the canvas and exported document. The existing report CSP prohibits executable chart libraries; changing the sandbox would expose report readers to user/agent content. Keyboard-accessible settings and move buttons provide layout control without a new drag dependency.
- Apply row/date filters before aggregation, preserve zero/missing values and label the aggregation/group limits. A sum of daily visitors is visitor-days; a mean of rates is unweighted. This builder does not infer business conclusions, calculate measures/DAX, join sources, schedule delivery or promise live refresh.
- Bound imported data and visual counts and enforce the existing report byte ceiling across both HTML and private editor data. This is a small-report editor, not a warehouse query engine; move aggregation server-side if larger datasets become a requirement.
- Keep unsaved editor changes on failed saves and warn on leaving. Existing report replacement semantics and public share state remain unchanged; concurrent editors still use last-write-wins, with no version history.
