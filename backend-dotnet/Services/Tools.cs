using Google.GenAI.Types;
using System.Collections.Generic;
using System.Text.Json.Nodes;

namespace backend_dotnet.Services;

public static class Tools
{
    public static Tool WeatherTool = new Tool
    {
        FunctionDeclarations = new List<FunctionDeclaration>
        {
            new FunctionDeclaration
            {
                Name = "get_current_weather",
                Description = "Get the current weather in a given location",
                Parameters = new Schema
                {
                    Type = Google.GenAI.Types.Type.Object,
                    Properties = new Dictionary<string, Schema>
                    {
                        ["location"] = new Schema
                        {
                            Type = Google.GenAI.Types.Type.String,
                            Description = "The city and state, e.g. San Francisco, CA"
                        }
                    },
                    Required = new List<string> { "location" }
                }
            }
        }
    };

    public static Dictionary<string, object> HandleGetCurrentWeather(Dictionary<string, object> args)
    {
        string location = "unknown";
        if (args != null && args.ContainsKey("location") && args["location"] != null)
        {
            location = args["location"].ToString();
        }
        else
        {
            return new Dictionary<string, object> { { "error", "location argument is required and must be a string" } };
        }

        // Mock data - in a real app this would call an external API
        return new Dictionary<string, object>
        {
            { "weather", "Sunny" },
            { "temperature", 25 },
            { "location", location },
            { "note", "This is mock data from the backend tool" }
        };
    }
}
