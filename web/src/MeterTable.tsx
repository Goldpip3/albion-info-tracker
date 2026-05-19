import type { PlayerSnapshot, Snapshot, Mode } from "./types.ts";

interface MeterTableProps {
	snapshot: Snapshot | null;
	mode: Mode;
}

// MeterTable renders one Skirmish-style table: ranked players with class
// chip, role subtitle, current/session bars, DPS, taken, heal.
export function MeterTable({ snapshot, mode }: MeterTableProps): React.ReactElement {
	const players = snapshot?.players ?? [];

	const sortKey = (p: PlayerSnapshot): number => {
		switch (mode) {
			case "damage":
				return p.currentDamage;
			case "heal":
				return p.currentHeal;
			case "taken":
				return p.currentTaken;
			case "mechanics":
				return p.currentDamage; // placeholder until T5 introduces mechanics data
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

	const sorted = [...players].sort((a, b) => sortKey(b) - sortKey(a));
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

	return (
		<div>
			<div className="grid grid-cols-[2rem_minmax(11rem,16rem)_1fr_4rem_5rem_5rem] gap-3 px-4 py-2 text-[10px] uppercase tracking-[0.15em] text-skirmish-muted border-b border-skirmish-line">
				<div>#</div>
				<div>Player</div>
				<div>{headerLabel[mode]}</div>
				<div className="text-right">{rateLabel[mode]}</div>
				<div className="text-right">↓ Taken</div>
				<div className="text-right">+ Heal</div>
			</div>
			<div>
				{sorted.map((p, i) => (
					<MeterRow
						key={p.userGuid}
						rank={i + 1}
						player={p}
						mode={mode}
						max={max}
					/>
				))}
			</div>
		</div>
	);
}

interface MeterRowProps {
	rank: number;
	player: PlayerSnapshot;
	mode: Mode;
	max: number;
}

function MeterRow({ rank, player, mode, max }: MeterRowProps): React.ReactElement {
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
	const rate =
		mode === "heal" ? player.currentHps : mode === "taken" ? 0 : player.currentDps;
	const pct = max > 0 ? (cur / max) * 100 : 0;

	// Class chip is a placeholder until T3 wires weapon-ID → role detection.
	// For now: local player → amber tinted "YOU", others → muted "???"
	const chipText = player.isLocal ? "YOU" : "—";

	return (
		<div className="group relative grid grid-cols-[2rem_minmax(11rem,16rem)_1fr_4rem_5rem_5rem] gap-3 px-4 py-1.5 items-center border-b border-skirmish-line/40 hover:bg-skirmish-bg2/40">
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
				<div
					className={`absolute inset-y-0 left-0 rounded-sm border ${
						player.isLocal
							? "bg-skirmish-amber/20 border-skirmish-amber"
							: "border-skirmish-amber-dim/60 bg-transparent"
					}`}
					style={{ width: `${pct}%` }}
					aria-hidden="true"
				/>
				<div className="relative z-10 pl-2 flex items-baseline gap-2">
					<span className="tnum text-sm text-skirmish-text">{formatNum(cur)}</span>
					<span className="tnum text-[11px] text-skirmish-muted">
						↓ {formatNum(ovr)} session
					</span>
				</div>
			</div>

			<div className="text-right tnum text-sm text-skirmish-text">
				{rate > 0 ? formatNum(rate) : "—"}
				<div className="text-[9px] uppercase text-skirmish-muted leading-none">
					{mode === "heal" ? "hps" : "dps"}
				</div>
			</div>

			<div className="text-right tnum text-sm text-skirmish-taken/90">
				↓ {player.currentTaken > 0 ? formatNum(player.currentTaken) : "0"}
			</div>

			<div className="text-right tnum text-sm text-skirmish-heal/90">
				+ {player.currentHeal > 0 ? formatNum(player.currentHeal) : "0"}
			</div>
		</div>
	);
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

// formatNum renders a number with k/m suffixes once it gets large.
function formatNum(n: number): string {
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 10_000) return `${(n / 1000).toFixed(1)}K`;
	if (n >= 1000) return `${(n / 1000).toFixed(2)}K`;
	return Math.round(n).toString();
}
