// Cloudflare Worker entry point.
//
// Two WebSocket endpoints:
//   /ingest    — accepts a single agent connection per token (your PC)
//   /view      — accepts any number of browser viewers per token
// Both require ?token=… or Authorization: Bearer …. The token alone
// determines which room you land in; share the token between agent and
// browser to see your own data.
//
// Plus one static page at / so visiting the URL in a browser without the
// React frontend deployed still tells you what's running.

import { extractToken, roomIdFromToken } from "./auth.js";
import { MeterRoom } from "./meterRoom.js";

interface Env {
	ROOMS: DurableObjectNamespace<MeterRoom>;
}

export { MeterRoom };

export default {
	async fetch(req: Request, env: Env): Promise<Response> {
		const url = new URL(req.url);

		if (url.pathname === "/ingest" || url.pathname === "/view") {
			if (req.headers.get("Upgrade") !== "websocket") {
				return new Response("expected websocket upgrade", { status: 426 });
			}
			const token = extractToken(req);
			if (!token) {
				return new Response("missing token", { status: 401 });
			}
			const roomId = await roomIdFromToken(token);
			const stub = env.ROOMS.get(env.ROOMS.idFromName(roomId));
			return stub.fetch(req);
		}

		if (url.pathname === "/healthz") {
			return new Response("ok", { status: 200 });
		}

		if (url.pathname === "/") {
			return new Response(landingHtml, {
				status: 200,
				headers: { "Content-Type": "text/html; charset=utf-8" },
			});
		}

		return new Response("not found", { status: 404 });
	},
} satisfies ExportedHandler<Env>;

const landingHtml = `<!doctype html>
<html><head>
<meta charset="utf-8">
<title>albion-meter</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 36rem; margin: 4rem auto; padding: 0 1rem; line-height: 1.5; color: #222; }
  code { background: #f3f3f3; padding: 0.1em 0.3em; border-radius: 3px; }
</style>
</head><body>
<h1>albion-meter</h1>
<p>Backend is running. WebSocket endpoints:</p>
<ul>
  <li><code>/ingest?token=…</code> — agent pushes snapshots here.</li>
  <li><code>/view?token=…</code> — viewers (browser) receive snapshots here.</li>
</ul>
<p>The React UI hasn't been deployed to this Worker yet; it lives at the
Cloudflare Pages project for this account.</p>
</body></html>
`;
