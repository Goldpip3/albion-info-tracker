using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Network;
using Serilog;
using System;
using System.Net.Sockets;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Services;

public class TrackingController
{
    public readonly EntityController EntityController;
    public readonly CombatController CombatController;
    private NetworkManager? _networkManager;

    public bool IsTrackingActive { get; private set; }
    public PacketProviderKind PacketProvider { get; set; } = PacketProviderKind.Npcap;

    public event Action<string>? OnTrackingError;
    public event Action<bool>? OnTrackingStateChanged;

    public TrackingController(EntityController entityController, CombatController combatController)
    {
        EntityController = entityController;
        CombatController = combatController;
    }

    public Task StartTrackingAsync()
    {
        if (_networkManager?.IsAnySocketActive() ?? false)
            return Task.CompletedTask;

        try
        {
            _networkManager = new NetworkManager(this, PacketProvider);
            _networkManager.Start();
            IsTrackingActive = true;
            OnTrackingStateChanged?.Invoke(true);
            Log.Information("Tracking started with provider: {Provider}", PacketProvider);
        }
        catch (Exception ex)
        {
            string errorMsg = GetTrackingErrorMessage(ex);
            Log.Error(ex, "StartTracking failed | provider={Provider} | msg={Msg}", PacketProvider, errorMsg);
            OnTrackingError?.Invoke(errorMsg);

            try { StopTracking(); } catch { /* ignored */ }
            IsTrackingActive = false;
            OnTrackingStateChanged?.Invoke(false);
        }

        return Task.CompletedTask;
    }

    public void StopTracking()
    {
        if (!IsTrackingActive) return;

        _networkManager?.Stop();
        IsTrackingActive = false;
        OnTrackingStateChanged?.Invoke(false);
        Log.Information("Tracking stopped");
    }

    private static string GetTrackingErrorMessage(Exception ex)
    {
        if (ex is SocketException se)
            return $"Socket failed with error code: {se.SocketErrorCode}. Run as Administrator for raw socket capture.";

        if (ex is UnauthorizedAccessException)
            return "Please start the application as Administrator for raw socket capture.";

        if (ex is DllNotFoundException d &&
            (d.Message.Contains("wpcap", StringComparison.OrdinalIgnoreCase) ||
             d.Message.Contains("npcap", StringComparison.OrdinalIgnoreCase)))
            return "Npcap is not installed. Please install Npcap from https://npcap.com/ and restart.";

        if (ex is TypeInitializationException { InnerException: DllNotFoundException inner } &&
            (inner.Message.Contains("wpcap", StringComparison.OrdinalIgnoreCase) ||
             inner.Message.Contains("npcap", StringComparison.OrdinalIgnoreCase)))
            return "Npcap is not installed. Please install Npcap from https://npcap.com/ and restart.";

        if (ex.GetType().Name.Equals("PcapException", StringComparison.OrdinalIgnoreCase))
            return "Failed to open Npcap capture. Ensure Npcap is installed and run as Administrator.";

        if (ex is InvalidOperationException)
            return "Capture start failed. Ensure the network device is available.";

        return $"Packet capture error: {ex.Message}";
    }
}
