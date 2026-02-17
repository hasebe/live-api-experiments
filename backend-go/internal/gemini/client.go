package gemini

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
}

func NewClient(ctx context.Context) (*Client, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	cfg := &genai.ClientConfig{
		Project:  projectID,
		Location: location,
	}

	c, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &Client{client: c}, nil
}

func (c *Client) Connect(ctx context.Context, model string, tools []*genai.Tool, systemInstruction string) (*genai.Session, error) {
	// Simple config for audio-only for now, plus tools
	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
		Tools:              tools,
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: "Puck",
				},
			},
		},
		ExplicitVADSignal: genai.Ptr(true),
	}

	if systemInstruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{
				{Text: systemInstruction},
			},
		}
	}

	return c.client.Live.Connect(ctx, model, config)
}
