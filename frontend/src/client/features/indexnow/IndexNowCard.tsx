import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Download } from "lucide-react";
import { useState, type FormEvent, type ReactNode } from "react";
import { ErrorState } from "@/client/components/ErrorState";
import { QueryError } from "@/client/components/QueryState";
import { Spinner } from "@/client/components/Spinner";
import { Button } from "@/client/components/ui/button";
import { Textarea } from "@/client/components/ui/textarea";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  checkIndexNowKeyFiles,
  indexNowSetupQueryOptions,
  setUpIndexNow,
  submitIndexNowUrls,
  type IndexNowSetup,
  type KeyFileStatus,
} from "./indexNowApi";

const KEY_FILE_STATUS: Record<KeyFileStatus, string> = {
  found: "Key file found.",
  missing: "No key file here yet.",
  wrong_content: "The file must contain only the key.",
  redirects: "This address redirects. Upload the file here itself.",
  moved: "Redirects to your other address. Fine if all your pages live there.",
  unreachable: "This address does not load.",
  unknown:
    "Couldn't check (a firewall or slow server). Submitting will still try.",
};

/**
 * Sets up IndexNow for the project: the key file the customer hosts, a check
 * that it is live, and a box to submit changed pages. Once set up, pages
 * published to WordPress from Uplumo are submitted automatically.
 */
export function IndexNowCard({ projectId }: { projectId: string }) {
  const setupQuery = useQuery(indexNowSetupQueryOptions(projectId));

  if (setupQuery.isPending) {
    return <Spinner size="sm" label="Checking IndexNow…" />;
  }
  if (setupQuery.isError) {
    return (
      <QueryError
        error={setupQuery.error}
        fallback="Couldn't load IndexNow."
        onRetry={() => void setupQuery.refetch()}
        isRetrying={setupQuery.isFetching}
      />
    );
  }
  const setup = setupQuery.data;
  if (!setup.included) {
    return (
      <Notice>
        IndexNow is part of Basic, Pro and Agency.{" "}
        <Link to="/billing/plan" className="underline underline-offset-4">
          See plans
        </Link>
      </Notice>
    );
  }
  if (!setup.domain) {
    return (
      <Notice>
        Add this project's website in{" "}
        <Link
          to="/p/$projectId/settings"
          params={{ projectId }}
          className="underline underline-offset-4"
        >
          project settings
        </Link>{" "}
        first.
      </Notice>
    );
  }
  if (!setup.key) {
    return <SetUp projectId={projectId} />;
  }
  return <Ready projectId={projectId} setup={setup} />;
}

function SetUp({ projectId }: { projectId: string }) {
  const queryClient = useQueryClient();
  const setUp = useMutation({
    mutationFn: () => setUpIndexNow(projectId),
    meta: { errorToast: false },
    onSuccess: (next) => {
      queryClient.setQueryData(
        indexNowSetupQueryOptions(projectId).queryKey,
        next,
      );
    },
  });
  return (
    <div className="space-y-3 rounded-lg border border-border bg-card p-4">
      <p className="text-sm text-muted-foreground">
        Tell Bing, Yandex, Naver, Seznam and Yep about new and changed pages, so
        they crawl them sooner. Google does not use IndexNow: for Google, keep
        using Search Console.
      </p>
      <Button
        size="sm"
        onClick={() => setUp.mutate()}
        disabled={setUp.isPending}
      >
        {setUp.isPending ? "Setting up…" : "Set up IndexNow"}
      </Button>
      {setUp.isError ? (
        <ErrorState
          variant="inline"
          message={getStandardErrorMessage(setUp.error)}
        />
      ) : null}
    </div>
  );
}

function Ready({
  projectId,
  setup,
}: {
  projectId: string;
  setup: IndexNowSetup;
}) {
  const fileName = `${setup.key}.txt`;
  const check = useMutation({
    mutationFn: () => checkIndexNowKeyFiles(projectId),
    meta: { errorToast: false },
  });

  return (
    <div className="space-y-4 rounded-lg border border-border bg-card p-4">
      <div className="space-y-2 text-sm">
        <p className="font-medium">1. Put the key file on your website</p>
        <p className="text-muted-foreground">
          Upload <span className="font-mono text-foreground">{fileName}</span>{" "}
          to the main folder of your site, so it opens at{" "}
          <span className="font-mono text-foreground break-all">
            https://{setup.domain}/{fileName}
          </span>
          . On WordPress, use your host's File Manager (the folder with
          wp-config.php). The file holds only the key; it is not a password.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => downloadKeyFile(fileName, setup.key)}
          >
            <Download className="size-4" />
            Download key file
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => check.mutate()}
            disabled={check.isPending}
          >
            {check.isPending ? "Checking…" : "Check key file"}
          </Button>
        </div>
        {check.isError ? (
          <ErrorState
            variant="inline"
            message={getStandardErrorMessage(check.error)}
          />
        ) : null}
        {check.data ? (
          <ul className="space-y-1">
            {check.data.files.map((file) => (
              <li key={file.host}>
                <span className="font-mono">{file.host}</span>:{" "}
                <span
                  className={
                    file.status === "found"
                      ? "text-primary"
                      : "text-muted-foreground"
                  }
                >
                  {KEY_FILE_STATUS[file.status]}
                </span>
              </li>
            ))}
          </ul>
        ) : null}
      </div>

      <div className="space-y-2 text-sm">
        <p className="font-medium">2. Submit pages</p>
        <p className="text-muted-foreground">
          Articles you publish to WordPress from Uplumo are sent automatically.
          For other pages, paste the ones that are new or changed. Only send
          changed pages: engines slow down sites that resend everything.
        </p>
        <SubmitForm projectId={projectId} domain={setup.domain} />
      </div>
    </div>
  );
}

function SubmitForm({
  projectId,
  domain,
}: {
  projectId: string;
  domain: string;
}) {
  const [text, setText] = useState("");
  const urls = text
    .split(/\s+/)
    .map((url) => url.trim())
    .filter(Boolean);
  const submit = useMutation({
    mutationFn: () => submitIndexNowUrls(projectId, urls),
    meta: { errorToast: false },
  });

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (urls.length > 0) submit.mutate();
  }

  return (
    <form onSubmit={onSubmit} className="space-y-2">
      <Textarea
        value={text}
        onChange={(event) => setText(event.target.value)}
        placeholder={`https://${domain}/new-page\nhttps://${domain}/updated-page`}
        rows={4}
        aria-label="Page URLs, one per line"
      />
      <Button
        type="submit"
        size="sm"
        disabled={submit.isPending || urls.length === 0 || urls.length > 10000}
      >
        {submit.isPending ? "Submitting…" : submitLabel(urls.length)}
      </Button>
      {urls.length > 10000 ? (
        <p className="text-sm text-muted-foreground">
          Up to 10,000 pages at a time.
        </p>
      ) : null}
      {submit.isError ? (
        <ErrorState
          variant="inline"
          message={getStandardErrorMessage(submit.error)}
        />
      ) : null}
      {submit.data ? (
        <ul className="space-y-1">
          {submit.data.hosts.map((host) => (
            <li key={host.host}>
              <span className="font-mono">{host.host}</span>: {host.message}
            </li>
          ))}
          {submit.data.skippedCount > 0 ? (
            <li className="text-muted-foreground">
              {submit.data.skippedCount} not sent: not on {domain}, or more than
              3 addresses at once.
            </li>
          ) : null}
        </ul>
      ) : null}
    </form>
  );
}

function submitLabel(count: number) {
  if (count === 0) return "Submit pages";
  return `Submit ${count} ${count === 1 ? "page" : "pages"}`;
}

function downloadKeyFile(fileName: string, key: string) {
  const url = URL.createObjectURL(new Blob([key], { type: "text/plain" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  link.click();
  URL.revokeObjectURL(url);
}

function Notice({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-lg border border-border bg-card p-4 text-sm text-muted-foreground">
      {children}
    </p>
  );
}
