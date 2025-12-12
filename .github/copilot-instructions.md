# Copilot Instructions for github-webhook

## Project Overview

This is a simple Go web service that consumes GitHub webhook push notifications. The server listens on port 8080 and accepts POST requests to the `/webhook` endpoint, parsing and displaying the JSON payload.

## Building and Running

- **Build**: `go build -o webhook-server`
- **Run**: `./webhook-server` (runs on port 8080)
- **Test**: `go test ./...`

## Code Style and Conventions

- Follow standard Go conventions and idioms
- Use `gofmt` for code formatting
- Keep error handling explicit and clear
- Log errors appropriately using the `log` package
- Use standard library packages where possible

## Project Structure

- `main.go`: Contains the webhook handler and server setup
- `go.mod`: Go module definition
- `README.md`: Project documentation
- `*_test.go`: Test files (follow this naming convention)

## Testing

- Write tests in `*_test.go` files
- Follow table-driven test patterns common in Go
- Test HTTP handlers using `httptest` package

## HTTP Endpoints

- `POST /webhook`: Accepts GitHub webhook notifications and displays the JSON payload
  - Returns `200 OK` with "Webhook received" message
  - Returns `405 Method Not Allowed` for non-POST requests
  - Returns `400 Bad Request` for invalid JSON or read errors

## Security Considerations

- Always validate request methods
- Handle errors gracefully
- Use `defer r.Body.Close()` for explicit resource management
- Log errors for debugging without exposing sensitive information
