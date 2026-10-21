import { describe, expect, it } from "vitest";
import { sameDraft } from "./draft-save";

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
