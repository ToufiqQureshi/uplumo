export const ACCOUNT_DELETION_PATH = "/api/auth/delete-account";

export const accountDeletionReasons: Readonly<Record<string, string>> = {
  just_testing: "I was just trying Uplumo",
  no_longer_needed: "I no longer need Uplumo",
  too_expensive: "Too expensive",
  missing_feature: "Missing a feature I need",
  hard_to_use: "Hard to use",
  technical_problem: "Something didn't work",
  other: "Other",
};
