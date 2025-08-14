## Common development targets for the waitroom-chatbot project

.PHONY: help run build test tidy

help:
	@echo "Makefile targets:"
	@echo "  make run    - run the HTTP server with 'go run'"
	@echo "  make build  - build the server binary"
	@echo "  make test   - run unit tests (none yet)"
	@echo "  make tidy   - tidy up go modules"

run:
	@echo "Starting server on port $${PORT:-8080}"
	@env $(shell if [ -f .env ]; then sed -e '/^$$/d' -e '/^#/d' .env | xargs -I {} echo {} ; fi) go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	@echo "No tests defined yet"

tidy:
	go mod tidy