using AlbionInfoTracker.Models;
using Serilog;
using System;
using System.Collections.Generic;

namespace AlbionInfoTracker.Network.Events;

public class PartyJoinedEvent
{
    public readonly Dictionary<Guid, string> PartyUsers = new();
    public Guid? SelfPlayerGuid { get; private set; }

    public PartyJoinedEvent(Dictionary<byte, object> parameters)
    {
        try
        {
            if (!parameters.ContainsKey(0) || parameters[0] == null)
                return;

            // Self GUID: key 3 (old format) or key 4 as raw byte[16] (new format)
            if (parameters.TryGetValue(3, out var selfGuidObj) && selfGuidObj is byte[] selfBytes3 && selfBytes3.Length == 16)
                SelfPlayerGuid = new Guid(selfBytes3);
            else if (parameters.TryGetValue(4, out var selfGuidObj4) && selfGuidObj4 is byte[] selfBytes4 && selfBytes4.Length == 16)
                SelfPlayerGuid = new Guid(selfBytes4);

            // Try multiple parsing strategies for the party member list
            ParsePartyUsers(parameters);

            Log.Information("PartyJoinedEvent parsed: {Count} users, selfGuid={SelfGuid}", PartyUsers.Count, SelfPlayerGuid);
            foreach (var kvp in PartyUsers)
                Log.Debug("PartyMember: {Guid} = {Name}", kvp.Key, kvp.Value);
        }
        catch (Exception e) { Log.Error(e, "PartyJoinedEvent parse error"); }
    }

    private void ParsePartyUsers(Dictionary<byte, object> parameters)
    {
        // Strategy 1: Old format - object[] of byte[] at key 4/5, string[] at key 5/6
        foreach (var guidKey in new byte[] { 4, 5 })
        {
            if (parameters.TryGetValue(guidKey, out var val) && val is object[] objArr && objArr.Length > 0 && objArr[0] is byte[])
            {
                var nameKey = (byte)(guidKey + 1);
                if (parameters.TryGetValue(nameKey, out var nameVal) && nameVal is string[] names)
                {
                    var guidDict = objArr.ToDictionary();
                    var nameDict = names.ToDictionary();
                    for (var i = 0; i < guidDict.Count && i < nameDict.Count; i++)
                    {
                        var guid = guidDict[i].ObjectToGuid();
                        if (guid != null && !string.IsNullOrEmpty(nameDict[i]))
                            PartyUsers.TryAdd((Guid)guid, nameDict[i]);
                    }
                    return;
                }
            }
        }

        // Strategy 2: New format - flat byte[] of concatenated GUIDs (N*16 bytes) at key 5, string[] at key 6
        foreach (var guidKey in new byte[] { 5, 4 })
        {
            if (parameters.TryGetValue(guidKey, out var val) && val is byte[] flatGuids && flatGuids.Length >= 16 && flatGuids.Length % 16 == 0)
            {
                var nameKey = (byte)(guidKey + 1);
                if (parameters.TryGetValue(nameKey, out var nameVal) && nameVal is string[] names)
                {
                    int count = flatGuids.Length / 16;
                    if (count != names.Length)
                    {
                        Log.Warning("PartyJoined guid count {GuidCount} != name count {NameCount}", count, names.Length);
                        count = Math.Min(count, names.Length);
                    }

                    for (int i = 0; i < count; i++)
                    {
                        var guidBytes = new byte[16];
                        Array.Copy(flatGuids, i * 16, guidBytes, 0, 16);
                        var guid = new Guid(guidBytes);
                        if (guid != Guid.Empty && !string.IsNullOrEmpty(names[i]))
                            PartyUsers.TryAdd(guid, names[i]);
                    }
                    return;
                }
            }
        }

        Log.Warning("PartyJoinedEvent: could not find guid+name arrays in parameters");
    }
}
