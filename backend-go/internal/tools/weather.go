package tools

import (
	"log"

	"google.golang.org/genai"
)

// WeatherTool definition for Gemini Function Calling
var WeatherTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name:        "get_current_weather",
			Description: "Get the current weather in a given location",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"location": {
						Type:        genai.TypeString,
						Description: "The city and state, e.g. San Francisco, CA",
					},
				},
				Required: []string{"location"},
			},
		},
	},
}

// HandleGetCurrentWeather executes the logic for the weather tool
func HandleGetCurrentWeather(args map[string]any) map[string]any {
	location, ok := args["location"].(string)
	if !ok {
		return map[string]any{"error": "location argument is required and must be a string"}
	}

	log.Printf("Executing tool get_current_weather for location: %s", location)

	// Mock data - in a real app this would call an external API
	return map[string]any{
		"weather":     "Sunny",
		"temperature": 25,
		"location":    location,
		"note":        "This is mock data from the backend tool",
	}
}
