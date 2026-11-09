import { describe, expect, it } from "vitest";
import {
  articleBodyHTML,
  articleHTMLFile,
  articleMarkdownFile,
  exportFileName,
} from "./article-export";

const BACKSLASH = String.fromCharCode(92);

describe("article HTML export", () => {
  it("builds a full page with escaped title, description, schema and theme", async () => {
    const file = await articleHTMLFile({
      title: 'Shoes "for" <flat> feet',
      description: "Best & safest picks",
      content:
        '# Shoes "for" <flat> feet\n\nIntro with **bold**.\n\n| a | b |\n|---|---|\n| 1 | 2 |',
      css: "h1{color:#7a2e0e}h2::after{content:'</style>'}",
      schemas: [
        { "@type": "FAQPage", text: "</script><script>alert(1)</script>" },
      ],
    });
    expect(file.startsWith("<!doctype html>")).toBe(true);
    expect(file).toContain(
      "<title>Shoes &quot;for&quot; &lt;flat&gt; feet</title>",
    );
    expect(file).toContain(
      '<meta name="description" content="Best &amp; safest picks">',
    );
    expect(file).not.toContain("</script><script>");
    expect(file).toContain(`${BACKSLASH}u003c/script>`);
    expect(file).toContain(`content:'${BACKSLASH}3c /style>'`);
    expect(file.match(/<\/style>/g)).toHaveLength(1);
    expect(file.match(/<h1>/g)).toHaveLength(1);
    expect(file).toContain("<strong>bold</strong>");
    expect(file).toContain("<table>");
    expect(file).not.toContain("class=");
  });

  it("uses a readable default style and skips empty metadata", async () => {
    const file = await articleHTMLFile({
      title: "",
      description: "  ",
      content: "नमस्ते दुनिया",
    });
    expect(file).toContain("<title>Untitled article</title>");
    expect(file).not.toContain('name="description"');
    expect(file).not.toContain("application/ld+json");
    expect(file).toContain("max-width:42rem");
    expect(file).toContain("<p>नमस्ते दुनिया</p>");
  });

  it("copies the body without the title heading or raw HTML", async () => {
    const body = await articleBodyHTML(
      "# My title\n\nText <script>alert(1)</script>",
      "My title",
    );
    expect(body).not.toContain("<h1>");
    expect(body).not.toContain("<script>");
    expect(body).toContain("<p>Text");
  });
});

describe("article Markdown export", () => {
  it("adds front matter that keeps quotes and colons valid", () => {
    expect(
      articleMarkdownFile({
        title: 'Guide: "best" shoes',
        description: "",
        slug: "best-shoes",
        content: "\nBody\n",
      }),
    ).toBe(
      `---\ntitle: "Guide: ${BACKSLASH}"best${BACKSLASH}" shoes"\nslug: "best-shoes"\n---\n\nBody\n`,
    );
  });
});

describe("export file name", () => {
  it.each([
    ["best-shoes", "Ignored", "best-shoes.html"],
    ["", "Best Shoes: 2026 Guide!", "best-shoes-2026-guide.html"],
    ["", "जूते कैसे चुनें", "जूते-कैसे-चुनें.html"],
    ["", "   ", "article.html"],
    ["", "!!!", "article.html"],
    ["", `${"a".repeat(79)} b`, `${"a".repeat(79)}.html`],
  ])("slug %j title %j -> %s", (slug, title, want) => {
    expect(exportFileName(slug, title, "html")).toBe(want);
  });
});
