import { createFileRoute } from "@tanstack/react-router";
import { CustomersPage } from "@/client/features/staff-console/CustomersPage";

export const Route = createFileRoute("/staff/customers")({
  component: CustomersPage,
});
