using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using AlbionInfoTracker.Network.Events;
using AlbionInfoTracker.Services;
using StatisticsAnalysisTool.Network;
using System;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Network.Handlers;

public class NewCharacterEventHandler : EventPacketHandler<NewCharacterEvent>
{
    private readonly TrackingController _trackingController;

    public NewCharacterEventHandler(TrackingController trackingController) : base((int)EventCodes.NewCharacter)
    {
        _trackingController = trackingController;
    }

    protected override async Task OnActionAsync(NewCharacterEvent value)
    {
        if (value.Guid != null && value.ObjectId != null)
        {
            _trackingController.EntityController.AddEntity(new Entity
            {
                ObjectId = value.ObjectId,
                UserGuid = value.Guid ?? Guid.Empty,
                Name = value.Name,
                Guild = value.GuildName,
                CharacterEquipment = value.CharacterEquipment,
                ObjectType = GameObjectType.Player,
                ObjectSubType = GameObjectSubType.Player
            });
        }
        await Task.CompletedTask;
    }
}
