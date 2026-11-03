import { createFileRoute } from "@tanstack/react-router";
import { CustomersPage } from "@/client/features/staff-customers/CustomersPage";

export const Route = createFileRoute("/_app/admin/customers")({
  component: CustomersPage,
});
