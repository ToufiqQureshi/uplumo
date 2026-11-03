import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

export type ReportInput = {
  kind: "bug" | "billing" | "suggestion";
  description: string;
  pagePath: string;
  appVersion: string;
  requestId?: string;
  submissionId: string;
  screenshot?: string;
};

export function submitReport(input: ReportInput) {
  return apiRequest(
    "/api/v1/support/reports",
    z.object({ ticketId: z.string().uuid() }),
    "POST",
    input,
  );
}

export function reportPath(value: string) {
  const path = value.split(/[?#]/u)[0] || "/support";
  return path.startsWith("/") && !path.startsWith("//") && !path.includes("\\")
    ? path.slice(0, 500)
    : "/support";
}

export async function readScreenshot(file: File): Promise<string> {
  if (
    !["image/png", "image/jpeg"].includes(file.type) ||
    file.size > 1024 * 1024
  ) {
    throw new Error("Choose a PNG or JPEG screenshot smaller than 1 MB.");
  }
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener("load", () =>
      typeof reader.result === "string"
        ? resolve(reader.result)
        : reject(new Error("Could not read the screenshot. Choose it again.")),
    );
    reader.addEventListener("error", () =>
      reject(new Error("Could not read the screenshot. Choose it again.")),
    );
    reader.readAsDataURL(file);
  });
}
