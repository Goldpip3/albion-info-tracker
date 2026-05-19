using StatisticsAnalysisTool.Network.Events;
using StatisticsAnalysisTool.Network.Manager;
using System.Threading.Tasks;

namespace StatisticsAnalysisTool.Network.Handler;

public class NewMountObjectEventHandler(TrackingController trackingController) : EventPacketHandler<NewMountObjectEvent>((int) EventCodes.NewMountObject)
{
    protected override Task OnActionAsync(NewMountObjectEvent value)
    {
        if (value.RiderUserGuid is { } guid)
        {
            trackingController.EntityController.BindGuidToRecentMountStart(guid);
        }

        return Task.CompletedTask;
    }
}
