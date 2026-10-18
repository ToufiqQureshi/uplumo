// @vitest-environment happy-dom
import { Markdown } from "@tiptap/markdown";
import { Editor } from "@tiptap/react";
import { afterEach, describe, expect, it } from "vitest";
import { articleExtensions } from "./visual-editor/extensions";
import {
  applyWriterTarget,
  captureWriterTarget,
  writerTargetMode,
} from "./writer-target";

describe("AI suggestion target", () => {
  const editors: Editor[] = [];
  const makeEditor = () => {
    const editor = new Editor({
      extensions: [...articleExtensions({ placeholder: "" }), Markdown],
      content: "First paragraph.\n\nSecond paragraph.",
      contentType: "markdown",
    });
    editors.push(editor);
    return editor;
  };
  afterEach(() => editors.splice(0).forEach((editor) => editor.destroy()));

  it("requires a selected passage for rewriting and replaces only that range", () => {
    const editor = makeEditor();
    expect(captureWriterTarget(editor, "tone_language", "draft-1")).toBeNull();
    editor.commands.setTextSelection({ from: 1, to: 16 });
    const target = captureWriterTarget(editor, "tone_language", "draft-1");
    expect(target?.selected).toBe("First paragraph");
    expect(
      target && applyWriterTarget(editor, target, "draft-1", "Better opening"),
    ).toBe(true);
    expect(editor.getText()).toContain("Better opening.");
    expect(editor.getText()).toContain("Second paragraph.");
    expect(editor.getText()).not.toContain("First paragraph.");
  });

  it("refuses stale or foreign draft results without touching the article", () => {
    const editor = makeEditor();
    editor.commands.setTextSelection({ from: 1, to: 16 });
    const target = captureWriterTarget(editor, "destuff", "draft-1")!;
    expect(applyWriterTarget(editor, target, "draft-2", "New words")).toBe(
      false,
    );
    editor.commands.insertContentAt(1, "Edited ");
    const changed = editor.getMarkdown();
    expect(applyWriterTarget(editor, target, "draft-1", "New words")).toBe(
      false,
    );
    expect(editor.getMarkdown()).toBe(changed);
  });

  it("keeps outlines, meta options, and fact notes out of the article", () => {
    const editor = makeEditor();
    for (const action of ["outline", "title_meta", "fact_check"] as const) {
      const target = captureWriterTarget(editor, action, "draft-1")!;
      expect(writerTargetMode(action)).toBe("review");
      expect(
        applyWriterTarget(editor, target, "draft-1", "Entire new article"),
      ).toBe(false);
    }
    expect(editor.getMarkdown()).toContain("Second paragraph.");
  });

  it("inserts a new section at the saved cursor, preserving existing paragraphs", () => {
    const editor = makeEditor();
    editor.commands.setTextSelection(17);
    const target = captureWriterTarget(editor, "write_section", "draft-1")!;
    expect(applyWriterTarget(editor, target, "draft-1", "New section")).toBe(
      true,
    );
    expect(editor.getText()).toContain("First paragraph.");
    expect(editor.getText()).toContain("Second paragraph.");
    expect(editor.getText()).toContain("New section");
  });
});
