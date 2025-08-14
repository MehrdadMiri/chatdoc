package core

import (
    "context"
    "fmt"

    "waitroom-chatbot/internal/llm"
)

// ChatService orchestrates the chat between a patient and the assistant.  In
// a real implementation this would use the OpenAI API (or another LLM) to
// generate a reply.  The MVP keeps stateful context in the database and
// generates a response for each patient message.
type ChatService struct {
    LLM llm.Client
}

// NewChatService constructs a new ChatService with the given LLM client.
func NewChatService(client llm.Client) *ChatService {
    return &ChatService{LLM: client}
}

// Reply generates a reply in Persian for a patient message.  This is a
// blocking call that delegates to the LLM.  On error a generic fallback
// message is returned.  The sessionID can be used to retrieve previous
// messages from the repository, but that logic belongs in the caller.
func (s *ChatService) Reply(ctx context.Context, sessionID string, message string) (string, error) {
    // Compose the prompt for the LLM.  In a full implementation you would
    // include the entire conversation history and system prompt.  Here we
    // simply pass the system prompt and the user message to a stubbed client.
    prompt := fmt.Sprintf("%s\n\nPatient: %s\n\nAssistant:", SystemPrompt, message)
    resp, err := s.LLM.Chat(ctx, prompt)
    if err != nil {
        // fallback generic response when the LLM call fails
        return "از توضیحات شما متشکرم. لطفاً کمی بیشتر دربارهٔ مشکل خود بگویید.", err
    }
    return resp, nil
}