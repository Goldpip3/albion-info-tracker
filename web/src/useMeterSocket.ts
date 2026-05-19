import { useEffect, useRef, useState } from "react";
import type { ConnectionState, Snapshot, Envelope } from "./types.ts";

interface UseMeterSocketArgs {
	url: string;
	token: string;
	enabled: boolean;
}

interface UseMeterSocketResult {
	state: ConnectionState;
	snapshot: Snapshot | null;
	lastMessageAt: number | null;
	error: string | null;
}

// useMeterSocket maintains a single WebSocket to the Worker /view endpoint
// with automatic reconnect on close/error. Returns the latest snapshot and
// a connection state suitable for rendering a status pill.
export function useMeterSocket({ url, token, enabled }: UseMeterSocketArgs): UseMeterSocketResult {
	const [state, setState] = useState<ConnectionState>("disconnected");
	const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
	const [lastMessageAt, setLastMessageAt] = useState<number | null>(null);
	const [error, setError] = useState<string | null>(null);
	const wsRef = useRef<WebSocket | null>(null);
	const reconnectTimerRef = useRef<number | null>(null);
	const backoffRef = useRef<number>(1000);

	useEffect(() => {
		if (!enabled || !url || !token) {
			setState("disconnected");
			return;
		}

		let cancelled = false;

		const connect = (): void => {
			if (cancelled) return;
			setState("connecting");
			setError(null);

			let target: string;
			try {
				const u = new URL(url);
				u.searchParams.set("token", token);
				target = u.toString();
			} catch (e) {
				setError(`bad url: ${String(e)}`);
				setState("disconnected");
				return;
			}

			const ws = new WebSocket(target);
			wsRef.current = ws;

			ws.addEventListener("open", () => {
				if (cancelled) return;
				setState("connected");
				backoffRef.current = 1000; // reset on success
			});

			ws.addEventListener("message", (ev) => {
				if (cancelled) return;
				try {
					const env = JSON.parse(ev.data as string) as Envelope;
					if (env.type === "snapshot" && env.snap) {
						setSnapshot(env.snap);
						setLastMessageAt(Date.now());
					}
				} catch {
					/* malformed payload, skip */
				}
			});

			const onCloseOrError = (e: Event): void => {
				if (cancelled) return;
				wsRef.current = null;
				setState("disconnected");
				if (e.type === "error") setError("connection error");
				// Exponential backoff with jitter, capped at 30s.
				const wait = Math.min(backoffRef.current, 30000);
				backoffRef.current = Math.min(backoffRef.current * 2, 30000);
				reconnectTimerRef.current = window.setTimeout(connect, wait + Math.random() * 500);
			};
			ws.addEventListener("close", onCloseOrError);
			ws.addEventListener("error", onCloseOrError);
		};

		connect();

		return (): void => {
			cancelled = true;
			if (reconnectTimerRef.current !== null) {
				clearTimeout(reconnectTimerRef.current);
				reconnectTimerRef.current = null;
			}
			if (wsRef.current) {
				wsRef.current.close();
				wsRef.current = null;
			}
		};
	}, [url, token, enabled]);

	return { state, snapshot, lastMessageAt, error };
}
