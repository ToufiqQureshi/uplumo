import { firstBy, sortBy } from "remeda";
import type { CiteFixProof, CiteFixReport } from "./citeFixApi";

type Prompt = NonNullable<CiteFixProof["visibility"]>["prompts"][number];

/** cited: named in the latest check; lost: named before, not now. */
export type CellState = "cited" | "lost" | "never";

export interface Cell {
  prompt: string;
  engine: string;
  state: CellState;
  latestRank: number | null;
  citedAnswers: number;
  answers: number;
  /** Sites the latest answer cited instead, when it missed the customer. */
  citedInstead: string[];
}

export interface Matrix {
  prompts: string[];
  engines: string[];
  /** cells[prompt][engine]; missing when that engine did not answer it. */
  cells: Map<string, Map<string, Cell>>;
}

function cellState(row: Prompt): CellState {
  if (row.latestRank !== null) return "cited";
  return row.citedAnswers > 0 ? "lost" : "never";
}

/**
 * Every tracked question against every engine, from the saved history of
 * comparable checks. Questions and engines keep the order they first appear.
 */
export function buildMatrix(proof: CiteFixProof | null): Matrix | null {
  const rows = proof?.visibility?.prompts ?? [];
  if (rows.length === 0) return null;
  const instead = new Map(
    proof?.misses.map((miss) => [`${miss.prompt} ${miss.engine}`, miss.cited]),
  );
  const prompts: string[] = [];
  const engines: string[] = [];
  const cells = new Map<string, Map<string, Cell>>();
  for (const row of rows) {
    if (!prompts.includes(row.prompt)) prompts.push(row.prompt);
    if (!engines.includes(row.engine)) engines.push(row.engine);
    const byEngine = cells.get(row.prompt) ?? new Map<string, Cell>();
    byEngine.set(row.engine, {
      prompt: row.prompt,
      engine: row.engine,
      state: cellState(row),
      latestRank: row.latestRank,
      citedAnswers: row.citedAnswers,
      answers: row.answers,
      citedInstead: instead.get(`${row.prompt} ${row.engine}`) ?? [],
    });
    cells.set(row.prompt, byEngine);
  }
  return { prompts, engines, cells };
}

export function allCells(matrix: Matrix): Cell[] {
  return [...matrix.cells.values()].flatMap((row) => [...row.values()]);
}

interface Headline {
  cited: Cell[];
  /** Named in earlier checks but not the latest, most often named first. */
  lost: Cell[];
  /** Engines that never named the customer for any tracked question. */
  neverEngines: string[];
  /** The question and engine that names the customer most often. */
  strongest: Cell | null;
}

export function headline(matrix: Matrix): Headline {
  const cells = allCells(matrix);
  const cited = cells.filter((cell) => cell.state === "cited");
  const lost = sortBy(
    cells.filter((cell) => cell.state === "lost"),
    [(cell) => cell.citedAnswers, "desc"],
  );
  const neverEngines = matrix.engines.filter((engine) =>
    cells
      .filter((cell) => cell.engine === engine)
      .every((cell) => cell.state === "never"),
  );
  const strongest =
    firstBy(cited, [(cell) => cell.citedAnswers, "desc"]) ?? null;
  return { cited, lost, neverEngines, strongest };
}

export interface RivalRank {
  domain: string;
  averageRank: number | null;
  /** How many answers of the latest check cited it. */
  latestCount: number;
  you: boolean;
}

/**
 * The sites AI names most in the latest check, with their average position
 * across checks, and the customer's own row when it was ever cited.
 */
export function rivalRanks(proof: CiteFixProof, limit = 6): RivalRank[] {
  const own = proof.domain;
  const isOwn = (domain: string) =>
    domain === own || domain.endsWith(`.${own}`);
  const domains = proof.visibility?.domains ?? [];
  if (domains.length === 0) {
    return proof.rivals.slice(0, limit).map((rival) => ({
      domain: rival.domain,
      averageRank: null,
      latestCount: rival.count,
      you: false,
    }));
  }
  const rows = domains.map((row) => ({
    domain: row.name.replace(/^www\./, ""),
    averageRank: row.averageRank,
    latestCount: row.latestCount,
    you: isOwn(row.name.replace(/^www\./, "")),
  }));
  const top = sortBy(
    rows.filter((row) => !row.you && row.latestCount > 0),
    [(row) => row.latestCount, "desc"],
    [(row) => row.averageRank ?? Infinity, "asc"],
  ).slice(0, limit);
  const you = rows.find((row) => row.you);
  return you ? [...top, you] : top;
}

type Comparison = CiteFixReport["rivals"][number];

export interface ListPage {
  url: string;
  title: string;
  domain: string;
  listsRivals: string[];
  engines: string[];
  prompts: string[];
}

/**
 * Pages AI cited that link to at least two other cited sites and not to the
 * customer: lists and directories the customer could ask to join.
 */
export function listPages(comparisons: Comparison[]): ListPage[] {
  const pages = new Map<string, ListPage>();
  for (const comparison of comparisons) {
    if (comparison.linksYou || comparison.listsRivals.length < 2) continue;
    const page = pages.get(comparison.rival.url) ?? {
      url: comparison.rival.url,
      title: comparison.rival.title,
      domain: comparison.rival.domain,
      listsRivals: comparison.listsRivals,
      engines: [],
      prompts: [],
    };
    if (!page.engines.includes(comparison.engine))
      page.engines.push(comparison.engine);
    if (!page.prompts.includes(comparison.prompt))
      page.prompts.push(comparison.prompt);
    pages.set(page.url, page);
  }
  return sortBy(
    [...pages.values()],
    [(page) => page.listsRivals.length, "desc"],
  );
}

/**
 * The rival pages to compare signal by signal: company pages, not lists,
 * one per page, at most `limit`.
 */
export function signalRivals(comparisons: Comparison[], limit = 2) {
  const seen = new Set<string>();
  const picks: Comparison[] = [];
  for (const comparison of comparisons) {
    if (comparison.rival.appShell || comparison.listsRivals.length >= 2)
      continue;
    if (seen.has(comparison.rival.domain)) continue;
    seen.add(comparison.rival.domain);
    picks.push(comparison);
    if (picks.length === limit) break;
  }
  return picks;
}

export type StepKind = "scan" | "track" | "win_back" | "get_listed" | "fix";

export interface Step {
  kind: StepKind;
  title: string;
  why: string;
  proof: string;
}

/**
 * What to do, in order, from the report's own facts. A step appears only
 * when the data shows its problem; nothing here is generic advice.
 */
export function nextSteps(
  report: CiteFixReport,
  matrix: Matrix | null,
  lists: ListPage[],
  engineName: (engine: string) => string,
): Step[] {
  const steps: Step[] = [];
  if (!report.proof || report.proof.answers === 0) {
    steps.push({
      kind: "track",
      title: "Check which sites AI names for your questions",
      why: "Cite Fix needs one saved AI Visibility check to show who AI cites instead of you.",
      proof: "No AI Visibility check is saved for this website yet.",
    });
  }
  if (report.state !== "ready") {
    steps.push({
      kind: "scan",
      title: "Scan your site so we can compare your pages",
      why: "Without a current scan we can show what rival pages have, but not what yours are missing.",
      proof:
        report.state === "no_scan"
          ? "This website has no site scan yet."
          : "Your last scan is older than the citation checks.",
    });
  }
  const facts = matrix ? headline(matrix) : null;
  const lost = facts?.lost[0];
  if (lost) {
    const instead = lost.citedInstead.slice(0, 2).join(" and ");
    steps.push({
      kind: "win_back",
      title: `Win back ${engineName(lost.engine)} on “${lost.prompt}”`,
      why: `It named you in ${lost.citedAnswers} of ${lost.answers} checks, then stopped. Compare your page with the ones it cites now and close the gaps.`,
      proof: instead
        ? `In the latest check it cited ${instead}.`
        : "It cited other sites in the latest check.",
    });
  }
  if (lists.length > 0) {
    const first = lists[0];
    steps.push({
      kind: "get_listed",
      title: `Get listed on ${lists.length === 1 ? "the page" : `the ${lists.length} pages`} that name your rivals`,
      why: "AI answers quote these lists. A listing there can reach every AI that reads them.",
      proof: `${first.domain} links to ${first.listsRivals.slice(0, 3).join(", ")} and not to you.`,
    });
  }
  if (report.state === "ready" && report.fixes.length > 0) {
    const fix = report.fixes[0];
    steps.push({
      kind: "fix",
      title: fix.title,
      why: fix.why,
      proof: fix.pagesAffected
        ? `${fix.pagesAffected} page${fix.pagesAffected === 1 ? "" : "s"} affected.`
        : "Affects the whole site.",
    });
  }
  return steps;
}
