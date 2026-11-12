import { useQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { QueryError } from "@/client/components/QueryState";
import { SkeletonPage } from "@/client/components/SkeletonPresets";
import { Button } from "@/client/components/ui/button";
import {
  formatAmount,
  formatPaidOn,
  receiptQueryOptions,
} from "@/client/features/billing/receipts/receiptsApi";
import { SUPPORT_EMAIL } from "@/client/lib/support";

export const Route = createFileRoute("/_app/billing_/receipts/$paymentId")({
  component: ReceiptPage,
});

// Printing shows only the receipt sheet, without the app around it.
const printCss = `@media print {
  body * { visibility: hidden; }
  #receipt, #receipt * { visibility: visible; }
  #receipt { position: absolute; inset: 0 auto auto 0; width: 100%; border: 0; box-shadow: none; }
}`;

function ReceiptPage() {
  const { paymentId } = Route.useParams();
  const receipt = useQuery(receiptQueryOptions(paymentId));

  if (receipt.isPending) return <SkeletonPage />;
  if (receipt.isError)
    return (
      <div className="mx-auto max-w-2xl space-y-4 px-4 py-6">
        <QueryError
          error={receipt.error}
          fallback="We couldn't load this receipt."
          onRetry={() => void receipt.refetch()}
          isRetrying={receipt.isFetching}
        />
        <Button variant="ghost" render={<Link to="/billing" />}>
          Back to billing
        </Button>
      </div>
    );

  const r = receipt.data;
  const b = r.billedTo;
  return (
    <div className="mx-auto max-w-2xl space-y-4 px-4 py-6 pb-24">
      <style>{printCss}</style>
      <div className="flex flex-wrap justify-between gap-2">
        <Button variant="ghost" render={<Link to="/billing" />}>
          Back to billing
        </Button>
        <Button onClick={() => window.print()}>Download PDF</Button>
      </div>
      {!b.name ? (
        <p className="text-sm text-muted-foreground">
          Add your name or company and address under Billing details on the
          billing page to show them on this receipt.
        </p>
      ) : null}
      <article
        id="receipt"
        className="space-y-6 rounded-md bg-white p-6 text-sm text-neutral-900 shadow sm:p-9"
      >
        <div className="flex flex-wrap justify-between gap-4">
          <p className="font-serif text-3xl">Uplumo</p>
          <div className="text-right">
            <p className="text-xs font-medium uppercase tracking-widest">
              Payment receipt
            </p>
            <p className="font-mono">{r.receiptNo}</p>
          </div>
        </div>
        <div className="grid gap-5 sm:grid-cols-2">
          <div>
            <p className="mb-1 text-xs uppercase tracking-wider text-neutral-500">
              From
            </p>
            <p className="font-semibold">Uplumo</p>
            <p>{SUPPORT_EMAIL}</p>
          </div>
          {b.name ? (
            <div>
              <p className="mb-1 text-xs uppercase tracking-wider text-neutral-500">
                Billed to
              </p>
              <p className="font-semibold">{b.name}</p>
              {b.address ? (
                <p className="whitespace-pre-line">{b.address}</p>
              ) : null}
              {b.country ? <p>{b.country}</p> : null}
              {b.taxId ? (
                <p>
                  Tax ID: <span className="font-mono">{b.taxId}</span>
                </p>
              ) : null}
            </div>
          ) : null}
        </div>
        <dl className="grid gap-3 border-y border-neutral-200 py-3 sm:grid-cols-3">
          <div>
            <dt className="text-xs uppercase tracking-wider text-neutral-500">
              Paid on
            </dt>
            <dd>{formatPaidOn(r.paidAt)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wider text-neutral-500">
              Paid via
            </dt>
            <dd>Razorpay</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wider text-neutral-500">
              Payment ID
            </dt>
            <dd className="font-mono break-all">{r.paymentId}</dd>
          </div>
        </dl>
        <table className="w-full">
          <thead>
            <tr className="border-b border-neutral-200 text-left text-xs uppercase tracking-wider text-neutral-500">
              <th className="py-2 font-medium">Description</th>
              <th className="py-2 text-right font-medium">Amount</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-neutral-200">
              <td className="py-2">{r.item}</td>
              <td className="py-2 text-right tabular-nums">
                {formatAmount(r.amount, r.currency)}
              </td>
            </tr>
            {r.refunded > 0 ? (
              <tr className="border-b border-neutral-200">
                <td className="py-2">Refunded</td>
                <td className="py-2 text-right tabular-nums">
                  −{formatAmount(r.refunded, r.currency)}
                </td>
              </tr>
            ) : null}
            <tr className="font-semibold">
              <td className="py-2">Total paid</td>
              <td className="py-2 text-right tabular-nums">
                {formatAmount(r.amount - r.refunded, r.currency)}
              </td>
            </tr>
          </tbody>
        </table>
        <p className="border-t border-neutral-200 pt-3 text-xs text-neutral-500">
          This is a receipt for a payment received. It is not a tax invoice. The
          seller is not registered under GST, so no GST was charged.
        </p>
      </article>
    </div>
  );
}
