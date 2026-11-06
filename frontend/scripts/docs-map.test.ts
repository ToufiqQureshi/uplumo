import { readdirSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { describe, expect, it } from "vitest";

// CLAUDE.md section 7: the repository keeps three general docs, feature docs
// with four fixed names next to the code, and content files that code reads.
// The owner also permits one exact DataForSEO reference snapshot path.
// Any other Markdown file is a stray doc.
const repoRoot = resolve(import.meta.dirname, "../..");
const skipDirs = new Set(["node_modules", ".git", "dist", "testdata"]);

const generalDocs = new Set([
  "CLAUDE.md",
  "imp_sessions_talked.md",
  "docs/DEPLOY.md",
]);
const providerReference = "backend/internal/platform/dataforseo/reference.md";

const content = [
  /^\.agents\/skills\//,
  /^backend\/internal\/sam\/skills\/[^/]+\.md$/,
  /^backend\/internal\/sam\/product_fact_sheet\.md$/,
  /^backend\/internal\/site\/blog\/[^/]+\.md$/,
  /^backend\/internal\/analytics\/geo\/ATTRIBUTION\.md$/,
  /^frontend\/src\/client\/features\/ai-mcp\/agentSetupPrompt\.md$/,
];
const featureDoc =
  /^(backend\/internal\/.+|frontend\/src\/client\/features\/[^/]+)\/(README|flow|how-it-works|business-logic)\.md$/;

function markdownFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    if (entry.isDirectory()) {
      return skipDirs.has(entry.name)
        ? []
        : markdownFiles(join(dir, entry.name));
    }
    return entry.name.endsWith(".md")
      ? [relative(repoRoot, join(dir, entry.name)).replaceAll("\\", "/")]
      : [];
  });
}

describe("documentation map", () => {
  it("allows only the docs CLAUDE.md permits", () => {
    const stray = markdownFiles(repoRoot).filter(
      (path) =>
        !generalDocs.has(path) &&
        path !== providerReference &&
        !featureDoc.test(path) &&
        !content.some((pattern) => pattern.test(path)),
    );
    expect(stray, "see CLAUDE.md section 7").toEqual([]);
  });
});
