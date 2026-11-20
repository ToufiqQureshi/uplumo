# How Search Console Page Works

## For the user

1. **Connect**: Link your Google Search Console account via Settings > Integrations.

2. **See your numbers**: The page shows four metrics:
   - **Clicks**: How many times someone clicked your result in Google
   - **Impressions**: How many times your result appeared in search
   - **CTR**: Click-through rate (clicks ÷ impressions)
   - **Position**: Average ranking position (lower is better)

3. **Compare periods**: By default, your numbers are compared to the same period last month. Use the date picker to compare custom periods. The percentage change shows up next to each metric (green is good, red is declining).

4. **See what changed**: The "What changed" card shows your biggest wins and losses:
   - Top keyword gains/losses (with reasons)
   - Top page losses
   - Which country sends the most clicks
   - Keywords close to the first page (positions 4–20)

5. **Filter and drill down**:
   - Use the preset date selector for quick ranges ("Last 7 days", "This month", etc.)
   - Filter by search type (Web, Images, Videos, News), device (Desktop, Mobile, Tablet), or country
   - Click any row in the Keywords, Pages, Countries, or Devices table to filter further

6. **Export**: Export keywords or pages as CSV or Google Sheets for analysis in your tools.

## For engineers

### Data refresh

- The page loads the report on first render via TanStack Query
- Filters and date changes re-fetch automatically
- Each report fetch queries Google Search Console 9 times in parallel (all free, no cost)
- Results are cached for the same project + filters for 5 minutes

### Table data

- Keywords and Pages tables are paginated (25/50/100 rows per page)
- Each page load makes one more Search Console read
- Countries and Devices tables load from the report (no extra read needed)

### Insights

The "What changed" card is computed from the report data in the browser:

- No API call; no processing cost
- Pure function: given a report, return at most 4 insights
- Each insight includes real numbers and a link to drill down

### Comparison math

If `comparison_available` is false (data is older than 16 months), no gains/losses are shown.

Lower is better for position (5 is better than 10), so the % change is flipped:

- Position 10 → 5 is green (position improved; % change negative but good)
- Position 5 → 10 is red (position worsened; % change positive but bad)
