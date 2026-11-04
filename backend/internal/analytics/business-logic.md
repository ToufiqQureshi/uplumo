# Analytics: rules that protect trust

- **We never guess what a lead is.** The owner picks lead actions; until then
  no lead number, rate or insight is shown. (An earlier version guessed from
  event names and told a site without WhatsApp that nobody tapped WhatsApp.)
- **Broken tracking hides the numbers.** Page views without visits mean an old
  snippet; sources and rates would be wrong, so only the fix is shown.
- **Visitors are visitor-days.** The visitor hash rotates daily with a secret
  salt, so nobody can be followed across days. The UI says "once per day".
- **A visit belongs to the source of its first page**, so source shares add up
  and a visit crossing the range edge keeps its real source.
- **Leads tie to a visit only with a session.** Old snippets send none; their
  leads count in the total but not in a source.
- **Insights state only facts the data shows**: no generic advice, no copy
  that assumes a button, country or business type.
- **No personal data:** no IP, phone number, email or form answer is stored.
- **Free:** Analytics makes no paid call; nothing here can cost credits.
- Lead actions: at most 20 names of 1 to 64 characters per project; saving
  replaces the set in one transaction so a reader never sees a mix.
