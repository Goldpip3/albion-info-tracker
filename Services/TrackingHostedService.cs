using AlbionInfoTracker.Enums;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Hosting;
using Serilog;
using System;
using System.Threading;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Services;

public class TrackingHostedService : IHostedService
{
    private readonly TrackingController _trackingController;
    private readonly ItemController _itemController;

    public TrackingHostedService(TrackingController trackingController, ItemController itemController, IConfiguration configuration)
    {
        _trackingController = trackingController;
        _itemController = itemController;

        var providerName = configuration.GetValue("PacketProvider", nameof(PacketProviderKind.Npcap));
        if (Enum.TryParse<PacketProviderKind>(providerName, ignoreCase: true, out var kind))
            _trackingController.PacketProvider = kind;
        else
            Log.Warning("Unknown PacketProvider value '{Name}', falling back to {Default}", providerName, _trackingController.PacketProvider);

        _trackingController.OnTrackingError += msg => Log.Error("Tracking error: {Msg}", msg);
        _trackingController.OnTrackingStateChanged += state => Log.Information("Tracking state: {State}", state ? "active" : "stopped");
    }

    public async Task StartAsync(CancellationToken cancellationToken)
    {
        _ = Task.Run(async () => await _itemController.LoadItemsAsync(), cancellationToken);
        await _trackingController.StartTrackingAsync();
    }

    public Task StopAsync(CancellationToken cancellationToken)
    {
        _trackingController.StopTracking();
        return Task.CompletedTask;
    }
}
