# Support report decisions

- Reuse Help and its email fallback; a separate ticket dashboard is unnecessary.
- A ticket means the report was saved, not that an email was delivered. Backend
  notification failure must not ask the customer to submit and pay attention twice.
- Preserve an unchanged submission UUID after ambiguous failure so retries cannot
  create duplicate reports. Disable edits during dispatch and guard same-tick
  double clicks before React renders the disabled button.
- Automatically collect only page pathname, app version and server request ID.
  Query strings and fragments can contain secrets and must never be attached.
- Screenshots are explicit and optional, restricted to PNG/JPEG and 1 MiB on
  both sides. Server validation remains authoritative; users are reminded to hide
  private information before selecting an image.
- No paid provider or checkout is invoked by this feature.
