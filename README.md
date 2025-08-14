# Patient Waitroom Chatbot (MVP)

This repository contains a minimal scaffold for the **Patient Waitroom Chatbot**
described in the accompanying technical design document.  The goal of the
project is to collect a patient’s chief complaint and history via free text
messages, produce a live updating summary for the clinician, and persist all
data in PostgreSQL.  The code here is intentionally kept lightweight and
annotated to make it easier to evolve into a full implementation during
subsequent development sprints.

## Project layout

The repository loosely follows a standard Go application layout:

```
waitroom-chatbot/
├── cmd/
│   └── server/        # entry point for the HTTP server
├── internal/
│   ├── http/          # HTTP handlers and templates
│   │   ├── handlers.go
│   │   ├── sse.go
│   │   └── templates/
│   │       ├── doctor.html
│   │       └── patient.html
│   ├── core/          # chat orchestration and summarisation stubs
│   │   ├── chat.go
│   │   ├── summarize.go
│   │   └── prompts.go
│   ├── db/            # database schema and repository layer
│   │   ├── schema.sql
│   │   ├── repo.go
│   │   └── notify.go
│   └── llm/           # wrappers around the OpenAI API (placeholder)
│       └── openai.go
├── pkg/
│   └── types.go       # shared data structures
├── migrations/
│   └── 001_initial.sql
├── .env.example       # environment variables to configure the app
└── Makefile           # common build and run targets
```

### How to run the server

1. **Install dependencies**: This project uses Go modules.  Ensure you have
   Go 1.20+ installed, then run:

   ```bash
   cd waitroom-chatbot
   go mod tidy
   ```

2. **Configure the environment**: Copy `.env.example` to `.env` and fill in
   your database URL and OpenAI API key.  A sample `.env.example` is provided.

3. **Run the server**: Use the Makefile to build and run the server:

   ```bash
   make run
   ```

   This will start an HTTP server on `:8080` by default.

4. **Apply database migrations**: The `migrations/001_initial.sql` file contains
   the SQL required to create the initial tables.  You can apply this manually
   or integrate it into your preferred migration tool.

### Why Server‑Sent Events (SSE)?

The doctor dashboard displays a live summary that updates as the patient
continues the conversation.  Server‑Sent Events (SSE) provide a simple
unidirectional streaming mechanism that allows the server to push messages to
connected clients over a single HTTP connection.  Compared to WebSockets,
SSEs are easier to implement and are supported natively in modern browsers.
Articles on SSE note that it uses a single connection and automatically
reconnects if the connection is lost【774445465098668†L29-L52】.  Because the
doctor view only needs one‑way updates, SSEs are sufficient and avoid the
complexity of full duplex WebSockets.

The HTMX library has first‑class support for SSE.  With the `hx-sse` attribute
in your HTML, HTMX can automatically establish a connection to an SSE endpoint
and swap the server‑sent content into the page.  For example, an element can
use `hx-sse="connect:/event_stream swap:eventName"` to connect to a stream
and update itself when events named `eventName` arrive【666865936018005†L64-L72】.

### Note

This scaffold provides only minimal functionality to get the project off the
ground.  Almost every layer (database, LLM orchestration, summarisation,
templates) contains TODOs to be expanded.  Refer back to the technical
specification to guide further development.