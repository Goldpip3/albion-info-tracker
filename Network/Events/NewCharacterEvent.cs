using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class NewCharacterEvent
{
    public long? ObjectId { get; }
    public Guid? Guid { get; }
    public string Name { get; } = string.Empty;
    public string GuildName { get; } = string.Empty;
    public CharacterEquipment CharacterEquipment { get; } = new();

    public NewCharacterEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(0, out object? objectId))
                ObjectId = objectId.ObjectToLong();
            if (parameters.TryGetValue(1, out object? name))
                Name = name?.ToString() ?? string.Empty;
            if (parameters.TryGetValue(7, out object? guid))
                Guid = guid.ObjectToGuid();
            if (parameters.TryGetValue(8, out object? guildName))
                GuildName = guildName?.ToString() ?? string.Empty;

            if (parameters.ContainsKey(40))
            {
                var valueType = parameters[40].GetType();
                switch (valueType.IsArray)
                {
                    case true when typeof(byte[]).Name == valueType.Name:
                        CharacterEquipment = GetEquipment(((byte[])parameters[40]).ToDictionary());
                        break;
                    case true when typeof(short[]).Name == valueType.Name:
                        CharacterEquipment = GetEquipment(((short[])parameters[40]).ToDictionary());
                        break;
                    case true when typeof(int[]).Name == valueType.Name:
                        CharacterEquipment = GetEquipment(((int[])parameters[40]).ToDictionary());
                        break;
                }
            }
        }
        catch (Exception e) { Log.Error(e, "NewCharacterEvent parse error"); }
    }

    private static CharacterEquipment GetEquipment<T>(IReadOnlyDictionary<int, T> values) where T : notnull
    {
        return new CharacterEquipment
        {
            MainHand = values[0].ObjectToInt(),
            OffHand = values[1].ObjectToInt(),
            Head = values[2].ObjectToInt(),
            Chest = values[3].ObjectToInt(),
            Shoes = values[4].ObjectToInt(),
            Bag = values[5].ObjectToInt(),
            Cape = values[6].ObjectToInt(),
            Mount = values[7].ObjectToInt(),
            Potion = values[8].ObjectToInt(),
            BuffFood = values[9].ObjectToInt()
        };
    }
}
