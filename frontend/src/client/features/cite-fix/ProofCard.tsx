import { Link } from "@tanstack/react-router";
import { Sparkles } from "lucide-react";
import { Button } from "@/client/components/ui/button";

/**
 * Shown until the project has a saved AI Visibility check: Cite Fix reads
 * that check to show who AI cites, and never runs one itself.
 */
export function ProofCard({ projectId }: { projectId: string }) {
  return (
    <section
      aria-label="AI citation proof"
      className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed p-4 text-sm"
    >
      <p className="text-muted-foreground">
        See which sites AI assistants cite for your customers' questions: run a
        check in the AI Visibility Tracker. Cite Fix then shows who AI picks
        instead of you and what to change.
      </p>
      <Button
        size="sm"
        variant="outline"
        nativeButton={false}
        render={
          <Link to="/p/$projectId/prompt-tracking" params={{ projectId }} />
        }
      >
        <Sparkles aria-hidden />
        AI Visibility Tracker
      </Button>
    </section>
  );
}
