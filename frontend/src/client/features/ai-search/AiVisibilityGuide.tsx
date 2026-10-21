import { Link } from "@tanstack/react-router";
import { cn } from "cn";

type Tool = "brand-lookup" | "prompt-explorer" | "prompt-tracking";

const TOOLS: { id: Tool; name: string; line: string }[] = [
  {
    id: "brand-lookup",
    name: "Brand Lookup",
    line: "Where AI already talks about your brand: questions people asked, pages it cites, your share against competitors.",
  },
  {
    id: "prompt-explorer",
    name: "Prompt Explorer",
    line: "Ask one question of ChatGPT, Claude, Gemini and Perplexity right now and compare their answers.",
  },
  {
    id: "prompt-tracking",
    name: "AI Visibility Tracker",
    line: "Save the questions that matter and re-check them on a schedule to see if AI recommends you over time.",
  },
];

/**
 * One line for each of the three AI visibility tools, so a customer can tell
 * them apart and pick the right one. The current tool is highlighted; the
 * others link to their pages.
 */
export function AiVisibilityGuide({
  current,
  projectId,
}: {
  current: Tool;
  projectId: string;
}) {
  return (
    <ul className="grid gap-2 text-sm md:grid-cols-3">
      {TOOLS.map((tool) => {
        const active = tool.id === current;
        return (
          <li
            key={tool.id}
            className={cn(
              "rounded-lg border p-3",
              active ? "border-primary bg-primary/5" : "text-muted-foreground",
            )}
          >
            {active ? (
              <span className="font-medium text-foreground">
                {tool.name} (this page)
              </span>
            ) : (
              <Link
                to={`/p/$projectId/${tool.id}`}
                params={{ projectId }}
                className="font-medium underline underline-offset-2"
              >
                {tool.name}
              </Link>
            )}
            <span className="block">{tool.line}</span>
          </li>
        );
      })}
    </ul>
  );
}
