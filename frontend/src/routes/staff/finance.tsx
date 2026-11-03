import { createFileRoute } from "@tanstack/react-router";
import { FinancePage } from "@/client/features/staff-console/FinancePage";

export const Route = createFileRoute("/staff/finance")({
  component: FinancePage,
});
