package tools

import (
	"context"
	"fmt"
	"log"
	"os"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/genai"
)

// SearchResult represents the search result to be returned to the model
type SearchResult struct {
	Contexts []struct {
		Text      string  `json:"text"`
		SourceURI string  `json:"source_uri"`
		Distance  float64 `json:"distance"` // Similarity distance
	} `json:"contexts"`
}

var RagTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name: "search_zero_trust_docs",
			Description: `Retrieves information from the internal knowledge base specifically related to Zero Trust Architecture.
				
IMPORTANT: When generating the 'query' argument, follow these rules:
1. Reformulate the prompt to a concise, fully specified and context-independent query.
2. Include time information to the query if the prompt is time sensitive.
3. Include location information to the query if the prompt is location sensitive.
4. Never ask for clarification.
`,
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"query": {
						Type:        genai.TypeString,
						Description: "The reformulated search query for Zero Trust information.",
					},
				},
				Required: []string{"query"},
			},
		},
	},
}

// searchZeroTrustDocs directly calls the Vertex AI RAG Engine
func searchZeroTrustDocs(ctx context.Context, query string) (*SearchResult, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

	// Fallback mechanism for separated Live API and RAG API locations
	location := os.Getenv("RAG_LOCATION")
	if location == "" {
		location = os.Getenv("GOOGLE_CLOUD_LOCATION")
	}

	ragCorpusID := os.Getenv("RAG_CORPUS_ID")
	if ragCorpusID == "" {
		return nil, fmt.Errorf("RAG_CORPUS_ID is not set")
	}

	apiEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
	client, err := aiplatform.NewVertexRagClient(ctx, option.WithEndpoint(apiEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create rag client: %w", err)
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	ragCorpusName := fmt.Sprintf("%s/ragCorpora/%s", parent, ragCorpusID)

	req := &aiplatformpb.RetrieveContextsRequest{
		Parent: parent,
		DataSource: &aiplatformpb.RetrieveContextsRequest_VertexRagStore_{
			VertexRagStore: &aiplatformpb.RetrieveContextsRequest_VertexRagStore{
				RagResources: []*aiplatformpb.RetrieveContextsRequest_VertexRagStore_RagResource{
					{RagCorpus: ragCorpusName},
				},
			},
		},
		Query: &aiplatformpb.RagQuery{
			Query: &aiplatformpb.RagQuery_Text{
				Text: query,
			},
			RagRetrievalConfig: &aiplatformpb.RagRetrievalConfig{
				TopK: 5,
				Filter: &aiplatformpb.RagRetrievalConfig_Filter{
					VectorDbThreshold: &aiplatformpb.RagRetrievalConfig_Filter_VectorDistanceThreshold{
						VectorDistanceThreshold: 0.5,
					},
				},
			},
		},
	}

	resp, err := client.RetrieveContexts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve contexts: %w", err)
	}

	result := &SearchResult{}
	for _, ctx := range resp.Contexts.Contexts {
		dist := 0.0
		if ctx.Score != nil {
			dist = *ctx.Score
		}
		result.Contexts = append(result.Contexts, struct {
			Text      string  `json:"text"`
			SourceURI string  `json:"source_uri"`
			Distance  float64 `json:"distance"`
		}{
			Text:      ctx.Text,
			SourceURI: ctx.SourceUri,
			Distance:  dist,
		})
	}

	return result, nil
}

// HandleSearchZeroTrustDocs executes the logic for the RAG tool
func HandleSearchZeroTrustDocs(ctx context.Context, args map[string]any) map[string]any {
	query, ok := args["query"].(string)
	if !ok {
		return map[string]any{"error": "query argument is required and must be a string"}
	}

	log.Printf("Executing tool search_zero_trust_docs with query: %s", query)

	searchResult, err := searchZeroTrustDocs(ctx, query)
	if err != nil {
		log.Printf("Search Error: %v", err)
		return map[string]any{"error": err.Error()}
	}
	log.Printf("Found %d documents from RAG.", len(searchResult.Contexts))

	// Simple transformation from Struct to map[string]any
	return map[string]any{
		"contexts": searchResult.Contexts,
	}
}
