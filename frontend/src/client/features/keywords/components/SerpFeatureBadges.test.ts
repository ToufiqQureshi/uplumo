import { describe, expect, it } from "vitest";
import { serpFeatureLabels } from "./SerpFeatureBadges";

describe("serpFeatureLabels", () => {
  it("turns Google result types into plain labels, most important first", () => {
    const labels = serpFeatureLabels([
      "organic",
      "people_also_ask",
      "ai_overview",
      "images",
      "hotels_pack",
    ]).map((feature) => feature.label);
    expect(labels).toEqual(["Maps", "AI", "Q&A"]);
  });

  it("shows one badge for several types with the same meaning", () => {
    const labels = serpFeatureLabels([
      "local_pack",
      "hotels_pack",
      "google_hotels",
      "video",
      "short_videos",
    ]).map((feature) => feature.label);
    expect(labels).toEqual(["Maps", "Video"]);
  });

  it("has nothing to show for plain results or results cached before features", () => {
    expect(serpFeatureLabels(["organic", "images"])).toEqual([]);
    expect(serpFeatureLabels([])).toEqual([]);
    expect(serpFeatureLabels(undefined)).toEqual([]);
  });
});
