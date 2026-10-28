import { useRef, useState } from "react";
import { useForm } from "@tanstack/react-form";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  deleteAudit,
  getAuditHistory,
  startAudit,
} from "@/client/features/audit/auditApi";
import {
  DEFAULT_LAUNCH_FORM_VALUES,
  getMaxPagesLimit,
  clampMaxPagesInput,
  type LaunchFormValues,
} from "@/client/features/audit/launch/types";
import {
  createFormValidationErrors,
  shouldValidateFieldOnChange,
} from "@/client/lib/forms";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { useAuditCapabilities } from "@/client/features/audit/shared";

function getLaunchValidationErrors(
  value: LaunchFormValues,
  shouldValidateUntouchedField: boolean,
) {
  if (value.url.trim()) {
    return null;
  }

  if (!shouldValidateUntouchedField) {
    return null;
  }

  return createFormValidationErrors({
    fields: {
      url: "Please enter a URL.",
    },
  });
}

export function useLaunchController({
  projectId,
  initialUrl,
  isFreePlan,
  onAuditStarted,
}: {
  projectId: string;
  initialUrl: string;
  isFreePlan: boolean;
  onAuditStarted: (auditId: string) => void;
}) {
  const capabilitiesQuery = useAuditCapabilities(projectId);
  const historyQuery = useQuery({
    queryKey: ["audit-history", projectId],
    queryFn: () => getAuditHistory({ data: { projectId } }),
    // Keep in-progress rows live so status and page counts don't need a reload.
    refetchInterval: (query) =>
      query.state.data?.some((audit) => audit.status === "running")
        ? 5000
        : false,
  });
  const { startMutation, deleteMutation } = useLaunchMutations({ projectId });
  // Pages of a large crawl waiting for the user to confirm it.
  const [largeCrawlPages, setLargeCrawlPages] = useState<number | null>(null);
  const largeCrawlConfirmed = useRef(false);

  const launchForm = useForm({
    defaultValues: { ...DEFAULT_LAUNCH_FORM_VALUES, url: initialUrl },
    validators: {
      onChange: ({ formApi, value }) =>
        getLaunchValidationErrors(
          value,
          shouldValidateFieldOnChange(formApi, "url"),
        ),
      onSubmit: ({ value }) => getLaunchValidationErrors(value, true),
    },
    onSubmit: async ({ formApi, value }) => {
      const effectiveMaxPages = commitMaxPagesInput(
        launchForm,
        getMaxPagesLimit(isFreePlan),
      );
      formApi.setErrorMap({ onSubmit: undefined });

      if (effectiveMaxPages > 500 && !largeCrawlConfirmed.current) {
        setLargeCrawlPages(effectiveMaxPages);
        return;
      }
      largeCrawlConfirmed.current = false;

      try {
        const result = await startMutation.mutateAsync({
          projectId,
          startUrl: value.url,
          maxPages: effectiveMaxPages,
          lighthouseStrategy: value.runLighthouse ? "auto" : "none",
          renderJavaScript: value.renderJavaScript,
        });
        toast.success("Audit started!");
        onAuditStarted(result.auditId);
      } catch (error) {
        const message = getStandardErrorMessage(error, "Failed to start audit");
        formApi.setErrorMap({
          onSubmit: createFormValidationErrors({ form: message }),
        });
      }
    },
  });

  const maxPagesLimit = getMaxPagesLimit(isFreePlan);
  // What an upgrade offers with the current options, so the upgrade hint
  // never promises pages a rendered audit cannot crawl.
  const paidMaxPagesLimit = getMaxPagesLimit(false);

  return {
    launchForm,
    historyQuery,
    canRenderJavaScript: capabilitiesQuery.data?.canRenderJavaScript,
    maxJavaScriptCredits: capabilitiesQuery.data?.maxJavaScriptCredits ?? 0,
    maxPagesLimit,
    paidMaxPagesLimit,
    commitMaxPagesInput: () => commitMaxPagesInput(launchForm, maxPagesLimit),
    deleteAudit: (auditId: string) => deleteMutation.mutate(auditId),
    largeCrawlPages,
    cancelLargeCrawl: () => setLargeCrawlPages(null),
    confirmLargeCrawl: () => {
      setLargeCrawlPages(null);
      largeCrawlConfirmed.current = true;
      void launchForm.handleSubmit();
    },
  };
}

function useLaunchMutations({ projectId }: { projectId: string }) {
  const queryClient = useQueryClient();
  // Invalidate rather than refetch: starting an audit navigates away from the
  // list, so the stale mark is what makes it reload when the user comes back.
  const invalidateHistory = () =>
    queryClient.invalidateQueries({ queryKey: ["audit-history", projectId] });

  const startMutation = useMutation({
    meta: { errorToast: false },
    mutationFn: (data: {
      projectId: string;
      startUrl: string;
      maxPages: number;
      lighthouseStrategy: "auto" | "none";
      renderJavaScript: boolean;
    }) => startAudit({ data }),
    onSuccess: () => {
      void invalidateHistory();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (auditId: string) =>
      deleteAudit({ data: { projectId, auditId } }),
    onSuccess: () => {
      void invalidateHistory();
      toast.success("Audit deleted");
    },
  });

  return { startMutation, deleteMutation };
}

function commitMaxPagesInput(
  launchForm: {
    state: { values: { maxPagesInput: string } };
    setFieldValue: (field: "maxPagesInput", value: string) => void;
  },
  maxPagesLimit: number,
) {
  const safeValue = clampMaxPagesInput(
    launchForm.state.values.maxPagesInput,
    maxPagesLimit,
  );
  launchForm.setFieldValue("maxPagesInput", String(safeValue));
  return safeValue;
}
