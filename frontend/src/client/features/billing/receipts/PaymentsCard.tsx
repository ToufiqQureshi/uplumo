import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { revalidateLogic } from "@tanstack/react-form";
import { Link } from "@tanstack/react-router";
import { toast } from "sonner";
import { z } from "zod";
import { useAppForm } from "@/client/components/form/useAppForm";
import { Button } from "@/client/components/ui/button";
import {
  billingDetailsQueryOptions,
  formatAmount,
  formatPaidOn,
  paymentsQueryOptions,
  refundStatus,
  saveBillingDetails,
  type BillingDetails,
} from "./receiptsApi";

/**
 * The workspace's payments with a receipt for each, and who the receipts
 * are made out to. Owners and admins only: the server refuses others.
 */
export function PaymentsCard() {
  const payments = useQuery(paymentsQueryOptions);
  if (payments.isError)
    return (
      <p role="alert" className="text-sm">
        Could not load your payments.{" "}
        <Button variant="ghost" onClick={() => void payments.refetch()}>
          Retry
        </Button>
      </p>
    );
  // A workspace that never paid has nothing to show here.
  if (!payments.data?.length) return null;
  return (
    <section className="space-y-5 rounded-xl border border-border p-5">
      <div>
        <h2 className="text-lg font-semibold">Payments and receipts</h2>
        <p className="text-sm text-muted-foreground">
          Every payment from this workspace. Open a receipt to save it as a PDF
          for your accounts.
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[560px] text-sm">
          <thead>
            <tr className="border-b border-border text-left text-xs text-muted-foreground">
              <th className="py-2 pr-3 font-medium">Date</th>
              <th className="py-2 pr-3 font-medium">Receipt no.</th>
              <th className="py-2 pr-3 font-medium">For</th>
              <th className="py-2 pr-3 text-right font-medium">Amount</th>
              <th className="py-2 pr-3 font-medium">Status</th>
              <th className="py-2" />
            </tr>
          </thead>
          <tbody>
            {payments.data.map((p) => (
              <tr key={p.paymentId} className="border-b border-border">
                <td className="py-3 pr-3">{formatPaidOn(p.paidAt)}</td>
                <td className="py-3 pr-3 font-mono text-xs">{p.receiptNo}</td>
                <td className="py-3 pr-3">{p.item}</td>
                <td className="py-3 pr-3 text-right tabular-nums">
                  {formatAmount(p.amount, p.currency)}
                </td>
                <td className="py-3 pr-3">{refundStatus(p)}</td>
                <td className="py-3 text-right">
                  <Button
                    variant="outline"
                    size="sm"
                    render={
                      <Link
                        to="/billing/receipts/$paymentId"
                        params={{ paymentId: p.paymentId }}
                      />
                    }
                  >
                    View receipt
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <BillingDetailsSection />
    </section>
  );
}

const detailsSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Enter the name to put on receipts.")
    .max(200, "Use 200 characters or fewer."),
  address: z.string().trim().max(500, "Use 500 characters or fewer."),
  country: z.string().trim().max(80, "Use 80 characters or fewer."),
  taxId: z.string().trim().max(40, "Use 40 characters or fewer."),
});

function BillingDetailsSection() {
  const details = useQuery(billingDetailsQueryOptions);
  if (details.isPending) return null;
  if (details.isError)
    return (
      <p role="alert" className="text-sm">
        Could not load your billing details.{" "}
        <Button variant="ghost" onClick={() => void details.refetch()}>
          Retry
        </Button>
      </p>
    );
  return <BillingDetailsForm initial={details.data} />;
}

function BillingDetailsForm({ initial }: { initial: BillingDetails }) {
  const queryClient = useQueryClient();
  const save = useMutation({
    mutationFn: saveBillingDetails,
    retry: false,
    onSuccess: (saved) => {
      queryClient.setQueryData(billingDetailsQueryOptions.queryKey, saved);
      void queryClient.invalidateQueries({
        queryKey: ["billing", "receipt"],
      });
      toast.success("Billing details saved");
    },
  });
  const form = useAppForm({
    defaultValues: initial,
    validationLogic: revalidateLogic(),
    validators: { onDynamic: detailsSchema },
    onSubmit: async ({ value }) => {
      await save.mutateAsync(value);
    },
  });
  return (
    <div className="space-y-3 border-t border-border pt-5">
      <div>
        <h3 className="font-semibold">Billing details</h3>
        <p className="text-sm text-muted-foreground">
          Printed on every receipt, including ones you open again later.
        </p>
      </div>
      <form.AppForm>
        <form.Form className="grid max-w-2xl gap-3 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <form.AppField name="name">
              {(field) => (
                <field.TextField label="Name or company name" required />
              )}
            </form.AppField>
          </div>
          <div className="sm:col-span-2">
            <form.AppField name="address">
              {(field) => <field.TextareaField label="Address" rows={2} />}
            </form.AppField>
          </div>
          <form.AppField name="country">
            {(field) => <field.TextField label="Country" />}
          </form.AppField>
          <form.AppField name="taxId">
            {(field) => (
              <field.TextField
                label="Tax ID (optional)"
                description="GSTIN, VAT number or similar, if your accountant needs it."
              />
            )}
          </form.AppField>
          <div className="sm:col-span-2">
            <form.SubmitButton>Save details</form.SubmitButton>
          </div>
        </form.Form>
      </form.AppForm>
    </div>
  );
}
