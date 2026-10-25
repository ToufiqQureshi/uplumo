import { describe, expect, it } from "vitest";
import {
  searchPreviewTitle,
  searchPreviewUrl,
  titleForDraft,
  sameDraft,
  isArticleContentWithinLimit,
  parseTargetTopics,
  primaryTargetTopic,
} from "./draft";

const draft = {
  title: "",
  targetKeyword: "",
  slug: "draft",
  metaDescription: "",
  content: "# Article H1\n\nBody",
};

describe("editor title and search preview", () => {
  it("uses the H1 while the page title is empty and only says Untitled when both are empty", () => {
    expect(titleForDraft(draft)).toBe("Article H1");
    expect(searchPreviewTitle(draft)).toBe("Article H1");
    expect(titleForDraft({ ...draft, content: "Body" })).toBe("Untitled draft");
  });

  it("uses the connected workspace domain and slug", () => {
    expect(searchPreviewUrl("revmerito.com", "guide")).toBe(
      "https://revmerito.com/guide",
    );
    expect(searchPreviewUrl("", "guide")).toContain("Connect a website");
  });
});

describe("isArticleContentWithinLimit", () => {
  it("allows content at the API's byte limit and rejects the next byte", () => {
    expect(isArticleContentWithinLimit("a".repeat(150_000))).toBe(true);
    expect(isArticleContentWithinLimit("a".repeat(150_001))).toBe(false);
  });

  it("counts UTF-8 bytes for Hindi content", () => {
    expect(isArticleContentWithinLimit("ह".repeat(50_001))).toBe(false);
  });
});

const base = {
  title: "Teal saree guide",
  targetKeyword: "teal saree",
  slug: "teal-saree-guide",
  metaDescription: "",
  content: "Teal is a deep blue-green shade.",
};

describe("sameDraft", () => {
  it("sees text typed while an autosave was in flight", () => {
    expect(
      sameDraft({ ...base, content: base.content + " A teal" }, base),
    ).toBe(false);
  });

  it("matches an unchanged draft", () => {
    expect(sameDraft({ ...base }, base)).toBe(true);
  });
});

describe("target topics", () => {
  it("splits comma and newline values and removes case-insensitive duplicates", () => {
    expect(parseTargetTopics("SEO, seo\ncontent strategy")).toEqual([
      "SEO",
      "content strategy",
    ]);
    expect(primaryTargetTopic("SEO, content strategy")).toBe("SEO");
  });

  it("limits the list to five topics", () => {
    expect(parseTargetTopics("a,A,b,c,d,e,f")).toEqual([
      "a",
      "b",
      "c",
      "d",
      "e",
    ]);
  });
});
