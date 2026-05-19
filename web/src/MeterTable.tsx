import type { PlayerSnapshot, Snapshot } from "./types.ts";

interface MeterTableProps {
	snapshot: Snapshot | null;
	mode: "damage" | "heal" | "taken";
}

// MeterTable renders one Skada-style table: players sorted by the mode's
// metric, each row showing CURRENT (this fight) | OVERALL (session) side by
// side with a bar-fill background proportional to the top performer.
export function MeterTable({ snapshot, mode }: MeterTableProps): React.ReactElement {
	const players = snapshot?.players ?? [];

	const valueFor = (p: PlayerSnapshot): { cur: number; ovr: number; rate: number } => {
		switch (mode) {
			case "damage":
				return { cur: p.currentDamage, ovr: p.overallDamage, rate: p.currentDps };
			case "heal":
				return { cur: p.currentHeal, ovr: p.overallHeal, rate: p.currentHps };
			case "taken":
				return { cur: p.currentTaken, ovr: p.overallTaken, rate: 0 };
		}
	};

	const sorted = [...players].sort((a, b) => valueFor(b).cur - valueFor(a).cur);
	const max = sorted[0] ? valueFor(sorted[0]).cur : 0;

	if (sorted.length === 0) {
		return (
			<div className="text-neutral-500 text-sm px-4 py-8 text-center">
				No players tracked yet. Re-zone in Albion (walk through any portal)
				to populate.
			</div>
		);
	}

	return (
		<div className="divide-y divide-neutral-900">
			<div className="grid grid-cols-[1fr_5rem_5rem_5rem] gap-2 px-3 py-1.5 text-[10px] uppercase tracking-wider text-neutral-500">
				<div>Player</div>
				<div className="text-right">Current</div>
				<div className="text-right">Overall</div>
				<div className="text-right">/sec</div>
			</div>
			{sorted.map((p) => {
				const { cur, ovr, rate } = valueFor(p);
				const pct = max > 0 ? (cur / max) * 100 : 0;
				return (
					<div key={p.userGuid} className="relative px-3 py-1.5">
						<div
							className="absolute inset-0 bg-amber-500/15"
							style={{ width: `${pct}%` }}
							aria-hidden="true"
						/>
						<div className="relative grid grid-cols-[1fr_5rem_5rem_5rem] gap-2 items-center">
							<div className="truncate">
								{p.isLocal && (
									<span className="mr-1 text-amber-400" title="you">
										★
									</span>
								)}
								<span className="text-neutral-100">{p.name || "(unknown)"}</span>
								{p.guild && (
									<span className="ml-1 text-neutral-500 text-xs">
										[{p.guild}]
									</span>
								)}
							</div>
							<div className="text-right tnum">{formatNum(cur)}</div>
							<div className="text-right tnum text-neutral-400">{formatNum(ovr)}</div>
							<div className="text-right tnum text-neutral-500">{formatNum(rate)}</div>
						</div>
					</div>
				);
			})}
		</div>
	);
}

// formatNum renders a number with k/m suffixes once it gets large. Damage
// meters scroll long enough that "1.2k" beats "1234" for column width.
function formatNum(n: number): string {
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}m`;
	if (n >= 10_000) return `${(n / 1000).toFixed(1)}k`;
	if (n >= 1000) return `${(n / 1000).toFixed(2)}k`;
	return Math.round(n).toString();
}
