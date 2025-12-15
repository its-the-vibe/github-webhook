# github-webhook

A simple web service which consumes GitHub webhook push notifications and publishes to Redis pub/sub

## Features

- Receives and parses GitHub webhook POST requests
- Verifies GitHub webhook signatures using HMAC SHA256
- Event filtering with configuration file support
- Publishes webhook payloads to event-specific Redis pub/sub channels
- Configurable log levels (DEBUG, INFO, WARN, ERROR)
- Configurable port via environment variable
- Configurable Redis connection via environment variables
- Docker and Docker Compose support for easy deployment

## Configuration

### Event Configuration

The webhook server uses a JSON configuration file to map GitHub event types to Redis pub/sub channels. Events not defined in the configuration file are ignored.

**Configuration File Format:**

Create a `config.json` file (or specify a custom path via `CONFIG_FILE` environment variable):

```json
[
  {
    "github-event-type": "push",
    "channel": "github-webhook-push"
  },
  {
    "github-event-type": "pull_request",
    "channel": "github-webhook-pull-request"
  }
]
```

**Environment Variables:**

- `CONFIG_FILE`: Path to the configuration file (default: `config.json`)

The server will examine the `X-GitHub-Event` header in incoming webhook requests and publish to the corresponding Redis channel. If an event type is not configured, the webhook will be acknowledged but not processed.

**Example:**

```bash
# Use default config.json
./webhook-server

# Use custom configuration file
CONFIG_FILE=/path/to/my-config.json ./webhook-server
```

### Log Level Configuration

Control the verbosity of logging with the `LOG_LEVEL` environment variable.

**Available Log Levels:**

- `DEBUG`: Most verbose, includes webhook payloads
- `INFO`: Standard operational messages (default)
- `WARN`: Warning messages only
- `ERROR`: Error messages only

**Environment Variables:**

- `LOG_LEVEL`: Sets the logging level (default: `INFO`)

**Note:** Webhook payloads are only logged when `LOG_LEVEL` is set to `DEBUG`. This prevents sensitive data from appearing in logs during normal operation.

**Example:**

```bash
# Use INFO level (default)
./webhook-server

# Use DEBUG level to see webhook payloads
LOG_LEVEL=DEBUG ./webhook-server

# Use WARN level for minimal logging
LOG_LEVEL=WARN ./webhook-server
```

### Port Configuration

The server port can be configured via the `PORT` environment variable. If not set, it defaults to `8080`.

```bash
# Run on default port 8080
./webhook-server

# Run on custom port
PORT=3000 ./webhook-server
```

### Redis Configuration

The webhook service publishes received webhooks to Redis pub/sub channels based on the event configuration. Each event type is routed to its configured channel.

**Environment Variables:**

- `REDIS_HOST`: Redis server hostname (default: `localhost`)
- `REDIS_PORT`: Redis server port (default: `6379`)

**Note:** If the Redis connection fails, the application will log a warning and continue to work without Redis publishing. This ensures the webhook service remains operational even if Redis is unavailable.

```bash
# Run with Redis configuration
REDIS_HOST=redis.example.com REDIS_PORT=6379 ./webhook-server

# Run with default Redis settings (connects to localhost:6379)
./webhook-server
```

### Webhook Secret

To enable GitHub webhook signature verification:

1. Create a `.secret` file in the application directory
2. Add your GitHub webhook secret to this file (the same secret you configured in GitHub)
3. The application will automatically load this secret on startup

**Note:** If the `.secret` file is not found, the application will start but webhook signature verification will be skipped (with a warning logged).

#### Setting up GitHub Webhook Secret

1. Go to your GitHub repository settings
2. Navigate to Webhooks → Add webhook (or edit existing webhook)
3. Set the Payload URL to your server's `/webhook` endpoint
4. Choose `application/json` as the content type
5. Set a secret token (you'll use this in your `.secret` file)
6. Select the events you want to receive

Example `.secret` file:
```
your-secret-token-here
```

**Security:** The `.secret` file is excluded from version control via `.gitignore`.

## Building and Running

### Local Development

```bash
# Build the application
go build -o webhook-server

# Run the server (requires config.json)
./webhook-server

# Run with custom configuration file
CONFIG_FILE=my-config.json ./webhook-server

# Run with custom port
PORT=3000 ./webhook-server

# Run with DEBUG logging
LOG_LEVEL=DEBUG ./webhook-server

# Run with Redis configuration
REDIS_HOST=redis.example.com REDIS_PORT=6379 ./webhook-server

# Run with all options
LOG_LEVEL=DEBUG CONFIG_FILE=config.json REDIS_HOST=localhost PORT=8080 ./webhook-server
```

### Using Docker

```bash
# Build the Docker image
docker build -t github-webhook .

# Run the container (mount config.json)
docker run -p 8080:8080 -v $(pwd)/config.json:/app/config.json:ro github-webhook

# Run with custom log level
docker run -p 8080:8080 -e LOG_LEVEL=DEBUG -v $(pwd)/config.json:/app/config.json:ro github-webhook

# Run with custom port
docker run -p 3000:8080 -e PORT=8080 -v $(pwd)/config.json:/app/config.json:ro github-webhook

# Run with secret file
docker run -p 8080:8080 -v $(pwd)/.secret:/app/.secret:ro -v $(pwd)/config.json:/app/config.json:ro github-webhook

# Run with Redis configuration (connecting to Redis on host machine)
docker run -p 8080:8080 -e REDIS_HOST=host.docker.internal -e REDIS_PORT=6379 -v $(pwd)/config.json:/app/config.json:ro github-webhook
```

### Using Docker Compose

The easiest way to run the application:

```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down
```

To use custom configuration with Docker Compose, you can set environment variables:

```bash
# Custom port
PORT=3000 docker-compose up -d

# Custom log level
LOG_LEVEL=DEBUG docker-compose up -d

# Custom configuration file
CONFIG_FILE=/path/to/config.json docker-compose up -d

# Redis configuration
REDIS_HOST=192.168.1.100 REDIS_PORT=6379 docker-compose up -d
```

The docker-compose configuration automatically mounts the `.secret` and `config.json` files if they exist.

## API Endpoints

### POST /webhook

Accepts GitHub webhook notifications and displays the JSON payload.

### POST /webhook

Accepts GitHub webhook notifications. The event type is determined from the `X-GitHub-Event` header and routed to the corresponding Redis channel based on the configuration file.

**Headers:**
- `X-GitHub-Event`: GitHub event type (e.g., "push", "pull_request") - **required**
- `X-Hub-Signature-256`: GitHub webhook signature (verified if secret is configured)

**Response:**
- `200 OK`: Webhook received and processed successfully
- `200 OK` (with message): Webhook received but event type not configured (event ignored)
- `401 Unauthorized`: Invalid webhook signature
- `405 Method Not Allowed`: Non-POST request
- `400 Bad Request`: Invalid JSON or request body error

## Testing

```bash
# Run all tests
go test ./...
```

### Manual Testing with curl

```bash
# Test with a configured event type (push)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{"test": "data", "repository": {"name": "test-repo"}}'

# Test with an unconfigured event type (will be ignored)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: issues" \
  -d '{"test": "data"}'

# With signature (requires .secret file)
# Generate signature: echo -n '{"test":"data"}' | openssl dgst -sha256 -hmac "your-secret"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=<computed-hash>" \
  -d '{"test":"data"}'
```

## Development

The project follows standard Go conventions:
- Use `gofmt` for code formatting
- Explicit error handling
- Standard library packages preferred

## License

MIT

