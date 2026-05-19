# albion-meter-agent (Go)

Local agent that captures Albion Online network traffic, decodes the Photon
protocol, maintains combat/party/entity state, and pushes events to a remote
backend over WebSocket.

This is a clean-room Go reimplementation of the relevant subset of
[StatisticsAnalysisTool](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis)
(the `src/` tree in this repo), targeting a web-served damage meter as the
first deliverable.

## Status

Phase 1: packet capture skeleton. Captures Photon UDP datagrams on Windows
using raw sockets (no libpcap/Npcap required). Requires administrator.

## Layout

```
agent/
├── cmd/agent/main.go              entry point
├── internal/
│   ├── capture/                   raw-socket packet capture (Windows)
│   └── photonports/               port + magic-byte heuristics
└── go.mod
```

## Build & run

```powershell
cd agent
go build ./cmd/agent
# Run elevated — raw socket SIO_RCVALL requires admin.
.\agent.exe
```

Expected output while Albion is running and you are in-game:

```
#000001   5055 → 49152  len= 120  f203...
#000002  49152 →  5055  len=  40  f102...
```

Stop with Ctrl+C.

## Roadmap

See task list in the parent project. Phases:

1. ✅ Packet capture skeleton
2. Photon protocol parser port (`Protocol18`, `PhotonPackageParser`)
3. Game data extractor (DES-decrypted Albion `.bin` files)
4. Domain engine (entities, combat, party — mirrors SAT's controllers)
5. Agent → server WebSocket push protocol
6. Cloudflare Worker + Durable Object backend
7. React damage meter frontend
