package llm

import (
	"context"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// Client defines the methods required by the chat and summariser.  An
// implementation could call the OpenAI API or any other LLM provider.
type Client interface {
	Chat(ctx context.Context, prompt string) (string, error)
	Summarize(ctx context.Context, prompt string) (string, error)
}

// OpenAIClient calls the OpenAI API for chat and summarisation responses.
// API credentials and model names are loaded from environment variables.
type OpenAIClient struct {
	client       *openai.Client
	chatModel    string
	summaryModel string
}

// NewOpenAIClient constructs an OpenAI-backed LLM client.  It reads the API key
// and model names from the environment and falls back to sensible defaults.
func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	c := openai.NewClient(apiKey)

	chatModel := os.Getenv("OPENAI_MODEL_CHAT")
	if chatModel == "" {
		chatModel = openai.GPT3Dot5Turbo
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

// Chat sends the prompt to the OpenAI chat completion API and returns the
// model's response.
func (c *OpenAIClient) Chat(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.chatModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
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
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}
