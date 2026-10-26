import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { BrandLogo, EngineLogo } from "./BrandLogo";

describe("BrandLogo", () => {
  it("shows the real engine logo for API engine ids", () => {
    expect(renderToStaticMarkup(<EngineLogo engine="chat_gpt" />)).toContain(
      "<img",
    );
    expect(
      renderToStaticMarkup(<EngineLogo engine="google_ai_overview" />),
    ).toContain("<img");
    expect(renderToStaticMarkup(<EngineLogo engine="unknown" />)).toBe("");
  });

  it("labels a logo for screen readers only when it stands alone", () => {
    expect(renderToStaticMarkup(<BrandLogo brand="gemini" />)).toContain(
      'alt=""',
    );
    expect(
      renderToStaticMarkup(<BrandLogo brand="gemini" decorative={false} />),
    ).toContain('alt="Gemini"');
    expect(
      renderToStaticMarkup(
        <BrandLogo brand="search_console" decorative={false} />,
      ),
    ).toContain('aria-label="Google Search Console"');
  });
});
