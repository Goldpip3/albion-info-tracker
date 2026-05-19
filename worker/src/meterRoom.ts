// MeterRoom is one user's live damage-meter state plus the set of browser
// clients watching it. Backed by a Cloudflare Durable Object so all traffic
// for a given user lands on the same instance no matter which Cloudflare
// edge it hits.
//
// We use the WebSocket Hibernation API: accepted WebSockets are not held
// by an in-process listener but by the DO runtime, which dispatches
// webSocketMessage / Close / Error back into our class as method calls.
// That means the DO can scale to thousands of hibernating connections per
// instance without burning CPU.

import { DurableObject } from "cloudflare:workers";

interface Env {
	ROOMS: DurableObjectNamespace<MeterRoom>;
}

interface SnapshotEnvelope {
	v: number;
	type: "snapshot";
	ts: number;
	snap: unknown;
}

interface HelloEnvelope {
	v: number;
	type: "hello";
	ts: number;
	hello: { agentVersion: string; localGuid?: string };
}

interface CommandEnvelope {
	v: number;
	type: "command";
	ts?: number;
	command: { action: string };
}

type AnyMessage = SnapshotEnvelope | HelloEnvelope | CommandEnvelope;

// Tag set when accepting an agent socket vs a viewer socket. Used in
// dispatch so we know which role each WS is playing without storing extra
// state per connection.
const TAG_AGENT = "agent";
const TAG_VIEWER = "viewer";

export class MeterRoom extends DurableObject<Env> {
	// In-memory: the latest snapshot pushed by the agent. Lost on hibernate
	// (every ~30s of idleness) which is fine — agents republish every 500ms.
	private latest: SnapshotEnvelope | null = null;

	async fetch(req: Request): Promise<Response> {
		const url = new URL(req.url);
		const role = url.pathname.endsWith("/ingest") ? TAG_AGENT : TAG_VIEWER;

		const pair = new WebSocketPair();
		const [client, server] = [pair[0], pair[1]];
		this.ctx.acceptWebSocket(server, [role]);

		// Push the current state to a freshly-connected viewer so its UI
		// renders immediately, not after the next agent tick.
		if (role === TAG_VIEWER && this.latest) {
			try {
				server.send(JSON.stringify(this.latest));
			} catch {
				/* connection died mid-handshake, ignore */
			}
		}

		return new Response(null, { status: 101, webSocket: client });
	}

	override async webSocketMessage(ws: WebSocket, message: string | ArrayBuffer): Promise<void> {
		if (typeof message !== "string") return;

		let parsed: AnyMessage;
		try {
			parsed = JSON.parse(message) as AnyMessage;
		} catch {
			return;
		}
		if (parsed.v !== 1) return;

		const tags = this.ctx.getTags(ws);

		if (tags.includes(TAG_AGENT)) {
			// Agent → viewers. Snapshots get cached so a late-joining viewer
			// gets the latest state immediately; hello is informational.
			if (parsed.type === "snapshot") {
				this.latest = parsed as SnapshotEnvelope;
				const viewers = this.ctx.getWebSockets(TAG_VIEWER);
				for (const v of viewers) {
					try { v.send(message); } catch { /* closed */ }
				}
			}
			return;
		}

		if (tags.includes(TAG_VIEWER)) {
			// Viewer → agents. Currently only "command" envelopes are forwarded.
			// Any agent in this room (typically just one) gets the message.
			if (parsed.type === "command") {
				const agents = this.ctx.getWebSockets(TAG_AGENT);
				for (const a of agents) {
					try { a.send(message); } catch { /* closed */ }
				}
			}
		}
	}

	override async webSocketClose(ws: WebSocket, code: number, _reason: string, _wasClean: boolean): Promise<void> {
		try {
			ws.close(code, "closed");
		} catch {
			/* already closed */
		}
	}

	override async webSocketError(ws: WebSocket, _error: unknown): Promise<void> {
		try {
			ws.close(1011, "error");
		} catch {
			/* already closed */
		}
	}
}
