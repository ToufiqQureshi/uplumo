package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/citefix"
)

func citeFixReportTool() *tool {
	return &tool{
		Name: "get_cite_fix_report", Title: "Get Cite Fix report",
		Description:  "Read why AI assistants (ChatGPT, Perplexity, Google AI) may not quote the project's site: a 0-100 citation-readiness score in six categories, fixes ranked by points gained with the pages each affects, the weakest pages, and who the latest AI Visibility check saw cited. Built from the latest completed site audit; if state is no_scan or needs_rescan, run run_site_audit first. Free; reads Uplumo state. The paid per-page Cite Fix agent runs only from the Cite Fix page.",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"projectId":{"type":"string","minLength":1}},"required":["projectId"]}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"state":{"type":"string","enum":["ready","no_scan","needs_rescan"]},"score":{"type":["integer","null"]},"categories":{"type":"array"},"fixes":{"type":"array"},"pages":{"type":"array"},"proof":{"type":["object","null"]}},"required":["state"],"additionalProperties":true}`),
		Annotations:  readOnlyAnnotations(),
		Handler:      handleCiteFixReport,
	}
}

func handleCiteFixReport(ctx context.Context, raw json.RawMessage, env *callEnv) (*callResult, error) {
	var args struct {
		ProjectID string `json:"projectId"`
	}
	if err := json.Unmarshal(raw, &args); err != nil || strings.TrimSpace(args.ProjectID) == "" {
		return nil, newAppErrorf("VALIDATION_ERROR", "projectId is required")
	}
	access, err := env.h.authorizeProject(ctx, env.auth, args.ProjectID)
	if err != nil {
		return nil, err
	}
	report, err := citefix.Store{DB: env.deps.DB}.Report(ctx, access.Auth.OrganizationID, access.Project.ID)
	if err != nil {
		return nil, err
	}
	// The agent offer depends on the plan and server; the page shows it with
	// the price, so the tool leaves it out rather than report a wrong one.
	structured := map[string]any{
		"state": report.State, "running": report.Running, "scan": report.Scan, "score": report.Score,
		"categories": report.Categories, "fixes": report.Fixes, "pages": report.Pages,
		"pagesChecked": report.PagesChecked, "pagesTotal": report.PagesTotal, "proof": report.Proof,
	}
	return mcpResponse(citeFixText(report), structured, metaFields{ProjectID: access.Project.ID, URL: buildDashboardURL(access.Auth.BaseURL, "/p/"+access.Project.ID+"/cite-fix", nil)}), nil
}

func citeFixText(report citefix.Report) string {
	var b strings.Builder
	switch report.State {
	case citefix.StateNoScan:
		b.WriteString("Cite Fix: no completed site audit yet. Run run_site_audit, then read this again.")
	case citefix.StateNeedsRescan:
		b.WriteString("Cite Fix: the latest audit predates Cite Fix and lacks citation signals. Run run_site_audit again.")
	default:
		score := 0
		if report.Score != nil {
			score = *report.Score
		}
		fmt.Fprintf(&b, "Cite Fix score %d/100 (%d of %d pages checked).", score, report.PagesChecked, report.PagesTotal)
		for i, fix := range report.Fixes {
			if i == 5 {
				break
			}
			fmt.Fprintf(&b, "\n%d. %s (+%.1f points, %s impact, %d pages)", i+1, fix.Title, fix.Gain, fix.Impact, fix.PagesAffected)
		}
	}
	if report.Proof != nil {
		fmt.Fprintf(&b, "\nLatest AI Visibility check: %d of %d answers cite %s.", report.Proof.CitingYou, report.Proof.Answers, report.Proof.Domain)
	}
	return b.String()
}
