using System.Collections.Generic;

namespace AlbionInfoTracker.Models;

public record PlayerSnapshot(
    string PlayerId,
    string Name,
    string? WeaponItemId,
    long TotalDamage,
    long TotalHeal,
    long TotalTaken,
    double Dps,
    double Hps,
    long Fame,
    long Silver);

public record HelloMessage(string Type, long Ts, string SessionId, IReadOnlyList<PlayerSnapshot> Players)
{
    public static HelloMessage Create(string sessionId, IReadOnlyList<PlayerSnapshot> players, long ts)
        => new("hello", ts, sessionId, players);
}

public record PlayersUpdateMessage(string Type, long Ts, IReadOnlyList<PlayerSnapshot> Players)
{
    public static PlayersUpdateMessage Create(IReadOnlyList<PlayerSnapshot> players, long ts)
        => new("playersUpdate", ts, players);
}

public record PlayerJoinedMessage(string Type, long Ts, string PlayerId, string Name, string? WeaponItemId)
{
    public static PlayerJoinedMessage Create(string playerId, string name, string? weaponItemId, long ts)
        => new("playerJoined", ts, playerId, name, weaponItemId);
}

public record PlayerLeftMessage(string Type, long Ts, string PlayerId)
{
    public static PlayerLeftMessage Create(string playerId, long ts)
        => new("playerLeft", ts, playerId);
}

public record WeaponEquippedMessage(string Type, long Ts, string PlayerId, string? WeaponItemId)
{
    public static WeaponEquippedMessage Create(string playerId, string? weaponItemId, long ts)
        => new("weaponEquipped", ts, playerId, weaponItemId);
}

public record FameUpdateMessage(string Type, long Ts, long Fame, long CombatFame, long Silver)
{
    public static FameUpdateMessage Create(long fame, long combatFame, long silver, long ts)
        => new("fameUpdate", ts, fame, combatFame, silver);
}

public record SessionResetMessage(string Type, long Ts, string SessionId)
{
    public static SessionResetMessage Create(string sessionId, long ts)
        => new("sessionReset", ts, sessionId);
}

public record ClientCommand(string? Type);
