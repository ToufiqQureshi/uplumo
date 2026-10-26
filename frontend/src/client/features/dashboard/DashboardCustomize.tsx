import { useState } from "react";
import { ArrowDown, ArrowUp, SlidersHorizontal } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { Switch } from "@/client/components/ui/switch";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/client/components/ui/sheet";
import { QueryError } from "@/client/components/QueryState";
import {
  cardLabel,
  moveCard,
  type ResolvedCard,
} from "@/client/features/dashboard/dashboardLayout";

/**
 * Lets a member show, hide and reorder dashboard cards. Edits stay local
 * until Save; Reset drops the saved layout so new cards appear by default.
 */
export function DashboardCustomize({
  cards,
  onSave,
  onReset,
  saving,
  error,
}: {
  cards: ResolvedCard[];
  onSave: (cards: ResolvedCard[]) => Promise<unknown>;
  onReset: () => Promise<unknown>;
  saving: boolean;
  error: unknown;
}) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState(cards);

  const close = async (action: () => Promise<unknown>) => {
    try {
      await action();
      setOpen(false);
    } catch {
      // The error stays visible in the sheet; the saved layout is unchanged.
    }
  };

  return (
    <Sheet
      open={open}
      onOpenChange={(next) => {
        if (next) setDraft(cards);
        setOpen(next);
      }}
    >
      <SheetTrigger
        render={
          <Button variant="outline" size="sm">
            <SlidersHorizontal className="size-4" aria-hidden />
            Customize
          </Button>
        }
      />
      <SheetContent className="flex flex-col">
        <SheetHeader>
          <SheetTitle>Customize your dashboard</SheetTitle>
          <SheetDescription>
            Turn cards on or off and move them up or down. Only you see this
            layout.
          </SheetDescription>
        </SheetHeader>
        <ul className="flex-1 space-y-2 overflow-y-auto px-4">
          {draft.map((card, index) => {
            const label = cardLabel(card.id);
            return (
              <li
                key={card.id}
                className="flex items-center gap-3 rounded-lg border border-border px-3 py-2"
              >
                <Switch
                  checked={!card.hidden}
                  onCheckedChange={(checked) =>
                    setDraft(
                      draft.map((item) =>
                        item.id === card.id
                          ? { ...item, hidden: !checked }
                          : item,
                      ),
                    )
                  }
                  aria-label={`Show ${label}`}
                />
                <span
                  className={`flex-1 text-sm ${card.hidden ? "text-muted-foreground" : ""}`}
                >
                  {label}
                </span>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label={`Move ${label} up`}
                  disabled={index === 0}
                  onClick={() => setDraft(moveCard(draft, index, -1))}
                >
                  <ArrowUp className="size-4" aria-hidden />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label={`Move ${label} down`}
                  disabled={index === draft.length - 1}
                  onClick={() => setDraft(moveCard(draft, index, 1))}
                >
                  <ArrowDown className="size-4" aria-hidden />
                </Button>
              </li>
            );
          })}
        </ul>
        {error ? (
          <div className="px-4">
            <QueryError
              error={error}
              fallback="Could not save your layout. Your dashboard is unchanged."
            />
          </div>
        ) : null}
        <SheetFooter className="flex-row justify-between">
          <Button
            variant="ghost"
            disabled={saving}
            onClick={() => void close(onReset)}
          >
            Reset to default
          </Button>
          <Button
            pending={saving}
            onClick={() => void close(() => onSave(draft))}
          >
            Save layout
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
