import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { CiteFix } from "./citeFixApi";
import { FixRow } from "./FixRow";

const fix: CiteFix = {
  id: "not-thin",
  category: "content",
  title: "Pages with under 300 words",
  label: "Under 300 words",
  why: "Short pages rarely answer a question fully.",
  how: "Expand the page.",
  impact: "high",
  gain: 5.7,
  pagesAffected: 2,
  pages: [
    {
      url: "https://shop.example/offer/a",
      title: "Training",
      alsoAt: [
        "https://shop.example/offer/a?from=home",
        "https://shop.example/offer/a?from=work",
      ],
    },
    { url: "https://shop.example/contact", title: "Contact" },
  ],
};

describe("FixRow", () => {
  it("lists a page once and says where else it was found", () => {
    const html = renderToStaticMarkup(<FixRow fix={fix} position={1} />);
    expect(html.match(/Training/g)).toHaveLength(1);
    expect(html).toContain("Same page also at 2 other addresses");
    expect(html).not.toContain("?from=home</a>");
  });

  it("adds no note to a page found at one address", () => {
    const html = renderToStaticMarkup(
      <FixRow fix={{ ...fix, pages: [fix.pages[1]] }} position={1} />,
    );
    expect(html).toContain("Contact");
    expect(html).not.toContain("also at");
  });
});
