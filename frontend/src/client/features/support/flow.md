# Support report flow

1. Help opens the form. Its initial affected page comes from the latest failed
   API request's originating page, otherwise a same-origin referrer or current
   page. Customers can correct this path; later failures do not overwrite edits.
2. The form attaches the app package version and, when available, the server's
   request ID. It does not read cookies, credentials, local storage or URL query
   strings. `uplumoApi.ts` keeps only this recent failure context in memory.
3. An explicit Send report posts to `/api/v1/support/reports`. A screenshot is
   read only after the customer selects a bounded PNG/JPEG file.
4. Failed or ambiguous responses preserve the form and submission UUID for a
   manual retry. There is no automatic retry. Editing starts a new submission
   identity; successful saving displays the ticket and offers a fresh report.
5. Go owns validation, durable persistence, tenant access and notification.
