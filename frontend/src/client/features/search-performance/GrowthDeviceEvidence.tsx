import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";
import { getGrowthDevices, type GrowthReport } from "./growthApi";
import { GrowthMobileResults } from "./GrowthMobileResults";
import { growthCount, growthRate } from "./growthFormat";
export function GrowthDeviceEvidence({
  projectId,
  report,
}: {
  projectId: string;
  report: GrowthReport;
}) {
  const devices = useMutation({
    mutationFn: () =>
      getGrowthDevices(projectId, {
        ...report.range,
        goalEvent: report.goalEvent,
      }),
    retry: false,
  });
  return (
    <>
      <Button
        variant="outline"
        size="sm"
        disabled={devices.isPending}
        onClick={() => devices.mutate()}
      >
        Compare devices
      </Button>
      {devices.isError ? (
        <QueryError
          error={devices.error}
          fallback="Could not inspect device performance."
        />
      ) : null}
      {devices.data ? (
        <div className="space-y-2 text-xs">
          {devices.data.limited ? (
            <p>
              Analytics data is limited. Avoid drawing business conclusions from
              these rates.
            </p>
          ) : null}
          {devices.data.rows.length === 0 ? (
            <p>No device rows returned.</p>
          ) : (
            <>
              <GrowthMobileResults
                label="Google organic device results"
                rows={devices.data.rows.map((row) => ({
                  key: row.deviceCategory,
                  title: row.deviceCategory,
                  metrics: [
                    { label: "Sessions", value: growthCount(row.sessions) },
                    {
                      label: report.goalEvent
                        ? `${report.goalEvent} rate`
                        : "Key-event rate",
                      value: growthRate(
                        (report.goalEvent
                          ? row.goalRate
                          : row.sessionKeyEventRate) ?? null,
                      ),
                    },
                    {
                      label: "All key events",
                      value: growthCount(row.keyEvents),
                    },
                  ],
                }))}
              />
              <div className="hidden overflow-x-auto sm:block">
                <table className="w-full text-left">
                  <caption className="sr-only">
                    Google organic traffic by device
                  </caption>
                  <thead>
                    <tr>
                      <th className="p-2">Device</th>
                      <th className="p-2">Sessions</th>
                      <th className="p-2">All key events</th>
                      <th className="p-2">
                        {report.goalEvent
                          ? `${report.goalEvent} session rate`
                          : "Session key-event rate"}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {devices.data.rows.map((row) => (
                      <tr key={row.deviceCategory}>
                        <td className="p-2">{row.deviceCategory}</td>
                        <td className="p-2">{row.sessions ?? "Unavailable"}</td>
                        <td className="p-2">
                          {row.keyEvents ?? "Unavailable"}
                        </td>
                        <td className="p-2">
                          {(report.goalEvent
                            ? row.goalRate
                            : row.sessionKeyEventRate) == null
                            ? "Unavailable"
                            : `${((report.goalEvent ? (row.goalRate ?? 0) : (row.sessionKeyEventRate ?? 0)) * 100).toFixed(1)}%`}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
          <p>
            Site-wide comparison for {report.range.startDate} –{" "}
            {report.range.endDate}. Small samples and different audience/page
            mixes can explain differences. This does not measure speed or
            establish a cause.
          </p>
        </div>
      ) : null}
    </>
  );
}
