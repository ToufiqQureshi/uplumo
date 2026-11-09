import { useState, type ReactNode } from "react";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/client/components/ui/alert-dialog";
import { Button } from "@/client/components/ui/button";

/**
 * Asks before an action that cannot be undone: a delete, or one that spends
 * credits. Render it only while it is open. `onConfirm` does not close it, so
 * the caller closes it right away or when the mutation settles. While
 * `pending`, Cancel and Escape do nothing.
 */
export function ConfirmDialog({
  title,
  confirmLabel,
  onConfirm,
  onClose,
  pending = false,
  destructive = false,
  children,
}: {
  title: string;
  confirmLabel: string;
  onConfirm: () => void;
  onClose: () => void;
  pending?: boolean;
  destructive?: boolean;
  children: ReactNode;
}) {
  return (
    <AlertDialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{children}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>Cancel</AlertDialogCancel>
          <Button
            variant={destructive ? "destructive" : "default"}
            pending={pending}
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

type ConfirmRequest = {
  title: string;
  body: ReactNode;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => void;
};

/**
 * The in-app replacement for window.confirm: call `ask(...)` instead of
 * running the action and render `dialog` anywhere on the page. The action
 * runs only after Confirm; the dialog closes first.
 */
export function useConfirm(): {
  ask: (request: ConfirmRequest) => void;
  dialog: ReactNode;
} {
  const [request, setRequest] = useState<ConfirmRequest | null>(null);
  const dialog = request ? (
    <ConfirmDialog
      title={request.title}
      confirmLabel={request.confirmLabel}
      destructive={request.destructive}
      onClose={() => setRequest(null)}
      onConfirm={() => {
        setRequest(null);
        request.onConfirm();
      }}
    >
      {request.body}
    </ConfirmDialog>
  ) : null;
  return { ask: setRequest, dialog };
}
