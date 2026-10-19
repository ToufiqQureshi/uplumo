import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { MarkdownAnswer } from "./MarkdownAnswer";

function render(text: string) {
  return renderToStaticMarkup(createElement(MarkdownAnswer, { text }));
}

describe("MarkdownAnswer", () => {
  it("shows formatting instead of raw markdown", () => {
    const html = render("## Top picks\n\nCompare **MMR Hotels** first.");
    expect(html).toContain("<h2");
    expect(html).toMatch(/<strong[^>]*>MMR Hotels<\/strong>/);
    expect(html).not.toContain("**");
    expect(html).not.toContain("##");
  });

  it("shows an AI vendor's image as a link with its title, without loading it", () => {
    const html = render(
      "![Revenue guide | Medium](https://images.openai.com/static/x.png)",
    );
    expect(html).not.toContain("<img");
    expect(html).toContain("Image: Revenue guide | Medium");
    expect(html).toContain('href="https://images.openai.com/static/x.png"');
  });

  it("never links an image with an unsafe address", () => {
    const html = render("![x](javascript:alert(1))");
    expect(html).not.toContain("javascript:");
    expect(html).toContain("Image: x");
  });
});
