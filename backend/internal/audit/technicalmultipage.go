package audit

// findCanonicalAndHreflangIssues compares only targets seen in this audit.
// An unfetched target is unknown, not an error or proof of a missing link.
func findCanonicalAndHreflangIssues(pages []SlimPage) []DetectedIssue {
	byURL := make(map[string]SlimPage, len(pages))
	for _, page := range pages {
		if key := normalizedCanonical(page.URL, page.URL); key != "" {
			byURL[key] = page
		}
	}
	var issues []DetectedIssue
	for _, page := range pages {
		if !isOKHTMLPage(page) {
			continue
		}
		pageKey := normalizedCanonical(page.URL, page.URL)
		canonical := page.CanonicalURL
		if canonical == "" {
			canonical = page.HeaderCanonicalURL
		}
		if target, ok := byURL[normalizedCanonical(canonical, page.URL)]; ok && target.StatusCode != nil {
			if *target.StatusCode >= 300 || !target.IsIndexable {
				pageID := page.ID
				issues = append(issues, DetectedIssue{IssueType: IssueBadCanonicalTarget, PageID: &pageID, PageURL: page.URL, Details: map[string]any{"target": target.URL, "statusCode": *target.StatusCode, "indexable": target.IsIndexable}})
			}
		}
		for _, alt := range page.HreflangTargets {
			targetKey := normalizedCanonical(alt.URL, page.URL)
			target, ok := byURL[targetKey]
			if !ok || target.StatusCode == nil || targetKey == pageKey {
				continue
			}
			pageID := page.ID
			if *target.StatusCode >= 300 {
				issues = append(issues, DetectedIssue{IssueType: IssueHreflangBadTarget, PageID: &pageID, PageURL: page.URL, DedupeKey: target.URL, Details: map[string]any{"target": target.URL, "statusCode": *target.StatusCode}})
				continue
			}
			if target.SignalsVersion < CitationSignalsVersion {
				continue
			}
			reciprocal := false
			for _, back := range target.HreflangTargets {
				if normalizedCanonical(back.URL, target.URL) == pageKey {
					reciprocal = true
					break
				}
			}
			if !reciprocal {
				issues = append(issues, DetectedIssue{IssueType: IssueHreflangNotReciprocal, PageID: &pageID, PageURL: page.URL, DedupeKey: target.URL, Details: map[string]any{"target": target.URL}})
			}
		}
	}
	return issues
}
