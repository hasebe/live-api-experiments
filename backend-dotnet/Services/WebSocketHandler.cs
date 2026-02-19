using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using Google.GenAI;
using Google.GenAI.Types;
using Microsoft.Extensions.Logging;

namespace backend_dotnet.Services;

public class WebSocketHandler
{
    private readonly ILogger<WebSocketHandler> _logger;

    public WebSocketHandler(ILogger<WebSocketHandler> logger)
    {
        _logger = logger;
    }

    public async Task Handle(WebSocket ws, CancellationToken cancellationToken)
    {
        using var cts = new CancellationTokenSource();
        using var linkedCts = CancellationTokenSource.CreateLinkedTokenSource(cts.Token, cancellationToken);
        
        // Initialize Gemini Client
        // Note: SDK v0.12.0
        var apiKey = System.Environment.GetEnvironmentVariable("GOOGLE_API_KEY") ?? System.Environment.GetEnvironmentVariable("GEMINI_API_KEY");
        
        Google.GenAI.Client client;
        if (!string.IsNullOrEmpty(apiKey))
        {
            client = new Google.GenAI.Client(apiKey: apiKey);
        }
        else
        {
            client = new Google.GenAI.Client();
        }

        var model = "gemini-live-2.5-flash-native-audio";
        _logger.LogInformation($"Connecting to Gemini Live API with model: {model}");

        var config = new LiveConnectConfig
        {
            SystemInstruction = new Content 
            {
                Parts = new List<Part> 
                { 
                    new Part
                    {
                        Text = Instruction.SystemInstruction
                    } 
                }
            },
            ResponseModalities = new List<Modality> { Modality.Audio },
            SpeechConfig = new SpeechConfig
            {
                VoiceConfig = new VoiceConfig
                {
                    PrebuiltVoiceConfig = new PrebuiltVoiceConfig { VoiceName = "Puck" }
                }
            },
            Tools = new List<Google.GenAI.Types.Tool> { backend_dotnet.Services.WeatherTool.Tool, backend_dotnet.Services.RagTool.Tool },
            ExplicitVadSignal = true
        };

        try
        {
            await using var session = await client.Live.ConnectAsync(model, config);
            _logger.LogInformation("Connected to Gemini Live API");

            // Receive Loop (Gemini -> WebSocket)
            var receiveTask = Task.Run(async () =>
            {
                try
                {
                    while (!linkedCts.Token.IsCancellationRequested)
                    {
                        LiveServerMessage serverMessage = await session.ReceiveAsync(linkedCts.Token);
                        if (serverMessage == null) break;

                        if (serverMessage.VoiceActivity != null)
                        {
                            _logger.LogInformation($"Signal: {serverMessage.VoiceActivity}");
                        }
                        if (serverMessage.VoiceActivityDetectionSignal != null)
                        {
                            _logger.LogInformation($"Voice Activity Detection Signal (Allowlisted): {serverMessage.VoiceActivityDetectionSignal}");
                        }

                        if (serverMessage.ToolCall != null)
                        {
                            foreach (var fc in serverMessage.ToolCall.FunctionCalls)
                            {
                                _logger.LogInformation($"Received Tool Call: {fc.Name}");
                                
                                Dictionary<string, object> result;
                                if (fc.Name == "search_zero_trust_docs")
                                {
                                    var args = fc.Args != null ? new Dictionary<string, object>(fc.Args) : new Dictionary<string, object>();
                                    result = await backend_dotnet.Services.RagTool.HandleSearchZeroTrustDocsAsync(args);
                                }
                                else if (fc.Name == "get_current_weather")
                                {
                                    var args = fc.Args != null ? new Dictionary<string, object>(fc.Args) : new Dictionary<string, object>();
                                    result = backend_dotnet.Services.WeatherTool.HandleGetCurrentWeather(args);
                                }
                                else
                                {
                                    result = new Dictionary<string, object> { ["error"] = "Unknown function" };
                                }

                                await session.SendToolResponseAsync(new LiveSendToolResponseParameters
                                {
                                    FunctionResponses = new List<FunctionResponse>
                                    {
                                        new FunctionResponse
                                        {
                                            Name = fc.Name ?? "unknown",
                                            Id = fc.Id ?? "",
                                            Response = result
                                        }
                                    }
                                });
                                _logger.LogInformation($"Sent tool response: {fc.Name}");
                            }
                        }

                        var json = JsonSerializer.Serialize(serverMessage);
                        var bytes = Encoding.UTF8.GetBytes(json);
                        if (ws.State == WebSocketState.Open)
                        {
                            await ws.SendAsync(new ArraySegment<byte>(bytes), WebSocketMessageType.Text, true, linkedCts.Token);
                        }
                    }
                }
                catch (Exception ex)
                {
                    _logger.LogError($"Error in Receive Loop: {ex.Message}");
                    cts.Cancel();
                }
            });

            // Send Loop (WebSocket -> Gemini)
            var buffer = new byte[32 * 1024];
            while (ws.State == WebSocketState.Open && !linkedCts.Token.IsCancellationRequested)
            {
                var result = await ws.ReceiveAsync(new ArraySegment<byte>(buffer), linkedCts.Token);
                
                if (result.MessageType == WebSocketMessageType.Close)
                {
                    await ws.CloseAsync(WebSocketCloseStatus.NormalClosure, "Closed by client", CancellationToken.None);
                    break;
                }

                if (result.Count > 0)
                {
                    var data = new byte[result.Count];
                    Array.Copy(buffer, data, result.Count);

                    await session.SendRealtimeInputAsync(new LiveSendRealtimeInputParameters
                    {
                        Media = new Google.GenAI.Types.Blob
                        {
                            MimeType = "audio/pcm;rate=16000",
                            Data = data
                        }
                    });
                }
            }

            cts.Cancel();
            await receiveTask;
        }
        catch (Exception ex)
        {
            _logger.LogError($"WebSocket Handler Error: {ex.Message}");
        }
    }
}
