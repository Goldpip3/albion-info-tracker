import type { ConnectionState, Mode, Snapshot } from "./types.ts";
import { fmtDuration, fmtRate } from "./format.ts";

interface HeaderProps {
	state: ConnectionState;
	stale: boolean;
	snapshot: Snapshot | null;
	mode: Mode;
	setMode: (m: Mode) => void;
	onSettings: () => void;
	onReset: () => void;
	onToggleLog: () => void;
	showLog: boolean;
	hideTabs?: boolean;
}

// Header = TitleBar (dynamic per-user name + agent status + controls)
// stacked on top of FightHeader (combat state + duration + fight label +
// optional mode tabs + party DPS + top + composition).
export function Header({
	state, stale, snapshot, mode, setMode,
	onSettings, onReset, onToggleLog, showLog, hideTabs,
}: HeaderProps): React.ReactElement {
	const localPlayer = snapshot?.players.find((p) => p.isLocal);
	const localName = localPlayer?.name;

	// Keep the document title in sync so the browser tab shows whose meter
	// this is. Falls back to a generic label until the local-player record
	// arrives in a snapshot.
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
				onReset={onReset}
				onToggleLog={onToggleLog}
				showLog={showLog}
			/>
			<FightHeader snapshot={snapshot} mode={mode} setMode={setMode} hideTabs={hideTabs} />
		</div>
	);
}

interface TitleBarProps {
	localName?: string;
	state: ConnectionState;
	stale: boolean;
	onSettings: () => void;
	onReset: () => void;
	onToggleLog: () => void;
	showLog: boolean;
}

function TitleBar({
	localName, state, stale, onSettings, onReset, onToggleLog, showLog,
}: TitleBarProps): React.ReactElement {
	const title = localName ? `${localName}'s Data Analytics` : "Combat Analytics";
	return (
		<div
			className="flex items-center justify-between px-4 py-2.5"
			style={{
				borderBottom: "1px solid var(--sk-line)",
				background: "linear-gradient(180deg, var(--sk-bg-1) 0%, var(--sk-bg-0) 100%)",
			}}
		>
			<div
				className="truncate"
				style={{
					fontSize: 15,
					fontWeight: 500,
					letterSpacing: "-0.005em",
					color: "var(--sk-fg-0)",
				}}
			>
				{title}
			</div>
			<div className="flex items-center" style={{ gap: 12 }}>
				<AgentPill state={state} stale={stale} />
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
	const label = isLive
		? "Agent · live"
		: state === "connecting"
		? "Dialing…"
		: stale
		? "Agent · stale"
		: "Offline";
	return (
		<span
			className="sk-upper inline-flex items-center"
			style={{ gap: 6, color: "var(--sk-fg-3)" }}
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
			{label}
		</span>
	);
}

interface FightHeaderProps {
	snapshot: Snapshot | null;
	mode: Mode;
	setMode: (m: Mode) => void;
	hideTabs?: boolean;
}

const TABS: Array<{ id: Mode; label: string; disabled?: boolean }> = [
	{ id: "damage", label: "Damage" },
	{ id: "heal",   label: "Healing" },
	{ id: "taken",  label: "Taken" },
	{ id: "mechanics", label: "Mechanics", disabled: true },
];

function FightHeader({ snapshot, mode, setMode, hideTabs }: FightHeaderProps): React.ReactElement {
	const inCombat = snapshot?.fight?.inCombat ?? false;
	const elapsedSec = (snapshot?.fight?.elapsedMs ?? 0) / 1000;
	const fightN = snapshot?.fight?.number ?? 0;
	const players = snapshot?.players ?? [];
	const partyDps = players.reduce((s, p) => s + (p.currentDps ?? 0), 0);
	const top = [...players].sort((a, b) => (b.currentDamage ?? 0) - (a.currentDamage ?? 0))[0];

	return (
		<div
			className="flex items-stretch px-4 pt-3 pb-2.5"
			style={{ borderBottom: "1px solid var(--sk-line)", gap: 16 }}
		>
			<div className="flex items-center min-w-0" style={{ gap: 14 }}>
				<div className="flex items-center" style={{ gap: 8 }}>
					<span
						style={{
							width: 7,
							height: 7,
							borderRadius: 99,
							background: inCombat ? "var(--sk-damage)" : "var(--sk-fg-3)",
							boxShadow: inCombat
								? "0 0 0 3px color-mix(in oklab, var(--sk-damage) 18%, transparent)"
								: "none",
							animation: inCombat ? "sk-pulse 1.4s var(--sk-ease) infinite" : "none",
						}}
					/>
					<span
						className="sk-upper"
						style={{
							fontWeight: 600,
							color: inCombat ? "var(--sk-damage)" : "var(--sk-fg-2)",
						}}
					>
						{inCombat ? "In Combat" : "Out of Combat"}
					</span>
				</div>
				<span className="sk-mono" style={{ color: "var(--sk-fg-0)", fontSize: 18, fontWeight: 500 }}>
					{fmtDuration(elapsedSec)}
				</span>
				<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>
					{fightN > 0 ? `Fight ${fightN.toString().padStart(2, "0")}` : "Awaiting first fight"}
				</span>
			</div>

			{!hideTabs && (
				<div
					className="flex items-center"
					style={{
						gap: 2,
						background: "var(--sk-bg-2)",
						padding: 3,
						borderRadius: 6,
						border: "1px solid var(--sk-line)",
					}}
				>
					{TABS.map((t) => {
						const active = t.id === mode;
						return (
							<button
								key={t.id}
								disabled={t.disabled}
								onClick={() => !t.disabled && setMode(t.id)}
								style={{
									appearance: "none",
									border: 0,
									cursor: t.disabled ? "not-allowed" : "pointer",
									padding: "5px 11px",
									borderRadius: 4,
									background: active ? "var(--sk-bg-3)" : "transparent",
									color: t.disabled
										? "var(--sk-fg-3)"
										: active
										? "var(--sk-fg-0)"
										: "var(--sk-fg-1)",
									fontFamily: "var(--sk-font-sans)",
									fontSize: 11.5,
									fontWeight: 500,
									letterSpacing: "0.02em",
									boxShadow: active ? "inset 0 0 0 1px var(--sk-line-2)" : "none",
									transition: "all 160ms var(--sk-ease)",
								}}
							>
								{t.label}
								{t.disabled && (
									<span style={{ marginLeft: 5, fontSize: 9, color: "var(--sk-fg-3)" }}>soon</span>
								)}
							</button>
						);
					})}
				</div>
			)}

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
