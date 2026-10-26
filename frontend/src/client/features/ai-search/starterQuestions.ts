import type { TrackingConfig } from "./trackingApi";

type Prompt = TrackingConfig["prompts"][number];

const ask = (topic: string, text: string): Prompt => ({
  topic,
  text,
  paused: false,
});

/**
 * Questions a real customer of this business would ask an AI assistant,
 * built from what it sells and where (free, no AI call). Broad industry
 * questions ("best luxury hotels in India") measure the wrong market and
 * report a useless 0%, so every question names the business type and city.
 */
export function starterQuestions({
  brand,
  business,
  city,
  competitors,
}: {
  brand: string;
  business: string;
  city: string;
  competitors: string[];
}): Prompt[] {
  const what = business.trim();
  const where = city.trim();
  const name = brand.trim();
  if (!what) return [];
  const inCity = where ? ` in ${where}` : "";
  const questions = [
    ask("Find a business", `Best ${what}${inCity}`),
    ask("Find a business", `Top rated ${what}${inCity} with good reviews`),
    ask("Find a business", `Affordable ${what}${inCity}`),
    ask(
      "Find a business",
      `Which ${what}${inCity} do customers recommend, and why?`,
    ),
  ];
  if (name) {
    questions.push(
      ask("Your brand", `Is ${name}${inCity} good? What do customers say?`),
      ask("Your brand", `What is ${name} known for?`),
    );
    for (const rival of competitors
      .map((c) => c.trim())
      .filter(Boolean)
      .slice(0, 2)) {
      questions.push(ask("Compare", `${name} vs ${rival}: which is better?`));
    }
  }
  return questions;
}

/** Fills blank rows first, skips questions already there, keeps the cap. */
export function addQuestions(
  current: Prompt[],
  suggested: Prompt[],
  max = 30,
): Prompt[] {
  const seen = new Set(
    current.map((p) => p.text.trim().toLowerCase()).filter(Boolean),
  );
  const fresh = suggested.filter((p) => {
    const key = p.text.toLowerCase();
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  const kept = current.filter((p) => p.text.trim());
  return [...kept, ...fresh].slice(0, max);
}
