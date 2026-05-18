using AlbionInfoTracker.Enums;
using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class JoinResponseEvent
{
    public long? UserObjectId;
    public Guid? UserGuid { get; }
    public string Username { get; } = string.Empty;
    public Guid? InteractGuid { get; }
    public string GuildName { get; } = string.Empty;
    public string AllianceName { get; } = string.Empty;

    public JoinResponseEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.ContainsKey(0))
                UserObjectId = parameters[0].ObjectToLong();
            if (parameters.ContainsKey(1))
                UserGuid = parameters[1].ObjectToGuid();
            if (parameters.TryGetValue(2, out object? username))
                Username = username?.ToString() ?? string.Empty;
            if (parameters.ContainsKey(54))
                InteractGuid = parameters[54].ObjectToGuid();
            if (parameters.ContainsKey(58))
                GuildName = string.IsNullOrEmpty(parameters[58]?.ToString()) ? string.Empty : parameters[58].ToString()!;
            if (parameters.ContainsKey(80))
                AllianceName = string.IsNullOrEmpty(parameters[80]?.ToString()) ? string.Empty : parameters[80].ToString()!;
        }
        catch (Exception e)
        {
            Log.Error(e, "JoinResponseEvent parse error");
        }
    }
}
