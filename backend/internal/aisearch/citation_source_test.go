package aisearch

import "testing"

func TestGeminiCitationsUseSafeOriginalSource(t *testing.T) {
	const redirect = "https://vertexaisearch.cloud.google.com/grounding-api-redirect/example"
	for _, tc := range []struct {
		name, direct, fallback, wantURL, wantDomain string
		wantMatched                                 bool
	}{
		{"original publisher", "https://acme.com/review", redirect, "https://acme.com/review", "acme.com", true},
		{"unsafe original", "javascript:alert(1)", redirect, redirect, "vertexaisearch.cloud.google.com", false},
		{"missing original", "", redirect, redirect, "vertexaisearch.cloud.google.com", false},
		{"credentialed original", "https://user:password@acme.com", redirect, redirect, "vertexaisearch.cloud.google.com", false},
		{"both unsafe", "javascript:alert(1)", "data:text/html,unsafe", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.on(geminiResponsePath, func(int, string) taskReply {
				return okReply(map[string]any{
					"model_name": "gemini-2.5-pro", "web_search": true,
					"items": []any{map[string]any{
						"type": "message", "sections": []any{map[string]any{
							"text": "A useful answer.", "annotations": []any{map[string]any{
								"title": "Source", "url": tc.fallback, "direct_url": tc.direct,
							}},
						}},
					}},
				})
			})
			svc := newTestService(t, api)
			in := promptInput(modelGemini)
			in.HighlightBrand = "acme.com"
			result, err := svc.ExplorePrompt(t.Context(), newOrg(), "p", in)
			if err != nil {
				t.Fatal(err)
			}
			got := firstSuccess(t, result, 0)
			if tc.wantURL == "" {
				if len(got.Citations) != 0 {
					t.Fatalf("unsafe citations retained: %+v", got.Citations)
				}
			} else if len(got.Citations) != 1 || got.Citations[0].URL != tc.wantURL || got.Citations[0].Domain == nil || *got.Citations[0].Domain != tc.wantDomain || got.Citations[0].MatchedBrand != tc.wantMatched {
				t.Fatalf("citations=%+v, want URL=%s domain=%s matched=%v", got.Citations, tc.wantURL, tc.wantDomain, tc.wantMatched)
			}
			if calls := len(api.requestsTo(geminiResponsePath)); calls != 1 {
				t.Fatalf("paid calls=%d, want 1", calls)
			}
		})
	}
}
