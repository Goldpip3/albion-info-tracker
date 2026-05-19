import type { PlayerSnapshot } from "./types.ts";

interface DrillInProps {
	player: PlayerSnapshot;
	onClose: () => void;
}

// DrillIn renders the per-player breakdown screen modeled after the design
// canvas. For T4 only the Abilities tab carries data — Targets / Taken /
// Healing received / Timeline are stubs visible but inert.
export function DrillIn({ player, onClose }: DrillInProps): React.ReactElement {
	const spells = player.spells ?? [];
	const max = spells[0]?.totalDamage ?? 0;

	return (
		<div className="fixed inset-0 z-40 bg-black/70 backdrop-blur-sm flex items-stretch justify-center p-2 md:p-6" onClick={onClose}>
			<div
				className="w-full max-w-3xl flex flex-col bg-skirmish-bg2 border border-skirmish-line rounded-md overflow-hidden"
				onClick={(e) => e.stopPropagation()}
			>
				<DrillHeader player={player} onClose={onClose} />
				<DrillTabs />
				<DrillAbilities spells={spells} max={max} />
			</div>
		</div>
	);
}

function DrillHeader({ player, onClose }: { player: PlayerSnapshot; onClose: () => void }): React.ReactElement {
	const chipText = player.classCode && player.classCode !== "—" ? player.classCode : "—";
	return (
		<header className="flex items-center gap-3 px-4 py-3 border-b border-skirmish-line">
			<button onClick={onClose} className="text-skirmish-muted hover:text-skirmish-text text-xl leading-none">
				←
			</button>
			<span className="shrink-0 inline-flex items-center justify-center w-10 h-7 text-[11px] tracking-[0.12em] font-medium border rounded-sm bg-skirmish-amber/15 text-skirmish-amber border-skirmish-amber/40">
				{chipText}
			</span>
			<div className="min-w-0 mr-auto">
				<div className={`truncate text-base font-medium ${player.isLocal ? "text-skirmish-amber" : "text-skirmish-text"}`}>
					{player.name || "(unknown)"}
				</div>
				<div className="truncate text-[10px] uppercase tracking-wider text-skirmish-muted">
					{player.roleLabel || player.guild || "—"}
				</div>
			</div>
			<HeaderStat label="Damage" value={formatNum(player.currentDamage)} />
			<HeaderStat label="DPS"    value={formatNum(player.currentDps)} />
			<HeaderStat label="Hits"   value={(player.spells ?? []).reduce((s, x) => s + x.hits, 0).toString()} />
			<HeaderStat label="Max"    value={formatNum(maxHit(player.spells ?? []))} />
		</header>
	);
}

function HeaderStat({ label, value }: { label: string; value: string }): React.ReactElement {
	return (
		<div className="text-right">
			<div className="text-[9px] uppercase tracking-[0.15em] text-skirmish-muted leading-none">{label}</div>
			<div className="tnum text-base text-skirmish-text">{value}</div>
		</div>
	);
}

function DrillTabs(): React.ReactElement {
	const tabs = [
		{ id: "abilities", label: "Abilities", active: true },
		{ id: "targets", label: "Targets", active: false },
		{ id: "taken", label: "Taken", active: false },
		{ id: "healing", label: "Healing received", active: false },
		{ id: "timeline", label: "Timeline", active: false },
	];
	return (
		<div className="flex gap-2 px-4 py-2 border-b border-skirmish-line text-[11px] uppercase tracking-[0.12em]">
			{tabs.map((t) => (
				<button
					key={t.id}
					disabled={!t.active}
					className={t.active
						? "px-2 py-0.5 rounded bg-skirmish-text text-skirmish-bg"
						: "px-2 py-0.5 rounded text-skirmish-muted/60 cursor-not-allowed"
					}
					title={t.active ? "" : "Coming soon"}
				>
					{t.label}
				</button>
			))}
		</div>
	);
}

function DrillAbilities({ spells, max }: { spells: PlayerSnapshot["spells"]; max: number }): React.ReactElement {
	if (!spells || spells.length === 0) {
		return (
			<div className="flex-1 px-4 py-16 text-center text-sm text-skirmish-dim">
				No abilities recorded in the current fight yet.
			</div>
		);
	}
	return (
		<div className="flex-1 overflow-auto">
			<div className="grid grid-cols-[1fr_5rem_5rem_5rem] gap-3 px-4 py-2 text-[10px] uppercase tracking-[0.15em] text-skirmish-muted border-b border-skirmish-line">
				<div>Ability</div>
				<div className="text-right">Total</div>
				<div className="text-right">Hits</div>
				<div className="text-right">Max hit</div>
			</div>
			<div>
				{spells.map((s) => {
					const pct = max > 0 ? (s.totalDamage / max) * 100 : 0;
					return (
						<div key={s.index} className="relative grid grid-cols-[1fr_5rem_5rem_5rem] gap-3 px-4 py-1.5 items-center border-b border-skirmish-line/40">
							<div className="relative">
								<div
									className="absolute inset-y-0 left-0 -ml-2 rounded-sm bg-skirmish-amber/15 border border-skirmish-amber/40"
									style={{ width: `calc(${pct}% + 1rem)` }}
									aria-hidden="true"
								/>
								<div className="relative pl-1 truncate text-sm text-skirmish-text">
									{prettyName(s.name) || `#${s.index}`}
								</div>
							</div>
							<div className="text-right tnum text-sm">{formatNum(s.totalDamage)}</div>
							<div className="text-right tnum text-sm text-skirmish-dim">{s.hits}</div>
							<div className="text-right tnum text-sm text-skirmish-dim">{formatNum(s.maxHit)}</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}

function maxHit(spells: PlayerSnapshot["spells"]): number {
	if (!spells || spells.length === 0) return 0;
	return spells.reduce((m, s) => (s.maxHit > m ? s.maxHit : m), 0);
}

// prettyName turns Albion's TOKEN_CASE into a more readable form.
//   FIREBALL_AOE        → "Fireball aoe"
//   PASSIVE_AA_STACK    → "Passive aa stack"
function prettyName(name?: string): string {
	if (!name) return "";
	const parts = name.split("_").filter(Boolean);
	if (parts.length === 0) return name;
	const first = parts[0].charAt(0).toUpperCase() + parts[0].slice(1).toLowerCase();
	const rest = parts.slice(1).map((p) => p.toLowerCase()).join(" ");
	return rest ? `${first} ${rest}` : first;
}

function formatNum(n: number): string {
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 10_000) return `${(n / 1000).toFixed(1)}K`;
	if (n >= 1000) return `${(n / 1000).toFixed(2)}K`;
	return Math.round(n).toString();
}
