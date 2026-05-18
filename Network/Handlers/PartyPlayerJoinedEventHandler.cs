using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using Serilog;
using StatisticsAnalysisTool.Network;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class PartyPlayerJoinedEventHandler : EventPacketHandler<PartyPlayerJoinedEvent>
{
    private readonly TrackingController _trackingController;

    public PartyPlayerJoinedEventHandler(TrackingController trackingController) : base((int)EventCodes.PartyPlayerJoined)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(PartyPlayerJoinedEvent value)
    {
        Log.Information("PartyPlayerJoined: {Username} guid={Guid}", value.Username, value.UserGuid);
        _trackingController.EntityController.AddEntity(new Entity
        {
            UserGuid = value.UserGuid,
            Name = value.Username,
            ObjectType = GameObjectType.Player,
            ObjectSubType = GameObjectSubType.Player
        });
        _trackingController.EntityController.AddToParty(value.UserGuid);
        await Task.CompletedTask;
    }
}
