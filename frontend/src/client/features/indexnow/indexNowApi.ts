import { z } from "zod";
import { apiRequest, shouldRetryApiError } from "@/client/lib/uplumoApi";

const setupSchema = z.object({
  included: z.boolean(),
  domain: z.string(),
  // "" until IndexNow is set up for the project.
  key: z.string(),
});
export type IndexNowSetup = z.infer<typeof setupSchema>;

const keyFilesSchema = z.object({
  files: z.array(
    z.object({
      host: z.string(),
      status: z.enum([
        "found",
        "missing",
        "wrong_content",
        "redirects",
        "moved",
        "unreachable",
        "unknown",
      ]),
    }),
  ),
});
export type KeyFileStatus = z.infer<
  typeof keyFilesSchema
>["files"][number]["status"];

const submitSchema = z.object({
  hosts: z.array(
    z.object({
      host: z.string(),
      urlCount: z.number(),
      status: z.enum(["submitted", "failed"]),
      message: z.string(),
    }),
  ),
  skipped: z.array(z.string()),
  skippedCount: z.number(),
});
type IndexNowSubmission = z.infer<typeof submitSchema>;

const indexNowPath = (projectId: string, action: string) =>
  `/api/v1/projects/${encodeURIComponent(projectId)}/indexnow/${action}`;

export const indexNowSetupQueryOptions = (projectId: string) => ({
  queryKey: ["indexNow", projectId],
  queryFn: () =>
    apiRequest(indexNowPath(projectId, "get"), setupSchema, "POST", {}),
  retry: shouldRetryApiError,
});

export function setUpIndexNow(projectId: string): Promise<IndexNowSetup> {
  return apiRequest(indexNowPath(projectId, "setup"), setupSchema, "POST", {});
}

export function checkIndexNowKeyFiles(projectId: string) {
  return apiRequest(
    indexNowPath(projectId, "check"),
    keyFilesSchema,
    "POST",
    {},
  );
}

export function submitIndexNowUrls(
  projectId: string,
  urls: string[],
): Promise<IndexNowSubmission> {
  return apiRequest(indexNowPath(projectId, "submit"), submitSchema, "POST", {
    urls,
  });
}
