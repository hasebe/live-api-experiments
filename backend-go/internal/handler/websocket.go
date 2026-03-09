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

	log.Printf("Connecting to Gemini Live API with model: %s", model)
	session, err := h.Client.Connect(ctx, model, []*genai.Tool{tools.WeatherTool, tools.RagTool}, SystemInstruction)
	if err != nil {
		log.Printf("Failed to connect to Gemini: %v", err)
		return
	}
	defer session.Close()

	// Channel for sending messages to Gemini session safely
	// We use a interface{} to handle different types of messages (RealtimeInput, ToolResponse)
	// or we can define a wrapper struct.
	type sessionMsg struct {
		RealtimeInput *genai.LiveRealtimeInput
		ToolResponse  *genai.LiveToolResponseInput
	}
	sendCh := make(chan sessionMsg, 100) // Buffered channel

	// Channel to signal internal errors or completion
	done := make(chan struct{})

	// Dedicated Write Loop for Gemini Session
	go func() {
		defer close(done)
		for msg := range sendCh {
			var err error
			if msg.RealtimeInput != nil {
				err = session.SendRealtimeInput(*msg.RealtimeInput)
			} else if msg.ToolResponse != nil {
				err = session.SendToolResponse(*msg.ToolResponse)
			}

			if err != nil {
				log.Printf("Gemini write error: %v", err)
				return // Exit loop on write error
			}
		}
	}()

	// Goroutine: Gemini -> Client (Read from Gemini)
	go func() {
		// Close sendCh when we stop receiving from Gemini to stop the write loop
		defer func() {
			// recover in case verify sendCh is not closed? 
			// Actually, we should just let the main handler close things or use a separate signal.
			// But for simplicity, if receive fails, we assume session is dead.
		}()
		
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

					// Execute tool asynchronously to not block receive loop
					go func(fc *genai.FunctionCall) {
						// Use tool handler
						var result map[string]any
						switch fc.Name {
						case "get_current_weather":
							result = tools.HandleGetCurrentWeather(fc.Args)
						case "search_zero_trust_docs":
							result = tools.HandleSearchZeroTrustDocs(ctx, fc.Args)
						default:
							result = map[string]any{"error": "Unknown function"}
						}

						// Send response via channel
						resp := &genai.LiveToolResponseInput{
							FunctionResponses: []*genai.FunctionResponse{
								{
									Name:     fc.Name,
									ID:       fc.ID,
									Response: result,
								},
							},
						}
						
						// Non-blocking send or blocking? 
						// Blocking is safer for order preservation, but tool calls are parallelizable.
						// The write loop will serialize them.
						select {
						case sendCh <- sessionMsg{ToolResponse: resp}:
							log.Printf("Sent function response for: %s", fc.Name)
						case <-done:
							log.Printf("Failed to queue function response (closed): %s", fc.Name)
						}
					}(fc)
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
			// msg is *genai.LiveServerMessage
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

	// Main loop: Client -> Gemini (Read from WebSocket)
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
			// Send audio data via channel
			input := &genai.LiveRealtimeInput{
				Media: &genai.Blob{
					MIMEType: "audio/pcm;rate=16000",
					Data:     buf[:n],
				},
			}
			
			select {
			case sendCh <- sessionMsg{RealtimeInput: input}:
			case <-done:
				break
			}
		}

		select {
		case <-done:
			return
		default:
		}
	}
	
	// Cleanup happens via defer
	// Ensure write loop exits
	close(sendCh)
}
