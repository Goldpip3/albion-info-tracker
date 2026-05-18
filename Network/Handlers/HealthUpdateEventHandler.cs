using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using StatisticsAnalysisTool.Network;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class HealthUpdateEventHandler : EventPacketHandler<HealthUpdateEvent>
{
    private readonly TrackingController _trackingController;

    public HealthUpdateEventHandler(TrackingController trackingController) : base((int)EventCodes.HealthUpdate)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(HealthUpdateEvent value)
    {
        await _trackingController.CombatController.AddDamage(value.AffectedObjectId, value.CauserId, value.HealthChange, value.NewHealthValue, value.CausingSpellIndex);
        await _trackingController.CombatController.AddTakenDamage(value.AffectedObjectId, value.CauserId, value.HealthChange, value.NewHealthValue, value.CausingSpellIndex);
    }
}
