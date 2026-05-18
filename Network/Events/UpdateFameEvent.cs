using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class UpdateFameEvent
{
    public long FameWithZoneMultiplier;
    public bool IsPremiumBonus;
    public long SatchelFame;

    public UpdateFameEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(2, out object? fameRaw))
                FameWithZoneMultiplier = fameRaw.ObjectToLong() ?? 0;
            if (parameters.TryGetValue(5, out object? premiumRaw))
                IsPremiumBonus = premiumRaw.ObjectToBool();
            if (parameters.TryGetValue(10, out object? satchelRaw))
                SatchelFame = satchelRaw.ObjectToLong() ?? 0;
        }
        catch (Exception e)
        {
            Log.Error(e, "UpdateFameEvent parse error");
        }
    }
}
