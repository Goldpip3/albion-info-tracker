import type { DungeonRun } from "./types.ts";
import { fmt, fmtDuration } from "./format.ts";

interface DungeonStripProps {
	dungeon: DungeonRun | undefined;
}

// DungeonStrip surfaces an active run-scoped scope above the meter
// when the local player is inside a dungeon-pattern zone (solo /
// group / avalonian / mists / hellgate / corrupted). Shows the
// delta-since-entry for the four currency tracks. Auto-hides when the
// agent isn't currently in a dungeon. Closed runs persist briefly so
// you can read the final numbers before leaving the entrance zone.
export function DungeonStrip({ dungeon }: DungeonStripProps): React.ReactElement | null {
	if (!dungeon) return null;
	const inFlight = !dungeon.endedAt;
	const seconds = Math.floor((dungeon.durationMs ?? 0) / 1000);
	const fame = dungeon.fameGained / 10_000;
	const silver = dungeon.silverGained / 10_000;
	const respec = dungeon.respecGained / 10_000;
	const might = dungeon.mightGained / 10_000;

	return (
		<div
			className="flex items-center"
			style={{
				gap: 18,
				padding: "8px 16px",
				background: inFlight
					? "color-mix(in oklab, var(--sk-card-fame) 8%, var(--sk-bg-inset))"
					: "var(--sk-bg-inset)",
				borderBottom: "1px solid var(--sk-line)",
				fontSize: 11,
			}}
		>
			<div className="flex items-center" style={{ gap: 8 }}>
				<span
					style={{
						width: 7,
						height: 7,
						borderRadius: 99,
						background: inFlight ? "var(--sk-card-fame)" : "var(--sk-fg-3)",
						boxShadow: inFlight ? "0 0 0 3px color-mix(in oklab, var(--sk-card-fame) 22%, transparent)" : "none",
						animation: inFlight ? "sk-pulse 1.4s var(--sk-ease) infinite" : "none",
					}}
				/>
				<span className="sk-upper" style={{ color: inFlight ? "var(--sk-card-fame)" : "var(--sk-fg-3)", fontWeight: 600 }}>
					{inFlight ? "In dungeon" : "Last run"}
				</span>
				<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>· {dungeon.type}</span>
			</div>
			<span style={{ color: "var(--sk-fg-1)", fontSize: 12 }}>{dungeon.zone}</span>
			<span className="sk-mono" style={{ color: "var(--sk-fg-2)", fontSize: 11 }}>
				{fmtDuration(seconds)}
			</span>
			<div className="flex items-center ml-auto" style={{ gap: 18 }}>
				<RunStat label="Fame"   value={fmt(fame)}   accent="var(--sk-card-fame)" />
				<RunStat label="Silver" value={fmt(silver)} accent="var(--sk-card-silver)" />
				<RunStat label="Respec" value={kFmt(respec)} accent="var(--sk-card-respec)" />
				<RunStat label="Might"  value={kFmt(might)}  accent="var(--sk-card-might)" />
				{dungeon.deathsInRun !== undefined && dungeon.deathsInRun > 0 && (
					<RunStat label="Deaths" value={dungeon.deathsInRun.toString()} accent="var(--sk-taken)" />
				)}
			</div>
		</div>
	);
}

function RunStat({ label, value, accent }: { label: string; value: string; accent: string }): React.ReactElement {
	return (
		<div className="flex flex-col items-end" style={{ gap: 1, lineHeight: 1 }}>
			<span className="sk-upper" style={{ fontSize: 8.5, color: "var(--sk-fg-3)" }}>{label}</span>
			<span className="sk-mono" style={{ fontSize: 12, color: accent, fontWeight: 600 }}>{value}</span>
		</div>
	);
}

function kFmt(n: number): string {
	if (n <= 0) return "0";
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
	return n.toFixed(0);
}
