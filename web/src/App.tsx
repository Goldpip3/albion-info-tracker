import { useEffect, useMemo, useState } from "react";
import { useMeterSocket } from "./useMeterSocket.ts";
import { MeterTable } from "./MeterTable.tsx";

type Mode = "damage" | "heal" | "taken";

// Persist connection settings + selected mode in localStorage so a page
// refresh doesn't drop you back to the setup screen.
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
	const [url, setUrl] = useStored("albion-meter:url", "");
	const [token, setToken] = useStored("albion-meter:token", "");
	const [modeStored, setModeStored] = useStored("albion-meter:mode", "damage");
	const mode = (["damage", "heal", "taken"].includes(modeStored) ? modeStored : "damage") as Mode;

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
		<div className="min-h-dvh flex flex-col">
			<header className="flex items-center gap-3 px-3 py-2 border-b border-neutral-900 bg-neutral-950">
				<div className="font-semibold tracking-tight">Albion Meter</div>
				<StatusPill state={state} stale={stale} error={error} />
				<div className="ml-auto flex gap-1 text-xs">
					<ModeButton current={mode} value="damage" onClick={() => setModeStored("damage")}>Dmg</ModeButton>
					<ModeButton current={mode} value="heal"   onClick={() => setModeStored("heal")}>Heal</ModeButton>
					<ModeButton current={mode} value="taken"  onClick={() => setModeStored("taken")}>Taken</ModeButton>
				</div>
				<button
					className="ml-1 text-xs text-neutral-500 hover:text-neutral-300"
					onClick={() => {
						setUrl("");
						setToken("");
					}}
					title="Reset connection settings"
				>
					⚙
				</button>
			</header>

			<main className="flex-1 overflow-auto">
				<MeterTable snapshot={snapshot} mode={mode} />
			</main>

			<footer className="px-3 py-1.5 text-[10px] text-neutral-600 border-t border-neutral-900">
				{snapshot
					? `updated ${new Date(snapshot.generatedAt).toLocaleTimeString()}`
					: "waiting for first snapshot…"}
			</footer>
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
		<div className="min-h-dvh flex items-center justify-center p-4">
			<form
				className="w-full max-w-sm space-y-4 bg-neutral-900 border border-neutral-800 rounded-lg p-6"
				onSubmit={(e) => {
					e.preventDefault();
					setUrl(localUrl.trim());
					setToken(localToken.trim());
				}}
			>
				<div>
					<h1 className="text-xl font-semibold">Albion Meter</h1>
					<p className="text-xs text-neutral-500 mt-1">
						Connect to your Worker to start viewing the live meter.
					</p>
				</div>

				<label className="block text-sm">
					<span className="text-neutral-300">Worker URL</span>
					<input
						type="text"
						value={localUrl}
						onChange={(e) => setLocalUrl(e.target.value)}
						placeholder="wss://albion-meter.<your>.workers.dev/view"
						className="mt-1 block w-full rounded bg-neutral-950 border border-neutral-800 px-2 py-1 text-sm focus:outline-none focus:border-amber-500"
					/>
				</label>

				<label className="block text-sm">
					<span className="text-neutral-300">Token</span>
					<input
						type="password"
						value={localToken}
						onChange={(e) => setLocalToken(e.target.value)}
						placeholder="same token you set in agent.json"
						className="mt-1 block w-full rounded bg-neutral-950 border border-neutral-800 px-2 py-1 text-sm focus:outline-none focus:border-amber-500"
					/>
				</label>

				<button
					type="submit"
					className="w-full bg-amber-500 hover:bg-amber-400 text-neutral-950 font-medium rounded py-1.5 text-sm"
				>
					Connect
				</button>

				<p className="text-[11px] text-neutral-600">
					The token alone authenticates you. Share it between the Go agent
					and this browser; anyone with the token sees the same meter.
				</p>
			</form>
		</div>
	);
}

interface StatusPillProps {
	state: "disconnected" | "connecting" | "connected";
	stale: boolean;
	error: string | null;
}

function StatusPill({ state, stale, error }: StatusPillProps): React.ReactElement {
	const tone =
		state === "connected" && !stale
			? "bg-emerald-500/15 text-emerald-300"
			: state === "connecting"
			? "bg-amber-500/15 text-amber-300"
			: "bg-rose-500/15 text-rose-300";
	const label =
		state === "connected"
			? stale
				? "stale"
				: "live"
			: state === "connecting"
			? "connecting"
			: error ?? "offline";
	return (
		<span className={`px-1.5 py-0.5 rounded text-[10px] uppercase tracking-wider ${tone}`}>
			{label}
		</span>
	);
}

interface ModeButtonProps {
	current: string;
	value: string;
	onClick: () => void;
	children: React.ReactNode;
}

function ModeButton({ current, value, onClick, children }: ModeButtonProps): React.ReactElement {
	const active = current === value;
	return (
		<button
			onClick={onClick}
			className={`px-2 py-0.5 rounded ${
				active
					? "bg-neutral-100 text-neutral-950"
					: "bg-neutral-800 text-neutral-400 hover:text-neutral-100"
			}`}
		>
			{children}
		</button>
	);
}
