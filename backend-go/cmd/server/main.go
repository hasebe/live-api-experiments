package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"live-api-demo/internal/gemini"
	"live-api-demo/internal/handler"

	"golang.org/x/net/websocket"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	client, err := gemini.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}

	h := &handler.WebSocketHandler{
		Client: client,
	}

	mux := http.NewServeMux()
	mux.Handle("/ws", websocket.Handler(h.Handle))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Server listening on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
