import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { Check, Code, Copy } from "lucide-react";
import { CardShell } from "@/client/components/CardShell";
import { ErrorState } from "@/client/components/ErrorState";
import { Button } from "@/client/components/ui/button";
import { useCopy } from "@/client/hooks/useCopy";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { createAnalyticsSite } from "./analyticsApi";

/**
 * Gets the site's tracking snippet (the API returns the existing one on every
 * later call) and shows it with a copy button and install steps.
 */
export function TrackingCodeCard({ projectId }: { projectId: string }) {
  const site = useMutation({
    mutationFn: () => createAnalyticsSite(projectId),
    meta: { errorToast: false },
  });
  const { copied, copy } = useCopy();
  const [recognition, setRecognition] = useState(false);
  const snippet = site.data?.snippet.replace(
    "<script ",
    recognition ? '<script data-recognition="true" ' : "<script ",
  );

  return (
    <CardShell
      title="Install the tracking code"
      icon={<Code className="size-4 text-primary" />}
    >
      <div className="space-y-4 text-sm">
        <ol className="list-decimal space-y-1 pl-5 text-muted-foreground">
          <li>Get your tracking code below.</li>
          <li>
            Paste it inside the <code>&lt;head&gt;</code> of every page (or your
            site builder&rsquo;s &ldquo;custom code&rdquo; box).
          </li>
          <li>Open your site once. Visits show up here within a minute.</li>
        </ol>
        <p className="text-muted-foreground">
          To count clicks on a button or link, add{" "}
          <code>data-uplumo-event=&quot;Signup&quot;</code> to it (use any
          name). Clicks show up under Custom events and Button and link
          destinations. External HTTP(S) links are tracked automatically.
        </p>
        <p className="text-muted-foreground">
          Links provide their destination automatically. For a JavaScript
          button, add <code>data-uplumo-destination="/booking"</code>. Other
          button destinations remain unknown. Query strings and fragments are
          removed from stored destinations; keep personal details out of paths
          and labels.
        </p>
        <p className="text-muted-foreground">
          To count a successful lead, booking or order, call{" "}
          <code>window.uplumo(&quot;LeadSubmitted&quot;)</code> only after your
          server confirms success. A button click alone does not confirm a
          completed lead or order.
        </p>
        <label className="flex items-start gap-2 rounded-lg border border-border p-3">
          <input
            type="checkbox"
            className="mt-1"
            checked={recognition}
            onChange={(e) => setRecognition(e.target.checked)}
          />
          <span>
            Enable new/returning browser recognition in this snippet
            <span className="mt-1 block text-xs text-muted-foreground">
              Optional: stores a random browser ID in first-party localStorage
              for 180 days. Only use the enabled snippet after the visitor
              grants the required analytics consent. Without recognition, these
              visits remain unclassified.
            </span>
          </span>
        </label>
        {recognition ? (
          <p className="text-muted-foreground">
            Connect your consent manager: load this snippet after permission;
            call <code>window.uplumoConsent(false)</code> to stop collection and
            clear Uplumo identifiers when permission is withdrawn. No banner or
            content is injected. Browsers without supported secure storage and
            Web Locks remain unclassified.
          </p>
        ) : null}
        <p className="text-muted-foreground">
          Keep names, email addresses and other personal details out of event
          names and UTM tags.
        </p>
        <p className="text-muted-foreground">
          Sessions use a random ID in first-party sessionStorage (30 minutes
          idle, 24 hours maximum). The tracker sets no cookies, honors Do Not
          Track and Global Privacy Control, and does not send data to ad
          platforms. Tell your visitors and update your privacy notice.
        </p>
        {site.data ? (
          <div className="space-y-2">
            <pre className="overflow-x-auto rounded-lg border border-border bg-muted p-3 font-mono text-xs whitespace-pre-wrap break-all">
              {snippet}
            </pre>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                snippet ? void copy(snippet, "Tracking code copied") : undefined
              }
            >
              {copied ? <Check /> : <Copy />}
              {copied ? "Copied" : "Copy code"}
            </Button>
          </div>
        ) : (
          <Button onClick={() => site.mutate()} disabled={site.isPending}>
            {site.isPending ? "Getting your code…" : "Get my tracking code"}
          </Button>
        )}
        {site.isError ? (
          <ErrorState
            variant="inline"
            message={getStandardErrorMessage(site.error)}
            onRetry={() => site.mutate()}
            isRetrying={site.isPending}
          />
        ) : null}
      </div>
    </CardShell>
  );
}
