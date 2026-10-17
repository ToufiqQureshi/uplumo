import { sortBy } from "remeda";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Textarea } from "@/client/components/ui/textarea";
import { ProjectPageHeader } from "@/client/features/projects/ProjectPageHeader";
import { enquiryApi, enquiryKey } from "./enquiryApi";
const date = (value: string) =>
  new Date(value).toLocaleDateString("en-IN", { timeZone: "UTC" });
const rate = (value: number) => `${value.toFixed(1)}%`;
const signals = {
  verify_tracking: "Verify event tracking first",
  low_data: "More visits needed",
  traffic_changed: "Traffic changed too",
  enquiry_drop: "Enquiry rate fell — investigate",
  no_goal_events: "No goal events on this page",
  stable: "No drop flag",
};
const actions = {
  verify_tracking:
    "Trigger a test enquiry and confirm that its event arrives. Check consent, blockers and tracker installation before interpreting zeros.",
  low_data:
    "Each week needs at least 30 visitor-days before a drop can be flagged. Review the actual counts while more data arrives.",
  traffic_changed:
    "Compare channel and device mix in Analytics. A change in visitor intent can affect enquiries without a broken form.",
  enquiry_drop:
    "Test the mobile form, WhatsApp destination and booking link. Check validation errors and consent. Record a baseline before making a change.",
  no_goal_events:
    "Confirm this page should drive the selected goal, then test the button and event wiring. A click is an intent signal, not a confirmed lead.",
  stable:
    "Keep the goal connected and check after material site changes. A missing flag does not prove that every enquiry flow works.",
};
export function EnquiriesPage({
  projectId,
  initialPath,
}: {
  projectId: string;
  initialPath: string;
}) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: enquiryKey(projectId),
    queryFn: () => enquiryApi.report(projectId),
    staleTime: 60000,
  });
  const [page, setPage] = useState(initialPath);
  const [note, setNote] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: enquiryKey(projectId) });
  const save = useMutation({
    mutationFn: () => enquiryApi.saveChange(projectId, page, note),
    onSuccess: async (data) => {
      setSelected(data.id);
      setNote("");
      await refresh();
    },
  });
  const remove = useMutation({
    mutationFn: (id: string) => enquiryApi.deleteChange(projectId, id),
    onSuccess: async () => {
      setSelected(null);
      await refresh();
    },
  });
  const compare = useQuery({
    queryKey: [...enquiryKey(projectId), "change", selected],
    queryFn: () => enquiryApi.compare(projectId, selected!),
    enabled: Boolean(selected),
    staleTime: 60000,
  });
  const data = query.data;
  if (query.isPending)
    return (
      <p role="status" className="p-6">
        Loading enquiry evidence…
      </p>
    );
  if (!data)
    return (
      <div role="alert" className="p-6">
        {query.error?.message}
        <Button onClick={() => void query.refetch()}>Retry</Button>
      </div>
    );
  const ordered = sortBy(data.pages, [
    (p) => (p.signal === "enquiry_drop" ? 0 : 1),
    "asc",
  ]);
  return (
    <div className="h-full overflow-auto px-4 py-6 pb-24 md:px-6">
      <div className="mx-auto max-w-6xl space-y-6">
        <ProjectPageHeader
          projectId={projectId}
          title="Missed Enquiries Finder"
        />
        <p className="max-w-3xl text-sm text-muted-foreground">
          Find pages where tracked enquiry actions fell, inspect the evidence,
          and measure a deliberate fix. This reads your Uplumo Analytics events
          without spending credits. It cannot count untracked enquiries or prove
          lost revenue.
        </p>
        {query.error ? (
          <p role="alert">
            Could not refresh; showing the last saved response.
          </p>
        ) : null}
        <GoalForm
          key={data.eventName}
          projectId={projectId}
          saved={data.eventName}
          onSaved={refresh}
        />
        {!data.siteConfigured ? (
          <p className="rounded-lg border p-4">
            Install the tracker from{" "}
            <Link
              to="/p/$projectId/analytics"
              params={{ projectId }}
              search={{ days: 30 }}
              className="underline"
            >
              Analytics
            </Link>{" "}
            to start collecting pageviews and events.
          </p>
        ) : null}
        {data.eventName && !data.goalObserved ? (
          <p role="status" className="rounded-lg border p-4">
            No matching event was recorded in the last 28 complete days. Verify
            instrumentation first; zero recorded events do not mean zero
            enquiries.
          </p>
        ) : null}
        <section className="space-y-3 rounded-xl border bg-card p-4">
          <h2 className="text-lg font-semibold">Lead Flow Guard</h2>
          <p className="text-xs text-muted-foreground">
            Two complete UTC weeks: {date(data.beforeStart)}–
            {date(data.currentStart)} versus {date(data.currentStart)}–
            {date(data.until)} (end dates excluded). A drop flag needs 30
            visitor-days in each week, at least 5 earlier goal visitors, traffic
            within ±20%, and a goal rate down at least 50%. Top 50 pages by
            visitor-days; flags appear here when this page loads.
          </p>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr>
                  <th className="p-2">Page</th>
                  <th className="p-2">Earlier week</th>
                  <th className="p-2">Latest week</th>
                  <th className="p-2">Evidence & next action</th>
                </tr>
              </thead>
              <tbody>
                {ordered.map((p) => (
                  <tr key={p.path} className="border-t align-top">
                    <td className="max-w-64 break-all p-2">
                      <Button
                        variant="link"
                        className="h-auto whitespace-normal p-0 text-left"
                        onClick={() => setPage(p.path)}
                      >
                        {p.path}
                      </Button>
                    </td>
                    <td className="p-2">
                      {p.before.goalVisitors}/{p.before.visitors}
                      <br />
                      {rate(p.before.rate)}
                    </td>
                    <td className="p-2">
                      {p.after.goalVisitors}/{p.after.visitors}
                      <br />
                      {rate(p.after.rate)}
                    </td>
                    <td className="max-w-md p-2">
                      <p
                        className={
                          p.signal === "enquiry_drop"
                            ? "font-semibold text-destructive"
                            : "font-medium"
                        }
                      >
                        {signals[p.signal]}
                      </p>
                      <p className="mt-1 text-xs leading-5 text-muted-foreground">
                        {actions[p.signal]}
                      </p>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!data.pages.length ? (
            <p className="text-sm">
              Choose an event and collect pageviews to see page evidence.
            </p>
          ) : null}
          <p className="text-xs text-muted-foreground">
            Counts are goal visitor-days / page visitor-days. Daily anonymous
            IDs rotate; one person can count on several days. Repeated clicks
            count once per visitor-day on each page, only with a matching
            pageview. This measures the selected event, not confirmed sales or
            people.
          </p>
        </section>
        <section className="space-y-3 rounded-xl border bg-card p-4">
          <h2 className="text-lg font-semibold">Measure a page change</h2>
          <p className="text-sm text-muted-foreground">
            Before editing or publishing, save a baseline of the last seven
            complete days and describe your intended change. After seven
            complete days following the change date, compare its goal rate. This
            is an observation, not an A/B test or proof of causation.
          </p>
          <label className="block space-y-1 text-sm">
            Page path
            <Input
              value={page}
              onChange={(e) => setPage(e.target.value)}
              placeholder="/services"
              maxLength={2048}
            />
          </label>
          <label className="block space-y-1 text-sm">
            Planned change
            <Textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              maxLength={500}
              placeholder="Move the WhatsApp button above the fold on mobile"
            />
          </label>
          <Button
            disabled={
              !data.eventName || !page.trim() || !note.trim() || save.isPending
            }
            onClick={() => save.mutate()}
          >
            Save measurement baseline
          </Button>
          {save.error ? (
            <p role="alert" className="text-sm text-destructive">
              {save.error.message}
            </p>
          ) : null}
          <ul className="space-y-2">
            {data.changes.map((c) => (
              <li
                key={c.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3 text-sm"
              >
                <div>
                  <p className="font-medium">
                    {c.path} · {date(c.createdAt)}
                  </p>
                  <p>{c.note}</p>
                  <p className="text-xs text-muted-foreground">
                    {c.eventName}: {c.baseline.goalVisitors}/
                    {c.baseline.visitors} · {rate(c.baseline.rate)}
                  </p>
                </div>
                <div>
                  <Button variant="outline" onClick={() => setSelected(c.id)}>
                    Compare
                  </Button>
                  <Button
                    variant="ghost"
                    disabled={remove.isPending}
                    onClick={() => {
                      if (
                        window.confirm(
                          "Delete this saved measurement baseline?",
                        )
                      )
                        remove.mutate(c.id);
                    }}
                  >
                    Delete
                  </Button>
                </div>
              </li>
            ))}
          </ul>
          {remove.error ? <p role="alert">{remove.error.message}</p> : null}
          {compare.isFetching ? (
            <p role="status">Loading comparison…</p>
          ) : compare.error ? (
            <p role="alert">{compare.error.message}</p>
          ) : compare.data ? (
            <div className="space-y-1 rounded-lg bg-muted p-4 text-sm">
              <h3 className="font-semibold">
                {compare.data.change.path} · {compare.data.change.eventName}
              </h3>
              {compare.data.ready ? (
                <>
                  <p>
                    Before: {compare.data.change.baseline.goalVisitors}/
                    {compare.data.change.baseline.visitors} ·{" "}
                    {rate(compare.data.change.baseline.rate)}
                  </p>
                  <p>
                    After: {compare.data.after.goalVisitors}/
                    {compare.data.after.visitors} ·{" "}
                    {rate(compare.data.after.rate)}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Post-change window: {date(compare.data.afterStart)}–
                    {date(compare.data.afterEnd)} (end excluded). Check traffic
                    mix, tracking and other changes before attributing any
                    difference to this edit.
                  </p>
                </>
              ) : (
                <p>
                  Collecting the first complete post-change week. Comparison
                  available {date(compare.data.afterEnd)} UTC.
                </p>
              )}
            </div>
          ) : null}
        </section>
      </div>
    </div>
  );
}
function GoalForm({
  projectId,
  saved,
  onSaved,
}: {
  projectId: string;
  saved: string;
  onSaved: () => Promise<unknown>;
}) {
  const [name, setName] = useState(saved);
  const mutation = useMutation({
    mutationFn: () => enquiryApi.saveGoal(projectId, name),
    onSuccess: onSaved,
  });
  return (
    <section className="space-y-3 rounded-xl border bg-card p-4">
      <h2 className="font-semibold">Which action matters?</h2>
      <p className="text-sm text-muted-foreground">
        Use the exact custom event name, such as WhatsApp, Booking or Enquiry
        submitted. Choose a success event for completed enquiries; button clicks
        measure intent. Add <code>data-uplumo-event="WhatsApp"</code> to the
        relevant button, or send <code>window.uplumo("Enquiry submitted")</code>{" "}
        only after your form succeeds.
      </p>
      <label className="block space-y-1 text-sm">
        Goal event name
        <Input
          value={name}
          disabled={mutation.isPending}
          maxLength={64}
          onChange={(e) => setName(e.target.value)}
        />
      </label>
      <Button
        variant="outline"
        disabled={!name.trim() || mutation.isPending}
        onClick={() => mutation.mutate()}
      >
        Save goal event
      </Button>
      {mutation.error ? <p role="alert">{mutation.error.message}</p> : null}
    </section>
  );
}
