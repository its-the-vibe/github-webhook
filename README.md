# github-webhook

A simple web service which consumes GitHub webhook push notifications and publishes to Redis pub/sub

## Features

- Receives and parses GitHub webhook POST requests
- Verifies GitHub webhook signatures using HMAC SHA256
- Publishes webhook payloads to Redis pub/sub
- Configurable port via environment variable
- Configurable Redis connection via environment variables
- Docker and Docker Compose support for easy deployment
- Logs formatted JSON payloads to console

## Configuration

### Port Configuration

The server port can be configured via the `PORT` environment variable. If not set, it defaults to `8080`.

```bash
# Run on default port 8080
./webhook-server

# Run on custom port
PORT=3000 ./webhook-server
```

### Redis Configuration

The webhook service can publish received webhooks to a Redis pub/sub channel. This is useful for integrating with other services that subscribe to the Redis channel.

**Environment Variables:**

- `REDIS_HOST`: Redis server hostname (default: `localhost`)
- `REDIS_PORT`: Redis server port (default: `6379`)
- `REDIS_CHANNEL`: Redis pub/sub channel name (default: `github-webhook`)

**Note:** If the Redis connection fails, the application will log a warning and continue to work without Redis publishing. This ensures the webhook service remains operational even if Redis is unavailable.

```bash
# Run with Redis configuration
REDIS_HOST=redis.example.com REDIS_PORT=6379 REDIS_CHANNEL=my-webhooks ./webhook-server

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

# Run the server
./webhook-server

# Run with custom port
PORT=3000 ./webhook-server

# Run with Redis configuration
REDIS_HOST=redis.example.com REDIS_PORT=6379 REDIS_CHANNEL=my-webhooks ./webhook-server
```

### Using Docker

```bash
# Build the Docker image
docker build -t github-webhook .

# Run the container
docker run -p 8080:8080 github-webhook

# Run with custom port
docker run -p 3000:8080 -e PORT=8080 github-webhook

# Run with secret file
docker run -p 8080:8080 -v $(pwd)/.secret:/app/.secret:ro github-webhook

# Run with Redis configuration (connecting to Redis on host machine)
docker run -p 8080:8080 -e REDIS_HOST=host.docker.internal -e REDIS_PORT=6379 github-webhook
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

To use a custom port with Docker Compose, set the `PORT` environment variable:

```bash
PORT=3000 docker-compose up -d
```

To configure Redis connection with Docker Compose, you can set environment variables:

```bash
REDIS_HOST=192.168.1.100 REDIS_PORT=6379 REDIS_CHANNEL=my-webhooks docker-compose up -d
```

The docker-compose configuration automatically mounts the `.secret` file if it exists.

## API Endpoints

### POST /webhook

Accepts GitHub webhook notifications and displays the JSON payload.

**Headers:**
- `X-Hub-Signature-256`: GitHub webhook signature (verified if secret is configured)

**Response:**
- `200 OK`: Webhook received and processed successfully
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
# Without signature (works only if no .secret file exists)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# With signature (requires .secret file)
# Generate signature: echo -n '{"test":"data"}' | openssl dgst -sha256 -hmac "your-secret"
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
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

