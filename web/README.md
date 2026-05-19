# albion-meter-web

React + Vite + Tailwind frontend for the Albion meter. Connects to the
Cloudflare Worker `/view` endpoint via WebSocket and renders the live
damage meter served from your Go agent.

## Dev

```powershell
cd web
npm install
npm run dev
```

Opens `http://localhost:5173`. The setup screen asks for:

- **Worker URL** — `wss://albion-meter.<your-subdomain>.workers.dev/view`
- **Token** — same string you put in `agent/agent.json` `pushToken`

Both are stored in `localStorage` so a page refresh keeps the connection.

## Build & deploy to Cloudflare Pages

```powershell
npm run build
npx wrangler pages deploy ./dist --project-name albion-meter-web
```

Wrangler prints the live URL (something like
`https://albion-meter-web.pages.dev`).

## Modes

Three columns: **Dmg** (damage dealt), **Heal**, **Taken**. Each row
shows CURRENT (this fight) | OVERALL (session) | per-second rate, with
a bar-fill proportional to the top performer in the selected mode.
