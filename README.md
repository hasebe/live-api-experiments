# Live API Demo Application

This project demonstrates the usage of Google Cloud Multimodal Live API using a Go backend and a Next.js (React) frontend.

## Prerequisites

- [Go](https://go.dev/) (1.22+)
- [Bun](https://bun.sh/) (1.0+)
- Google Cloud Project with Vertex AI API enabled.
- `gcloud` CLI authenticated or service account credentials.
- [.NET SDK](https://dotnet.microsoft.com/) (10.0+) - for .NET backend

## Folder Structure

- `backend-go/`: Go server acting as a proxy to Gemini Live API.
- `backend-dotnet/`: .NET server acting as a proxy to Gemini Live API.
- `frontend/`: Next.js web application for recording/playing audio.

## Setup & Run

### 1. Backend (Go)

```bash
cd backend-go
# Install dependencies
go mod tidy

# Run server
# Ensure GOOGLE_CLOUD_PROJECT is set
export GOOGLE_CLOUD_PROJECT="your-project-id"
export GOOGLE_CLOUD_LOCATION="us-central1" # Optional, Live API Location

# Optional: RAG Engine Configuration (for Zero Trust Search Tool, etc)
# If omitted, GOOGLE_CLOUD_PROJECT and GOOGLE_CLOUD_LOCATION will be used as fallbacks.
export RAG_CORPUS_ID="your-rag-corpus-id"
export RAG_LOCATION="us-central1" # e.g. "europe-west3"

go run cmd/server/main.go
```

Server listens on `http://localhost:8080`.

### 2. Backend (.NET)

```bash
cd backend-dotnet
# Restore dependencies
dotnet restore

# Run server
# Ensure GOOGLE_CLOUD_PROJECT is set
export GOOGLE_CLOUD_PROJECT="your-project-id"
export GOOGLE_CLOUD_LOCATION="us-central1" # Optional
dotnet run
```

Server listens on `http://localhost:5093` (default) or check output.

### 3. Frontend

```bash
cd frontend
# Install dependencies
bun install

# Run development server
bun dev
```

App runs on `http://localhost:3000`.

## Architecture

1.  **Browser** captures audio (16kHz PCM) via AudioWorklet.
2.  **Frontend** sends audio chunks via WebSocket to **Backend**.
3.  **Backend** wraps chunks in `LiveRealtimeInput` and forwards to **Gemini Live API** via `google.golang.org/genai` SDK.
4.  **Gemini** responds with audio.
5.  **Backend** forwards response to **Frontend**.
6.  **Frontend** plays audio via Web Audio API.

## Notes

- The demo assumes usage of `gemini-live-2.5-flash-native-audio` or similar model supporting Multimodal Live API.
- Audio is handled as raw PCM for low latency.
