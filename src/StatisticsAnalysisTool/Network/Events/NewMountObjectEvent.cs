using StatisticsAnalysisTool.Common;
using System;
using System.Collections.Generic;
using System.Reflection;
using StatisticsAnalysisTool.Diagnostics;

namespace StatisticsAnalysisTool.Network.Events;

public class NewMountObjectEvent
{
    public Guid? RiderUserGuid { get; }

    public NewMountObjectEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(5, out object riderUserGuid))
            {
                RiderUserGuid = riderUserGuid.ObjectToGuid();
            }
        }
        catch (Exception e)
        {
            DebugConsole.WriteError(MethodBase.GetCurrentMethod()?.DeclaringType, e);
        }
    }
}
