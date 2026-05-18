using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using Serilog;
using StatisticsAnalysisTool.Network;
using System;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class PartyJoinedEventHandler : EventPacketHandler<PartyJoinedEvent>
{
    private readonly TrackingController _trackingController;

    public PartyJoinedEventHandler(TrackingController trackingController) : base((int)EventCodes.PartyJoined)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(PartyJoinedEvent value)
    {
        Log.Information("PartyJoined received: {Count} members", value.PartyUsers.Count);

        _trackingController.EntityController.SetParty(value.PartyUsers);
        await Task.CompletedTask;
    }
}
