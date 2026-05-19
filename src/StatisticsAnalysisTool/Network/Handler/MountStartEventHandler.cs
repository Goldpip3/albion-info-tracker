using StatisticsAnalysisTool.Network.Events;
using StatisticsAnalysisTool.Network.Manager;
using System.Threading.Tasks;

namespace StatisticsAnalysisTool.Network.Handler;

public class MountStartEventHandler(TrackingController trackingController) : EventPacketHandler<MountStartEvent>((int) EventCodes.MountStart)
{
    protected override Task OnActionAsync(MountStartEvent value)
    {
        if (value.RiderObjectId is { } objectId)
        {
            trackingController.EntityController.StashMountStartObjectId(objectId);
        }

        return Task.CompletedTask;
    }
}
