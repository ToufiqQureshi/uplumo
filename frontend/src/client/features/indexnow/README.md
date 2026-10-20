# indexnow

The IndexNow card on Settings → Integrations (`#indexnow`). Rules, limits and
errors live in `backend/internal/indexnow/README.md`.

- `IndexNowCard`: Free workspaces see a plans link; a project without a
  website is sent to project settings; otherwise "Set up IndexNow", then the
  key file to download, a "Check key file" button and a box to paste URLs.
- `indexNowApi.ts`: the four routes, validated with Zod.
- The card says plainly that **Google does not use IndexNow**. Never change
  the copy to promise faster Google indexing.
- Pages published to WordPress from the content editor are submitted by the
  server; `site-publishing/PublishToSite.tsx` shows the result as a toast.
