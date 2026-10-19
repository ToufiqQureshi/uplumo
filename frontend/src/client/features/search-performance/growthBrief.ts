import type { GrowthPage, GrowthReport } from "./growthApi";

export function growthAnalystPrompt(report: GrowthReport, page?: GrowthPage) {
  const scope = page
    ? `Focus on my existing page: ${page.page}.`
    : "Prioritize the three most useful actions for my website.";
  const brands = report.brandTerms.length
    ? `My brand names are: ${report.brandTerms.join(", ")}.`
    : "I have not configured brand names yet.";
  const task = page
    ? "Read the page and prepare a focused improvement brief: what to keep, what to update, related search questions to cover, title and description options, and useful internal links you can verify. Include direct-answer improvements where they help readers. Help me decide whether to improve this page or create a separate page."
    : "Explain which pages need attention, which performing pages could benefit from more search traffic, and where I should check tracking or the next step for visitors.";
  return `Search growth analysis for ${report.range.startDate} to ${report.range.endDate}, using my connected Search Console and Google Analytics data. ${scope} ${brands}\n\n${task}\n\nKeep it simple: what changed, the evidence, possible explanations, what I should do next, and how to measure the result. Separate facts from guesses. Use existing data and ask before any extra paid research. Give suggestions for me to review; do not publish or change my site.`;
}
