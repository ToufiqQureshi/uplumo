import { useId, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";
import { Button } from "@/client/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/client/components/ui/alert-dialog";
import { resetAnalyticsUser } from "@/client/lib/posthog";
import { ApiError } from "@/client/lib/uplumoApi";
import { signOutAndRedirect } from "@/lib/auth-client";
import { accountDeletionReasons } from "@/shared/accountDeletion";

const deletionResponse = z.object({ success: z.literal(true) });
const deletionError = z.object({ code: z.string(), message: z.string() });
const retryMessage = "Could not delete your account. Please try again.";

async function requestDeletion(reason: string) {
  const response = await fetch("/api/auth/delete-account", {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ confirm: true, reason: reason || undefined }),
  }).catch(() => {
    throw new Error(retryMessage);
  });
  const data: unknown = await response.json().catch(() => undefined);
  if (!response.ok) {
    const result = deletionError.safeParse(data);
    throw new ApiError(
      response.status,
      result.data?.code ?? "unknown",
      result.data?.message ?? retryMessage,
    );
  }
  if (!deletionResponse.safeParse(data).success) {
    throw new Error(retryMessage);
  }
}

export function DeleteAccountDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const sending = useRef(false);
  const reasonID = useId();
  const [reason, setReason] = useState("");
  const deletion = useMutation({
    mutationFn: requestDeletion,
    retry: false,
    onSuccess: () => {
      resetAnalyticsUser();
      window.location.replace("/");
    },
    onError: () => {
      sending.current = false;
    },
  });
  const busy = deletion.isPending || deletion.isSuccess;
  const needsSignIn =
    deletion.error instanceof ApiError &&
    (deletion.error.status === 401 ||
      deletion.error.code === "RECENT_SIGN_IN_REQUIRED");

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (busy) return;
        deletion.reset();
        if (!next) setReason("");
        onOpenChange(next);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete your Uplumo account?</AlertDialogTitle>
          <AlertDialogDescription>
            This permanently deletes your account, ends your sessions, and
            removes your personal workspaces and their data. Payment records are
            kept for accounting.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="space-y-2">
          <label htmlFor={reasonID} className="block text-sm font-medium">
            Why are you leaving? (optional)
          </label>
          <select
            id={reasonID}
            value={reason}
            disabled={busy}
            onChange={(event) => setReason(event.target.value)}
            className="h-10 w-full rounded-lg border border-input bg-background px-2 text-sm focus-visible:outline-2 focus-visible:outline-ring disabled:opacity-50"
          >
            <option value="">Prefer not to say</option>
            {Object.entries(accountDeletionReasons).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
          <p className="text-xs text-muted-foreground">
            Your answer is counted without your account details.
          </p>
        </div>
        {deletion.error && (
          <p role="alert" className="text-sm text-destructive">
            {deletion.error.message}
          </p>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>Cancel</AlertDialogCancel>
          {needsSignIn ? (
            <Button onClick={signOutAndRedirect}>Sign in again</Button>
          ) : (
            <AlertDialogAction
              variant="destructive"
              disabled={busy}
              onClick={() => {
                if (sending.current) return;
                sending.current = true;
                deletion.mutate(reason);
              }}
            >
              {busy ? "Deleting account…" : "Delete my account"}
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
