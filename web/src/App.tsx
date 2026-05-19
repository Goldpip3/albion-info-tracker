import { useEffect, useMemo, useState } from "react";
import { useMeterSocket } from "./useMeterSocket.ts";
import { MeterTable } from "./MeterTable.tsx";
import { Header } from "./Header.tsx";
import { Footer } from "./Footer.tsx";
import { SettingsPanel } from "./SettingsPanel.tsx";
import { DrillIn } from "./DrillIn.tsx";
import { ActivityLog } from "./ActivityLog.tsx";
import { accentOklch, useSettings } from "./useSettings.ts";
import type { Mode, PlayerSnapshot } from "./types.ts";

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

	const { settings, update, reset } = useSettings();
	const [showSettings, setShowSettings] = useState(false);
	const [drillGuid, setDrillGuid] = useState<string | null>(null);

	const configured = url.trim() !== "" && token.trim() !== "";
	const { state, snapshot, lastMessageAt, error } = useMeterSocket({
		url: url.trim(),
		token: token.trim(),
		enabled: configured,
	});

	// Apply the chosen accent color as CSS variables so every component
	// re-tints without a re-render. Drives --sk-local + --sk-local-tint
	// across the whole tree.
	useEffect(() => {
		const { fg, tint } = accentOklch(settings.accent);
		document.documentElement.style.setProperty("--sk-local", fg);
		document.documentElement.style.setProperty("--sk-local-tint", tint);
	}, [settings.accent]);

	const stale = useMemo(() => {
		if (!lastMessageAt) return false;
		return Date.now() - lastMessageAt > 5000;
	}, [lastMessageAt, state, snapshot]);

	if (!configured) {
		return <SetupScreen url={url} token={token} setUrl={setUrl} setToken={setToken} />;
	}

	return (
		<div className="min-h-dvh flex flex-col" style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-0)" }}>
			<Header
				state={state}
				stale={stale}
				snapshot={snapshot}
				mode={mode}
				setMode={(m) => setModeStored(m)}
				onSettings={() => setShowSettings(true)}
				onReset={() => {
					if (confirm("Disconnect and clear settings?")) {
						setUrl("");
						setToken("");
					}
				}}
			/>
			<main className="flex-1 overflow-hidden flex flex-col">
				<div className="flex-1 overflow-auto">
					<MeterTable
						snapshot={snapshot}
						mode={mode}
						settings={settings}
						onDrillIn={(p: PlayerSnapshot) => setDrillGuid(p.userGuid)}
					/>
				</div>
				{error && (
					<div
						className="px-4 py-2 sk-upper"
						style={{ color: "var(--sk-err)", borderTop: "1px solid var(--sk-line)" }}
					>
						connection: {error}
					</div>
				)}
				<ActivityLog snapshot={snapshot} />
			</main>
			<Footer snapshot={snapshot} lastMessageAt={lastMessageAt} />
			{showSettings && (
				<SettingsPanel
					settings={settings}
					update={update}
					reset={reset}
					onClose={() => setShowSettings(false)}
				/>
			)}
			{drillGuid && snapshot && (() => {
				const p = snapshot.players.find((p) => p.userGuid === drillGuid);
				return p ? <DrillIn player={p} onClose={() => setDrillGuid(null)} /> : null;
			})()}
		</div>
	);
}

interface SetupScreenProps {
	url: string;
	token: string;
	setUrl: (v: string) => void;
	setToken: (v: string) => void;
}

const DEFAULT_VIEW_URL = "wss://albion-meter.goldpipe.workers.dev/view";

function SetupScreen({ url, token, setUrl, setToken }: SetupScreenProps): React.ReactElement {
	const [localUrl, setLocalUrl] = useState(url || DEFAULT_VIEW_URL);
	const [localToken, setLocalToken] = useState(token);
	const [copied, setCopied] = useState(false);

	const generate = (): void => {
		const bytes = new Uint8Array(16);
		crypto.getRandomValues(bytes);
		const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
		setLocalToken(hex);
		try {
			void navigator.clipboard.writeText(hex);
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		} catch {
			/* private mode etc */
		}
	};

	return (
		<div className="min-h-dvh flex items-center justify-center p-4" style={{ background: "var(--sk-bg-0)" }}>
			<form
				className="w-full max-w-sm"
				style={{
					background: "var(--sk-bg-1)",
					border: "1px solid var(--sk-line)",
					borderRadius: 8,
					padding: 22,
				}}
				onSubmit={(e) => {
					e.preventDefault();
					setUrl(localUrl.trim());
					setToken(localToken.trim());
				}}
			>
				<div className="flex items-center mb-2.5" style={{ gap: 7 }}>
					<svg width="18" height="18" viewBox="0 0 18 18" style={{ display: "block" }}>
						<rect x="1.5" y="3"  width="11" height="2" fill="var(--sk-damage)" />
						<rect x="1.5" y="8"  width="15" height="2" fill="var(--sk-fg-0)" />
						<rect x="1.5" y="13" width="6"  height="2" fill="var(--sk-local)" />
					</svg>
					<span
						style={{
							fontFamily: "var(--sk-font-mono)",
							fontSize: 12.5,
							fontWeight: 600,
							letterSpacing: "0.18em",
							color: "var(--sk-fg-0)",
							textTransform: "uppercase",
						}}
					>
						Skirmish
					</span>
				</div>

				<h1 style={{ fontSize: 17, fontWeight: 500, marginBottom: 6, color: "var(--sk-fg-0)" }}>
					Pair your agent
				</h1>
				<p style={{ fontSize: 12, color: "var(--sk-fg-2)", marginBottom: 18, lineHeight: 1.5 }}>
					Generate a token below and paste it into the agent when prompted —
					or generate one in the agent and paste it here. Both ends meet in
					the same backend room.
				</p>

				<label className="block">
					<div className="flex items-baseline justify-between" style={{ marginBottom: 6 }}>
						<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>Pairing token</span>
						<button
							type="button"
							onClick={generate}
							className="sk-upper"
							style={{
								appearance: "none",
								background: "transparent",
								border: 0,
								color: "var(--sk-local)",
								cursor: "pointer",
								padding: 0,
								textDecoration: "underline",
								textUnderlineOffset: 3,
							}}
						>
							{copied ? "✓ Copied" : "Generate"}
						</button>
					</div>
					<input
						type="text"
						value={localToken}
						onChange={(e) => setLocalToken(e.target.value)}
						placeholder="32 hex chars"
						className="sk-mono"
						style={{
							width: "100%",
							background: "var(--sk-bg-0)",
							border: "1px solid var(--sk-line)",
							borderRadius: 4,
							padding: "8px 10px",
							fontSize: 12,
							color: "var(--sk-fg-0)",
							outline: "none",
						}}
					/>
				</label>

				<details style={{ marginTop: 14 }}>
					<summary
						className="sk-upper"
						style={{ color: "var(--sk-fg-3)", cursor: "pointer", listStyle: "none" }}
					>
						Advanced — Worker URL
					</summary>
					<input
						type="text"
						value={localUrl}
						onChange={(e) => setLocalUrl(e.target.value)}
						style={{
							marginTop: 8,
							width: "100%",
							background: "var(--sk-bg-0)",
							border: "1px solid var(--sk-line)",
							borderRadius: 4,
							padding: "6px 9px",
							fontSize: 11,
							color: "var(--sk-fg-1)",
							outline: "none",
							fontFamily: "var(--sk-font-mono)",
						}}
					/>
				</details>

				<button
					type="submit"
					disabled={!localToken.trim()}
					style={{
						marginTop: 20,
						width: "100%",
						appearance: "none",
						background: "var(--sk-local)",
						color: "var(--sk-bg-0)",
						border: 0,
						borderRadius: 4,
						padding: "9px 12px",
						fontSize: 12.5,
						fontWeight: 600,
						cursor: localToken.trim() ? "pointer" : "not-allowed",
						opacity: localToken.trim() ? 1 : 0.5,
					}}
				>
					Connect
				</button>

				<p style={{ marginTop: 14, fontSize: 11, color: "var(--sk-fg-3)", lineHeight: 1.5 }}>
					Anyone with this token sees the same meter room. Keep it private
					unless you want to share.
				</p>
			</form>
		</div>
	);
}
