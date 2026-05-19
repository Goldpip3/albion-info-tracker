import { useEffect, useMemo, useState } from "react";
import { useMeterSocket } from "./useMeterSocket.ts";
import { MeterTable } from "./MeterTable.tsx";
import { Header } from "./Header.tsx";
import { Footer } from "./Footer.tsx";
import type { Mode } from "./types.ts";

function useStored(key: string, initial: string): [string, (v: string) => void] {
	const [v, setV] = useState<string>(() => {
		try {
			return localStorage.getItem(key) ?? initial;
		} catch {
			return initial;
		}
	});
	useEffect(() => {
		try {
			localStorage.setItem(key, v);
		} catch {
			/* private mode etc. */
		}
	}, [key, v]);
	return [v, setV];
}

export default function App(): React.ReactElement {
	const [url, setUrl] = useStored("skirmish:url", "");
	const [token, setToken] = useStored("skirmish:token", "");
	const [modeStored, setModeStored] = useStored("skirmish:mode", "damage");
	const mode = (["damage", "heal", "taken", "mechanics"].includes(modeStored)
		? modeStored
		: "damage") as Mode;

	const configured = url.trim() !== "" && token.trim() !== "";
	const { state, snapshot, lastMessageAt, error } = useMeterSocket({
		url: url.trim(),
		token: token.trim(),
		enabled: configured,
	});

	const stale = useMemo(() => {
		if (!lastMessageAt) return false;
		return Date.now() - lastMessageAt > 5000;
	}, [lastMessageAt, state, snapshot]);

	if (!configured) {
		return <SetupScreen url={url} token={token} setUrl={setUrl} setToken={setToken} />;
	}

	return (
		<div className="min-h-dvh flex flex-col bg-skirmish-bg text-skirmish-text">
			<Header
				state={state}
				stale={stale}
				snapshot={snapshot}
				mode={mode}
				setMode={(m) => setModeStored(m)}
				onSettings={() => alert("Settings panel — coming in T2")}
				onReset={() => {
					if (confirm("Disconnect and clear settings?")) {
						setUrl("");
						setToken("");
					}
				}}
			/>
			<main className="flex-1 overflow-auto">
				<MeterTable snapshot={snapshot} mode={mode} />
				{error && (
					<div className="px-4 py-2 text-xs text-rose-300">connection: {error}</div>
				)}
			</main>
			<Footer snapshot={snapshot} lastMessageAt={lastMessageAt} />
		</div>
	);
}

interface SetupScreenProps {
	url: string;
	token: string;
	setUrl: (v: string) => void;
	setToken: (v: string) => void;
}

function SetupScreen({ url, token, setUrl, setToken }: SetupScreenProps): React.ReactElement {
	const [localUrl, setLocalUrl] = useState(url);
	const [localToken, setLocalToken] = useState(token);

	return (
		<div className="min-h-dvh flex items-center justify-center p-4 bg-skirmish-bg">
			<form
				className="w-full max-w-sm space-y-4 bg-skirmish-bg2 border border-skirmish-line rounded-md p-6"
				onSubmit={(e) => {
					e.preventDefault();
					setUrl(localUrl.trim());
					setToken(localToken.trim());
				}}
			>
				<div className="space-y-1">
					<div className="text-xs tracking-[0.25em] uppercase text-skirmish-amber font-semibold">
						SKIRMISH
					</div>
					<h1 className="text-lg font-medium text-skirmish-text">Pair your agent</h1>
					<p className="text-xs text-skirmish-dim">
						The local capture agent shares a token with this page. Both end up in
						the same room on the backend.
					</p>
				</div>

				<label className="block text-sm">
					<span className="text-[10px] uppercase tracking-[0.15em] text-skirmish-muted">Worker URL</span>
					<input
						type="text"
						value={localUrl}
						onChange={(e) => setLocalUrl(e.target.value)}
						placeholder="wss://albion-meter.<you>.workers.dev/view"
						className="mt-1 block w-full rounded bg-skirmish-bg border border-skirmish-line px-2 py-1.5 text-sm focus:outline-none focus:border-skirmish-amber"
					/>
				</label>

				<label className="block text-sm">
					<span className="text-[10px] uppercase tracking-[0.15em] text-skirmish-muted">Pairing token</span>
					<input
						type="password"
						value={localToken}
						onChange={(e) => setLocalToken(e.target.value)}
						placeholder="same token as agent.json pushToken"
						className="mt-1 block w-full rounded bg-skirmish-bg border border-skirmish-line px-2 py-1.5 text-sm focus:outline-none focus:border-skirmish-amber"
					/>
				</label>

				<button
					type="submit"
					className="w-full bg-skirmish-amber hover:bg-skirmish-amber-dim text-skirmish-bg font-medium rounded py-1.5 text-sm transition"
				>
					Connect
				</button>
			</form>
		</div>
	);
}
