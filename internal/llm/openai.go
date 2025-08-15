package llm

import (
	"context"
	"errors"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// Message is a minimal chat message used by the core chat service.
// Role must be one of: "system", "user", or "assistant".
type Message struct {
	Role    string
	Content string
}

// Client defines the methods required by the chat and summariser.
// Chat accepts the full message history (system + prior turns + latest user).
type Client interface {
	Chat(ctx context.Context, messages []Message) (string, error)
	Summarize(ctx context.Context, prompt string) (string, error)
}

// OpenAIClient calls the OpenAI API for chat and summarisation responses.
// API credentials and model names are loaded from environment variables.
type OpenAIClient struct {
	client       *openai.Client
	chatModel    string
	summaryModel string
}

// NewOpenAIClient constructs an OpenAI-backed LLM client. It reads the API key
// and model names from the environment and falls back to sensible defaults.
func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	c := openai.NewClient(apiKey)

	chatModel := os.Getenv("OPENAI_MODEL_CHAT")
	if chatModel == "" {
		// default to a modern small model; can be overridden via env
		chatModel = "gpt-4o-mini"
	}
	summaryModel := os.Getenv("OPENAI_MODEL_SUMMARY")
	if summaryModel == "" {
		summaryModel = chatModel
	}

	return &OpenAIClient{
		client:       c,
		chatModel:    chatModel,
		summaryModel: summaryModel,
	}
}

// Chat sends the message history to the OpenAI chat completion API and returns
// the assistant's response.
func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (string, error) {
	if c.client == nil {
		return "", errors.New("openai client not initialized")
	}

	// Convert to OpenAI message type
	oaMsgs := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		role := m.Role
		if role != openai.ChatMessageRoleSystem && role != openai.ChatMessageRoleUser && role != openai.ChatMessageRoleAssistant {
			// coerce anything unknown to user
			role = openai.ChatMessageRoleUser
		}
		oaMsgs = append(oaMsgs, openai.ChatCompletionMessage{Role: role, Content: m.Content})
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.chatModel,
		Messages:    oaMsgs,
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}

// Summarize generates a short summary of the prompt using the OpenAI API.
func (c *OpenAIClient) Summarize(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.summaryModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "Summarize the following in Persian:"},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}
