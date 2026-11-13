import type { WriterAction } from "./writer-target";

// Plain words for writers who are not SEO people. The first four are the
// everyday actions; the rest sit under "More".
export const writerActions: Array<{
  id: WriterAction;
  label: string;
  hint: string;
}> = [
  {
    id: "analyze",
    label: "Analyze full draft",
    hint: "Intent, missing answers, evidence and next edits",
  },
  {
    id: "write_section",
    label: "Write this section",
    hint: "Writes the part your cursor is in",
  },
  {
    id: "title_meta",
    label: "Suggest title & Google text",
    hint: "Three options you can apply in one click",
  },
  {
    id: "tone_language",
    label: "Change tone or language",
    hint: "Select text first · your chosen language",
  },
  {
    id: "fact_check",
    label: "List facts to check",
    hint: "Select text first · your text is not changed",
  },
  { id: "outline", label: "Make an outline", hint: "Headings to start from" },
  {
    id: "fill_gap",
    label: "Add what's missing",
    hint: "Adds a part the article skips",
  },
  {
    id: "answer_first",
    label: "Rewrite as a direct answer",
    hint: "Select the opening paragraph first",
  },
  {
    id: "fix_top_issue",
    label: "Fix the top score issue",
    hint: "Select the passage first",
  },
  {
    id: "faq",
    label: "Write FAQ answers",
    hint: "Suggests questions from your article",
  },
  {
    id: "destuff",
    label: "Remove repeated keyword",
    hint: "Select text first",
  },
];
export const PRIMARY_ACTIONS = 5;
