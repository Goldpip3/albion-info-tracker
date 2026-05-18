using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class HealthUpdateEvent
{
    public long CauserId;
    public int CausingSpellIndex;
    public EffectOrigin EffectOrigin;
    public EffectType EffectType;
    public double HealthChange;
    public double NewHealthValue;
    public long AffectedObjectId;

    public HealthUpdateEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(0, out object? affectedObjectId))
                AffectedObjectId = affectedObjectId.ObjectToLong() ?? throw new ArgumentNullException();
            if (parameters.TryGetValue(2, out object? healthChange))
                HealthChange = healthChange.ObjectToDouble();
            if (parameters.TryGetValue(3, out object? newHealthValue))
                NewHealthValue = newHealthValue.ObjectToDouble();
            if (parameters.TryGetValue(4, out object? effectType))
                EffectType = (EffectType)(effectType as byte? ?? 0);
            if (parameters.TryGetValue(5, out object? effectOrigin))
                EffectOrigin = (EffectOrigin)(effectOrigin as byte? ?? 9);
            if (parameters.TryGetValue(6, out object? causerId))
                CauserId = causerId.ObjectToLong() ?? throw new ArgumentNullException();
            if (parameters.TryGetValue(7, out object? causingSpellType))
                CausingSpellIndex = causingSpellType.ObjectToShort();
        }
        catch (Exception e)
        {
            Log.Error(e, "HealthUpdateEvent parse error");
        }
    }
}
