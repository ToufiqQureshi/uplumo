import type { KeywordIntent } from "@/types/keywords";

export type ResultLimit = 150 | 300 | 500;

export type KeywordSource = "related" | "suggestions" | "ideas";
export type KeywordMode = "auto" | KeywordSource;
