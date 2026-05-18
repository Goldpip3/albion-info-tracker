using AlbionInfoTracker.Services;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;
using Serilog;
using System;
using System.IO;
using System.Threading.Tasks;

namespace AlbionInfoTracker;

public static class Program
{
    public static async Task<int> Main(string[] args)
    {
        var logDir = Path.Combine(AppContext.BaseDirectory, "logs");
        Directory.CreateDirectory(logDir);

        Log.Logger = new LoggerConfiguration()
            .MinimumLevel.Debug()
            .WriteTo.Console(restrictedToMinimumLevel: Serilog.Events.LogEventLevel.Information)
            .WriteTo.File(Path.Combine(logDir, "albion-info-tracker-.log"),
                rollingInterval: RollingInterval.Day,
                retainedFileCountLimit: 7)
            .CreateLogger();

        try
        {
            Log.Information("Albion Info Tracker starting");

            var host = Host.CreateDefaultBuilder(args)
                .UseSerilog()
                .ConfigureServices((ctx, services) =>
                {
                    services.AddSingleton<EntityController>();
                    services.AddSingleton<CombatController>();
                    services.AddSingleton<TrackingController>();
                    services.AddSingleton<ItemController>();
                    // Order matters: broadcaster subscribes to controller events before
                    // tracking service starts the packet capture.
                    services.AddHostedService<WebSocketBroadcaster>();
                    services.AddHostedService<TrackingHostedService>();
                })
                .Build();

            await host.RunAsync();
            return 0;
        }
        catch (Exception ex)
        {
            Log.Fatal(ex, "Host terminated unexpectedly");
            return 1;
        }
        finally
        {
            Log.CloseAndFlush();
        }
    }
}
