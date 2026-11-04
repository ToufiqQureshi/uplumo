import { Outlet, createFileRoute } from "@tanstack/react-router";
import { StaffLayout } from "@/client/features/staff-console/StaffLayout";

export const Route = createFileRoute("/staff")({
  component: () => (
    <StaffLayout>
      <Outlet />
    </StaffLayout>
  ),
});
