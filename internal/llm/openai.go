package llm

import (
    "context"
)

// Client defines the methods required by the chat and summariser.  An
// implementation could call the OpenAI API or any other LLM provider.
type Client interface {
    Chat(ctx context.Context, prompt string) (string, error)
    Summarize(ctx context.Context, prompt string) (string, error)
}

// OpenAIClient is a stub implementation of Client that always returns
// deterministic responses.  Replace this with real calls to the OpenAI API in
// production.  The API key and model names would normally be supplied via
// environment variables.
type OpenAIClient struct{}

// NewOpenAIClient constructs a stub LLM client.
func NewOpenAIClient() *OpenAIClient {
    return &OpenAIClient{}
}

// Chat returns a canned response in Persian based on the prompt.  This stub
// simply acknowledges the patient message.
func (c *OpenAIClient) Chat(ctx context.Context, prompt string) (string, error) {
    return "متشکرم از توضیحات شما. لطفاً بیشتر توضیح دهید.", nil
}

// Summarize returns a canned summary.  It returns the prompt truncated to
// 120 characters.  Replace this with a call to your LLM summarisation model.
func (c *OpenAIClient) Summarize(ctx context.Context, prompt string) (string, error) {
    // Return the last 120 runes of the prompt as a fake summary.
    runes := []rune(prompt)
    if len(runes) > 120 {
        runes = runes[:120]
    }
    return string(runes), nil
}