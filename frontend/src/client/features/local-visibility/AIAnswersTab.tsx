import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Label } from "@/client/components/ui/label";
import type { Business } from "./localVisibilityApi";

/**
 * The local question people ask AI assistants, started from the business's
 * own search term and edited by the customer. It makes no request: asking
 * the models and tracking the prompt happen in AI Visibility, on a click.
 */
export function AIAnswersTab({
  projectId,
  business,
}: {
  projectId: string;
  business: Business;
}) {
  const [prompt, setPrompt] = useState(`best ${business.searchTerm} near me`);
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        People ask ChatGPT, Gemini and Perplexity for local recommendations.
        Check whether they name {business.title}.
      </p>
      <div className="space-y-2">
        <Label htmlFor="local-prompt">
          Write the question the way your customers ask it, in their language
          and with your area
        </Label>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            id="local-prompt"
            value={prompt}
            maxLength={200}
            onChange={(e) => setPrompt(e.target.value)}
          />
          <Button
            variant="outline"
            disabled={!prompt.trim()}
            onClick={() =>
              void navigator.clipboard
                .writeText(prompt.trim())
                .then(() => toast.success("Prompt copied"))
            }
          >
            Copy
          </Button>
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          render={
            <Link
              to="/p/$projectId/ai-visibility/test"
              params={{ projectId }}
            />
          }
        >
          Ask the AI models once
        </Button>
        <Button
          variant="outline"
          render={
            <Link to="/p/$projectId/ai-visibility" params={{ projectId }} />
          }
        >
          Track a prompt every week
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        Paste a prompt there. Both show their credits before they run.
      </p>
    </div>
  );
}
