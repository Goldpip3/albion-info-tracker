using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using Serilog;
using StatisticsAnalysisTool.Network;
using System;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class JoinResponseHandler : ResponsePacketHandler<JoinResponseEvent>
{
    private readonly TrackingController _trackingController;

    public JoinResponseHandler(TrackingController trackingController) : base((int)OperationCodes.Join)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(JoinResponseEvent value)
    {
        Log.Information("JoinResponse received: User={Username} Guild={Guild} Guid={Guid}",
            value.Username, value.GuildName, value.UserGuid);

        if (value.UserGuid == null || value.UserObjectId == null)
            return;

        _trackingController.EntityController.AddEntity(new Entity
        {
            ObjectId = value.UserObjectId,
            UserGuid = value.UserGuid ?? Guid.Empty,
            InteractGuid = value.InteractGuid,
            Name = value.Username,
            Guild = value.GuildName,
            Alliance = value.AllianceName,
            ObjectType = GameObjectType.Player,
            ObjectSubType = GameObjectSubType.LocalPlayer
        });

        _trackingController.EntityController.AddToParty(value.UserGuid ?? Guid.Empty);

        // Reset damage meter on map change if setting is enabled
        _trackingController.CombatController.ResetDamageMeterByClusterChange();

        await Task.CompletedTask;
    }
}
