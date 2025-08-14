package core

import (
    "context"
    "encoding/json"
    "time"

    "waitroom-chatbot/internal/llm"
    "waitroom-chatbot/pkg"
)

// Summarizer coordinates extraction of structured data and free‑text summary from
// a transcript.  It uses the LLM client to perform summarisation and
// extraction.  In the MVP this is a simple stub.
type Summarizer struct {
    LLM llm.Client
}

// NewSummarizer constructs a summariser.
func NewSummarizer(client llm.Client) *Summarizer {
    return &Summarizer{LLM: client}
}

// Summarize analyses the transcript and produces a Summary.  The transcript
// should contain all messages in a session ordered chronologically.  The old
// summary can be passed in to support merging; new non‑empty values
// overwrite previous ones and arrays are deduplicated.  For the MVP, the
// summariser simply echoes the last patient message as free text and leaves
// the structured data empty.
func (s *Summarizer) Summarize(ctx context.Context, sessionID string, transcript []pkg.Message, old *pkg.Summary) (*pkg.Summary, error) {
    // Compose the prompt for the LLM.  In a full implementation you would
    // include the transcript and the existing structured data.  For now we
    // pass only the latest patient message to the stubbed summariser.
    var lastMsg string
    for i := len(transcript) - 1; i >= 0; i-- {
        if transcript[i].Role == pkg.RolePatient {
            lastMsg = transcript[i].Content
            break
        }
    }
    prompt := SummarizationInstruction + "\n\n" + lastMsg
    resp, err := s.LLM.Summarize(ctx, prompt)
    if err != nil {
        // fallback summary when the LLM call fails
        return &pkg.Summary{
            SessionID: sessionID,
            KeyPoints: []string{"گفت‌وگو انجام شد"},
            Structured: map[string]interface{}{},
            FreeText: "خلاصهٔ گفت‌وگو در دسترس نیست.",
            UpdatedAt: time.Now(),
        }, err
    }
    // The stubbed LLM client returns JSON for the structured field followed by
    // free text separated by a delimiter.  Since this is a placeholder, we
    // decode an empty JSON object and use the raw response as free text.
    var structured map[string]interface{}
    if err := json.Unmarshal([]byte("{}"), &structured); err != nil {
        structured = map[string]interface{}{}
    }
    return &pkg.Summary{
        SessionID: sessionID,
        KeyPoints: []string{resp},
        Structured: structured,
        FreeText: resp,
        UpdatedAt: time.Now(),
    }, nil
}