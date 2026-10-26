import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getDashboardActivation,
  getDashboardLayout,
  getDashboardOverview,
  saveDashboardLayout,
  refreshDashboardBacklinkSnapshot,
  markDashboardStepClicked,
  setDashboardStepDismissed,
} from "./dashboardApi";

afterEach(() => vi.restoreAllMocks());

describe("dashboard API", () => {
  it("encodes project IDs and validates activation state", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json({
        domain: null,
        ga4: {
          connected: false,
          propertyDisplayName: null,
          cardDismissedAt: null,
        },
        gsc: { connected: false, siteUrl: null },
        mcp: {
          authorizedAt: null,
          firstToolCallAt: null,
          cardDismissedAt: null,
        },
        competitorClickedAt: null,
        keywordsClickedAt: null,
        hasAudit: false,
        hasMultipleProjects: false,
        hasTeammate: false,
        dismissedSteps: [],
      }),
    );
    await expect(
      getDashboardActivation({ data: { projectId: "p/1" } }),
    ).resolves.toMatchObject({ hasAudit: false });
    expect(fetchMock.mock.calls[0]).toMatchObject([
      "/api/v1/projects/p%2F1/activation/get",
      { method: "POST", body: "{}" },
    ]);
    await expect(
      getDashboardActivation({ data: { projectId: "" } }),
    ).rejects.toThrow();
  });

  it("marks an explicit snapshot refresh so old automatic clients are rejected", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(Response.json({ ok: true }));
    await refreshDashboardBacklinkSnapshot({ data: { projectId: "p1" } });
    expect(fetchMock.mock.calls[0]).toMatchObject([
      "/api/v1/projects/p1/dashboard/backlinks/refresh",
      { method: "POST", body: JSON.stringify({ userInitiated: true }) },
    ]);
  });

  it("reads a missing layout as null and saves the card list as sent", async () => {
    const cards = [
      { id: "ai_visibility", hidden: false },
      { id: "gsc", hidden: true },
    ];
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(Response.json({ cards: null }))
      .mockResolvedValueOnce(Response.json({ cards }));
    await expect(
      getDashboardLayout({ data: { projectId: "p/1" } }),
    ).resolves.toBeNull();
    await expect(
      saveDashboardLayout({ data: { projectId: "p1", cards } }),
    ).resolves.toEqual({ cards });
    expect(fetchMock.mock.calls).toMatchObject([
      ["/api/v1/projects/p%2F1/dashboard/layout/get", { body: "{}" }],
      [
        "/api/v1/projects/p1/dashboard/layout/save",
        { method: "POST", body: JSON.stringify({ cards }) },
      ],
    ]);
  });

  it("accepts empty overview cards and sends only supported activation changes", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(Response.json({ audit: null, backlinks: null }))
      .mockResolvedValueOnce(Response.json({ ok: true }))
      .mockResolvedValueOnce(Response.json({ ok: true }));
    await expect(
      getDashboardOverview({ data: { projectId: "p1" } }),
    ).resolves.toEqual({
      audit: null,
      backlinks: null,
      backlinkRefreshAvailable: false,
    });
    await markDashboardStepClicked({
      data: { projectId: "p1", step: "competitor" },
    });
    await setDashboardStepDismissed({
      data: { projectId: "p1", step: "audit", dismissed: true },
    });
    expect(fetchMock.mock.calls.slice(1)).toMatchObject([
      [
        "/api/v1/projects/p1/activation/click",
        { body: JSON.stringify({ step: "competitor" }) },
      ],
      [
        "/api/v1/projects/p1/activation/dismiss",
        { body: JSON.stringify({ step: "audit", dismissed: true }) },
      ],
    ]);
  });
});
