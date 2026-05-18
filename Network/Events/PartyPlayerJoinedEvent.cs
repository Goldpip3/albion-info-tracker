using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class PartyPlayerJoinedEvent
{
    public Guid UserGuid { get; private set; } = Guid.Empty;
    public string Username { get; private set; } = string.Empty;

    public PartyPlayerJoinedEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (parameters.TryGetValue(1, out object? userGuid))
                UserGuid = userGuid.ObjectToGuid() ?? Guid.Empty;
            if (parameters.TryGetValue(2, out object? username) && username is string usernameString && !string.IsNullOrEmpty(usernameString))
                Username = usernameString;
        }
        catch (Exception e) { Log.Error(e, "PartyPlayerJoinedEvent parse error"); }
    }
}
