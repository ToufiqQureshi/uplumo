package aisearch

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestConcurrentPromptExplorersSharePaidAnswers(t *testing.T) {
	for _, separateOrganizations := range []bool{false, true} {
		t.Run(map[bool]string{false: "one organization", true: "separate organizations"}[separateOrganizations], func(t *testing.T) {
			api := newFakeAPI(t)
			for _, path := range []string{chatGPTResponsePath, claudeResponsePath} {
				api.on(path, func(int, string) taskReply {
					time.Sleep(30 * time.Millisecond)
					return okReply(answer(true, "Use Acme.", "https://acme.com/review"))
				})
			}
			svc := newTestService(t, api)
			org, otherOrg := newOrg(), newOrg()
			start := make(chan struct{})
			var workers sync.WaitGroup
			for i := range 8 {
				workers.Go(func() {
					<-start
					in := promptInput(modelChatGPT, modelClaude)
					in.WebSearch = false
					in.HighlightBrand = "Acme"
					selectedOrg := org
					if i%2 == 1 {
						in.Models = []string{modelClaude, modelChatGPT}
						in.HighlightBrand = "Nobody"
						if separateOrganizations {
							selectedOrg = otherOrg
						}
					}
					result, err := svc.ExplorePrompt(t.Context(), selectedOrg, "p", in)
					if err != nil {
						t.Error(err)
						return
					}
					for j, model := range in.Models {
						got := firstSuccess(t, result, j)
						if got.Model != model || got.BrandMentioned == nil || *got.BrandMentioned != (in.HighlightBrand == "Acme") {
							t.Errorf("caller order/brand changed: model=%s result=%+v", model, got)
						}
					}
				})
			}
			close(start)
			workers.Wait()
			want := 1
			if separateOrganizations {
				want = 2
			}
			for _, path := range []string{chatGPTResponsePath, claudeResponsePath} {
				if calls := len(api.requestsTo(path)); calls != want {
					t.Errorf("paid calls to %s=%d, want %d", path, calls, want)
				}
			}
		})
	}
}

func TestCancelledPromptExplorerDoesNotBuyAnswers(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := svc.ExplorePrompt(ctx, newOrg(), "p", promptInput(modelChatGPT)); err == nil {
		t.Fatal("cancelled prompt returned success")
	}
	if calls := len(api.requestsTo(chatGPTResponsePath)); calls != 0 {
		t.Fatalf("cancelled prompt bought %d answers", calls)
	}
}
