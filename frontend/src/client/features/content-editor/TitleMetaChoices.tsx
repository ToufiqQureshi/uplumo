import { Button } from "@/client/components/ui/button";
import { parseTitleMetaOptions } from "./title-meta";

/** One-click "use this" buttons for the AI's title and Google text ideas. */
export function TitleMetaChoices({
  suggestion,
  onUse,
}: {
  suggestion: string;
  onUse: (field: "title" | "metaDescription", value: string) => void;
}) {
  const options = parseTitleMetaOptions(suggestion);
  if (!options.length) return null;
  return (
    <ul className="space-y-2">
      {options.map((option) => (
        <li
          key={option.title}
          className="space-y-1 rounded-lg border p-2 text-sm"
        >
          <p className="font-medium">{option.title}</p>
          <p className="text-xs text-muted-foreground">{option.description}</p>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onUse("title", option.title)}
            >
              Use this title
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onUse("metaDescription", option.description)}
            >
              Use this Google text
            </Button>
          </div>
        </li>
      ))}
    </ul>
  );
}
