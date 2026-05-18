using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using StatisticsAnalysisTool.Network;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class PartyPlayerLeftEventHandler : EventPacketHandler<PartyPlayerLeftEvent>
{
    private readonly TrackingController _trackingController;

    public PartyPlayerLeftEventHandler(TrackingController trackingController) : base((int)EventCodes.PartyPlayerLeft)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(PartyPlayerLeftEvent value)
    {
        _trackingController.EntityController.RemoveFromParty(value.UserGuid);
        await Task.CompletedTask;
    }
}
