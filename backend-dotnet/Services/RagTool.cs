using System;
using System.Collections.Generic;
using System.Text.Json.Serialization;
using System.Threading.Tasks;

using AIPlatform = Google.Cloud.AIPlatform.V1;
using GenAITypes = Google.GenAI.Types;

namespace backend_dotnet.Services;

public class SearchResult
{
    [JsonPropertyName("contexts")]
    public List<ContextItem> Contexts { get; set; } = new();

    public class ContextItem
    {
        [JsonPropertyName("text")]
        public string Text { get; set; } = string.Empty;

        [JsonPropertyName("source_uri")]
        public string SourceUri { get; set; } = string.Empty;

        [JsonPropertyName("distance")]
        public double Distance { get; set; }
    }
}

public static class RagTool
{
    public static GenAITypes.Tool Tool = new GenAITypes.Tool
    {
        FunctionDeclarations = new List<GenAITypes.FunctionDeclaration>
        {
            new GenAITypes.FunctionDeclaration
            {
                Name = "search_zero_trust_docs",
                Description = @"Retrieves information from the internal knowledge base specifically related to Zero Trust Architecture.

IMPORTANT: When generating the 'query' argument, follow these rules:
1. Reformulate the prompt to a concise, fully specified and context-independent query.
2. Include time information to the query if the prompt is time sensitive.
3. Include location information to the query if the prompt is location sensitive.
4. Never ask for clarification.",
                Parameters = new GenAITypes.Schema
                {
                    Type = GenAITypes.Type.Object,
                    Properties = new Dictionary<string, GenAITypes.Schema>
                    {
                        {
                            "query", new GenAITypes.Schema
                            {
                                Type = GenAITypes.Type.String,
                                Description = "The reformulated search query for Zero Trust information."
                            }
                        }
                    },
                    Required = new List<string> { "query" }
                }
            }
        }
    };

    public static async Task<Dictionary<string, object>> HandleSearchZeroTrustDocsAsync(Dictionary<string, object> args)
    {
        if (args == null || !args.ContainsKey("query") || args["query"] == null)
        {
            return new Dictionary<string, object> { { "error", "query argument is required and must be a string" } };
        }

        string query = args["query"].ToString() ?? string.Empty;
        Console.WriteLine($"[RagTool] Executing tool search_zero_trust_docs with query: {query}");

        try
        {
            var searchResult = await SearchZeroTrustDocsAsync(query);
            Console.WriteLine($"[RagTool] Found {searchResult.Contexts.Count} documents from RAG.");
            
            return new Dictionary<string, object>
            {
                { "contexts", searchResult.Contexts }
            };
        }
        catch (Exception ex)
        {
            Console.WriteLine($"[RagTool] Search Error: {ex.Message}");
            return new Dictionary<string, object> { { "error", ex.Message } };
        }
    }

    private static async Task<SearchResult> SearchZeroTrustDocsAsync(string queryText)
    {
        string? projectId = Environment.GetEnvironmentVariable("GOOGLE_CLOUD_PROJECT");
        string? location = Environment.GetEnvironmentVariable("RAG_LOCATION");
        if (string.IsNullOrEmpty(location))
        {
            location = Environment.GetEnvironmentVariable("GOOGLE_CLOUD_LOCATION");
        }
        string? ragCorpusId = Environment.GetEnvironmentVariable("RAG_CORPUS_ID");

        if (string.IsNullOrEmpty(projectId) || string.IsNullOrEmpty(location) || string.IsNullOrEmpty(ragCorpusId))
        {
            throw new Exception("Missing RAG environment variables (GOOGLE_CLOUD_PROJECT, RAG_LOCATION/GOOGLE_CLOUD_LOCATION, or RAG_CORPUS_ID).");
        }

        var builder = new AIPlatform.VertexRagServiceClientBuilder
        {
            Endpoint = $"{location}-aiplatform.googleapis.com:443"
        };
        var ragClient = await builder.BuildAsync();

        string parent = $"projects/{projectId}/locations/{location}";
        string ragCorpusName = $"{parent}/ragCorpora/{ragCorpusId}";

        var request = new AIPlatform.RetrieveContextsRequest
        {
            Parent = parent,
            VertexRagStore = new AIPlatform.RetrieveContextsRequest.Types.VertexRagStore
            {
                RagResources =
                {
                    new AIPlatform.RetrieveContextsRequest.Types.VertexRagStore.Types.RagResource { RagCorpus = ragCorpusName }
                }
            },
            Query = new AIPlatform.RagQuery
            {
                Text = queryText,
                RagRetrievalConfig = new AIPlatform.RagRetrievalConfig
                {
                    TopK = 5,
                    Filter = new AIPlatform.RagRetrievalConfig.Types.Filter
                    {
                        VectorDistanceThreshold = 0.5
                    }
                }
            }
        };

        var response = await ragClient.RetrieveContextsAsync(request);

        var result = new SearchResult();
        if (response.Contexts != null)
        {
            foreach (var ctx in response.Contexts.Contexts)
            {
                result.Contexts.Add(new SearchResult.ContextItem
                {
                    Text = ctx.Text,
                    SourceUri = ctx.SourceUri,
                    Distance = ctx.Score
                });
            }
        }

        return result;
    }
}
