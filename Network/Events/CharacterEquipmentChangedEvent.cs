using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class CharacterEquipmentChangedEvent
{
    public long? ObjectId { get; }
    public CharacterEquipment CharacterEquipment { get; } = new();

    public CharacterEquipmentChangedEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(0, out object? objectId))
                ObjectId = objectId.ObjectToLong();

            if (parameters.TryGetValue(2, out object? equipmentObject))
            {
                var valueType = equipmentObject.GetType();
                if (valueType.IsArray)
                {
                    if (typeof(byte[]).Name == valueType.Name)
                        SetEquipment(((byte[])equipmentObject).ToDictionary());
                    else if (typeof(short[]).Name == valueType.Name)
                        SetEquipment(((short[])equipmentObject).ToDictionary());
                    else if (typeof(int[]).Name == valueType.Name)
                        SetEquipment(((int[])equipmentObject).ToDictionary());
                }
            }
        }
        catch (Exception e)
        {
            Log.Error(e, "CharacterEquipmentChangedEvent parse error");
        }
    }

    private void SetEquipment<T>(IReadOnlyDictionary<int, T> values) where T : notnull
    {
        CharacterEquipment.MainHand = values[0].ObjectToInt();
        CharacterEquipment.OffHand = values[1].ObjectToInt();
        CharacterEquipment.Head = values[2].ObjectToInt();
        CharacterEquipment.Chest = values[3].ObjectToInt();
        CharacterEquipment.Shoes = values[4].ObjectToInt();
        CharacterEquipment.Bag = values[5].ObjectToInt();
        CharacterEquipment.Cape = values[6].ObjectToInt();
        CharacterEquipment.Mount = values[7].ObjectToInt();
        CharacterEquipment.Potion = values[8].ObjectToInt();
        CharacterEquipment.BuffFood = values[9].ObjectToInt();
    }
}
