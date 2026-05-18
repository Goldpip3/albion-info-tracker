using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using StatisticsAnalysisTool.Network;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class UpdateFameEventHandler : EventPacketHandler<UpdateFameEvent>
{
    private readonly TrackingController _trackingController;

    public UpdateFameEventHandler(TrackingController trackingController) : base((int)EventCodes.UpdateFame)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(UpdateFameEvent value)
    {
        long combatFame = value.FameWithZoneMultiplier / 10000L;
        long premiumFame = value.IsPremiumBonus ? combatFame / 2 : 0;
        long satchelFame = value.SatchelFame / 10000L;
        long totalGainedFame = combatFame + premiumFame + satchelFame;

        await _trackingController.CombatController.AddFame(totalGainedFame);
    }
}
