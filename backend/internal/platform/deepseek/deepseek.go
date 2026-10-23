// Package deepseek streams chat completions from the DeepSeek API.
//
// It speaks DeepSeek's OpenAI-format chat completions API and nothing else: no
// retries (the caller decides, because a retry re-bills), no tool execution.
// DeepSeek reports tokens, not money, so the client prices every response from
// the published rate card (prices.go) and fills Usage.CostUSD.
package deepseek

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the DeepSeek API root (OpenAI format).
	DefaultBaseURL = "https://api.deepseek.com"
	// DefaultModel is the model used when none is configured.
	DefaultModel = "deepseek-flash"

	// maxLine bounds one SSE line so a hostile or broken upstream cannot grow
	// memory without limit.
	maxLine = 1 << 20
	// maxErrorBody bounds how much of an error response is read.
	maxErrorBody = 4 << 10
)

// Message is one chat message. Assistant messages may carry ToolCalls; tool
// messages carry the ToolCallID they answer.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is a model request to run one tool. Arguments is a JSON document.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall names the tool and carries its JSON arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool describes one callable function to the model.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// Request is one streaming completion.
type Request struct {
	Model     string
	Messages  []Message
	Tools     []Tool
	MaxTokens int
	// ReasoningEffort turns thinking mode on ("low", "high" or "max"). Empty
	// or "none" sends thinking disabled. With tools, thinking mode requires
	// every earlier reasoning text to be sent back, so tool-using callers keep
	// it off.
	ReasoningEffort string
	// CacheScope separates DeepSeek's prompt cache per customer. It is hashed
	// before it is sent, so no Uplumo id leaves the server.
	CacheScope string
}

// Usage is the token accounting DeepSeek reports, plus CostUSD, which the
// client computes from it. CostUSD is zero only when no usage arrived.
type Usage struct {
	PromptTokens          int     `json:"prompt_tokens"`
	CompletionTokens      int     `json:"completion_tokens"`
	PromptCacheHitTokens  int     `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int     `json:"prompt_cache_miss_tokens"`
	CostUSD               float64 `json:"-"`
}

// charsPerToken is a low guess at characters per token, for output that
// streamed before the stream broke and no usage arrived.
const charsPerToken = 3

// BillableUSD is what to charge for a call to model: the priced usage, or,
// when it was not priced (no usage arrived), an estimate from the token counts
// or, failing those, from generatedChars, the characters that streamed, at the
// model's peak rates (the most expensive model's for one missing from the
// rate card). estimated says which.
func (u Usage) BillableUSD(model string, generatedChars int) (usd float64, estimated bool) {
	if u.CostUSD > 0 && !math.IsInf(u.CostUSD, 0) {
		return u.CostUSD, false
	}
	outputTokens := u.CompletionTokens
	if outputTokens <= 0 && generatedChars > 0 {
		outputTokens = (generatedChars + charsPerToken - 1) / charsPerToken
	}
	inputTokens := max(u.PromptTokens, 0)
	if outputTokens <= 0 && inputTokens == 0 {
		return 0, false
	}
	rates := peakRates(model)
	return float64(inputTokens)*rates.miss + float64(max(outputTokens, 0))*rates.out, true
}

// MaxCostUSD is the most req can cost, for holding credits before it is sent:
// every byte of the messages and tool definitions counted as a third of a
// token (fewer bytes per token than real text, so more tokens), none of it a
// cache hit, and the full MaxTokens of output, at the model's peak rates. A
// model missing from the rate card is priced at the most expensive one. A
// request without MaxTokens has no ceiling, so it is refused with an error.
func MaxCostUSD(req Request) (float64, error) {
	if req.MaxTokens <= 0 {
		return 0, errors.New("a request without MaxTokens has no cost ceiling")
	}
	chars := 0
	for _, message := range req.Messages {
		chars += len(message.Content) + len(message.ToolCallID)
		for _, call := range message.ToolCalls {
			chars += len(call.ID) + len(call.Function.Name) + len(call.Function.Arguments)
		}
	}
	for _, tool := range req.Tools {
		chars += len(tool.Name) + len(tool.Description) + len(tool.Parameters)
	}
	inputTokens := (chars + charsPerToken - 1) / charsPerToken
	rates := peakRates(req.Model)
	return float64(inputTokens)*rates.miss + float64(req.MaxTokens)*rates.out, nil
}

// Result is the assembled response once the stream ends.
type Result struct {
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
}

// Handler receives deltas as they arrive. Either callback may be nil.
type Handler struct {
	OnText      func(delta string) error
	OnReasoning func(delta string) error
}

// Client calls the DeepSeek API.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
	// Now is the clock used to price peak and off-peak hours; nil is time.Now.
	Now func() time.Time
}

// StatusError is a non-2xx response from the API.
type StatusError struct {
	StatusCode int
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("deepseek returned %d: %s", e.StatusCode, e.Body)
}

// ErrStreamError means the upstream sent an error object mid-stream.
var ErrStreamError = errors.New("deepseek stream error")

type requestBody struct {
	Model           string            `json:"model"`
	Messages        []Message         `json:"messages"`
	Tools           []toolEnvelope    `json:"tools,omitempty"`
	Stream          bool              `json:"stream"`
	StreamOptions   map[string]bool   `json:"stream_options"`
	MaxTokens       int               `json:"max_tokens,omitempty"`
	Thinking        map[string]string `json:"thinking"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
}

type toolEnvelope struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type streamChunk struct {
	Error   *streamError `json:"error"`
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

type streamError struct {
	Message string `json:"message"`
}

// cacheScope turns an Uplumo id into an opaque value DeepSeek accepts as
// user_id ([a-zA-Z0-9_-], at most 512 characters).
func cacheScope(scope string) string {
	if scope == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("uplumo-cache:" + scope))
	return hex.EncodeToString(sum[:16])
}

// Stream sends req and feeds deltas to h until the response ends. On error the
// partial Result is still returned so the caller can bill and persist what was
// produced. Result.Usage.CostUSD is priced at the peak rate when the call
// started or ended in peak hours.
func (c *Client) Stream(ctx context.Context, req Request, h Handler) (Result, error) {
	body := requestBody{
		Model:         req.Model,
		Messages:      req.Messages,
		Stream:        true,
		StreamOptions: map[string]bool{"include_usage": true},
		MaxTokens:     req.MaxTokens,
		Thinking:      map[string]string{"type": "disabled"},
		UserID:        cacheScope(req.CacheScope),
	}
	if effort := req.ReasoningEffort; effort != "" && effort != "none" {
		body.Thinking = map[string]string{"type": "enabled"}
		body.ReasoningEffort = effort
	}
	for _, t := range req.Tools {
		params := t.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		body.Tools = append(body.Tools, toolEnvelope{Type: "function", Function: toolFunction{
			Name: t.Name, Description: t.Description, Parameters: params,
		}})
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Result{}, fmt.Errorf("encode deepseek request: %w", err)
	}

	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return Result{}, fmt.Errorf("build deepseek request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	client := c.HTTP
	if client == nil {
		// No overall timeout: a turn can legitimately stream for minutes. The
		// header timeout bounds how long the provider may take to start.
		client = &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 60 * time.Second}}
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	started := now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("call deepseek: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		closeErr := resp.Body.Close()
		statusErr := &StatusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(snippet))}
		if readErr != nil || closeErr != nil {
			return Result{}, errors.Join(statusErr, readErr, closeErr)
		}
		return Result{}, statusErr
	}
	result, streamErr := readStream(resp.Body, h)
	if closeErr := resp.Body.Close(); closeErr != nil {
		streamErr = errors.Join(streamErr, fmt.Errorf("close DeepSeek response: %w", closeErr))
	}
	result.Usage.CostUSD = priceUsage(req.Model, result.Usage, IsPeak(started) || IsPeak(now()))
	return result, streamErr
}

// readStream folds the SSE body into a Result.
func readStream(body io.Reader, h Handler) (Result, error) {
	var (
		result Result
		text   strings.Builder
		calls  = map[int]*ToolCall{}
	)
	finish := func() Result {
		result.Text = text.String()
		result.ToolCalls = nil
		for _, index := range slices.Sorted(maps.Keys(calls)) {
			call := calls[index]
			call.Type = "function"
			result.ToolCalls = append(result.ToolCalls, *call)
		}
		return result
	}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), maxLine)
	for scanner.Scan() {
		line := scanner.Text()
		// Comment lines (": keep-alive") keep the connection warm.
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return finish(), fmt.Errorf("decode deepseek chunk: %w", err)
		}
		if chunk.Error != nil {
			return finish(), fmt.Errorf("%w: %s", ErrStreamError, chunk.Error.Message)
		}
		if chunk.Usage != nil {
			result.Usage = *chunk.Usage
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.ReasoningContent != "" && h.OnReasoning != nil {
				if err := h.OnReasoning(choice.Delta.ReasoningContent); err != nil {
					return finish(), err
				}
			}
			if choice.Delta.Content != "" {
				text.WriteString(choice.Delta.Content)
				if h.OnText != nil {
					if err := h.OnText(choice.Delta.Content); err != nil {
						return finish(), err
					}
				}
			}
			for _, delta := range choice.Delta.ToolCalls {
				call := calls[delta.Index]
				if call == nil {
					call = &ToolCall{}
					calls[delta.Index] = call
				}
				if delta.ID != "" {
					call.ID = delta.ID
				}
				if delta.Function.Name != "" {
					call.Function.Name = delta.Function.Name
				}
				call.Function.Arguments += delta.Function.Arguments
			}
			if choice.FinishReason != nil {
				result.FinishReason = *choice.FinishReason
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return finish(), fmt.Errorf("read deepseek stream: %w", err)
	}
	return finish(), nil
}
