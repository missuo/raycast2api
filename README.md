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

- **RAYCAST_BEARER_TOKEN**: You need to intercept the request sent by Raycast with an Authorization header. You can obtain it by intercepting Raycast app traffic using Proxyman. 

```
I. Open Proxyman, then open Raycast and type any query like 'hello'.

II. In Proxyman, find any request sent by Raycast with an Authorization header.

III. The token is the part after 'Bearer ', e.g., 'Bearer xxxxxxxxxxxxx', where 'xxxxxxxxxxxxx' is your `RAYCAST_BEARER_TOKEN`.
```

- **API_KEY**: It is used for authentication in this project to prevent unauthorized access or abuse. You can define it to be anything you want, e.g., `sk-1234567890`.

3. Build and run the application:

```bash
# Download dependencies
go mod tidy

# Build the application
go build

# Run the application
./raycast2api
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

The following model list was updated on April 8, 2025, at 7:28 PM EDT. 

Some models in the following list still indicate that the model does not exist, even when sending the exact same request as the Raycast App. We suspect that the server has implemented authentication for the signature.

| Model ID | Owner | Availability |
|:---|:---|:---:|
| claude-3-5-haiku-latest | Anthropic | ✅ |
| claude-3-5-sonnet-latest | Anthropic | ✅ |
| claude-3-7-sonnet-latest | Anthropic | ✅ |
| claude-3-7-sonnet-latest-reasoning | Anthropic | ✅ |
| claude-3-opus-20240229 | Anthropic | ✅ |
| codestral-latest | Mistral | ✅ |
| deepseek-ai/DeepSeek-R1 | Together | ✅ |
| deepseek-ai/DeepSeek-V3 | Together | ✅ |
| deepseek-r1-distill-llama-70b | Groq | ✅ |
| gemini-1.5-flash | Google | ✅ |
| gemini-1.5-pro | Google | ✅ |
| gemini-2.0-flash | Google | ✅ |
| gemini-2.0-flash-thinking-exp-01-21 | Google | ✅ |
| gemini-2.5-pro-preview-03-25 | Google | ❌ |
| grok-2-latest | XAI | ✅ |
| gpt-4 | OpenAI | ✅ |
| gpt-4-turbo | OpenAI | ✅ |
| gpt-4o | OpenAI | ✅ |
| gpt-4o-mini | OpenAI | ✅ |
| llama-3.1-8b-instant | Groq | ✅ |
| llama-3.3-70b-versatile | Groq | ✅ |
| llama3-70b-8192 | Groq | ✅ |
| meta-llama/llama-4-scout-17b-16e-instruct | Groq | ✅ |
| meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo | Together | ✅ |
| mistral-large-latest | Mistral | ✅ |
| mistral-small-latest | Mistral | ✅ |
| o1-2024-12-17 | OpenAI O1 | ❌ |
| o1-mini | OpenAI O1 | ❌ |
| o1-preview | OpenAI O1 | ❌ |
| o3-mini | OpenAI O1 | ✅ |
| open-mistral-nemo | Mistral | ✅ |
| qwen-2.5-32b | Groq | ✅ |
| ray1 | Raycast | ✅ |
| ray1-mini | Raycast | ✅ |
| sonar | Perplexity | ✅ |
| sonar-pro | Perplexity | ✅ |
| sonar-reasoning | Perplexity | ✅ |
| sonar-reasoning-pro | Perplexity | ✅ |

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