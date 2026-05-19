# Skirmish

Personal Albion Online damage meter that streams to a web UI.

A small Windows agent sniffs Albion's network packets, decodes them, and
pushes live snapshots to a Cloudflare Worker. The website at
[**albion-meter-web.pages.dev**](https://albion-meter-web.pages.dev)
renders the live meter in any browser using your pairing token.

```
   ┌──────────────────┐    ┌─────────────────────┐    ┌─────────────────┐
   │  agent.exe       │ ─▶ │  Cloudflare Worker  │ ─▶ │  browser meter  │
   │  (your PC)       │    │  (auth + fan-out)   │    │  (any device)   │
   └──────────────────┘    └─────────────────────┘    └─────────────────┘
        sniffs Photon          per-token rooms        renders the data
        UDP, decodes,          via Durable Objects
        sends snapshots
```

## Quick start

You need:
- Windows 10/11
- Albion Online installed
- Admin rights (raw-socket packet capture requires it — same as Wireshark)

Steps:

1. **Clone or download this repo**:
   ```powershell
   git clone https://github.com/Goldpip3/albion-info-tracker.git
   cd albion-info-tracker
   git checkout go-port
   ```
2. **Build the agent once.** Needs [Go 1.26+](https://go.dev/) (`winget install GoLang.Go`):
   ```powershell
   cd agent
   go build -o agent.exe ./cmd/agent
   cd ..
   ```
3. **Double-click `Start Skirmish.lnk`** at the repo root.
   - Windows asks for admin — click **Yes**.
   - A small console window appears. First launch prompts for a
     pairing token. You can:
     - Paste a token generated on the website, **or**
     - Press Enter and the agent generates one — paste THAT into the
       website.
4. **Open https://albion-meter-web.pages.dev** in your browser.
   - Click **Generate** to make a token, or paste the one from the agent.
   - Hit **Connect**.
5. **Re-zone in Albion** (walk through any portal) so the meter
   populates with your character + any party members visible after
   the zone change.

## What you see

- Live damage meter with current-fight + session totals, role chips,
  composition strip, and a tunable color accent.
- Click any player row → per-ability drill-in.
- Activity log / kill feed below the meter (hits / heals / deaths).
- Settings ⚙ in the header for density, bar style, columns,
  pin-local-user, etc.

## Privacy

The Cloudflare Worker keeps no persistent storage. Snapshots live in a
Durable Object's memory only while at least one connection is open.
**Anyone with your token sees the same room**, so treat it like a
Discord invite link — share with your party, don't post publicly.

## Project layout

```
agent/      Go agent (Windows-only raw-socket Photon capture)
worker/     Cloudflare Worker + Durable Object (TypeScript)
web/        React + Vite + Tailwind v4 frontend
src/        Original C# SAT solution (sat-fork branch's tree)
```

Each subproject has its own README with build / deploy details.

## Self-hosting the backend

The default Worker URL (`albion-meter.goldpipe.workers.dev`) is the
maintainer's instance. To run your own:

1. `cd worker && npx wrangler login && npx wrangler deploy`
2. `cd ../web && npm run build && npx wrangler pages deploy ./dist --project-name <your-project>`
3. Set `pushUrl` in `agent/agent.json` (or the `ALBION_AGENT_URL` env
   var) to your Worker's `/ingest` URL.

## Branches

- **`go-port`** (active) — Go agent + Worker + React frontend.
- **`sat-fork`** — original C# WPF damage meter (working but no longer
  the focus).

See [CLAUDE.md](./CLAUDE.md) for the deeper notes — protocol quirks,
known gaps, what's deferred.
