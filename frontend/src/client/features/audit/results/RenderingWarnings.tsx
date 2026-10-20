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
            Some pages may need JavaScript to show their content.
          </AlertTitle>
          <AlertDescription>
            This audit checks initial HTML. JavaScript rendering is unavailable;
            inspect the affected pages in a browser and serve essential content
            in the initial HTML. This warning does not prove Google cannot index
            them.
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
          This saved audit requested rendering. That setting does not prove
          JavaScript was executed; verify the page in a browser.
        </p>
      )}
    </>
  );
}
