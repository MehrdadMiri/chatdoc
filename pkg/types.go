package pkg

import "time"

// Session represents a patient visit.  It is keyed by a UUID and
// optionally includes administrative information supplied by the patient.
type Session struct {
    ID           string     `json:"id"`
    CreatedAt    time.Time  `json:"created_at"`
    ClosedAt     *time.Time `json:"closed_at,omitempty"`
    MessageCap   int        `json:"message_cap"`
    PatientPhone *string    `json:"patient_phone,omitempty"`
    PatientID    *string    `json:"patient_national_id,omitempty"`
    ClientIP     *string    `json:"client_ip,omitempty"`
    UserAgent    *string    `json:"user_agent,omitempty"`
}

// MessageRole describes who authored a message.  In the MVP there are only
// two roles: patient and bot.
type MessageRole string

const (
    RolePatient MessageRole = "patient"
    RoleBot     MessageRole = "bot"
)

// Message represents a chat message in a session.
type Message struct {
    ID        int64       `json:"id"`
    SessionID string      `json:"session_id"`
    Role      MessageRole `json:"role"`
    Content   string      `json:"content"`
    CreatedAt time.Time   `json:"created_at"`
}

// Summary holds the doctor‑facing summary for a session.  The structured
// field stores machine‑readable data conforming to the JSON schema in the
// technical specification.  KeyPoints and FreeText are used for the doctor UI.
type Summary struct {
    ID         int64                  `json:"id"`
    SessionID  string                 `json:"session_id"`
    KeyPoints  []string               `json:"key_points"`
    Structured map[string]interface{} `json:"structured"`
    FreeText   string                 `json:"free_text"`
    UpdatedAt  time.Time              `json:"updated_at"`
}

// ChatRequest represents a request to send a message from the patient.
type ChatRequest struct {
    Content string `json:"content"`
}

// ChatResponse contains the bot's reply and whether the session is
// capped due to exceeding the message limit.
type ChatResponse struct {
    Reply string `json:"reply"`
    Capped bool   `json:"capped"`
}

// DoctorSessionPreview is returned in the list of active sessions for the
// doctor dashboard.  It includes a few key points and the last update time.
type DoctorSessionPreview struct {
    SessionID   string    `json:"session_id"`
    KeyPoints   []string  `json:"key_points"`
    UpdatedAt   time.Time `json:"updated_at"`
    LastMessage time.Time `json:"last_message"`
}