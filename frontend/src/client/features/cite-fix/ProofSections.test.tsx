import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { FixProofCard, PitchButton, WhatAISaysCard } from "./ProofSections";

describe("WhatAISaysCard", () => {
  it("quotes each answer with its engine's name", () => {
    const html = renderToStaticMarkup(
      <WhatAISaysCard
        said={[
          {
            prompt: "best firms",
            engine: "chat_gpt",
            excerpt: "Shop Co is fast.",
          },
        ]}
      />,
    );
    expect(html).toContain("ChatGPT · “best firms”");
    expect(html).toContain("Shop Co is fast.");
  });

  it("is hidden when no answer names the customer", () => {
    expect(renderToStaticMarkup(<WhatAISaysCard said={[]} />)).toBe("");
  });
});

const proof = {
  url: "https://shop.example/services",
  fixedAt: "2026-10-20T12:00:00Z",
  days: 10,
  aiVisitorsBefore: 1,
  aiVisitorsAfter: 4,
  leadsBefore: 0,
  leadsAfter: 2,
  leadsTracked: true,
  citedNow: true,
};

describe("FixProofCard", () => {
  it("shows before and after for each fixed page", () => {
    const html = renderToStaticMarkup(<FixProofCard proofs={[proof]} />);
    expect(html).toContain("/services");
    expect(html).toContain("1 → 4");
    expect(html).toContain("0 → 2");
    expect(html).toContain("Yes");
    expect(html).toContain("Likely linked, not proven");
  });

  it("does not show leads until lead actions are chosen", () => {
    const html = renderToStaticMarkup(
      <FixProofCard
        proofs={[{ ...proof, leadsTracked: false, citedNow: false }]}
      />,
    );
    expect(html).not.toContain("0 → 2");
    expect(html).toContain("Not yet");
    expect(html).toContain("choose your lead actions in Analytics");
  });

  it("is hidden with no fixed pages", () => {
    expect(renderToStaticMarkup(<FixProofCard proofs={[]} />)).toBe("");
  });
});

describe("PitchButton", () => {
  it("shows the server's credit cap before any click", () => {
    const html = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <PitchButton
          projectId="p1"
          url="https://dir.example/"
          maxCredits={25}
        />
      </QueryClientProvider>,
    );
    expect(html).toContain("Draft a pitch");
    expect(html).toContain("≤25 credits");
  });
});
