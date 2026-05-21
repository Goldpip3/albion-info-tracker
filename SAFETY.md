# SAFETY

Goldpipe Damage Agent (GDA) is a personal-fork damage meter for **Albion Online**.
This document explains exactly what the tool does and does not do, so users can
decide for themselves whether they're comfortable running it and so Sandbox
Interactive can audit its behaviour at a glance.

## What this tool is

A two-component pipeline:

1. **Agent** — a single Windows binary (`agent.exe`) that opens a raw socket
   on the local machine, reads Albion's UDP traffic, decodes Photon Protocol 18
   in user space, maintains in-memory combat/loot/zone state, and pushes JSON
   snapshots over an outbound WebSocket roughly every 400 ms (and at most
   every 15 s when idle).
2. **Web UI** — a React page hosted on Cloudflare Pages that subscribes to
   the same WebSocket room and renders the snapshots as a WoW-Details-style
   meter.

The agent and the browser share a per-user pairing token; the Cloudflare Worker
between them does nothing except forward bytes between the two sockets that
share that token. There is no account, no database, no telemetry.

## What this tool does NOT do

- **No game-process interaction.** The agent never opens a handle to the Albion
  client, never reads its memory, never injects code, and never loads a DLL into
  it. The two processes never touch.
- **No overlay.** Nothing draws on top of the game window. The meter renders in
  a separate browser tab and is the user's responsibility to position.
- **No input automation.** The agent does not synthesise mouse clicks,
  keystrokes, controller input, or any other user-input events. It cannot
  cast spells, click buttons, or play the game on the user's behalf.
- **No outbound traffic to game servers.** The agent is a passive consumer of
  packets the operating system has already received. It never sends, forges,
  replays, or modifies traffic to Albion's servers.
- **No file modification of the game install.** The agent reads
  `items.bin`, `spells.bin`, `localization.bin`, and `mobs.bin` from the
  installed game folder (decrypts them in memory the same way the official
  client does) so it can translate numeric item/spell indices into English
  display names. It never writes to the game directory.

## How the capture works

The agent opens a raw socket bound to the local interface and asks Windows for
`SIO_RCVALL` mode. That delivers a copy of every IPv4 packet the NIC sees to
the agent's userland buffer. Admin / elevated permission is required
**specifically because** Windows guards `SIO_RCVALL` — not because the agent
needs special access to Albion.

From that firehose the agent filters for UDP datagrams whose payload looks
like Photon Protocol 18 frames, reassembles fragments, and parses the events
it knows how to interpret (combat, equipment, party, loot, fame, silver,
zone changes). Everything else is discarded.

This is the same fundamental approach used by every well-known third-party
Albion meter (Triky313/AlbionOnline-StatisticsAnalysis, Albion-Online-Stats,
etc.). The Photon protocol decoder in this repo is a clean-room Go port of
publicly documented techniques; it does not contain any code copied from
SBI's client.

## Disclaimer

Sandbox Interactive's Terms of Service prohibit some kinds of third-party
tools. Whether a passive packet-decoding meter falls inside or outside that
prohibition is a judgement call that only SBI can make for a given account.
Historically the developer community has interpreted SBI's public statements
as **tolerating** passive network-reading meters while disallowing overlays,
automation, and memory-reading tools — but this is not an official policy
document and the position can change.

**If you want explicit clearance for your account, email
[support@albiononline.com](mailto:support@albiononline.com)** with a link to
this repository and a description of how you intend to use the tool. That is
the only authoritative answer.

Use of this software is at the user's own risk. The author makes no warranty
that it is safe to use on any particular account, in any particular region,
or under any particular future policy.
