import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/client/components/ui/dialog";
import { Input } from "@/client/components/ui/input";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { runUserAction, type UserAction } from "./staffApi";

const copy: Record<
  UserAction,
  { title: string; body: string; button: string }
> = {
  block: {
    title: "Block this user?",
    body: "They are signed out now and cannot sign in, use the app, API keys or MCP until unblocked. Their data stays. A paid plan keeps running: cancel it in Razorpay if they should stop paying.",
    button: "Block user",
  },
  unblock: {
    title: "Unblock this user?",
    body: "They can sign in again with their data as it was.",
    button: "Unblock user",
  },
  delete: {
    title: "Delete this user for good?",
    body: "Their account, their workspace and everything in it is erased. This cannot be undone. Payment records are kept for accounting without their email. A user with a running paid plan, or whose workspace has other members, cannot be deleted here.",
    button: "Delete forever",
  },
};

export function UserActionDialog({
  action,
  user,
  onClose,
}: {
  action: UserAction;
  user: { userId: string; email: string };
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [reason, setReason] = useState("");
  const [confirmEmail, setConfirmEmail] = useState("");
  const run = useMutation({
    mutationFn: () =>
      runUserAction(action, {
        userId: user.userId,
        ...(action === "block" ? { reason } : {}),
        ...(action === "delete" ? { confirmEmail } : {}),
      }),
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["staff"] });
      await queryClient.invalidateQueries({ queryKey: ["staffCustomers"] });
      onClose();
    },
  });
  const text = copy[action];
  const confirmed =
    action !== "delete" ||
    confirmEmail.trim().toLowerCase() === user.email.toLowerCase();

  return (
    <Dialog open onOpenChange={(open) => !open && !run.isPending && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{text.title}</DialogTitle>
          <DialogDescription>
            <strong>{user.email}</strong>. {text.body}
          </DialogDescription>
        </DialogHeader>
        {action === "block" ? (
          <Input
            aria-label="Reason (optional)"
            placeholder="Reason (optional, staff only)"
            maxLength={500}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        ) : null}
        {action === "delete" ? (
          <div className="space-y-1">
            <label htmlFor="confirm-email" className="text-sm">
              Type <strong>{user.email}</strong> to confirm
            </label>
            <Input
              id="confirm-email"
              autoComplete="off"
              value={confirmEmail}
              onChange={(e) => setConfirmEmail(e.target.value)}
            />
          </div>
        ) : null}
        {run.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {getStandardErrorMessage(run.error)}
          </p>
        ) : null}
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={run.isPending}>
            Cancel
          </Button>
          <Button
            variant={action === "unblock" ? "default" : "destructive"}
            disabled={!confirmed || run.isPending}
            onClick={() => run.mutate()}
          >
            {run.isPending ? "Working…" : text.button}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
