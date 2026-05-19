import type { Snapshot } from "./types.ts";
import { fmt, fmtDuration } from "./format.ts";

interface SessionStripProps {
	snapshot: Snapshot | null;
	onReset: () => void;
}

// SessionStrip: a horizontal economy bar with three zones.
//   LEFT   — session timer (compact)
//   CENTER — the big 5 economy cards: Fame, Silver, Respec, Might, Deaths.
//            Bigger numerals + per-hour rate; this is the headline area.
//   RIGHT  — "New session" reset button.
//
// All values live in the Go agent's RAM and are never written to disk;
// closing the agent clears them.
export function SessionStrip({ snapshot, onReset }: SessionStripProps): React.ReactElement {
	const s = snapshot?.session;
	const elapsedSec = (s?.elapsedMs ?? 0) / 1000;
	const hours = elapsedSec / 3600;

	const fame = s?.fameTotal ?? 0;
	const silver = (s?.silverTotal ?? 0) / 10_000;
	const respec = (s?.respecTotal ?? 0) / 10_000;
	const might = (s?.mightTotal ?? 0) / 10_000;
	const deaths = s?.deathsTotal ?? 0;

	const fameRate   = hours > 0 ? fame   / hours : 0;
	const silverRate = hours > 0 ? silver / hours : 0;
	const respecRate = hours > 0 ? respec / hours : 0;
	const mightRate  = hours > 0 ? might  / hours : 0;

	return (
		<div
			className="grid items-center px-4 py-2.5"
			style={{
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
				gridTemplateColumns: "auto 1fr auto",
				gap: 18,
			}}
		>
			<SessionTimer elapsedSec={elapsedSec} />

			<div
				className="flex items-stretch justify-center"
				style={{
					gap: 10,
				}}
			>
				<BigStat label="Fame"   value={fmt(fame)}             rate={ratePerHour(fameRate)}   accent="var(--sk-damage)" />
				<BigStat label="Silver" value={fmt(silver)}           rate={ratePerHour(silverRate)} accent="var(--sk-warn)" />
				<BigStat label="Respec" value={kFormat(respec)}       rate={ratePerHour(respecRate)} accent="var(--sk-role-support)" />
				<BigStat label="Might"  value={kFormat(might)}        rate={ratePerHour(mightRate)}  accent="var(--sk-role-control)" />
				<BigStat label="Deaths" value={deaths.toString()}     accent={deaths > 0 ? "var(--sk-taken)" : "var(--sk-fg-1)"} />
			</div>

			<button
				onClick={onReset}
				className="sk-upper"
				style={{
					appearance: "none",
					border: "1px solid var(--sk-line-2)",
					background: "var(--sk-bg-3)",
					color: "var(--sk-fg-0)",
					padding: "6px 12px",
					borderRadius: 4,
					cursor: "pointer",
					fontSize: 10.5,
					fontWeight: 600,
					letterSpacing: "0.08em",
					whiteSpace: "nowrap",
				}}
			>
				New session
			</button>
		</div>
	);
}

function SessionTimer({ elapsedSec }: { elapsedSec: number }): React.ReactElement {
	return (
		<div className="flex flex-col" style={{ gap: 1, minWidth: 0 }}>
			<span className="sk-upper" style={{ color: "var(--sk-fg-3)", fontSize: 9.5 }}>Session</span>
			<span
				className="sk-mono"
				style={{
					fontSize: 16,
					fontWeight: 600,
					color: "var(--sk-fg-0)",
					lineHeight: 1.1,
					fontVariantNumeric: "tabular-nums",
				}}
			>
				{fmtDuration(elapsedSec)}
			</span>
		</div>
	);
}

function BigStat({
	label, value, rate, accent,
}: {
	label: string;
	value: string;
	rate?: string;
	accent?: string;
}): React.ReactElement {
	return (
		<div
			className="flex flex-col items-center justify-center"
			style={{
				padding: "6px 14px",
				borderRadius: 6,
				background: "var(--sk-bg-2)",
				border: "1px solid var(--sk-line)",
				minWidth: 92,
				gap: 2,
			}}
		>
			<span className="sk-upper" style={{ color: "var(--sk-fg-3)", fontSize: 9.5, letterSpacing: "0.1em" }}>
				{label}
			</span>
			<span
				className="sk-mono"
				style={{
					fontSize: 19,
					fontWeight: 700,
					color: accent ?? "var(--sk-fg-0)",
					lineHeight: 1.05,
					fontVariantNumeric: "tabular-nums",
					letterSpacing: "-0.005em",
				}}
			>
				{value}
			</span>
			{rate && (
				<span
					className="sk-mono"
					style={{
						fontSize: 10,
						color: "var(--sk-fg-3)",
						lineHeight: 1,
						fontVariantNumeric: "tabular-nums",
					}}
				>
					{rate}
				</span>
			)}
		</div>
	);
}

function ratePerHour(n: number): string {
	if (n <= 0) return "0 /hr";
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M /hr`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K /hr`;
	return `${Math.round(n)} /hr`;
}

function kFormat(n: number): string {
	if (n <= 0) return "0";
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
	return n.toFixed(0);
}
