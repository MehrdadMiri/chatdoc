package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"waitroom-chatbot/internal/core"
	"waitroom-chatbot/internal/db"
	httpserver "waitroom-chatbot/internal/http"
	"waitroom-chatbot/internal/llm"

	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}
	// Default message cap is 50
	capStr := os.Getenv("MESSAGE_CAP")
	messageCap := 50
	if capStr != "" {
		if v, err := strconv.Atoi(capStr); err == nil {
			messageCap = v
		}
	}
	// Open database connection
	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dbConn.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	if err := db.Migrate(context.Background(), dbConn); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	repo := db.NewRepository(dbConn)
	// Initialize OpenAI LLM client (uses env: OPENAI_API_KEY, OPENAI_MODEL_CHAT)
	llmClient := llm.NewOpenAIClient()
	chatService := core.NewChatService(llmClient)
	// Create HTTP server
	srv, err := httpserver.NewServer(repo, chatService, messageCap)
	if err != nil {
		log.Fatalf("failed to construct server: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
