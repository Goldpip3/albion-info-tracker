using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class TakeSilverEvent
{
    public long YieldPreTax;
    public long GuildTax;

    public long YieldAfterTax => YieldPreTax - GuildTax;

    public TakeSilverEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(3, out object? yieldRaw))
                YieldPreTax = yieldRaw.ObjectToLong() ?? 0;
            if (parameters.TryGetValue(5, out object? taxRaw))
                GuildTax = taxRaw.ObjectToLong() ?? 0;
        }
        catch (Exception e)
        {
            Log.Error(e, "TakeSilverEvent parse error");
        }
    }
}
