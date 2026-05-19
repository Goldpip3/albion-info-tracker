import type { Snapshot } from "./types.ts";
import { fmt, fmtDuration } from "./format.ts";

interface SessionStripProps {
	snapshot: Snapshot | null;
	onReset: () => void;
}

// SessionStrip is the compact row above the meter showing economy
// counters: fame, silver, respec, deaths, session duration. Plus the
// "New session" button on the right.
export function SessionStrip({ snapshot, onReset }: SessionStripProps): React.ReactElement {
	const s = snapshot?.session;
	const elapsedSec = (s?.elapsedMs ?? 0) / 1000;
	const hours = elapsedSec / 3600;

	// FixPoint divisor: SAT stores silver and respec credits as
	// internal long * 10_000. Fame is a raw int. Divide where appropriate.
	const fame = s?.fameTotal ?? 0;
	const silver = (s?.silverTotal ?? 0) / 10_000;
	const respec = (s?.respecTotal ?? 0) / 10_000;

	const fameRate = hours > 0 ? fame / hours : 0;
	const silverRate = hours > 0 ? silver / hours : 0;

	return (
		<div
			className="flex items-center px-4 py-2"
			style={{
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
				gap: 18,
			}}
		>
			<Stat label="Session" value={fmtDuration(elapsedSec)} accent="var(--sk-fg-0)" />
			<Divider />
			<Stat label="Fame"   value={fmt(fame)}   rate={fmtRatePerHour(fameRate, "/hr")}   accent="var(--sk-damage)" />
			<Stat label="Silver" value={fmt(silver)} rate={fmtRatePerHour(silverRate, "/hr")} accent="var(--sk-warn)" />
			<Stat label="Respec" value={respecOnly(respec)} accent="var(--sk-role-support)" />
			<Stat label="Deaths" value={(s?.deathsTotal ?? 0).toString()} accent={(s?.deathsTotal ?? 0) > 0 ? "var(--sk-taken)" : "var(--sk-fg-1)"} />

			<button
				onClick={onReset}
				className="ml-auto sk-upper"
				style={{
					appearance: "none",
					border: "1px solid var(--sk-line-2)",
					background: "var(--sk-bg-3)",
					color: "var(--sk-fg-0)",
					padding: "5px 12px",
					borderRadius: 4,
					cursor: "pointer",
					fontSize: 10.5,
					fontWeight: 600,
					letterSpacing: "0.08em",
				}}
			>
				New session
			</button>
		</div>
	);
}

function Stat({
	label,
	value,
	rate,
	accent,
}: {
	label: string;
	value: string;
	rate?: string;
	accent?: string;
}): React.ReactElement {
	return (
		<div className="flex flex-col" style={{ gap: 1 }}>
			<span className="sk-upper" style={{ color: "var(--sk-fg-3)" }}>{label}</span>
			<div className="flex items-baseline" style={{ gap: 6 }}>
				<span className="sk-mono" style={{ fontSize: 14, fontWeight: 600, color: accent ?? "var(--sk-fg-0)" }}>
					{value}
				</span>
				{rate && (
					<span className="sk-mono" style={{ fontSize: 10.5, color: "var(--sk-fg-2)" }}>
						{rate}
					</span>
				)}
			</div>
		</div>
	);
}

function Divider(): React.ReactElement {
	return (
		<span
			aria-hidden
			style={{ width: 1, alignSelf: "stretch", background: "var(--sk-line)" }}
		/>
	);
}

function fmtRatePerHour(n: number, suffix: string): string {
	if (n <= 0) return `0${suffix}`;
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M${suffix}`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K${suffix}`;
	return `${Math.round(n)}${suffix}`;
}

function respecOnly(credits: number): string {
	if (credits <= 0) return "0";
	if (credits >= 1000) return `${(credits / 1000).toFixed(1)}K`;
	return credits.toFixed(0);
}
