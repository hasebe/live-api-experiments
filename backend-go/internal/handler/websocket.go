package handler

import (
	"encoding/json"
	"io"
	"log"
	"strings"

	"live-api-demo/internal/gemini"
	"live-api-demo/internal/tools"

	"golang.org/x/net/websocket"
	"google.golang.org/genai"
)

type WebSocketHandler struct {
	Client *gemini.Client
}

// Handle manages the WebSocket connection
func (h *WebSocketHandler) Handle(ws *websocket.Conn) {
	defer ws.Close()

	ctx := ws.Request().Context()
	// Use a default model, ensure it's one that supports Multimodal Live API
	model := "gemini-live-2.5-flash-native-audio"
	// model := "gemini-live-2.5-flash-preview-native-audio-09-2025"

	// Define tools
	// Used from internal/tools package

	log.Printf("Connecting to Gemini Live API with model: %s", model)
	session, err := h.Client.Connect(ctx, model, []*genai.Tool{tools.WeatherTool})
	if err != nil {
		log.Printf("Failed to connect to Gemini: %v", err)
		return
	}
	defer session.Close()

	// Channel to signal internal errors or completion
	done := make(chan struct{})

	// Goroutine: Gemini -> Client
	go func() {
		defer close(done)
		for {
			msg, err := session.Receive()
			if err != nil {
				// Ignore error if session is closed or context canceled
				if strings.Contains(err.Error(), "use of closed network connection") || strings.Contains(err.Error(), "context canceled") {
					log.Println("Gemini session closed")
					return
				}
				log.Printf("Gemini receive error: %v", err)
				return
			}

			// Handle Voice Activity messages (New in v1.42.0)
			if msg.VoiceActivity != nil {
				log.Printf("Voice Activity Notification: %+v", msg.VoiceActivity)
				log.Printf("Voice Activity Type: %+v", msg.VoiceActivity.VoiceActivityType)
			}
			if msg.VoiceActivityDetectionSignal != nil {
				log.Printf("Voice Activity Detection Signal (Allowlisted): %+v", msg.VoiceActivityDetectionSignal)
			}

			// Handle Tool Calls (Top-level field in Live API)
			if msg.ToolCall != nil {
				for _, fc := range msg.ToolCall.FunctionCalls {
					log.Printf("Received Tool Call: %s(%v)", fc.Name, fc.Args)

					// Use tool handler
					var result map[string]any
					if fc.Name == "get_current_weather" {
						result = tools.HandleGetCurrentWeather(fc.Args)
					} else {
						result = map[string]any{"error": "Unknown function"}
					}

					// Send response
					err := session.SendToolResponse(genai.LiveToolResponseInput{
						FunctionResponses: []*genai.FunctionResponse{
							{
								Name:     fc.Name,
								ID:       fc.ID,
								Response: result,
							},
						},
					})
					if err != nil {
						log.Printf("Failed to send function response: %v", err)
					} else {
						log.Printf("Sent function response: %v", result)
					}
				}
			}

			// Handle Text/Audio Content
			if msg.ServerContent != nil && msg.ServerContent.ModelTurn != nil {
				for _, part := range msg.ServerContent.ModelTurn.Parts {
					if part.FunctionCall != nil {
						// Fallback: older models might send it here, but unlikely for Live API
						log.Printf("Received FunctionCall in ModelTurn (unexpected): %s", part.FunctionCall.Name)
					}
				}
			}

			// Extract audio parts and send to client
			// Simple approach: forward the raw JSON or just audio?
			// For this demo, let's extract audio if present.
			// msg is *genai.LiveServerMessage

			// We can filter for ServerContent and inline data
			// TODO: Structure this message for the frontend
			// For now, let's just marshal the whole message and send it
			// The frontend can parse it.

			respBytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Marshal error: %v", err)
				continue
			}

			if _, err := ws.Write(respBytes); err != nil {
				log.Printf("Websocket write error: %v", err)
				return
			}
		}
	}()

	// Main loop: Client -> Gemini
	// We expect the client to send audio chunks.
	// For simplicity, assume client sends raw binary audio (PCM 16kHz or 24kHz)
	// OR client sends JSON messages.
	// Let's assume client sends JSON with "audio" field for now, or raw blobs?
	// To keep it robust, let's assume client sends JSON messages:
	// { "audio": "base64..." } or just raw binary if we control frontend.
	// Let's go with raw binary for audio chunks for max efficiency if we can,
	// but standard WebSocket in JS sends Blobs or ArrayBuffers.

	buf := make([]byte, 4096)
	for {
		// Read from Websocket
		n, err := ws.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Websocket read error: %v", err)
			}
			break
		}

		if n > 0 {
			// Assume it's audio data (PCM 16kHz 1 channel linear16 usually expected)
			// We need to wrap it in LiveRealtimeInput

			err = session.SendRealtimeInput(genai.LiveRealtimeInput{
				Media: &genai.Blob{
					MIMEType: "audio/pcm;rate=16000",
					Data:     buf[:n],
				},
			})
			if err != nil {
				log.Printf("Gemini send error: %v", err)
				break
			}
		}

		select {
		case <-done:
			return
		default:
		}
	}
}
