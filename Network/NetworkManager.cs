using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Network.Handlers;
using AlbionInfoTracker.Network.PacketProviders;
using AlbionInfoTracker.Services;
using Serilog;
using StatisticsAnalysisTool.Abstractions;
using StatisticsAnalysisTool.Network;

namespace AlbionInfoTracker.Network;

public class NetworkManager
{
    private readonly PacketProvider _packetProvider;

    public NetworkManager(TrackingController trackingController, PacketProviderKind providerKind)
    {
        IPhotonReceiver photonReceiver = Build(trackingController);

        if (providerKind == PacketProviderKind.Npcap)
        {
            _packetProvider = new LibpcapPacketProvider(photonReceiver);
            Log.Information("Using packet provider: {Provider}", PacketProviderKind.Npcap);
        }
        else
        {
            _packetProvider = new SocketsPacketProvider(photonReceiver);
            Log.Information("Using packet provider: {Provider}", PacketProviderKind.Sockets);
        }
    }

    private static IPhotonReceiver Build(TrackingController trackingController)
    {
        ReceiverBuilder builder = ReceiverBuilder.Create();

        // Events - combat related only
        builder.AddEventHandler(new HealthUpdateEventHandler(trackingController));
        builder.AddEventHandler(new HealthUpdatesEventHandler(trackingController));
        builder.AddEventHandler(new InCombatStateUpdateEventHandler(trackingController));
        builder.AddEventHandler(new NewCharacterEventHandler(trackingController));
        builder.AddEventHandler(new CharacterEquipmentChangedEventHandler(trackingController));
        builder.AddEventHandler(new PartyDisbandedEventHandler(trackingController));
        builder.AddEventHandler(new PartyJoinedEventHandler(trackingController));
        builder.AddEventHandler(new PartyPlayerJoinedEventHandler(trackingController));
        builder.AddEventHandler(new PartyPlayerLeftEventHandler(trackingController));

        builder.AddEventHandler(new UpdateFameEventHandler(trackingController));
        builder.AddEventHandler(new UpdateReSpecPointsEventHandler(trackingController));
        builder.AddEventHandler(new TakeSilverEventHandler(trackingController));

        // Response
        builder.AddResponseHandler(new JoinResponseHandler(trackingController));

        return builder.Build();
    }

    public void Start()
    {
        _packetProvider.Start();
        Log.Information("Network capture started");
    }

    public void Stop()
    {
        _packetProvider.Stop();
        Log.Information("Network capture stopped");
    }

    public bool IsAnySocketActive() => _packetProvider.IsRunning;
}
