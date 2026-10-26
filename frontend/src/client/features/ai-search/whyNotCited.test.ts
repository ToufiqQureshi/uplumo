import { describe, expect, it } from "vitest";
import type { TrackingAnswer } from "./trackingApi";
import { whyNotCited } from "./whyNotCited";

let next = 0;
function answer(over: Partial<TrackingAnswer> & { cites?: string[] }) {
  const { cites = [], ...rest } = over;
  return {
    id: `a${next++}`,
    prompt: "hotel near kemps corner",
    topic: "",
    engine: "chat_gpt",
    status: "completed",
    answer: "Some answer",
    errorCode: "",
    collectedAt: null,
    citations: cites.map((url) => ({
      url,
      domain: new URL(url).hostname,
      title: null,
      matchedBrand: false,
    })),
    brands: [
      {
        name: "Shalimar",
        domain: "theshalimarhotel.com",
        mentioned: false,
        cited: false,
      },
    ],
    recommended: [],
    ...rest,
  } satisfies TrackingAnswer;
}

const own = "theshalimarhotel.com";

describe("whyNotCited", () => {
  it("tells a local business which listing sites and pages AI used instead", () => {
    const result = whyNotCited(
      [
        answer({
          cites: [
            "https://www.booking.com/hotel/a",
            "https://www.tripadvisor.in/b",
            "https://www.justdial.com/c",
            "https://hotelkempscorner.com/rooms",
          ],
          recommended: [{ name: "Hotel Kemps Corner", position: 1 }],
        }),
        answer({
          cites: ["https://in.hotels.com/x", "https://www.reddit.com/r/mumbai"],
          recommended: [{ name: "Hotel Kemps Corner", position: 1 }],
        }),
      ],
      own,
    );
    expect(result).toMatchObject({ answered: 2, cited: 0 });
    expect(result.gaps.map((g) => g.kind)).toEqual([
      "listings",
      "not_named",
      "competitor_pages",
      "community",
    ]);
    expect(result.gaps[0]).toMatchObject({
      answers: 2,
      action: { kind: "external" },
    });
    expect(result.gaps[0].evidence).toEqual(
      expect.arrayContaining(["booking.com", "in.hotels.com", "justdial.com"]),
    );
    expect(result.gaps[2]).toMatchObject({
      evidence: ["https://hotelkempscorner.com/rooms"],
      action: { kind: "write", keyword: "hotel near kemps corner" },
    });
    expect(result.gaps[1].evidence).toEqual(["Hotel Kemps Corner"]);
  });

  it("leaves out answers that already cite the site, including its subdomains", () => {
    const result = whyNotCited(
      [
        answer({ cites: ["https://www.theshalimarhotel.com/dining"] }),
        answer({ cites: ["https://book.theshalimarhotel.com/"] }),
        answer({
          brands: [
            { name: "Shalimar", domain: own, mentioned: true, cited: true },
          ],
          cites: ["https://www.booking.com/x"],
        }),
      ],
      own,
    );
    expect(result).toEqual({ answered: 3, cited: 3, gaps: [] });
  });

  it("separates being named without a link from not being named", () => {
    const result = whyNotCited(
      [
        answer({
          prompt: "best seafood in mumbai",
          brands: [
            { name: "Shalimar", domain: own, mentioned: true, cited: false },
          ],
        }),
      ],
      own,
    );
    expect(result.gaps).toEqual([
      expect.objectContaining({
        kind: "named_not_linked",
        answers: 1,
        action: {
          kind: "write",
          label: "Write the answer page",
          keyword: "best seafood in mumbai",
        },
      }),
    ]);
  });

  it("ignores failed, pending and empty answers", () => {
    const result = whyNotCited(
      [
        answer({ status: "failed", cites: ["https://www.booking.com/a"] }),
        answer({ status: "queued", cites: ["https://www.booking.com/a"] }),
        answer({ answer: "  ", cites: ["https://www.booking.com/a"] }),
      ],
      own,
    );
    expect(result).toEqual({ answered: 0, cited: 0, gaps: [] });
  });

  it("keeps the editor keyword within its 200 character limit", () => {
    const long = "q".repeat(300);
    const result = whyNotCited(
      [answer({ prompt: long, cites: ["https://rival.example/page"] })],
      own,
    );
    const action = result.gaps[0].action;
    expect(action.kind === "write" && action.keyword.length).toBe(200);
  });
});
