using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class InCombatStateUpdateEvent
{
    public readonly bool InActiveCombat;
    public readonly bool InPassiveCombat;
    public long? ObjectId;

    public InCombatStateUpdateEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.ContainsKey(0))
                ObjectId = parameters[0].ObjectToLong();
            if (parameters.TryGetValue(1, out object? inActiveCombat))
                InActiveCombat = inActiveCombat as bool? ?? false;
            if (parameters.TryGetValue(2, out object? inPassiveCombat))
                InPassiveCombat = inPassiveCombat as bool? ?? false;
        }
        catch (Exception e) { Log.Error(e, "InCombatStateUpdateEvent parse error"); }
    }
}
