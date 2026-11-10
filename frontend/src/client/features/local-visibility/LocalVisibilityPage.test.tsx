// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { LocalVisibilityPage } from "./LocalVisibilityPage";

vi.mock("@/client/features/billing/HostedPlanGate", () => ({
  useHostedPlanGate: () => "paid",
}));

const prices = {
  find: 3,
  check: 40,
  grid3: 14,
  grid5: 39,
  reviews: 58,
  watchPerCompetitor: 10,
  reply: 1,
};
const business = {
  cid: "100",
  placeId: "p",
  title: "Smile Studio",
  address: "Koregaon Park",
  category: "Dentist",
  latitude: 18.5,
  longitude: 73.9,
  searchTerm: "dentist",
  languageCode: "en",
  updatedAt: "2026-11-10T10:00:00Z",
};
const checkResult = {
  searchTerm: "dentist",
  yourRank: 4,
  you: {
    place: {
      rank: 4,
      cid: "100",
      placeId: "",
      title: "Smile Studio",
      address: "",
      category: "",
      categories: null,
      rating: null,
      reviews: 0,
      latitude: 0,
      longitude: 0,
      domain: "",
    },
    profile: null,
    postsLast30: null,
    lastPostAt: null,
  },
  rivals: [],
  fixes: [
    {
      status: "missing",
      title: "Add the category “Cosmetic dentist”",
      detail: "3 of 3 competitors use it.",
      you: "not listed",
      others: "3 of 3",
    },
  ],
  summary: "You rank #4 on Google Maps for “dentist” at your address.",
  missing: [],
};

function run(kind: string, status: string, result: unknown = null) {
  return {
    id: `00000000-0000-0000-0000-00000000000${kind.length}`,
    kind,
    status,
    result,
    error: "",
    createdAt: "2026-11-10T10:00:00Z",
    finishedAt: null,
    cached: false,
  };
}

let overview: Record<string, unknown>;
let responses: Record<string, () => Response>;
const calls: { path: string; body: unknown }[] = [];

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let host: HTMLDivElement;
let root: Root;

beforeEach(() => {
  calls.length = 0;
  overview = {
    business,
    runs: {},
    watch: {
      enabled: false,
      nextRunAt: null,
      competitors: [],
      creditsPerWeek: 0,
    },
    prices,
    aiReady: true,
  };
  // The server's newest check run, which the overview also returns.
  let serverCheck: unknown;
  responses = {
    overview: () =>
      json({ ...overview, runs: serverCheck ? { check: serverCheck } : {} }),
    "runs/start": () => json((serverCheck = run("check", "running"))),
    "runs/get": () => json((serverCheck = run("check", "done", checkResult))),
  };
  vi.stubGlobal(
    "fetch",
    vi.fn(async (path: string, init: RequestInit) => {
      const name = path.split("/local/")[1];
      calls.push({
        path: name,
        body: typeof init.body === "string" ? JSON.parse(init.body) : undefined,
      });
      return responses[name]();
    }),
  );
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});

afterEach(() => {
  act(() => root.unmount());
  host.remove();
  vi.unstubAllGlobals();
});

async function settle(action: () => void | Promise<void> = () => {}) {
  await act(async () => {
    await action();
    await new Promise((resolve) => setTimeout(resolve, 50));
  });
}

async function render() {
  const client = new QueryClient();
  await settle(() =>
    root.render(
      <QueryClientProvider client={client}>
        <LocalVisibilityPage projectId="project-1" />
      </QueryClientProvider>,
    ),
  );
  await settle();
}

function button(text: string) {
  const found = [...host.querySelectorAll("button")].find((b) =>
    b.textContent?.includes(text),
  );
  if (!found) throw new Error(`no button "${text}" in ${host.textContent}`);
  return found;
}

function paths() {
  return calls.map((c) => c.path);
}

it("opening the page reads only the free overview", async () => {
  await render();
  expect(paths()).toEqual(["overview"]);
  expect(button("Run profile check").textContent).toContain("up to 40 credits");
});

it("a double click starts one check, then polls the free status until it is done", async () => {
  await render();
  await settle(() => {
    button("Run profile check").click();
    button("Run profile check").click();
  });
  await vi.waitFor(() =>
    expect(host.textContent).toContain("You rank #4 on Google Maps"),
  );
  expect(paths().filter((p) => p === "runs/start")).toHaveLength(1);
  expect(paths()).toContain("runs/get");
  expect(host.textContent).toContain("You rank #4 on Google Maps");
  expect(host.textContent).toContain("Add the category “Cosmetic dentist”");
});

it("a refused start shows the server's message and is not retried", async () => {
  responses["runs/start"] = () =>
    json(
      {
        error: {
          code: "insufficient_credits",
          message: "Not enough credits for this.",
        },
      },
      402,
    );
  await render();
  await settle(() => button("Run profile check").click());
  await settle();
  expect(paths().filter((p) => p === "runs/start")).toHaveLength(1);
  expect(host.querySelector('[role="alert"]')?.textContent).toBeTruthy();
});

it("without a business, search is paid on click and saving the pick is free", async () => {
  overview.business = null;
  responses.find = () =>
    json({
      places: [
        {
          rank: 1,
          cid: "100",
          placeId: "p",
          title: "Smile Studio",
          address: "Koregaon Park",
          category: "Dentist",
          categories: ["Dentist"],
          rating: 4.6,
          reviews: 212,
          latitude: 18.5,
          longitude: 73.9,
          domain: "",
        },
      ],
      cached: false,
    });
  responses.business = () => json(business);
  await render();
  expect(paths()).toEqual(["overview"]);
  const input = host.querySelector<HTMLInputElement>(
    'input[aria-label="Business name"]',
  )!;
  await settle(() => {
    Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value",
    )!.set!.call(input, "Smile Studio Pune");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await settle(() => button("Search").click());
  expect(calls.find((c) => c.path === "find")?.body).toEqual({
    query: "Smile Studio Pune",
  });
  await settle(async () =>
    (await vi.waitFor(() => button("Smile Studio"))).click(),
  );
  await settle(async () => (await vi.waitFor(() => button("Save"))).click());
  await vi.waitFor(() => expect(paths()).toContain("business"));
  expect(calls.find((c) => c.path === "business")?.body).toEqual({
    cid: "100",
    searchTerm: "Dentist",
  });
  expect(paths().filter((p) => p === "find")).toHaveLength(1);
});
