import { describe, expect, it } from "vitest";
import { draftAfterSave } from "./draft-save";

const base = {
  title: "Teal saree guide",
  targetKeyword: "teal saree",
  slug: "teal-saree-guide",
  metaDescription: "",
  content: "Teal is a deep blue-green shade.",
};

describe("draftAfterSave", () => {
  it("keeps text typed while the autosave was in flight", () => {
    const sent = base;
    const current = { ...base, content: base.content + " A teal silk saree" };
    expect(draftAfterSave(current, sent, { ...sent })).toBe(current);
  });

  it("takes the server copy when nothing changed during the save", () => {
    const saved = { ...base, slug: "teal-saree" };
    expect(draftAfterSave({ ...base }, base, saved)).toBe(saved);
  });
});
