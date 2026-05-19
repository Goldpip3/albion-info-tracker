# albion-meter-worker

Cloudflare Worker + Durable Object backend for the Go agent. Receives
snapshot pushes from one agent per user, fans them out to any number of
browser viewers, and serves a tiny landing page.

## Routes

| Path        | Protocol  | Who connects | What happens |
|-------------|-----------|--------------|--------------|
| `/ingest`   | WebSocket | the Go agent | every 500 ms, the agent sends a `{v:1, type:"snapshot", snap:{…}}` envelope; the Durable Object stores the latest and re-broadcasts to viewers |
| `/view`     | WebSocket | the browser  | receives every snapshot the agent sends, plus an immediate replay of the latest one on connect |
| `/healthz`  | HTTP      | anything     | returns `ok` |
| `/`         | HTTP      | anything     | tiny landing page |

Auth: `?token=…` query param or `Authorization: Bearer …` header. The
token hashes to a Durable Object room name; any agent + any browser
using the same token share a room. Pick something high-entropy.

## First-time setup

```powershell
cd worker
npm install
npx wrangler login            # opens browser, authorize once
npx wrangler deploy           # uploads code + creates the DO migration
```

Wrangler prints a URL like `https://albion-meter.<your-subdomain>.workers.dev`.

## Wire the agent to it

Create `agent/agent.json` next to `agent.exe`:

```json
{
  "pushUrl": "wss://albion-meter.<your-subdomain>.workers.dev/ingest",
  "pushToken": "<your-token>"
}
```

Or set env vars instead:

```powershell
$env:ALBION_AGENT_URL = "wss://albion-meter.<your-subdomain>.workers.dev/ingest"
$env:ALBION_AGENT_TOKEN = "<your-token>"
.\agent.exe
```

## Local dev

```powershell
npx wrangler dev
```

Spins up the Worker locally on `http://localhost:8787`. Durable Objects
run in the same process. Useful for iterating on the message protocol
before deploying.

## Cost

Cloudflare's free tier covers ~100k Worker requests/day and 1M DO
requests/month. A solo user pushing 2 snapshots/sec for 4 hours/day
is ~28k DO requests/day — well within free.
