import { ShieldAlert } from "lucide-react";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/client/components/ui/alert";
import type { AuditResultsData } from "@/client/features/audit/results/types";

/** Report notices about JavaScript: app shells, and pages a render missed. */
export function RenderingWarnings({ data }: { data: AuditResultsData }) {
  const rendered = data.audit.config.renderJavaScript;
  const hasShells = data.issues.some(
    (issue) => issue.issueType === "javascript-rendering-suspected",
  );
  const unreadCount = data.pages.filter(
    (page) => page.fetchClass === "error",
  ).length;

  return (
    <>
      {hasShells && (
        <Alert variant="warning">
          <ShieldAlert />
          <AlertTitle>
            Some pages still look like JavaScript app shells.
          </AlertTitle>
          <AlertDescription>
            {rendered
              ? "Rendering was requested for up to 20 flagged pages, but these pages still did not return readable rendered content. Check them in a browser and make sure crawlers can read the main text."
              : "This audit checked initial HTML only. Check the affected pages and make sure crawlers can read the main text; a browser displaying content does not show what the crawler received."}
          </AlertDescription>
        </Alert>
      )}

      {rendered && unreadCount > 0 && (
        <Alert variant="warning">
          <ShieldAlert />
          <AlertTitle>
            We couldn't read {unreadCount}{" "}
            {unreadCount === 1 ? "page" : "pages"}.
          </AlertTitle>
          <AlertDescription>
            To see them, open Pages and filter Crawl result by Failed.
          </AlertDescription>
        </Alert>
      )}

      {rendered && (
        <p className="text-sm text-muted-foreground">
          This audit opted into JavaScript rendering for flagged app-shell pages
          only, with a 20-page maximum.
        </p>
      )}
    </>
  );
}
