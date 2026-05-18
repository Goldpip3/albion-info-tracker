# Albion Info Tracker

A headless Windows service that passively captures **Albion Online** network traffic, decodes Photon packets, and broadcasts combat events (damage, healing, fame, silver) over a localhost **WebSocket** for any frontend to render — browser tab, OBS overlay, Discord bot, etc.

Forked from [akashi-sym/Minimal-Albion-Online-DPS-Meter](https://github.com/akashi-sym/Minimal-Albion-Online-DPS-Meter) (a WinUI 3 desktop meter) and rewritten as a .NET 9 console host.

## Requirements

- Windows 10/11 (build 22621+)
- [.NET 9 SDK](https://dotnet.microsoft.com/download/dotnet/9.0)
- [Npcap](https://npcap.com/#download) (recommended) with **"WinPcap API-compatible mode"** checked during install — or run as Administrator to use raw sockets
- Node.js 20+ (only if you want to run the included web UI under `web/`)

## Build & run

```powershell
dotnet build -c Release
dotnet bin\Release\net9.0\AlbionInfoTracker.dll
```

The service listens on `ws://127.0.0.1:9696` (localhost only) and begins packet capture immediately. Start it **before** zoning in so it can register party members from the `NewCharacter` packet.

### Web UI (`web/`)

A Vite + React meter UI is included as a sibling project. In a second terminal:

```powershell
cd web
npm install         # one time
npm run dev
```

Vite prints a `http://localhost:5173/` URL — open it in any browser. The UI auto-connects to `ws://127.0.0.1:9696`, auto-reconnects every 2s if the service is down, and ships a `Reset` button that sends `{"type":"reset"}` to the service. `npm run build` produces a static bundle in `web/dist/` that can be served from any static host or used as an OBS browser source.

### Configuration

Pass flags or set env vars (standard `Microsoft.Extensions.Configuration` precedence — CLI > env > defaults):

| Setting | Default | Notes |
|---|---|---|
| `--Port` / `Port` env | `9696` | TCP port for the WebSocket listener. Bind host is always `127.0.0.1`. |
| `--PacketProvider` / `PacketProvider` env | `Npcap` | `Npcap` or `Sockets`. Sockets requires Administrator. |

## WebSocket protocol

### Server → client messages

All messages are single JSON objects with `type` and `ts` (unix milliseconds). Field names are camelCase.

| `type` | Sent when | Payload |
|---|---|---|
| `hello` | A client connects | `sessionId`, `players[]` snapshot |
| `playersUpdate` | Combat occurs (~1Hz throttled) | `players[]` full snapshot |
| `playerJoined` | A player enters your party | `playerId`, `name`, `weaponItemId` |
| `playerLeft` | A player leaves your party | `playerId` |
| `weaponEquipped` | A party member swaps main hand | `playerId`, `weaponItemId` |
| `fameUpdate` | Fame, combat fame or silver ticks | `fame`, `combatFame`, `silver` (session totals) |
| `sessionReset` | After a client sends `{"type":"reset"}` | new `sessionId` |

### Client → server messages

| `type` | Effect |
|---|---|
| `reset` | Clears session damage/heal/fame/silver totals and issues a new `sessionId` |

### `PlayerSnapshot` shape

```jsonc
{
  "playerId":     "<guid hex, no dashes>",
  "name":         "PlayerName",
  "weaponItemId": "T8_MAIN_FIRESTAFF",   // null until item data loads
  "totalDamage":  12345,
  "totalHeal":    0,
  "totalTaken":   1200,
  "dps":          412.5,                 // damage / actual combat time
  "hps":          0.0,                   // heal   / actual combat time
  "fame":         0,    // see note
  "silver":       0     // see note
}
```

> `dps` / `hps` divide by the player's accumulated **combat time** (only seconds the player was actually in combat), not wall-clock since first sight — so they match what other Albion tools show and don't get diluted by downtime between fights.

> **Note on `fame`/`silver`:** The game only sends your *own* fame and silver over the wire, so these fields are non-zero **only for the local player** in `players[]`. Other party members will always show `0` for these two. Per-player fame attribution is not possible from packet capture alone.

### Quick test (no Albion required)

```powershell
$ws = New-Object System.Net.WebSockets.ClientWebSocket
$ws.ConnectAsync([Uri]"ws://127.0.0.1:9696", [Threading.CancellationToken]::None).Wait()
$buf = New-Object byte[] 8192
$seg = [ArraySegment[byte]]::new($buf)
$r = $ws.ReceiveAsync($seg, [Threading.CancellationToken]::None).GetAwaiter().GetResult()
[Text.Encoding]::UTF8.GetString($buf, 0, $r.Count)   # → {"type":"hello",...}
```

## How it works

```
UDP packets ─► Libpcap / raw sockets
            │
            ▼
   StatisticsAnalysisTool.Network  (Photon decoder, vendored)
            │
            ▼
   Event handlers (NewCharacter, HealthUpdate, PartyJoined, …)
            │
            ▼
   EntityController + CombatController  (party-scoped aggregation)
            │
            ▼
   WebSocketBroadcaster  (Fleck, 127.0.0.1:9696, snapshot stream)
            │
            ▼
        Your frontend
```

| Component | Role |
|---|---|
| `Network/NetworkManager.cs` | Wires the chosen `PacketProvider` to the SAT receiver and registers all handlers |
| `Services/TrackingController.cs` | Composition root for tracking; owns the network-manager lifecycle |
| `Services/EntityController.cs` | Tracks known players + party membership; emits `OnProfileOrPartyChanged` |
| `Services/CombatController.cs` | Per-player damage/heal/taken aggregates + session fame/silver counters; emits throttled `OnDamageUpdate` and `OnFameOrSilverUpdate` |
| `Services/ItemController.cs` | Maps item indices → unique names from [ao-data/ao-bin-dumps](https://github.com/ao-data/ao-bin-dumps) (cached daily under `%LOCALAPPDATA%/AlbionDpsMeter/`) |
| `Services/WebSocketBroadcaster.cs` | Fleck `IHostedService`; subscribes to the controllers, serializes events to JSON, broadcasts to connected clients, handles `reset` |
| `Services/TrackingHostedService.cs` | Starts/stops packet capture as part of the generic host lifecycle |
| `vendor/AlbionOnline-StatisticsAnalysis/` | Vendored [Triky313/AlbionOnline-StatisticsAnalysis](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis) source pinned at commit `883ea17a` (last `net9.0` state) — provides Photon decoding |

## Constraints (by design)

- **Capture-only.** No packet modification, no injection, no memory reading, no game-client overlay.
- **Party-scope only.** Combat aggregation is filtered to party members in `CombatController.AddDamage` and `AddTakenDamage`. This is the policy boundary set by Sandbox Interactive ([forum policy thread](https://forum.albiononline.com/index.php/Thread/124819-)).
- **Localhost only.** WebSocket binds to `127.0.0.1`, never `0.0.0.0`.

## Logs

Rolling daily, kept 7 days, written next to the binary:

```
bin/Release/net9.0/logs/albion-info-tracker-YYYYMMDD.log
```

Console shows `Information+`; the file captures `Debug+` (including every outbound WebSocket message).

## Troubleshooting

- **"Npcap is not installed" / `DllNotFoundException 'pcap'`** — install [Npcap](https://npcap.com/#download) with WinPcap-compatible mode, or use `--PacketProvider Sockets` and run as Administrator.
- **Service starts but `players[]` is always empty** — the meter only sees players whose `NewCharacter` packet you captured. Start the service *before* zoning in, or have your party leave + rejoin.
- **`fame`/`silver` never increment** — only the local player's fame/silver are sent over the wire; party members will always be `0` for these.

## Credits

- Original WinUI app: [akashi-sym](https://github.com/akashi-sym/Minimal-Albion-Online-DPS-Meter)
- Photon decoder + event/op-code mapping: [Triky313/AlbionOnline-StatisticsAnalysis](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis)
- Item data: [ao-data/ao-bin-dumps](https://github.com/ao-data/ao-bin-dumps)

## License

Personal / educational use. Albion Online is a registered trademark of Sandbox Interactive GmbH.
