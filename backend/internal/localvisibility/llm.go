package localvisibility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// LLM streams one completion; *openrouter.Client implements it.
type LLM interface {
	Stream(context.Context, openrouter.Request, openrouter.Handler) (openrouter.Result, error)
}

// Meter holds credits before a model call; billing's AI meter implements it.
type Meter interface {
	HoldWithIdempotencyKey(ctx context.Context, organizationID string, ceilingUSD float64, key string) (spend.Hold, error)
}

// Model and its price cap (USD per million tokens), sent to OpenRouter as
// max_price so no provider bills above what the hold assumed.
const model = "deepseek/deepseek-v4.1-flash"

var price = openrouter.MaxPrice{Prompt: 0.30, Completion: 1.20}

// Size limits that make the most a call can cost a fixed number, shown on
// the button before the click.
const (
	themesSystemBytes  = 1600
	themesInputBytes   = 48000
	themesMaxTokens    = 4000
	replyInputBytes    = 2600
	replyMaxTokens     = 400
	reviewSnippetRunes = 400
	maxThemes          = 8
	callTimeout        = 90 * time.Second
)

// Ceilings: every input byte as a third of a token, plus all output tokens.
var (
	themesCeilingUSD = ceilingUSD(themesSystemBytes+themesInputBytes, themesMaxTokens)
	replyCeilingUSD  = ceilingUSD(replyInputBytes, replyMaxTokens)
)

func ceilingUSD(inputBytes, maxTokens int) float64 {
	return (float64((inputBytes+2)/3)*price.Prompt + float64(maxTokens)*price.Completion) / 1_000_000
}

func requestBytes(req openrouter.Request) int {
	n := 0
	for _, m := range req.Messages {
		n += len(m.Content)
	}
	return n
}

// callCost is what OpenRouter reported, or the price cap applied to the
// tokens when it reported nothing.
func callCost(result openrouter.Result, req openrouter.Request) float64 {
	if result.Usage.CostUSD > 0 {
		return result.Usage.CostUSD
	}
	output := max(result.Usage.CompletionTokens, (len(result.Text)+2)/3)
	input := max(result.Usage.PromptTokens, (requestBytes(req)+2)/3)
	return (float64(input)*price.Prompt + float64(output)*price.Completion) / 1_000_000
}

// call holds ceiling, makes one model call and settles: the customer pays
// the call when parse accepts the answer, Uplumo pays it otherwise.
func (s *Service) call(ctx context.Context, org, key string, req openrouter.Request, ceiling float64, parse func(string) error) error {
	if s.LLM == nil || s.Meter == nil {
		return errors.New("AI is not configured")
	}
	if bytes := requestBytes(req); ceilingUSD(bytes, req.MaxTokens) > ceiling {
		return fmt.Errorf("AI request of %d bytes is above its quoted ceiling", bytes)
	}
	hold, err := s.Meter.HoldWithIdempotencyKey(ctx, org, ceiling, key)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	result, callErr := s.LLM.Stream(callCtx, req, openrouter.Handler{})
	cancel()
	provider := callCost(result, req)
	customer := 0.0
	if callErr == nil {
		if callErr = parse(result.Text); callErr == nil {
			customer = min(provider, ceiling)
		}
	}
	settleCtx, cancelSettle := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelSettle()
	return errors.Join(callErr, spend.SettleProviderCost(settleCtx, hold, provider, customer, "openrouter/local_visibility"))
}

func providerPreferences() *openrouter.ProviderPreferences {
	p := price
	return &openrouter.ProviderPreferences{MaxPrice: &p, DataCollection: "deny", RequireParameters: true}
}

type reviewSet struct {
	Key     string
	Reviews []Review
}

const themesSystem = `You group Google reviews of local businesses into themes.
Input: reviews, each starting with an id in square brackets, then the star rating.
Find at most 8 themes customers mention (for example a service, staff, waiting, price, cleanliness). Name each theme in 1-3 plain English words.
For each theme list the ids of the reviews that speak well of it in "pos" and of those that complain about it in "neg".
Use only ids from the input. Do not invent themes nobody mentions.
Answer with compact JSON only: {"themes":[{"name":"...","pos":["y1","a4"],"neg":["y7"]}]}`

// themes asks the model which reviews mention which theme, and counts the
// mentions itself: the model never reports a number, only review ids that
// are checked against the input.
func (s *Service) themes(ctx context.Context, run *Run, sets []reviewSet) (map[string][]Theme, error) {
	var b strings.Builder
	known := map[string]string{}
	// Ids are a letter per business and the review's index (y3, a12): short
	// ids keep the answer small enough to finish inside its token cap.
	for n, set := range sets {
		letter := string(rune('y' + n))
		if n > 0 {
			letter = string(rune('a' + n - 1))
		}
		for i, r := range set.Reviews {
			text := strings.Join(strings.Fields(r.Text), " ")
			if text == "" {
				continue
			}
			id := letter + strconv.Itoa(i)
			line := fmt.Sprintf("[%s] %d★ %s\n", id, r.Rating, truncate(text, reviewSnippetRunes))
			if b.Len()+len(line) > themesInputBytes {
				break
			}
			b.WriteString(line)
			known[id] = set.Key
		}
	}
	if len(known) == 0 {
		return map[string][]Theme{}, nil
	}
	req := openrouter.Request{
		Model:      model,
		Messages:   []openrouter.Message{{Role: "system", Content: themesSystem}, {Role: "user", Content: b.String()}},
		MaxTokens:  themesMaxTokens,
		JSONObject: true,
		Provider:   providerPreferences(),
	}
	var out map[string][]Theme
	err := s.call(ctx, run.OrganizationID, "local-themes:"+run.ID, req, themesCeilingUSD, func(text string) error {
		var err error
		out, err = countThemes(text, known)
		return err
	})
	return out, err
}

func countThemes(text string, known map[string]string) (map[string][]Theme, error) {
	var answer struct {
		Themes []struct {
			Name string   `json:"name"`
			Pos  []string `json:"pos"`
			Neg  []string `json:"neg"`
		} `json:"themes"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &answer); err != nil {
		return nil, fmt.Errorf("review themes are not JSON: %w", err)
	}
	out := map[string][]Theme{}
	for _, t := range answer.Themes[:min(maxThemes, len(answer.Themes))] {
		name := truncate(strings.Join(strings.Fields(t.Name), " "), 40)
		if name == "" {
			continue
		}
		perSet := map[string]*Theme{}
		seen := map[string]bool{}
		count := func(ids []string, add func(*Theme)) {
			for _, id := range ids {
				set, ok := known[id]
				if !ok || seen[id] {
					continue
				}
				seen[id] = true
				if perSet[set] == nil {
					perSet[set] = &Theme{Name: name}
				}
				add(perSet[set])
			}
		}
		count(t.Pos, func(theme *Theme) { theme.Positive++ })
		count(t.Neg, func(theme *Theme) { theme.Negative++ })
		for set, theme := range perSet {
			out[set] = append(out[set], *theme)
		}
	}
	return out, nil
}

const replySystem = `You write the business owner's public reply to a Google review.
Write in the language of the review. Thank the reviewer, by first name when one is given.
Answer the specific points the review makes. For a complaint, apologise without arguing and invite them to contact the business directly.
Never promise refunds, discounts or anything the review does not mention, and never state facts about the business you were not given.
At most 80 words. Plain text only: no quotes, no placeholders, no signature.`

// DraftReply writes, or returns the saved, reply to one of the unanswered
// reviews of the newest reviews run. The review comes from stored data, not
// from the request, so the call cannot be used as a free-form AI writer.
func (s *Service) DraftReply(ctx context.Context, org, projectID, reviewID string, rewrite bool, idempotencyKey string) (string, error) {
	run, err := s.Store.LatestDone(ctx, projectID, KindReviews, s.now().Add(time.Minute))
	if err != nil {
		return "", err
	}
	if run == nil {
		return "", ErrNotFound
	}
	var result ReviewsResult
	if err := json.Unmarshal(run.Result, &result); err != nil {
		return "", fmt.Errorf("decode reviews result: %w", err)
	}
	var review *Review
	for i, r := range result.Unanswered {
		if r.ID == reviewID {
			review = &result.Unanswered[i]
		}
	}
	if review == nil {
		return "", ErrNotFound
	}
	if !rewrite {
		drafts, err := s.Store.ReplyDrafts(ctx, projectID, []string{reviewID})
		if err != nil {
			return "", err
		}
		if draft, ok := drafts[reviewID]; ok {
			return draft, nil
		}
	}
	business, err := s.Store.Business(ctx, projectID)
	if err != nil {
		return "", err
	}
	if business == nil {
		return "", ErrNoBusiness
	}
	prompt := fmt.Sprintf("Business: %s (%s)\nReviewer: %s\nRating: %d of 5\nReview: %s",
		truncateBytes(business.Title, 160), truncateBytes(business.Category, 100), truncateBytes(review.Author, 80), review.Rating, truncateBytes(review.Text, 1500))
	if rewrite {
		prompt += "\nWrite a different reply from any earlier one."
	}
	req := openrouter.Request{
		Model:     model,
		Messages:  []openrouter.Message{{Role: "system", Content: replySystem}, {Role: "user", Content: prompt}},
		MaxTokens: replyMaxTokens,
		Provider:  providerPreferences(),
	}
	var reply string
	key := "local-reply:" + projectID + ":" + reviewID + ":" + idempotencyKey
	if err := s.call(ctx, org, key, req, replyCeilingUSD, func(text string) error {
		reply = truncate(strings.Trim(strings.TrimSpace(text), `"`), 1200)
		if reply == "" {
			return errors.New("the model wrote an empty reply")
		}
		return nil
	}); err != nil {
		return "", err
	}
	if err := s.Store.SaveReplyDraft(context.WithoutCancel(ctx), projectID, reviewID, reply); err != nil {
		return "", err
	}
	return reply, nil
}

// truncateBytes keeps at most limit bytes, ending on a whole character, so
// a prompt in any script stays inside its quoted ceiling.
func truncateBytes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	cut := 0
	for i := range value {
		if i > limit {
			break
		}
		cut = i
	}
	return value[:cut]
}
