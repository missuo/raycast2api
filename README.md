# Raycast to OpenAI API

- [Setup](#setup)
- [Usage](#usage)
- [Available Models](#available-models)

This project provides a relay server that allows you to use Raycast AI models through an OpenAI-compatible API interface, implemented in Go with Gin framework.

## Deployment

### Prerequisites

- [Go 1.24.2](https://golang.org/dl/)

### Setup

1. Clone this repository

```bash
git clone https://github.com/missuo/raycast2api
cd raycast2api
```

2. Set environment variables:

```bash
# Set Raycast Bearer Token
export RAYCAST_BEARER_TOKEN=your_raycast_bearer_token

# Optional: Set API Key for authentication
export API_KEY=your_optional_api_key

# Optional: Set custom port (default is 8080)
export PORT=8080
```

3. Build and run the application:

```bash
# Download dependencies
go mod tidy

# Build the application
go build

# Run the application
./raycast-relay
```

### Docker Deployment

```bash
# Clone the repository
git clone https://github.com/missuo/raycast2api
cd raycast2api

# Run Docker container
docker compose up -d --build
```

## Usage

Once deployed, you can use the server as an OpenAI-compatible API endpoint:

```
http://localhost:8080/v1
```

### API Endpoints

| Endpoint | Method | Description |
|:---------|:-------|:------------|
| `/v1/models` | GET | List available models |
| `/v1/chat/completions` | POST | Create a chat completion |
| `/v1/refresh-models` | GET | Manually refresh model cache |
| `/health` | GET | Health check endpoint |

### Authentication

If you've set an `API_KEY`, include it in your requests:

```
Authorization: Bearer your-api-key
```

## Use with Cursor

Unlike the previous version, this Go implementation works seamlessly with Cursor:

1. Use the generated endpoint (e.g., `http://localhost:8080/v1`)
2. No additional configuration needed
3. Manually add the models you want to use to the `Models Name` section in Cursor

## Available Models

Here's a list of all the model IDs:

- ray1
- ray1-mini
- gpt-4
- gpt-4-turbo
- gpt-4o
- gpt-4o-mini
- o1-preview
- o1-mini
- o1-2024-12-17
- o3-mini
- claude-3-5-haiku-latest
- claude-3-5-sonnet-latest
- claude-3-7-sonnet-latest
- claude-3-opus-20240229
- sonar
- sonar-pro
- sonar-reasoning
- sonar-reasoning-pro
- llama-3.3-70b-versatile
- llama-3.1-8b-instant
- llama3-70b-8192
- meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo
- open-mistral-nemo
- mistral-large-latest
- mistral-small-latest
- codestral-latest
- deepseek-r1-distill-llama-70b
- gemini-1.5-flash
- gemini-1.5-pro
- gemini-2.0-flash
- gemini-2.0-flash-thinking-exp-01-21
- deepseek-ai/DeepSeek-R1
- grok-2-latest

You can view the full list by calling the `/v1/models` endpoint.

## Configuration

Configuration is managed through environment variables:

| Variable | Description | Default |
|:---------|:------------|:--------|
| `RAYCAST_BEARER_TOKEN` | **Required** Raycast API token | None |
| `API_KEY` | Optional authentication key | None |
| `PORT` | Server listening port | `8080` |

## Special Thanks

Thanks to [Charles](https://github.com/szcharlesji) for developing the original [raycast-relay](https://github.com/szcharlesji/raycast-relay) and providing the initial implementation.

## License

MIT License