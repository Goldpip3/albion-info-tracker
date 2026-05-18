using AlbionInfoTracker.Models;
using Fleck;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Hosting;
using Serilog;
using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;

namespace AlbionInfoTracker.Services;

public class WebSocketBroadcaster : IHostedService
{
    private readonly TrackingController _trackingController;
    private readonly ItemController _itemController;
    private readonly string _host;
    private readonly int _port;

    private static readonly JsonSerializerOptions JsonOpts = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        WriteIndented = false,
    };

    private readonly ConcurrentDictionary<Guid, IWebSocketConnection> _connections = new();
    private WebSocketServer? _server;
    private string _sessionId = Guid.NewGuid().ToString("N");

    // Snapshot of last-seen party state for diffing -> playerJoined/playerLeft/weaponEquipped
    private readonly Dictionary<Guid, (string Name, int MainHandIndex)> _lastPartySnapshot = new();
    private readonly object _diffLock = new();

    public WebSocketBroadcaster(TrackingController trackingController, ItemController itemController, IConfiguration configuration)
    {
        _trackingController = trackingController;
        _itemController = itemController;
        _host = "127.0.0.1";
        _port = configuration.GetValue("Port", 9696);
    }

    public Task StartAsync(CancellationToken cancellationToken)
    {
        var url = $"ws://{_host}:{_port}";
        _server = new WebSocketServer(url) { RestartAfterListenError = true };
        FleckLog.LogAction = (level, message, ex) =>
        {
            switch (level)
            {
                case LogLevel.Debug: Log.Debug(ex, "[Fleck] {Message}", message); break;
                case LogLevel.Info: Log.Information(ex, "[Fleck] {Message}", message); break;
                case LogLevel.Warn: Log.Warning(ex, "[Fleck] {Message}", message); break;
                case LogLevel.Error: Log.Error(ex, "[Fleck] {Message}", message); break;
            }
        };

        _server.Start(socket =>
        {
            socket.OnOpen = () => HandleOpen(socket);
            socket.OnClose = () => HandleClose(socket);
            socket.OnMessage = message => HandleMessage(socket, message);
            socket.OnError = ex => Log.Warning(ex, "WebSocket error for {Id}", socket.ConnectionInfo.Id);
        });

        Log.Information("WebSocket broadcaster listening on {Url}", url);

        _trackingController.CombatController.OnDamageUpdate += OnDamageUpdate;
        _trackingController.CombatController.OnFameOrSilverUpdate += OnFameOrSilverUpdate;
        _trackingController.EntityController.OnProfileOrPartyChanged += OnPartyChanged;

        return Task.CompletedTask;
    }

    public Task StopAsync(CancellationToken cancellationToken)
    {
        _trackingController.CombatController.OnDamageUpdate -= OnDamageUpdate;
        _trackingController.CombatController.OnFameOrSilverUpdate -= OnFameOrSilverUpdate;
        _trackingController.EntityController.OnProfileOrPartyChanged -= OnPartyChanged;

        foreach (var conn in _connections.Values)
        {
            try { conn.Close(); } catch { /* ignored */ }
        }
        _connections.Clear();

        _server?.Dispose();
        _server = null;
        Log.Information("WebSocket broadcaster stopped");
        return Task.CompletedTask;
    }

    private void HandleOpen(IWebSocketConnection socket)
    {
        _connections[socket.ConnectionInfo.Id] = socket;
        Log.Information("Client connected: {Id} from {Ip}", socket.ConnectionInfo.Id, socket.ConnectionInfo.ClientIpAddress);

        var hello = HelloMessage.Create(_sessionId, BuildPlayersSnapshot(), Now());
        SendTo(socket, hello);
    }

    private void HandleClose(IWebSocketConnection socket)
    {
        _connections.TryRemove(socket.ConnectionInfo.Id, out _);
        Log.Information("Client disconnected: {Id}", socket.ConnectionInfo.Id);
    }

    private void HandleMessage(IWebSocketConnection socket, string message)
    {
        try
        {
            var cmd = JsonSerializer.Deserialize<ClientCommand>(message, JsonOpts);
            if (cmd?.Type == "reset")
            {
                Log.Information("Reset requested by client {Id}", socket.ConnectionInfo.Id);
                _trackingController.CombatController.ResetDamageMeter();
                _sessionId = Guid.NewGuid().ToString("N");
                Broadcast(SessionResetMessage.Create(_sessionId, Now()));
            }
            else
            {
                Log.Debug("Unknown client message from {Id}: {Msg}", socket.ConnectionInfo.Id, message);
            }
        }
        catch (Exception ex)
        {
            Log.Warning(ex, "Failed to parse client message from {Id}: {Msg}", socket.ConnectionInfo.Id, message);
        }
    }

    private void OnDamageUpdate(List<KeyValuePair<Guid, PlayerGameObject>> _)
    {
        Broadcast(PlayersUpdateMessage.Create(BuildPlayersSnapshot(), Now()));
    }

    private void OnFameOrSilverUpdate(long fame, long combatFame, long silver)
    {
        Broadcast(FameUpdateMessage.Create(fame, combatFame, silver, Now()));
    }

    private void OnPartyChanged()
    {
        var current = _trackingController.EntityController.GetAllEntitiesInParty();
        var localGuid = _trackingController.EntityController.GetLocalEntity()?.Key;

        lock (_diffLock)
        {
            var currentByGuid = current.ToDictionary(
                kv => kv.Key,
                kv => (Name: kv.Value.Name ?? string.Empty, MainHand: kv.Value.CharacterEquipment?.MainHand ?? 0));

            // Joins
            foreach (var (guid, info) in currentByGuid)
            {
                if (!_lastPartySnapshot.ContainsKey(guid))
                {
                    var weapon = _itemController.GetUniqueNameByIndex(info.MainHand);
                    Broadcast(PlayerJoinedMessage.Create(GuidToId(guid), info.Name, weapon, Now()));
                }
                else if (_lastPartySnapshot[guid].MainHandIndex != info.MainHand)
                {
                    var weapon = _itemController.GetUniqueNameByIndex(info.MainHand);
                    Broadcast(WeaponEquippedMessage.Create(GuidToId(guid), weapon, Now()));
                }
            }

            // Leaves
            foreach (var guid in _lastPartySnapshot.Keys.Where(g => !currentByGuid.ContainsKey(g)).ToList())
            {
                Broadcast(PlayerLeftMessage.Create(GuidToId(guid), Now()));
            }

            _lastPartySnapshot.Clear();
            foreach (var (guid, info) in currentByGuid)
                _lastPartySnapshot[guid] = info;
        }
    }

    private IReadOnlyList<PlayerSnapshot> BuildPlayersSnapshot()
    {
        var party = _trackingController.EntityController.GetAllEntitiesInParty();
        var localGuid = _trackingController.EntityController.GetLocalEntity()?.Key;
        var fame = _trackingController.CombatController.SessionFame;
        var silver = _trackingController.CombatController.SessionSilver;

        var snapshots = new List<PlayerSnapshot>(party.Count);
        foreach (var (guid, p) in party)
        {
            var isLocal = localGuid.HasValue && guid == localGuid.Value;
            var weapon = _itemController.GetUniqueNameByIndex(p.CharacterEquipment?.MainHand ?? 0);
            snapshots.Add(new PlayerSnapshot(
                PlayerId: GuidToId(guid),
                Name: p.Name ?? string.Empty,
                WeaponItemId: weapon,
                TotalDamage: p.Damage,
                TotalHeal: p.Heal,
                TotalTaken: p.TakenDamage,
                Dps: p.Dps,
                Hps: p.Hps,
                Fame: isLocal ? fame : 0,
                Silver: isLocal ? silver : 0));
        }
        return snapshots;
    }

    private void Broadcast<T>(T message)
    {
        var json = JsonSerializer.Serialize(message, JsonOpts);
        Log.Debug("Broadcast {Json}", json);
        foreach (var conn in _connections.Values)
        {
            if (conn.IsAvailable)
            {
                try { conn.Send(json); }
                catch (Exception ex) { Log.Warning(ex, "Send failed for {Id}", conn.ConnectionInfo.Id); }
            }
        }
    }

    private void SendTo<T>(IWebSocketConnection socket, T message)
    {
        var json = JsonSerializer.Serialize(message, JsonOpts);
        Log.Debug("Send to {Id} {Json}", socket.ConnectionInfo.Id, json);
        try { socket.Send(json); }
        catch (Exception ex) { Log.Warning(ex, "SendTo failed for {Id}", socket.ConnectionInfo.Id); }
    }

    private static long Now() => DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

    private static string GuidToId(Guid g) => g.ToString("N");
}
