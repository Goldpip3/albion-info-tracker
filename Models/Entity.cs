using AlbionInfoTracker.Enums;
using System;

namespace AlbionInfoTracker.Models;

public class Entity
{
    public long? ObjectId { get; set; }
    public Guid UserGuid { get; set; }
    public Guid? InteractGuid { get; set; }
    public string Name { get; set; } = string.Empty;
    public string Guild { get; set; } = string.Empty;
    public string Alliance { get; set; } = string.Empty;
    public CharacterEquipment? CharacterEquipment { get; set; }
    public GameObjectType ObjectType { get; set; }
    public GameObjectSubType ObjectSubType { get; set; }
}
