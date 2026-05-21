import { useEffect, useMemo, useState } from "react";
import { useMeterSocket } from "./useMeterSocket.ts";
import { MeterTable } from "./MeterTable.tsx";
import { Header } from "./Header.tsx";
import { Footer } from "./Footer.tsx";
import { SettingsPanel } from "./SettingsPanel.tsx";
import { PlayerPicker } from "./PlayerPicker.tsx";
import { DrillIn } from "./DrillIn.tsx";
import { ActivityLog } from "./ActivityLog.tsx";
import { SessionStrip } from "./SessionStrip.tsx";
import { DungeonStrip } from "./DungeonStrip.tsx";
import { LootBody } from "./LootBody.tsx";
import { PartyBody } from "./PartyBody.tsx";
import { SessionsBody } from "./SessionsBody.tsx";
import { ProgressChart } from "./ProgressChart.tsx";
import { TabBar } from "./TabBar.tsx";
import { tabsFor, type TabId } from "./tabs.ts";
import { isDemoMode, useDemoSnapshot } from "./demo.ts";
import { accentOklch, useSettings, type Settings } from "./useSettings.ts";
import type { FightArchive, Mode, PlayerSnapshot, Snapshot, SubMetric } from "./types.ts";

// PANES is the fixed catalog of meter panes — one per (metric, scope).
// Listing Current and Session adjacent lets the user open both for a
// metric and watch the current-fight leader beside the session leader.
const PANES: Array<{ sub: SubMetric; mode: Exclude<Mode, "mechanics">; label: string; tone: string }> = [
	{ sub: "damageCurrent", mode: "damage", label: "Damage · Current", tone: "var(--sk-damage)" },
	{ sub: "damageTotal",   mode: "damage", label: "Damage · Session", tone: "var(--sk-damage)" },
	{ sub: "healCurrent",   mode: "heal",   label: "Healing · Current", tone: "var(--sk-heal)" },
	{ sub: "healTotal",     mode: "heal",   label: "Healing · Session", tone: "var(--sk-heal)" },
	{ sub: "takenCurrent",  mode: "taken",  label: "Tank · Current",    tone: "var(--sk-taken)" },
	{ sub: "takenTotal",    mode: "taken",  label: "Tank · Session",    tone: "var(--sk-taken)" },
];

// MeterPanes renders every enabled pane as a rounded card. Each pane is a
// MeterTable locked to one SubMetric, with a colored header bar naming it.
// The arrangement follows settings.paneLayout:
//   auto    — responsive card grid that fills the area for 1–2 panes and
//             wraps into a tidy 2×N grid for 3–4 (380px anti-crush floor).
//   columns — side-by-side, horizontal-scrolls instead of crushing.
//   rows    — full-width stacked cards.
// Replaces the old fixed `repeat(N,1fr)` grid, which crushed each table
// below its ~290px content width once 2–3 panes were open.
function MeterPanes({ meterSnapshot, settings, onDrillIn }: {
	meterSnapshot: Snapshot | null;
	settings: Settings;
	onDrillIn: (p: PlayerSnapshot) => void;
}): React.ReactElement {
	const active = PANES.filter((p) => settings.panes[p.sub]);
	if (active.length === 0) active.push(PANES[0]); // defensive: never blank
	const layout = settings.paneLayout;

	const outerClass = layout === "columns" ? "flex-1 overflow-hidden" : "flex-1 overflow-y-auto";
	const container: React.CSSProperties =
		layout === "columns"
			? { display: "flex", flexDirection: "row", flexWrap: "nowrap", gap: 10, padding: 10, height: "100%", overflowX: "auto" }
			: layout === "rows"
			? { display: "flex", flexDirection: "column", gap: 10, padding: 10, minHeight: "100%" }
			: { display: "flex", flexWrap: "wrap", gap: 10, padding: 10, minHeight: "100%", alignContent: "stretch" };
	const paneFlex = layout === "columns" ? "1 1 0" : layout === "rows" ? "1 1 240px" : "1 1 380px";

	return (
		<div className={outerClass}>
			<div style={container}>
				{active.map((p) => (
					<div
						key={p.sub}
						className="flex flex-col"
						style={{
							flex: paneFlex,
							minWidth: layout === "columns" ? 320 : 0,
							minHeight: 240,
							border: "1px solid var(--sk-line)",
							borderRadius: "var(--sk-radius-lg, 10px)",
							background: "var(--sk-bg-1)",
							overflow: "hidden",
							boxShadow: "0 1px 3px rgba(0,0,0,0.18)",
						}}
					>
						<div
							className="sk-upper flex items-center"
							style={{
								padding: "8px 14px",
								borderBottom: "1px solid var(--sk-line)",
								color: p.tone,
								fontWeight: 600,
								background: "var(--sk-bg-2)",
								gap: 8,
							}}
						>
							<span style={{ width: 6, height: 6, borderRadius: 99, background: p.tone }} />
							{p.label}
						</div>
						<div className="flex-1 overflow-auto">
							<MeterTable snapshot={meterSnapshot} mode={p.mode} sub={p.sub} settings={settings} onDrillIn={onDrillIn} />
						</div>
					</div>
				))}
			</div>
		</div>
	);
}

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

const DEFAULT_VIEW_URL = "wss://albion-meter.goldpipe.workers.dev/view";

// isLocalMode is true when the meter is talking to a local agent rather than
// the Cloudflare-hosted site — either served by the agent's own server
// (http://localhost:<port>) or bundled inside the GDA desktop shell
// (tauri.localhost / the tauri: scheme). In both cases we connect straight
// to the agent's /view WebSocket — no pairing token, no Cloudflare in the
// data path, so it never touches the daily request budget.
function isLocalMode(): boolean {
	const h = window.location.hostname;
	return (
		h === "localhost" || h === "127.0.0.1" || h === "::1" ||
		h === "tauri.localhost" || window.location.protocol === "tauri:"
	);
}

// localViewWsUrl derives the agent's /view WebSocket URL. When the page was
// served by the agent itself it's the same origin; inside the desktop shell
// the bundle loads from tauri.localhost, so we point at the agent's fixed
// loopback port (8787, the agent default).
function localViewWsUrl(): string {
	const h = window.location.hostname;
	if (h === "localhost" || h === "127.0.0.1" || h === "::1") {
		const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
		return `${proto}//${window.location.host}/view`;
	}
	return "ws://localhost:8787/view";
}

// readPairFromURL returns ?pair=<token> from the current URL and strips the
// query param so a refresh / bookmark doesn't keep showing it.
function readPairFromURL(): string | null {
	try {
		const params = new URLSearchParams(window.location.search);
		const pair = params.get("pair");
		if (!pair) return null;
		// Strip the param so the URL stays clean. replaceState avoids a
		// navigation but updates the address bar + bookmarks behaviour.
		params.delete("pair");
		const qs = params.toString();
		const newUrl = window.location.pathname + (qs ? "?" + qs : "") + window.location.hash;
		window.history.replaceState({}, "", newUrl);
		return pair.trim();
	} catch {
		return null;
	}
}

export default function App(): React.ReactElement {
	// Single mounted shell. The pathname picks the body to render
	// inside LiveApp / DemoApp without unmounting the WebSocket or
	// scroll position. Soft-nav from TabBar dispatches popstate, the
	// listener below flips the local path and re-renders the body.
	const [path, setPath] = useState<string>(() =>
		typeof window !== "undefined" ? window.location.pathname.replace(/\/$/, "") : "",
	);
	useEffect(() => {
		if (typeof window === "undefined") return;
		const onPop = (): void => setPath(window.location.pathname.replace(/\/$/, ""));
		window.addEventListener("popstate", onPop);
		return () => window.removeEventListener("popstate", onPop);
	}, []);

	const demo = isDemoMode();
	if (demo) return <DemoApp path={path} />;
	return <LiveApp path={path} />;
}

function pathToTabId(path: string): TabId {
	if (path === "/loot")     return "loot";
	if (path === "/party")    return "party";
	if (path === "/sessions") return "sessions";
	return "meter";
}

function LiveApp({ path }: { path: string }): React.ReactElement {
	const [url, setUrl] = useStored("skirmish:url", "");
	const [token, setToken] = useStored("skirmish:token", "");
	const tab = pathToTabId(path);

	// Auto-pair from a magic link generated by the local agent on first run.
	// /?pair=<32-hex-token> wires the meter to that agent without anyone
	// typing a token. Runs exactly once on mount; the URL is cleaned even
	// when the token in the link matches what's already stored.
	useEffect(() => {
		const fromURL = readPairFromURL();
		if (!fromURL) return;
		if (!url) setUrl(DEFAULT_VIEW_URL);
		if (fromURL !== token) setToken(fromURL);
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, []);

	const { settings, update, reset } = useSettings();
	const [showSettings, setShowSettings] = useState(false);
	const [showPicker, setShowPicker] = useState(false);
	const [drillGuid, setDrillGuid] = useState<string | null>(null);
	const [viewingFight, setViewingFight] = useState<number | null>(null);

	// Local mode (served by the agent at localhost): connect straight to the
	// agent's /view socket on this origin, no token, no Cloudflare. Otherwise
	// fall back to the saved Cloudflare URL + pairing token.
	const localMode = isLocalMode();
	const configured = localMode || (url.trim() !== "" && token.trim() !== "");
	const { state, snapshot, lastMessageAt, error, sendCommand } = useMeterSocket({
		url: localMode ? localViewWsUrl() : url.trim(),
		token: localMode ? "local" : token.trim(),
		enabled: configured,
	});

	const onNewSession = (): void => {
		if (!confirm("Reset the session? This wipes combat stats, fight count, fame/silver/respec totals, and the activity log on the agent.")) {
			return;
		}
		const ok = sendCommand("resetSession");
		if (!ok) {
			alert("Couldn't reach the agent — connection isn't open. Make sure agent.exe is running.");
		}
	};

	const onClearMeter = (): void => {
		if (!confirm("Clear the meter? This drops everyone else off the meter and clears the party roster. Your own numbers and the session totals are kept.")) {
			return;
		}
		const ok = sendCommand("clearMeter");
		if (!ok) {
			alert("Couldn't reach the agent — connection isn't open. Make sure agent.exe is running.");
		}
	};

	// Apply the chosen accent color as CSS variables so every component
	// re-tints without a re-render. Drives --sk-local + --sk-local-tint
	// across the whole tree.
	useEffect(() => {
		const { fg, tint } = accentOklch(settings.accent);
		document.documentElement.style.setProperty("--sk-local", fg);
		document.documentElement.style.setProperty("--sk-local-tint", tint);
	}, [settings.accent]);

	// Sync the agent's meter scope whenever the connection opens or
	// the user changes the scope control. Without this, a returning
	// user whose saved scope differs from the agent's partyGuild
	// default would see the wrong set until they re-picked the scope.
	useEffect(() => {
		if (state !== "connected") return;
		sendCommand("setLootFilter", settings.meterScope);
	}, [state, settings.meterScope, sendCommand]);

	// Global hotkeys. Stay out of the way when the user is typing in
	// an input or holding a modifier — those are browser / OS chords
	// and we don't want to steal them.
	useEffect(() => {
		const onKey = (e: KeyboardEvent): void => {
			if (e.metaKey || e.ctrlKey || e.altKey) return;
			const target = e.target as HTMLElement | null;
			const tag = target?.tagName;
			if (tag === "INPUT" || tag === "TEXTAREA" || target?.isContentEditable) return;
			let dest: string | null = null;
			switch (e.key.toLowerCase()) {
				case "m": dest = "/"; break;
				case "l": dest = "/loot"; break;
				case "p": dest = "/party"; break;
				case "s": dest = "/sessions"; break;
				case "escape":
					if (drillGuid) {
						e.preventDefault();
						setDrillGuid(null);
					} else if (showPicker) {
						e.preventDefault();
						setShowPicker(false);
					} else if (showSettings) {
						e.preventDefault();
						setShowSettings(false);
					}
					return;
				case "d":
				case "h":
				case "t": {
					// Toggle the Current pane of each metric (the primary one).
					const map: Record<string, SubMetric> = { d: "damageCurrent", h: "healCurrent", t: "takenCurrent" };
					const key = map[e.key.toLowerCase()];
					const next = { ...settings.panes, [key]: !settings.panes[key] };
					if (Object.values(next).some(Boolean)) {
						e.preventDefault();
						update("panes", next);
					}
					return;
				}
				default:
					return;
			}
			if (dest) {
				e.preventDefault();
				window.history.pushState({}, "", dest);
				window.dispatchEvent(new PopStateEvent("popstate"));
			}
		};
		window.addEventListener("keydown", onKey);
		return () => window.removeEventListener("keydown", onKey);
	}, [drillGuid, showPicker, showSettings, settings.panes, update]);

	const stale = useMemo(() => {
		if (!lastMessageAt) return false;
		// 70s grace: must exceed the agent's 60s idle heartbeat (see Footer)
		// so a healthy-but-idle agent isn't falsely flagged stale.
		return Date.now() - lastMessageAt > 70000;
	}, [lastMessageAt, state, snapshot]);

	if (!configured) {
		return <SetupScreen url={url} token={token} setUrl={setUrl} setToken={setToken} />;
	}

	// When the user picks a past fight from the history dropdown, swap the
	// snapshot the meter renders. The session strip + agent pill always
	// reflect the live state; only the table + drill-in are frozen.
	const archived = viewingFight !== null
		? snapshot?.recent?.find((r) => r.number === viewingFight)
		: undefined;
	const meterSnapshot: Snapshot | null = archived
		? archiveToSnapshot(archived, snapshot)
		: snapshot;

	return (
		<div className="min-h-dvh flex flex-col" style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-0)" }}>
			<Header
				state={state}
				stale={stale}
				snapshot={snapshot}
				panes={settings.panes}
				togglePane={(key) => {
					const next = { ...settings.panes, [key]: !settings.panes[key] };
					const stillOn = Object.values(next).some(Boolean);
					if (stillOn) update("panes", next);
				}}
				viewingFight={viewingFight}
				setViewingFight={setViewingFight}
				onSettings={() => setShowSettings(true)}
				onNewSession={onNewSession}
				onClearMeter={onClearMeter}
				onReset={() => {
					if (confirm("Disconnect and clear settings?")) {
						setUrl("");
						setToken("");
					}
				}}
				onToggleLog={() => update("showActivityLog", !settings.showActivityLog)}
				showLog={settings.showActivityLog}
				meterScope={settings.meterScope}
				onScopeChange={(s) => {
					update("meterScope", s);
					sendCommand("setLootFilter", s);
				}}
			/>
			<TabBar tabs={tabsFor(snapshot, token)} active={tab} />
			{tab === "meter" && <SessionStrip snapshot={snapshot} />}
			{tab === "meter" && <DungeonStrip dungeon={snapshot?.dungeon} />}
			{tab === "meter" && (
				<div
					className="flex items-center justify-end"
					style={{ gap: 10, padding: "6px 16px", borderBottom: "1px solid var(--sk-line)", background: "var(--sk-bg-1)" }}
				>
					<button
						onClick={() => setShowPicker(true)}
						className="sk-upper"
						title="Pick which players the meter tracks"
						style={{
							appearance: "none",
							border: "1px solid var(--sk-line)",
							background: "var(--sk-bg-2)",
							color: "var(--sk-fg-1)",
							padding: "4px 12px",
							borderRadius: 5,
							cursor: "pointer",
							fontSize: 10,
							fontWeight: 700,
							letterSpacing: "0.06em",
						}}
					>
						Track Players{snapshot?.roster?.length ? ` · ${snapshot.roster.length}` : ""}
					</button>
				</div>
			)}
			<main className="flex-1 overflow-hidden flex flex-col">
				{tab === "loot" && (
					<LootBody
						loot={snapshot?.loot ?? []}
						looterTotals={snapshot?.looterTotals ?? []}
						players={snapshot?.players ?? []}
						session={snapshot?.session ?? null}
						generatedAt={snapshot?.generatedAt}
					/>
				)}
				{tab === "party" && (
					<PartyBody
						players={snapshot?.players ?? []}
						generatedAt={snapshot?.generatedAt}
						visiblePlayers={snapshot?.visiblePlayers ?? []}
						sendCommand={sendCommand}
					/>
				)}
				{tab === "sessions" && (
					<>
						<ProgressChart daily={snapshot?.daily ?? []} liveSession={snapshot?.session ?? undefined} />
						<SessionsBody
							sessions={snapshot?.sessions ?? []}
							onDelete={(id) => sendCommand("deleteSession", id)}
						/>
					</>
				)}
				{tab === "meter" && (
					<MeterPanes
						meterSnapshot={meterSnapshot}
						settings={settings}
						onDrillIn={(p: PlayerSnapshot) => setDrillGuid(p.userGuid)}
					/>
				)}
				{error && (
					<div
						className="px-4 py-2 sk-upper"
						style={{ color: "var(--sk-err)", borderTop: "1px solid var(--sk-line)" }}
					>
						connection: {error}
					</div>
				)}
				{settings.showActivityLog && <ActivityLog snapshot={snapshot} />}
			</main>
			<Footer snapshot={snapshot} lastMessageAt={lastMessageAt} />
			{showSettings && (
				<SettingsPanel
					settings={settings}
					update={update}
					reset={reset}
					onClose={() => setShowSettings(false)}
					sendCommand={sendCommand}
				/>
			)}
			{showPicker && (
				<PlayerPicker
					roster={snapshot?.roster ?? []}
					onClose={() => setShowPicker(false)}
					sendCommand={sendCommand}
				/>
			)}
			{drillGuid && meterSnapshot && (() => {
				const p = meterSnapshot.players.find((p) => p.userGuid === drillGuid);
				return p ? <DrillIn player={p} onClose={() => setDrillGuid(null)} /> : null;
			})()}
		</div>
	);
}

// DemoApp renders the meter using a hand-crafted snapshot from demo.ts
// — no agent, no WebSocket. Activates when the URL has ?demo=1.
// Useful for verifying class chips, role labels, per-class bar colours,
// and the FarmStrip with realistic data even when not in-game.
function DemoApp({ path }: { path: string }): React.ReactElement {
	const { settings, update, reset } = useSettings();
	const [showSettings, setShowSettings] = useState(false);
	const [drillGuid, setDrillGuid] = useState<string | null>(null);
	const [viewingFight, setViewingFight] = useState<number | null>(null);
	const { snapshot, lastMessageAt } = useDemoSnapshot();
	const tab = pathToTabId(path);

	useEffect(() => {
		const { fg, tint } = accentOklch(settings.accent);
		document.documentElement.style.setProperty("--sk-local", fg);
		document.documentElement.style.setProperty("--sk-local-tint", tint);
	}, [settings.accent]);


	return (
		<div className="min-h-dvh flex flex-col" style={{ background: "var(--sk-bg-0)", color: "var(--sk-fg-0)" }}>
			<Header
				state="connected"
				stale={false}
				snapshot={snapshot}
				panes={settings.panes}
				togglePane={(key) => {
					const next = { ...settings.panes, [key]: !settings.panes[key] };
					const stillOn = Object.values(next).some(Boolean);
					if (stillOn) update("panes", next);
				}}
				viewingFight={viewingFight}
				setViewingFight={setViewingFight}
				onSettings={() => setShowSettings(true)}
				onNewSession={() => {/* demo no-op */}}
				onClearMeter={() => {/* demo no-op */}}
				onReset={() => {/* demo no-op */}}
				onToggleLog={() => update("showActivityLog", !settings.showActivityLog)}
				showLog={settings.showActivityLog}
				meterScope={settings.meterScope}
				onScopeChange={(s) => update("meterScope", s)}
			/>
			<TabBar tabs={tabsFor(snapshot, "")} active={tab} />
			{tab === "meter" && <SessionStrip snapshot={snapshot} />}
			{tab === "meter" && <DungeonStrip dungeon={snapshot?.dungeon} />}
			<main className="flex-1 overflow-hidden flex flex-col">
				{tab === "loot" && (
					<LootBody
						loot={snapshot?.loot ?? []}
						looterTotals={snapshot?.looterTotals ?? []}
						players={snapshot?.players ?? []}
						session={snapshot?.session ?? null}
						generatedAt={snapshot?.generatedAt}
					/>
				)}
				{tab === "party" && (
					<PartyBody
						players={snapshot?.players ?? []}
						generatedAt={snapshot?.generatedAt}
						visiblePlayers={snapshot?.visiblePlayers ?? []}
					/>
				)}
				{tab === "sessions" && (
					<>
						<ProgressChart daily={snapshot?.daily ?? []} liveSession={snapshot?.session ?? undefined} />
						<SessionsBody sessions={snapshot?.sessions ?? []} onDelete={() => {/* demo no-op */}} />
					</>
				)}
				{tab === "meter" && (
					<MeterPanes
						meterSnapshot={snapshot}
						settings={settings}
						onDrillIn={(p: PlayerSnapshot) => setDrillGuid(p.userGuid)}
					/>
				)}
				{settings.showActivityLog && <ActivityLog snapshot={snapshot} />}
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
			{drillGuid && (() => {
				const p = snapshot.players.find((p) => p.userGuid === drillGuid);
				return p ? <DrillIn player={p} onClose={() => setDrillGuid(null)} /> : null;
			})()}
		</div>
	);
}

// archiveToSnapshot wraps a FightArchive in a Snapshot-shaped object so
// MeterTable / DrillIn render past fights without their own code path.
// The wrapper carries over the live session block + composition for
// chrome that should always reflect "now"; only players + fight are
// derived from the archive.
function archiveToSnapshot(arch: FightArchive, live: Snapshot | null): Snapshot {
	const players: PlayerSnapshot[] = arch.players.map((p) => ({
		userGuid: p.userGuid,
		name: p.name,
		classCode: p.classCode,
		role: p.role,
		roleLabel: p.roleLabel,
		isLocal: p.isLocal,
		deaths: p.deaths,
		overheal: p.overheal,
		currentDamage: p.damage,
		currentDps: p.dps,
		overallDamage: p.damage,
		overallDps: p.dps,
		currentHeal: p.heal,
		currentHps: p.hps,
		overallHeal: p.heal,
		overallHps: p.hps,
		currentTaken: p.taken,
		overallTaken: p.taken,
		spells: p.spells,
	}));
	return {
		generatedAt: arch.endedAt,
		players,
		composition: live?.composition,
		fight: {
			number: arch.number,
			elapsedMs: arch.durationMs,
			inCombat: false,
		},
		session: live?.session,
		recent: live?.recent,
		events: live?.events,
	};
}

interface SetupScreenProps {
	url: string;
	token: string;
	setUrl: (v: string) => void;
	setToken: (v: string) => void;
}

function SetupScreen({ url, token, setUrl, setToken }: SetupScreenProps): React.ReactElement {
	const [localUrl, setLocalUrl] = useState(url || DEFAULT_VIEW_URL);
	const [localToken, setLocalToken] = useState(token);

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
				<div className="flex items-center mb-2.5" style={{ gap: 10 }}>
					<img
						src="/assets/icon-GDA-64.png"
						srcSet="/assets/icon-GDA-32.png 1x, /assets/icon-GDA-64.png 2x, /assets/icon-GDA-128.png 4x"
						alt="GDA"
						width={22}
						height={22}
						style={{ display: "block", borderRadius: 5 }}
					/>
					<span
						className="sk-mono"
						style={{
							fontSize: 14,
							fontWeight: 600,
							letterSpacing: "0.04em",
							color: "var(--sk-fg-0)",
						}}
					>
						gda
					</span>
				</div>

				<h1 style={{ fontSize: 17, fontWeight: 500, marginBottom: 6, color: "var(--sk-fg-0)" }}>
					Manual pairing
				</h1>
				<p style={{ fontSize: 12, color: "var(--sk-fg-2)", marginBottom: 18, lineHeight: 1.5 }}>
					You usually don't need this screen — the agent opens this page for
					you on first run and auto-pairs. Use this only if you closed that
					tab, or you want to point a second browser at the same agent. Copy
					the <span className="sk-mono" style={{ color: "var(--sk-fg-0)" }}>pushToken</span> value from
					your <span className="sk-mono" style={{ color: "var(--sk-fg-0)" }}>agent/agent.json</span>.
				</p>

				<label className="block">
					<div className="flex items-baseline justify-between" style={{ marginBottom: 6 }}>
						<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>Pairing token</span>
					</div>
					<input
						type="text"
						value={localToken}
						onChange={(e) => setLocalToken(e.target.value)}
						placeholder="paste pushToken from agent.json"
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
