import React from "react";
import type { ConnectionState, Snapshot } from "./types.ts";
import type { PaneSet } from "./useSettings.ts";
import { fmtDuration, fmtRate } from "./format.ts";

interface HeaderProps {
	state: ConnectionState;
	stale: boolean;
	snapshot: Snapshot | null;
	panes: PaneSet;
	togglePane: (m: keyof PaneSet) => void;
	viewingFight: number | null;
	setViewingFight: (n: number | null) => void;
	onSettings: () => void;
	onNewSession: () => void;
	onReset: () => void;
	onToggleLog: () => void;
	showLog: boolean;
}

// Header = TitleBar (dynamic per-user name + agent status + controls)
// stacked on top of FightHeader (combat state + duration + fight label +
// tab-style pane toggles + party DPS + top + composition).
export function Header({
	state, stale, snapshot, panes, togglePane,
	viewingFight, setViewingFight,
	onSettings, onNewSession, onReset, onToggleLog, showLog,
}: HeaderProps): React.ReactElement {
	const localPlayer = snapshot?.players.find((p) => p.isLocal);
	const localName = localPlayer?.name;

	const docTitle = localName ? `${localName}'s Data Analytics` : "Combat Analytics";
	if (typeof document !== "undefined" && document.title !== docTitle) {
		document.title = docTitle;
	}

	return (
		<div>
			<TitleBar
				localName={localName}
				state={state}
				stale={stale}
				onSettings={onSettings}
				onNewSession={onNewSession}
				onReset={onReset}
				onToggleLog={onToggleLog}
				showLog={showLog}
			/>
			<FightHeader
				snapshot={snapshot}
				panes={panes}
				togglePane={togglePane}
				viewingFight={viewingFight}
				setViewingFight={setViewingFight}
			/>
		</div>
	);
}

interface TitleBarProps {
	localName?: string;
	state: ConnectionState;
	stale: boolean;
	onSettings: () => void;
	onNewSession: () => void;
	onReset: () => void;
	onToggleLog: () => void;
	showLog: boolean;
}

function TitleBar({
	localName, state, stale, onSettings, onNewSession, onReset, onToggleLog, showLog,
}: TitleBarProps): React.ReactElement {
	const breadcrumb = localName ? `meter / ${localName.toLowerCase()}` : "meter / (waiting)";
	return (
		<div
			className="flex items-center justify-between"
			style={{
				padding: "0 16px",
				height: 44,
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-inset)",
			}}
		>
			<div className="flex items-center min-w-0" style={{ gap: 14 }}>
				<div className="flex items-center" style={{ gap: 10 }}>
					<img
						src="/assets/icon-GDA-64.png"
						srcSet="/assets/icon-GDA-32.png 1x, /assets/icon-GDA-64.png 2x, /assets/icon-GDA-128.png 4x"
						alt="GDA"
						width={22}
						height={22}
						style={{
							display: "block",
							borderRadius: 5,
							flex: "0 0 auto",
						}}
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
				<span style={{ fontFamily: "var(--sk-font-sans)", fontSize: 12, color: "var(--sk-fg-1)" }}>
					Combat Meter
				</span>
				<span style={{ width: 1, height: 16, background: "var(--sk-line)" }} />
				<span
					className="sk-mono truncate"
					style={{ fontSize: 11, color: "var(--sk-fg-2)", letterSpacing: "0.04em" }}
				>
					{breadcrumb}
				</span>
			</div>
			<div className="flex items-center" style={{ gap: 14 }}>
				<AgentPill state={state} stale={stale} />
				<button
					onClick={onNewSession}
					className="sk-upper"
					style={{
						appearance: "none",
						border: "1px solid var(--sk-line-2)",
						background: "var(--sk-bg-3)",
						color: "var(--sk-fg-0)",
						padding: "4px 9px",
						borderRadius: 4,
						cursor: "pointer",
						fontSize: 10,
						fontWeight: 600,
						letterSpacing: "0.08em",
					}}
					title="Reset all session counters"
				>
					New session
				</button>
				<IconToggle onClick={onToggleLog} active={showLog} title="Activity log">≡</IconToggle>
				<IconBtn onClick={onSettings} title="Settings">⋯</IconBtn>
				<IconBtn onClick={onReset} title="Disconnect">⟲</IconBtn>
			</div>
		</div>
	);
}

function AgentPill({ state, stale }: { state: ConnectionState; stale: boolean }): React.ReactElement {
	const isLive = state === "connected" && !stale;
	const dotColor = isLive
		? "var(--sk-ok)"
		: state === "connecting"
		? "var(--sk-warn)"
		: "var(--sk-err)";
	const liveLabel = isLive
		? "live"
		: state === "connecting"
		? "dialing"
		: stale
		? "stale"
		: "offline";
	const sub = isLive ? "agent · paired" : state === "connecting" ? "agent · pairing" : "agent · disconnected";
	return (
		<span className="inline-flex items-center" style={{ gap: 12 }}>
			<span
				className="sk-mono inline-flex items-center"
				style={{
					gap: 6,
					fontSize: 11,
					color: isLive ? "var(--sk-ok)" : dotColor,
					letterSpacing: "0.04em",
				}}
			>
				<span
					style={{
						width: 6,
						height: 6,
						borderRadius: 99,
						background: dotColor,
						animation: isLive ? "sk-pulse 1.6s var(--sk-ease) infinite" : "none",
					}}
				/>
				{liveLabel}
			</span>
			<span
				className="sk-mono"
				style={{ fontSize: 11, color: "var(--sk-fg-2)", letterSpacing: "0.04em" }}
			>
				{sub}
			</span>
		</span>
	);
}

interface FightHeaderProps {
	snapshot: Snapshot | null;
	panes: PaneSet;
	togglePane: (m: keyof PaneSet) => void;
	viewingFight: number | null;
	setViewingFight: (n: number | null) => void;
}

const TABS: Array<{ id: keyof PaneSet; label: string }> = [
	{ id: "damage", label: "Damage" },
	{ id: "heal",   label: "Healing" },
	{ id: "taken",  label: "Taken" },
];

function FightHeader({ snapshot, panes, togglePane, viewingFight, setViewingFight }: FightHeaderProps): React.ReactElement {
	const liveInCombat = snapshot?.fight?.inCombat ?? false;
	const liveElapsedSec = (snapshot?.fight?.elapsedMs ?? 0) / 1000;
	const liveFightN = snapshot?.fight?.number ?? 0;
	const recent = snapshot?.recent ?? [];

	const isViewingPast = viewingFight !== null;
	const pastArch = isViewingPast ? recent.find((r) => r.number === viewingFight) : undefined;
	const inCombat = isViewingPast ? false : liveInCombat;
	const elapsedSec = isViewingPast ? (pastArch?.durationMs ?? 0) / 1000 : liveElapsedSec;
	const fightLabel = isViewingPast ? (viewingFight ?? 0) : liveFightN;

	const players = snapshot?.players ?? [];
	const partyDps = players.reduce((s, p) => s + (p.currentDps ?? 0), 0);
	const top = [...players].sort((a, b) => (b.currentDamage ?? 0) - (a.currentDamage ?? 0))[0];

	return (
		<div
			className="flex items-stretch"
			style={{
				padding: "12px 16px 10px",
				borderBottom: "1px solid var(--sk-line)",
				gap: 16,
				justifyContent: "space-between",
			}}
		>
			<div className="flex items-center min-w-0" style={{ gap: 16 }}>
				<span className="sk-index">{String(fightLabel).padStart(2, "0")}</span>
				<div className="flex items-center" style={{ gap: 8 }}>
					<span
						style={{
							width: 7,
							height: 7,
							borderRadius: 99,
							background: inCombat ? "var(--sk-damage)" : "var(--sk-fg-3)",
							boxShadow: inCombat
								? "0 0 0 3px color-mix(in oklab, var(--sk-damage) 22%, transparent)"
								: "none",
							animation: inCombat ? "sk-pulse 1.4s var(--sk-ease) infinite" : "none",
						}}
					/>
					<span
						className="sk-upper"
						style={{
							fontWeight: 600,
							color: inCombat ? "var(--sk-damage)" : isViewingPast ? "var(--sk-warn)" : "var(--sk-fg-2)",
						}}
					>
						{isViewingPast ? "Past fight" : inCombat ? "In Combat" : "Out of Combat"}
					</span>
				</div>
				<span className="sk-display" style={{ color: "var(--sk-fg-0)", fontSize: 22, lineHeight: 1 }}>
					{fmtDuration(elapsedSec)}
				</span>
				<FightPicker
					currentFightN={liveFightN}
					recent={recent}
					viewingFight={viewingFight}
					setViewingFight={setViewingFight}
				/>
				{isViewingPast && (
					<button
						onClick={() => setViewingFight(null)}
						className="sk-upper"
						style={{
							appearance: "none",
							border: "1px solid var(--sk-warn)",
							background: "color-mix(in oklab, var(--sk-warn) 18%, transparent)",
							color: "var(--sk-warn)",
							padding: "3px 8px",
							borderRadius: 4,
							cursor: "pointer",
							fontSize: 10.5,
							fontWeight: 600,
							letterSpacing: "0.08em",
						}}
					>
						Return to live
					</button>
				)}
			</div>

			<div
				className="flex items-center"
				style={{
					gap: 2,
					background: "var(--sk-bg-2)",
					padding: 3,
					borderRadius: 6,
					border: "1px solid var(--sk-line)",
				}}
				title="Click to add/remove this metric as a pane. At least one must stay on."
			>
				{TABS.map((t) => {
					const active = panes[t.id];
					const onCount = (panes.damage ? 1 : 0) + (panes.heal ? 1 : 0) + (panes.taken ? 1 : 0);
					const wouldRemoveLast = active && onCount === 1;
					return (
						<button
							key={String(t.id)}
							onClick={() => !wouldRemoveLast && togglePane(t.id)}
							style={{
								appearance: "none",
								border: 0,
								cursor: wouldRemoveLast ? "not-allowed" : "pointer",
								padding: "5px 11px",
								borderRadius: 4,
								background: active ? "var(--sk-bg-3)" : "transparent",
								color: active ? "var(--sk-fg-0)" : "var(--sk-fg-1)",
								fontFamily: "var(--sk-font-sans)",
								fontSize: 11.5,
								fontWeight: 500,
								letterSpacing: "0.02em",
								boxShadow: active ? "inset 0 0 0 1px var(--sk-line-2)" : "none",
								transition: "all 160ms var(--sk-ease)",
								opacity: wouldRemoveLast ? 0.85 : 1,
							}}
							title={active ? (wouldRemoveLast ? "At least one pane must stay on" : "Click to hide this pane") : "Click to show this pane"}
						>
							{t.label}
						</button>
					);
				})}
				<button
					disabled
					style={{
						appearance: "none",
						border: 0,
						cursor: "not-allowed",
						padding: "5px 11px",
						borderRadius: 4,
						background: "transparent",
						color: "var(--sk-fg-3)",
						fontFamily: "var(--sk-font-sans)",
						fontSize: 11.5,
						fontWeight: 500,
						letterSpacing: "0.02em",
					}}
					title="Mechanics breakdown — coming soon"
				>
					Mechanics
					<span style={{ marginLeft: 5, fontSize: 9, color: "var(--sk-fg-3)" }}>soon</span>
				</button>
			</div>

			<div className="flex items-center ml-auto" style={{ gap: 18 }}>
				<SummaryStat label="Party DPS" value={fmtRate(partyDps)} accent="var(--sk-damage)" />
				<SummaryStat label="Top" value={top?.name ?? "—"} mono={false} />
				<div className="flex flex-col items-end" style={{ gap: 3 }}>
					<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>
						Composition · {snapshot?.composition?.total ?? 0}/20
					</span>
					<CompStrip snapshot={snapshot} />
				</div>
			</div>
		</div>
	);
}

function FightPicker({
	currentFightN,
	recent,
	viewingFight,
	setViewingFight,
}: {
	currentFightN: number;
	recent: Snapshot["recent"];
	viewingFight: number | null;
	setViewingFight: (n: number | null) => void;
}): React.ReactElement {
	const [open, setOpen] = React.useState(false);
	const label = viewingFight !== null
		? `Fight ${viewingFight.toString().padStart(2, "0")}`
		: currentFightN > 0
		? `Fight ${currentFightN.toString().padStart(2, "0")}`
		: "Awaiting first fight";
	const past = (recent ?? []).slice().reverse();
	return (
		<div style={{ position: "relative" }}>
			<button
				onClick={() => setOpen((o) => !o)}
				className="sk-upper inline-flex items-center"
				style={{
					appearance: "none",
					border: "1px solid var(--sk-line)",
					background: "var(--sk-bg-2)",
					color: "var(--sk-fg-2)",
					padding: "3px 8px",
					borderRadius: 4,
					cursor: past.length > 0 ? "pointer" : "default",
					fontSize: 10.5,
					gap: 6,
				}}
				disabled={past.length === 0 && viewingFight === null}
				title={past.length === 0 ? "No completed fights yet" : "Pick a past fight to view"}
			>
				{label}
				{past.length > 0 && <span style={{ color: "var(--sk-fg-3)" }}>▼</span>}
			</button>
			{open && past.length > 0 && (
				<div
					style={{
						position: "absolute",
						top: "calc(100% + 4px)",
						left: 0,
						minWidth: 200,
						background: "var(--sk-bg-2)",
						border: "1px solid var(--sk-line-2)",
						borderRadius: 6,
						boxShadow: "0 12px 32px -8px rgba(0,0,0,0.6)",
						zIndex: 10,
						padding: 4,
					}}
				>
					<MenuItem
						active={viewingFight === null}
						onClick={() => { setViewingFight(null); setOpen(false); }}
					>
						<span style={{ color: "var(--sk-damage)" }}>●</span> Current (live)
					</MenuItem>
					<div style={{ height: 1, background: "var(--sk-line)", margin: "4px 0" }} />
					{past.map((a) => {
						const secs = Math.floor(a.durationMs / 1000);
						const m = Math.floor(secs / 60);
						const s = secs % 60;
						return (
							<MenuItem
								key={a.number}
								active={viewingFight === a.number}
								onClick={() => { setViewingFight(a.number); setOpen(false); }}
							>
								<span>Fight {a.number.toString().padStart(2, "0")}</span>
								<span className="sk-mono" style={{ marginLeft: "auto", color: "var(--sk-fg-3)", fontSize: 10 }}>
									{m}:{s.toString().padStart(2, "0")}
								</span>
							</MenuItem>
						);
					})}
				</div>
			)}
		</div>
	);
}

function MenuItem({
	active, onClick, children,
}: { active: boolean; onClick: () => void; children: React.ReactNode }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			className="flex items-center w-full"
			style={{
				appearance: "none",
				border: 0,
				background: active ? "var(--sk-bg-3)" : "transparent",
				color: active ? "var(--sk-fg-0)" : "var(--sk-fg-1)",
				padding: "5px 10px",
				borderRadius: 3,
				cursor: "pointer",
				fontSize: 11.5,
				gap: 6,
				textAlign: "left",
			}}
		>
			{children}
		</button>
	);
}

function SummaryStat({
	label, value, accent, mono = true,
}: {
	label: string; value: string; accent?: string; mono?: boolean;
}): React.ReactElement {
	return (
		<div className="flex flex-col items-end" style={{ gap: 1 }}>
			<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>{label}</span>
			<span
				className={mono ? "sk-mono" : ""}
				style={{ fontSize: 13, color: accent ?? "var(--sk-fg-0)", fontWeight: 500 }}
			>
				{value}
			</span>
		</div>
	);
}

function CompStrip({ snapshot }: { snapshot: Snapshot | null }): React.ReactElement {
	const c = snapshot?.composition;
	const entries: Array<{ key: string; short: string; n: number }> = [
		{ key: "tank",    short: "T", n: c?.tank ?? 0 },
		{ key: "healer",  short: "H", n: c?.healer ?? 0 },
		{ key: "rdps",    short: "R", n: c?.ranged ?? 0 },
		{ key: "mdps",    short: "M", n: c?.melee ?? 0 },
		{ key: "support", short: "S", n: c?.support ?? 0 },
	].filter((r) => r.n > 0);

	if (entries.length === 0) {
		return <span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>—</span>;
	}

	return (
		<div className="inline-flex items-center" style={{ gap: 4 }}>
			{entries.map((r) => {
				const roleColor = `var(--sk-role-${r.key})`;
				return (
					<span
						key={r.key}
						title={`${r.n} × ${r.key}`}
						className="inline-flex items-center"
						style={{
							gap: 4,
							padding: "2px 5px 2px 4px",
							background: `color-mix(in oklab, ${roleColor} 12%, var(--sk-bg-2))`,
							border: `1px solid color-mix(in oklab, ${roleColor} 35%, var(--sk-line))`,
							borderRadius: 3,
						}}
					>
						<span style={{ fontFamily: "var(--sk-font-mono)", fontSize: 9, fontWeight: 700, color: roleColor }}>
							{r.short}
						</span>
						<span className="sk-mono" style={{ fontSize: 10.5, color: "var(--sk-fg-0)", fontWeight: 600 }}>
							{r.n}
						</span>
					</span>
				);
			})}
		</div>
	);
}

function IconBtn({
	onClick, title, children,
}: { onClick: () => void; title: string; children: React.ReactNode }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			title={title}
			aria-label={title}
			style={{
				appearance: "none",
				border: "1px solid var(--sk-line)",
				background: "var(--sk-bg-2)",
				color: "var(--sk-fg-1)",
				width: 24,
				height: 24,
				borderRadius: 4,
				display: "inline-flex",
				alignItems: "center",
				justifyContent: "center",
				cursor: "pointer",
				fontSize: 13,
				fontFamily: "var(--sk-font-sans)",
			}}
		>
			{children}
		</button>
	);
}

function IconToggle({
	onClick, active, title, children,
}: { onClick: () => void; active: boolean; title: string; children: React.ReactNode }): React.ReactElement {
	return (
		<button
			onClick={onClick}
			title={title}
			aria-label={title}
			aria-pressed={active}
			style={{
				appearance: "none",
				border: `1px solid ${active ? "var(--sk-local)" : "var(--sk-line)"}`,
				background: active ? "color-mix(in oklab, var(--sk-local) 18%, var(--sk-bg-2))" : "var(--sk-bg-2)",
				color: active ? "var(--sk-local)" : "var(--sk-fg-1)",
				width: 24,
				height: 24,
				borderRadius: 4,
				display: "inline-flex",
				alignItems: "center",
				justifyContent: "center",
				cursor: "pointer",
				fontSize: 13,
				fontFamily: "var(--sk-font-sans)",
			}}
		>
			{children}
		</button>
	);
}
