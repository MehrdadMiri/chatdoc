package core

import (
	"context"

	"waitroom-chatbot/internal/llm"
	"waitroom-chatbot/pkg"
)

// ChatService orchestrates patient chat with an LLM backend.
// It builds a Persian system prompt and passes recent transcript
// (mapped to OpenAI-style roles) plus the latest user message.
type ChatService struct {
	LLM llm.Client
}

// NewChatService constructs a new ChatService with the given LLM client.
func NewChatService(client llm.Client) *ChatService {
	return &ChatService{LLM: client}
}

// Reply is kept for backward compatibility; it delegates to ReplyWithContext
// with no history.
func (s *ChatService) Reply(ctx context.Context, nationalID string, message string) (string, error) {
	return s.ReplyWithContext(ctx, nationalID, message, nil)
}

// ReplyWithContext generates a reply using the last week's transcript provided
// by the caller (history). The history should be in chronological order.
func (s *ChatService) ReplyWithContext(ctx context.Context, nationalID, lastUserMsg string, history []pkg.Message) (string, error) {
	var msgs []llm.Message

	// System prompt (Persian) guiding tone & behavior.
	msgs = append(msgs, llm.Message{Role: "system", Content: SystemPrompt})

	// Add prior transcript as alternating user/assistant messages.
	for _, m := range history {
		role := "user"
		if m.Role == pkg.RoleBot {
			role = "assistant"
		}
		msgs = append(msgs, llm.Message{Role: role, Content: m.Content})
	}

	// Current patient message last.
	msgs = append(msgs, llm.Message{Role: "user", Content: lastUserMsg})

	// Delegate to LLM. On error we return it so the HTTP handler can surface
	// a proper 502 and the UI can show an error bubble.
	return s.LLM.Chat(ctx, msgs)
}
