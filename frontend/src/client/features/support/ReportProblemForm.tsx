import { useRef, useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { version } from "../../../../package.json";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Textarea } from "@/client/components/ui/textarea";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/client/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/client/components/ui/select";
import { getRecentApiFailure } from "@/client/lib/uplumoApi";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  readScreenshot,
  reportPath,
  submitReport,
  type ReportInput,
} from "./reportApi";

function initialContext() {
  const failure = getRecentApiFailure();
  if (failure) return { ...failure, pagePath: reportPath(failure.pagePath) };
  const referrer = document.referrer ? new URL(document.referrer) : undefined;
  return {
    pagePath: reportPath(
      referrer?.origin === window.location.origin
        ? referrer.pathname
        : window.location.pathname,
    ),
  };
}

export function ReportProblemForm() {
  const [context] = useState(initialContext);
  const [kind, setKind] = useState<ReportInput["kind"]>("bug");
  const [description, setDescription] = useState("");
  const [validateDescription, setValidateDescription] = useState(false);
  const [pagePath, setPagePath] = useState(context.pagePath);
  const [submissionId, setSubmissionId] = useState(() => crypto.randomUUID());
  const [screenshot, setScreenshot] = useState<string>();
  const [fileError, setFileError] = useState("");
  const [reading, setReading] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const descriptionInput = useRef<HTMLTextAreaElement>(null);
  const sending = useRef(false);
  const report = useMutation({ mutationFn: submitReport, retry: false });
  const busy = reading || report.isPending;
  const trimmedDescription = description.trim();
  // Count Unicode code points to match the server's rune count.
  const descriptionTooShort = Array.from(trimmedDescription).length < 10;
  const showDescriptionError = validateDescription && descriptionTooShort;

  function edited() {
    setSubmissionId(crypto.randomUUID());
    report.reset();
  }

  async function send(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (sending.current || busy || fileError) return;
    if (descriptionTooShort) {
      setValidateDescription(true);
      descriptionInput.current?.focus();
      return;
    }
    sending.current = true;
    try {
      await report.mutateAsync({
        kind,
        description: trimmedDescription,
        pagePath: reportPath(pagePath),
        appVersion: version,
        requestId: context.requestId,
        submissionId,
        screenshot,
      });
    } catch {
      // Keep the same submission ID and input after an ambiguous response.
    } finally {
      sending.current = false;
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <h2>Report a problem</h2>
        </CardTitle>
        <CardDescription>
          Tell us what happened and what you expected. You can also suggest an
          improvement.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {report.isSuccess ? (
          <div className="space-y-3">
            <p role="status">
              Your report is saved. Ticket number:{" "}
              <strong>{report.data.ticketId}</strong>. Keep this number if you
              contact support.
            </p>
            <Button
              variant="outline"
              onClick={() => {
                setDescription("");
                setValidateDescription(false);
                setScreenshot(undefined);
                setFileError("");
                edited();
              }}
            >
              Send another report
            </Button>
          </div>
        ) : (
          <form onSubmit={send} className="max-w-2xl space-y-4">
            <div className="space-y-2">
              <label htmlFor="report-kind" className="text-sm font-medium">
                Report type
              </label>
              <Select
                value={kind}
                onValueChange={(value) => {
                  if (
                    value === "bug" ||
                    value === "billing" ||
                    value === "suggestion"
                  ) {
                    setKind(value);
                    edited();
                  }
                }}
                disabled={busy}
              >
                <SelectTrigger id="report-kind" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="bug">Bug or something broken</SelectItem>
                  <SelectItem value="billing">Credits or payment</SelectItem>
                  <SelectItem value="suggestion">
                    Feature request or feedback
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <label
                htmlFor="report-description"
                className="text-sm font-medium"
              >
                What happened?
              </label>
              <Textarea
                ref={descriptionInput}
                id="report-description"
                aria-required="true"
                aria-invalid={showDescriptionError || undefined}
                maxLength={5000}
                rows={5}
                value={description}
                disabled={busy}
                onChange={(event) => {
                  setDescription(event.target.value);
                  edited();
                }}
                aria-describedby={
                  showDescriptionError
                    ? "report-description-error report-privacy"
                    : "report-privacy"
                }
              />
              {showDescriptionError && (
                <p
                  id="report-description-error"
                  role="alert"
                  className="text-sm text-destructive"
                >
                  Enter at least 10 characters describing what happened.
                </p>
              )}
              <p id="report-privacy" className="text-xs text-muted-foreground">
                Minimum 10 characters. Include the steps to reproduce it. Do not
                include passwords, API keys or payment details.
              </p>
            </div>
            <div className="space-y-2">
              <label htmlFor="report-page" className="text-sm font-medium">
                Affected page
              </label>
              <Input
                id="report-page"
                required
                maxLength={500}
                pattern="/.*"
                value={pagePath}
                disabled={busy}
                onChange={(event) => {
                  setPagePath(event.target.value);
                  edited();
                }}
                aria-describedby="report-context"
              />
              <p id="report-context" className="text-xs text-muted-foreground">
                Page path only, for example /billing/plan. URL queries and
                fragments are removed. App version
                {context.requestId
                  ? " and the recent failed request ID are"
                  : " is"}{" "}
                included automatically.
              </p>
            </div>
            <div className="space-y-2">
              <label
                htmlFor="report-screenshot"
                className="text-sm font-medium"
              >
                Screenshot (optional)
              </label>
              <Input
                ref={fileInput}
                id="report-screenshot"
                type="file"
                accept="image/png,image/jpeg"
                disabled={busy}
                aria-describedby="report-screenshot-help"
                onChange={async (event) => {
                  const file = event.target.files?.[0];
                  edited();
                  setScreenshot(undefined);
                  setFileError("");
                  if (!file) return;
                  setReading(true);
                  try {
                    setScreenshot(await readScreenshot(file));
                  } catch (error) {
                    setFileError(getStandardErrorMessage(error));
                  } finally {
                    setReading(false);
                  }
                }}
              />
              <p
                id="report-screenshot-help"
                className="text-xs text-muted-foreground"
              >
                PNG or JPEG, up to 1 MB. Hide any private information first.
              </p>
              {screenshot || fileError ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={busy}
                  onClick={() => {
                    setScreenshot(undefined);
                    setFileError("");
                    if (fileInput.current) fileInput.current.value = "";
                    edited();
                  }}
                >
                  Remove screenshot
                </Button>
              ) : null}
              {fileError ? (
                <p role="alert" className="text-sm text-destructive">
                  {fileError}
                </p>
              ) : null}
            </div>
            {report.isError ? (
              <p role="alert" className="text-sm text-destructive">
                {getStandardErrorMessage(
                  report.error,
                  "Could not send your report. Try again; your details are kept.",
                )}
              </p>
            ) : null}
            <Button type="submit" disabled={busy || Boolean(fileError)}>
              {reading
                ? "Reading screenshot…"
                : report.isPending
                  ? "Sending…"
                  : "Send report"}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
