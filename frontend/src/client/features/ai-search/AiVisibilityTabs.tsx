import { Link } from "@tanstack/react-router";

const TABS = [
  { to: "/p/$projectId/ai-visibility", label: "Overview", exact: true },
  { to: "/p/$projectId/ai-visibility/prompts", label: "Prompts" },
  { to: "/p/$projectId/ai-visibility/opportunities", label: "Opportunities" },
  { to: "/p/$projectId/ai-visibility/sentiment", label: "Sentiment" },
  { to: "/p/$projectId/ai-visibility/test", label: "Test a question" },
  { to: "/p/$projectId/ai-visibility/research", label: "Market research" },
  { to: "/p/$projectId/ai-visibility/ideas", label: "Prompt ideas" },
  { to: "/p/$projectId/ai-visibility/setup", label: "Setup & history" },
] as const;

/** The sections of the AI visibility page, as links so each has its own URL. */
export function AiVisibilityTabs({ projectId }: { projectId: string }) {
  return (
    <nav
      aria-label="AI visibility sections"
      className="-mx-1 overflow-x-auto overflow-y-hidden"
    >
      <ul className="flex min-w-max gap-1 border-b px-1">
        {TABS.map((tab) => (
          <li key={tab.to}>
            <Link
              to={tab.to}
              params={{ projectId }}
              activeOptions={{
                exact: "exact" in tab,
                includeSearch: false,
              }}
              className="-mb-px inline-flex min-h-11 items-center border-b-2 border-transparent px-3 text-sm font-medium text-muted-foreground hover:text-foreground aria-[current=page]:border-primary aria-[current=page]:text-foreground"
            >
              {tab.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}
