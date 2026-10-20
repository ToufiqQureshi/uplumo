import { z } from "zod";
const searchContextSchema = z.object({
  page: z.url().regex(/^https?:\/\//),
  queries: z.array(z.string().max(200)).max(10),
  reasons: z.array(z.string().max(2000)).max(10),
  startDate: z.string(),
  endDate: z.string(),
  created: z.number(),
});
type SearchContext = z.infer<typeof searchContextSchema>;
export function saveSearchContext(
  project: string,
  context: Omit<SearchContext, "created">,
) {
  sessionStorage.setItem(
    `growth-editor:${project}`,
    JSON.stringify(
      searchContextSchema.parse({ ...context, created: Date.now() }),
    ),
  );
}
export function readSearchContext(project: string): SearchContext | null {
  try {
    const raw = sessionStorage.getItem(`growth-editor:${project}`);
    if (!raw) return null;
    const parsed = searchContextSchema.safeParse(JSON.parse(raw));
    if (!parsed.success || Date.now() - parsed.data.created > 30 * 60 * 1000)
      return null;
    return parsed.data;
  } catch {
    return null;
  }
}
