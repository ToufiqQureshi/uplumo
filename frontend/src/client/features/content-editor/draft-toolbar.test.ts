import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { DraftToolbar } from "./EditorPanels";

describe("DraftToolbar", () => {
  it("counts visible words, not link URLs or image markup", () => {
    const html = renderToStaticMarkup(
      createElement(DraftToolbar, {
        draft: {
          title: "Example title",
          targetKeyword: "",
          slug: "",
          metaDescription: "",
          content:
            "# Guide\n\nSee [this link](https://example.org/a-long-path) and ![cover image](https://cdn.example.org/image.jpg)",
        },
        saving: false,
        dirty: false,
        saved: true,
        canSave: true,
        onPreview: () => {},
        onSave: () => {},
        onPublish: () => {},
      }),
    );

    expect(html).toContain("5 words");
  });
});
