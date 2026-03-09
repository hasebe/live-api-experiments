using backend_dotnet.Services;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddOpenApi();
builder.Services.AddTransient<WebSocketHandler>();

var app = builder.Build();

// Configure the HTTP request pipeline.
if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseWebSockets();

app.MapGet("/ws", async (HttpContext context, WebSocketHandler handler) =>
{
    if (context.WebSockets.IsWebSocketRequest)
    {
        string ragProtocol = context.Request.Query["rag_protocol"].ToString();
        if (string.IsNullOrEmpty(ragProtocol))
        {
            ragProtocol = "grpc";
        }

        using var ws = await context.WebSockets.AcceptWebSocketAsync();
        await handler.Handle(ws, context.RequestAborted, ragProtocol);
    }
    else
    {
        context.Response.StatusCode = StatusCodes.Status400BadRequest;
    }
});

app.Run();


