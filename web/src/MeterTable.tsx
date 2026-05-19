import { useState } from "react";
import type { PlayerSnapshot, Snapshot, Mode } from "./types.ts";
import type { Settings } from "./useSettings.ts";

interface MeterTableProps {
	snapshot: Snapshot | null;
	mode: Mode;
	settings: Settings;
}

export function MeterTable({ snapshot, mode, settings }: MeterTableProps): React.ReactElement {
	const players = snapshot?.players ?? [];
	const [hovered, setHovered] = useState<PlayerSnapshot | null>(null);

	const sortKey = (p: PlayerSnapshot): number => {
		switch (mode) {
			case "damage":
				return p.currentDamage;
			case "heal":
				return p.currentHeal;
			case "taken":
				return p.currentTaken;
			case "mechanics":
				return p.currentDamage;
		}
	};

	const primaryFor = (p: PlayerSnapshot): { cur: number; ovr: number } => {
		switch (mode) {
			case "damage":
				return { cur: p.currentDamage, ovr: p.overallDamage };
			case "heal":
				return { cur: p.currentHeal, ovr: p.overallHeal };
			case "taken":
				return { cur: p.currentTaken, ovr: p.overallTaken };
			case "mechanics":
				return { cur: p.currentDamage, ovr: p.overallDamage };
		}
	};

	let sorted = [...players].sort((a, b) => sortKey(b) - sortKey(a));

	// Pin local user to the top of the list if requested, regardless of rank.
	if (settings.pinLocal) {
		const local = sorted.find((p) => p.isLocal);
		if (local) {
			sorted = [local, ...sorted.filter((p) => p !== local)];
		}
	}

	const max = sorted[0] ? primaryFor(sorted[0]).cur : 0;

	if (sorted.length === 0) {
		return (
			<div className="text-skirmish-dim text-sm px-4 py-16 text-center">
				No players tracked yet. Re-zone in Albion (walk through any portal)
				to populate.
			</div>
		);
	}

	const headerLabel: Record<Mode, string> = {
		damage: "DAMAGE · CURRENT / SESSION",
		heal: "HEALING · CURRENT / SESSION",
		taken: "DAMAGE TAKEN · CURRENT / SESSION",
		mechanics: "MECHANICS · COUNTS",
	};
	const rateLabel: Record<Mode, string> = {
		damage: "DPS",
		heal: "HPS",
		taken: "DPS",
		mechanics: "TICKS",
	};

	const cols = visibleColumns(settings, mode);
	const gridCols = `2rem minmax(11rem,16rem) 1fr ${cols.rate ? "4rem " : ""}${cols.taken ? "5rem " : ""}${cols.heal ? "5rem" : ""}`.trim();

	return (
		<div className="relative">
			<div
				className="grid gap-3 px-4 py-2 text-[10px] uppercase tracking-[0.15em] text-skirmish-muted border-b border-skirmish-line"
				style={{ gridTemplateColumns: gridCols }}
			>
				<div>#</div>
				<div>Player</div>
				<div>{headerLabel[mode]}</div>
				{cols.rate && <div className="text-right">{rateLabel[mode]}</div>}
				{cols.taken && <div className="text-right">↓ Taken</div>}
				{cols.heal && <div className="text-right">+ Heal</div>}
			</div>
			<div>
				{sorted.map((p, i) => (
					<MeterRow
						key={p.userGuid}
						rank={i + 1}
						player={p}
						mode={mode}
						max={max}
						settings={settings}
						cols={cols}
						gridCols={gridCols}
						onHover={setHovered}
					/>
				))}
			</div>

			{hovered && <HoverTooltip player={hovered} />}
		</div>
	);
}

interface VisibleCols {
	rate: boolean;
	taken: boolean;
	heal: boolean;
}

function visibleColumns(s: Settings, mode: Mode): VisibleCols {
	return {
		rate: mode === "heal" ? s.columns.hps : s.columns.dps,
		taken: s.columns.damageTaken,
		heal: s.columns.healing,
	};
}

interface MeterRowProps {
	rank: number;
	player: PlayerSnapshot;
	mode: Mode;
	max: number;
	settings: Settings;
	cols: VisibleCols;
	gridCols: string;
	onHover: (p: PlayerSnapshot | null) => void;
}

function MeterRow({ rank, player, mode, max, settings, cols, gridCols, onHover }: MeterRowProps): React.ReactElement {
	const cur =
		mode === "damage"
			? player.currentDamage
			: mode === "heal"
			? player.currentHeal
			: mode === "taken"
			? player.currentTaken
			: player.currentDamage;
	const ovr =
		mode === "damage"
			? player.overallDamage
			: mode === "heal"
			? player.overallHeal
			: mode === "taken"
			? player.overallTaken
			: player.overallDamage;
	const rate = mode === "heal" ? player.currentHps : mode === "taken" ? 0 : player.currentDps;
	const pct = max > 0 ? (cur / max) * 100 : 0;

	const chipText = player.isLocal ? "YOU" : "—";
	const pyPad = settings.density === 24 ? "py-1" : settings.density === 28 ? "py-1.5" : "py-2";

	const barClass = barClassName(settings.barStyle, player.isLocal ?? false);

	return (
		<div
			className={`group relative grid gap-3 px-4 ${pyPad} items-center border-b border-skirmish-line/40 hover:bg-skirmish-bg2/40`}
			style={{ gridTemplateColumns: gridCols }}
			onMouseEnter={() => onHover(player)}
			onMouseLeave={() => onHover(null)}
		>
			<div className="text-skirmish-muted tnum text-xs">
				{rank.toString().padStart(2, "0")}
			</div>

			<div className="flex items-center gap-2 min-w-0">
				<ClassChip text={chipText} isLocal={player.isLocal ?? false} />
				<div className="min-w-0">
					<div className={`truncate text-sm ${player.isLocal ? "text-skirmish-amber" : "text-skirmish-text"}`}>
						{player.name || player.userGuid.slice(0, 8) + "…"}
					</div>
					<div className="truncate text-[10px] uppercase tracking-wider text-skirmish-muted">
						{player.guild || "—"}
					</div>
				</div>
			</div>

			<div className="relative h-7 flex items-center">
				<div className={`absolute inset-y-0 left-0 rounded-sm ${barClass}`} style={{ width: `${pct}%` }} aria-hidden="true" />
				<div className="relative z-10 pl-2 flex items-baseline gap-2">
					<span className="tnum text-sm text-skirmish-text">{formatNum(cur)}</span>
					<span className="tnum text-[11px] text-skirmish-muted">
						↓ {formatNum(ovr)} session
					</span>
				</div>
			</div>

			{cols.rate && (
				<div className="text-right">
					<div className="tnum text-sm text-skirmish-text">{rate > 0 ? formatNum(rate) : "—"}</div>
					<div className="text-[9px] uppercase text-skirmish-muted leading-none">
						{mode === "heal" ? "hps" : "dps"}
					</div>
				</div>
			)}

			{cols.taken && (
				<div className="text-right tnum text-sm text-skirmish-taken/90">
					↓ {player.currentTaken > 0 ? formatNum(player.currentTaken) : "0"}
				</div>
			)}

			{cols.heal && (
				<div className="text-right tnum text-sm text-skirmish-heal/90">
					+ {player.currentHeal > 0 ? formatNum(player.currentHeal) : "0"}
				</div>
			)}
		</div>
	);
}

function barClassName(style: Settings["barStyle"], isLocal: boolean): string {
	if (isLocal) {
		switch (style) {
			case "solid":
				return "bg-skirmish-amber";
			case "tint":
				return "bg-skirmish-amber/25";
			case "outline":
			default:
				return "bg-skirmish-amber/20 border border-skirmish-amber";
		}
	}
	switch (style) {
		case "solid":
			return "bg-skirmish-amber-dim/70";
		case "tint":
			return "bg-skirmish-amber-dim/20";
		case "outline":
		default:
			return "border border-skirmish-amber-dim/60";
	}
}

interface ClassChipProps {
	text: string;
	isLocal: boolean;
}

function ClassChip({ text, isLocal }: ClassChipProps): React.ReactElement {
	const tone = isLocal
		? "bg-skirmish-amber/15 text-skirmish-amber border-skirmish-amber/40"
		: "bg-skirmish-bg2 text-skirmish-dim border-skirmish-line";
	return (
		<span
			className={`shrink-0 inline-flex items-center justify-center w-9 h-6 text-[10px] tracking-[0.12em] font-medium border rounded-sm ${tone}`}
		>
			{text}
		</span>
	);
}

function HoverTooltip({ player }: { player: PlayerSnapshot }): React.ReactElement {
	return (
		<div className="fixed bottom-12 right-4 z-40 w-64 bg-skirmish-bg2/95 backdrop-blur border border-skirmish-line rounded-md p-3 text-xs shadow-2xl pointer-events-none">
			<div className="flex items-baseline gap-2 mb-2">
				<div className={`text-sm font-medium ${player.isLocal ? "text-skirmish-amber" : "text-skirmish-text"}`}>
					{player.name || "(unknown)"}
				</div>
				{player.guild && <div className="text-skirmish-muted text-[10px]">[{player.guild}]</div>}
			</div>
			<StatRow label="Damage current"  value={formatNum(player.currentDamage)} />
			<StatRow label="Damage session"  value={formatNum(player.overallDamage)} dim />
			<StatRow label="DPS"             value={formatNum(player.currentDps)} />
			<StatRow label="Healing current" value={formatNum(player.currentHeal)} />
			<StatRow label="Healing session" value={formatNum(player.overallHeal)} dim />
			<StatRow label="Taken current"   value={formatNum(player.currentTaken)} />
			<StatRow label="Taken session"   value={formatNum(player.overallTaken)} dim />
		</div>
	);
}

function StatRow({ label, value, dim }: { label: string; value: string; dim?: boolean }): React.ReactElement {
	return (
		<div className="flex justify-between py-0.5">
			<span className={`text-[10px] uppercase tracking-wider ${dim ? "text-skirmish-muted" : "text-skirmish-dim"}`}>
				{label}
			</span>
			<span className={`tnum text-xs ${dim ? "text-skirmish-dim" : "text-skirmish-text"}`}>{value}</span>
		</div>
	);
}

function formatNum(n: number): string {
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 10_000) return `${(n / 1000).toFixed(1)}K`;
	if (n >= 1000) return `${(n / 1000).toFixed(2)}K`;
	return Math.round(n).toString();
}
