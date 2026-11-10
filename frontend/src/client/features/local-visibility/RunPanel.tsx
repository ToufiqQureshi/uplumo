import type { ReactNode } from "react";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/client/components/ui/alert";
import { Button } from "@/client/components/ui/button";
import { Spinner } from "@/client/components/ui/spinner";
import { type Run, useRunPolling } from "./localVisibilityApi";

export function formatWhen(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** A paid action: the label always carries the most it can charge. */
export function PaidButton({
  label,
  credits,
  pending,
  disabled,
  onClick,
  variant = "default",
  type = "button",
}: {
  label: string;
  credits: number;
  pending: boolean;
  disabled?: boolean;
  onClick?: () => void;
  variant?: "default" | "outline";
  type?: "button" | "submit";
}) {
  return (
    <Button
      type={type}
      variant={variant}
      disabled={pending || disabled}
      onClick={onClick}
    >
      {pending ? <Spinner /> : null}
      {label} · up to {credits} credits
    </Button>
  );
}

/**
 * The state of a tab's newest run: collecting, failed, or the finished
 * result with where it came from. Polls (free) while the run is running.
 */
export function RunPanel({
  projectId,
  run,
  source,
  error,
  children,
}: {
  projectId: string;
  run: Run | undefined;
  source: string;
  error: Error | null;
  children: ReactNode;
}) {
  useRunPolling(projectId, run);
  return (
    <div className="space-y-4">
      {error ? (
        <Alert variant="destructive">
          <AlertTitle>Could not start</AlertTitle>
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      ) : null}
      {run?.status === "running" ? (
        <div
          role="status"
          className="flex items-center gap-2 rounded-lg border p-4 text-sm"
        >
          <Spinner />
          Collecting Google data. This usually takes about a minute; you can
          leave the page.
        </div>
      ) : null}
      {run?.status === "failed" ? (
        <Alert variant="destructive">
          <AlertTitle>The last check did not finish</AlertTitle>
          <AlertDescription>{run.error}</AlertDescription>
        </Alert>
      ) : null}
      {run?.status === "done" ? (
        <>
          <p className="text-xs text-muted-foreground">
            Source: {source} · DataForSEO, {formatWhen(run.createdAt)}
            {run.cached
              ? " · shown from the last 24 hours, nothing was charged"
              : ""}
          </p>
          {children}
        </>
      ) : null}
    </div>
  );
}

export function MissingNote({ missing }: { missing: string[] }) {
  if (missing.length === 0) return null;
  return (
    <p className="text-xs text-muted-foreground">
      Google did not return: {missing.join(", ")}. Those parts are left out.
    </p>
  );
}
