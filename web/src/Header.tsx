import type { ConnectionState, Snapshot, Mode } from "./types.ts";

interface HeaderProps {
	state: ConnectionState;
	stale: boolean;
	snapshot: Snapshot | null;
	mode: Mode;
	setMode: (m: Mode) => void;
	onSettings: () => void;
	onReset: () => void;
}

// Header renders the two stacked bars at the top of the meter:
//   row 1: SKIRMISH logo · breadcrumb · agent status · controls
//   row 2: combat state + fight info · mode tabs · party stats · composition
//
// Composition / fight info are placeholders until T3 introduces fight
// lifecycle + role aggregation.
export function Header({ state, stale, snapshot, mode, setMode, onSettings, onReset }: HeaderProps): React.ReactElement {
	const localPlayer = snapshot?.players.find((p) => p.isLocal);
	const localName = localPlayer?.name?.toLowerCase() ?? "—";

	return (
		<header className="border-b border-skirmish-line bg-skirmish-bg">
			<div className="flex items-center gap-4 px-4 h-10 text-sm">
				<div className="font-semibold tracking-[0.2em] text-skirmish-amber">
					SKIRMISH
				</div>
				<div className="text-skirmish-muted text-xs tracking-wider uppercase truncate">
					meter.skirmish.gg / <span className="text-skirmish-dim">{localName}</span>
				</div>
				<div className="ml-auto flex items-center gap-3">
					<AgentPill state={state} stale={stale} />
					<button
						className="text-skirmish-muted hover:text-skirmish-text text-sm"
						onClick={onSettings}
						title="Settings"
					>
						⚙
					</button>
					<button
						className="text-skirmish-muted hover:text-skirmish-text text-sm"
						onClick={onReset}
						title="Reset connection"
					>
						⟳
					</button>
				</div>
			</div>

			<div className="flex items-center gap-4 px-4 h-10 text-xs border-t border-skirmish-line/60">
				<CombatBadge snapshot={snapshot} />
				<FightInfo snapshot={snapshot} />

				<div className="ml-auto flex items-center gap-1">
					<ModeTab current={mode} value="damage" onClick={setMode}>Damage</ModeTab>
					<ModeTab current={mode} value="heal" onClick={setMode}>Healing</ModeTab>
					<ModeTab current={mode} value="taken" onClick={setMode}>Taken</ModeTab>
					<ModeTab current={mode} value="mechanics" onClick={setMode} disabled>
						Mechanics
					</ModeTab>
				</div>

				<div className="hidden md:flex items-center gap-4 pl-3 border-l border-skirmish-line/60">
					<PartyStat label="Party DPS" value={partyDps(snapshot)} />
					<PartyStat label="Top" value={topName(snapshot)} mono={false} />
					<CompositionStrip snapshot={snapshot} />
				</div>
			</div>
		</header>
	);
}

function AgentPill({ state, stale }: { state: ConnectionState; stale: boolean }): React.ReactElement {
	const isLive = state === "connected" && !stale;
	const dot = isLive ? "bg-emerald-400" : state === "connecting" ? "bg-amber-400" : "bg-rose-500";
	const label =
		state === "connected" ? (stale ? "STALE" : "AGENT") : state === "connecting" ? "DIALING" : "OFFLINE";
	return (
		<span className="inline-flex items-center gap-1.5 text-[10px] tracking-[0.18em] uppercase text-skirmish-dim">
			<span className={`w-1.5 h-1.5 rounded-full ${dot} ${isLive ? "animate-pulse" : ""}`} />
			{label}
		</span>
	);
}

function CombatBadge({ snapshot }: { snapshot: Snapshot | null }): React.ReactElement {
	const inCombat = snapshot?.fight?.inCombat ?? false;
	const dot = inCombat ? "bg-rose-500" : "bg-skirmish-muted";
	const label = inCombat ? "IN COMBAT" : "OUT OF COMBAT";
	return (
		<span className="inline-flex items-center gap-2 text-[11px] tracking-[0.18em] uppercase">
			<span className={`w-1.5 h-1.5 rounded-full ${dot} ${inCombat ? "animate-pulse" : ""}`} />
			<span className={inCombat ? "text-rose-300" : "text-skirmish-muted"}>{label}</span>
			<span className="tnum text-skirmish-dim">{formatElapsed(snapshot?.fight?.elapsedMs ?? 0)}</span>
		</span>
	);
}

function FightInfo({ snapshot }: { snapshot: Snapshot | null }): React.ReactElement {
	const n = snapshot?.fight?.number ?? 0;
	if (n === 0) {
		return (
			<div className="text-skirmish-muted tracking-wider uppercase truncate">
				Awaiting first fight
			</div>
		);
	}
	return (
		<div className="text-skirmish-dim tracking-wider uppercase truncate">
			Fight {n.toString().padStart(2, "0")}
			{/* TODO: zone name when LoadCluster handler ships */}
		</div>
	);
}

function formatElapsed(ms: number): string {
	if (ms <= 0) return "--:--";
	const totalSec = Math.floor(ms / 1000);
	const mm = Math.floor(totalSec / 60).toString().padStart(2, "0");
	const ss = (totalSec % 60).toString().padStart(2, "0");
	return `${mm}:${ss}`;
}

function PartyStat({ label, value, mono = true }: { label: string; value: string; mono?: boolean }): React.ReactElement {
	return (
		<div className="flex flex-col items-end leading-none">
			<span className="text-[9px] uppercase tracking-[0.15em] text-skirmish-muted">{label}</span>
			<span className={`text-sm text-skirmish-text ${mono ? "tnum" : ""}`}>{value}</span>
		</div>
	);
}

function ModeTab({
	current,
	value,
	onClick,
	disabled,
	children,
}: {
	current: Mode;
	value: Mode;
	onClick: (m: Mode) => void;
	disabled?: boolean;
	children: React.ReactNode;
}): React.ReactElement {
	const active = current === value;
	return (
		<button
			disabled={disabled}
			onClick={() => onClick(value)}
			className={`px-2.5 py-0.5 text-[11px] tracking-[0.12em] uppercase rounded-sm transition ${
				disabled
					? "text-skirmish-muted/50 cursor-not-allowed"
					: active
					? "bg-skirmish-text text-skirmish-bg"
					: "text-skirmish-dim hover:text-skirmish-text"
			}`}
		>
			{children}
		</button>
	);
}

function partyDps(snapshot: Snapshot | null): string {
	const total = (snapshot?.players ?? []).reduce((s, p) => s + p.currentDps, 0);
	if (total >= 1_000_000) return `${(total / 1_000_000).toFixed(2)}M`;
	if (total >= 1000) return `${(total / 1000).toFixed(1)}K`;
	return Math.round(total).toString();
}

function topName(snapshot: Snapshot | null): string {
	const players = snapshot?.players ?? [];
	if (players.length === 0) return "—";
	const top = [...players].sort((a, b) => b.currentDamage - a.currentDamage)[0];
	return top.name || "—";
}

function CompositionStrip({ snapshot }: { snapshot: Snapshot | null }): React.ReactElement {
	const c = snapshot?.composition;
	if (!c || c.total === 0) {
		return (
			<div className="flex flex-col items-end leading-none">
				<span className="text-[9px] uppercase tracking-[0.15em] text-skirmish-muted">Composition</span>
				<span className="text-sm text-skirmish-muted">—</span>
			</div>
		);
	}
	return (
		<div className="flex flex-col items-end leading-none gap-1">
			<span className="text-[9px] uppercase tracking-[0.15em] text-skirmish-muted">
				Composition {c.total}/20
			</span>
			<div className="flex gap-1">
				<RoleCount letter="T" count={c.tank}    tone="bg-blue-500/15 text-blue-300" />
				<RoleCount letter="H" count={c.healer}  tone="bg-emerald-500/15 text-emerald-300" />
				<RoleCount letter="R" count={c.ranged}  tone="bg-rose-500/15 text-rose-300" />
				<RoleCount letter="M" count={c.melee}   tone="bg-amber-500/15 text-amber-300" />
				<RoleCount letter="S" count={c.support} tone="bg-purple-500/15 text-purple-300" />
			</div>
		</div>
	);
}

function RoleCount({ letter, count, tone }: { letter: string; count: number; tone: string }): React.ReactElement {
	if (count === 0) {
		return (
			<span className="text-[10px] tnum text-skirmish-muted/60 px-1.5 py-0.5 rounded border border-skirmish-line">
				{letter}0
			</span>
		);
	}
	return (
		<span className={`text-[10px] tnum px-1.5 py-0.5 rounded ${tone}`}>
			{letter}
			{count}
		</span>
	);
}
