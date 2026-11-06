import { describe, expect, it } from "vitest";
import { lifetime, packsHeading } from "./TopUpsCard";

const credits = {
  id: "credits_5000",
  name: "5,000 credits",
  priceInr: 1499,
  priceUsd: 19,
  credits: 5000,
  sites: 0,
};
const site = {
  id: "sites_1",
  name: "1 more website",
  priceInr: 999,
  priceUsd: 12,
  credits: 0,
  sites: 1,
};

describe("TopUpsCard copy", () => {
  it("names only what the plan can buy", () => {
    expect(packsHeading([site])).toBe("Add more websites");
    expect(packsHeading([credits])).toBe("Buy more credits");
    expect(packsHeading([credits, site])).toBe("Buy more credits or websites");
  });

  it("says one website, not one websites", () => {
    expect(lifetime(site)).toBe(
      "Adds 1 website for 30 days from payment. Buy one for each extra website.",
    );
    expect(lifetime({ ...site, sites: 5 })).toBe(
      "Adds 5 websites for 30 days from payment.",
    );
  });
});
