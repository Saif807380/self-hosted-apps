// Package llm is Trove's one door to a language model: a single call that
// returns JSON matching a schema. Switching provider means changing only
// this file.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

const (
	// Model is Claude Sonnet 5.5: strong judgement at a fraction of Opus's
	// price, which matters for a call made once per game.
	Model     = "claude-sonnet-5-5"
	maxTokens = 16000
)

// ErrRefused means the model declined the request (a safety classifier),
// even after the server-side fallback.
var ErrRefused = errors.New("llm: request refused")

// Client calls Claude. A nil *Client means no API key is configured.
type Client struct {
	api anthropic.Client
	log *slog.Logger
}

// New returns a client, or nil when apiKey is empty.
func New(apiKey string, log *slog.Logger) *Client {
	if apiKey == "" {
		return nil
	}
	return &Client{api: anthropic.NewClient(option.WithAPIKey(apiKey)), log: log}
}

// CompleteJSON asks for a reply matching schema (a JSON Schema object) and
// decodes it into out. It returns the model that answered.
func (c *Client) CompleteJSON(ctx context.Context, system, user string, schema map[string]any, out any) (string, error) {
	resp, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     Model,
		MaxTokens: maxTokens,
		System:    []anthropic.BetaTextBlockParam{{Text: system}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user))},
		OutputConfig: anthropic.BetaOutputConfigParam{
			// Weighing a page of evidence doesn't need deep thought.
			Effort: anthropic.BetaOutputConfigEffortMedium,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: schema},
		},
		// A declined request is retried on Anthropic's recommended fallback
		// model inside the same call, instead of just failing.
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks: anthropic.BetaFallbacksParamOfDefault(),
	})
	if err != nil {
		return "", fmt.Errorf("llm call: %w", err)
	}
	c.log.Info("llm call", "model", resp.Model, "stop", resp.StopReason,
		"input_tokens", resp.Usage.InputTokens, "output_tokens", resp.Usage.OutputTokens)

	switch resp.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return "", fmt.Errorf("%w (%s)", ErrRefused, resp.StopDetails.Category)
	case anthropic.BetaStopReasonMaxTokens:
		return "", errors.New("llm: reply cut off at max_tokens")
	}
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			if err := json.Unmarshal([]byte(text.Text), out); err != nil {
				return "", fmt.Errorf("llm reply: %w", err)
			}
			return string(resp.Model), nil
		}
	}
	return "", errors.New("llm: reply had no text")
}
