import { useMemo, useState } from "react";
import type { DailyStat, Session } from "./types.ts";
import { fmt } from "./format.ts";

interface ProgressChartProps {
	daily: DailyStat[];
	// liveSession is only used as a fallback: if the agent predates daily
	// tracking (no `daily` in the snapshot) we synthesize a single "today"
	// bar from the live session so the chart isn't blank. When daily data
	// IS present it already includes today's live gains (the agent accrues
	// per-event), so we must NOT add the session on top — that would
	// double-count.
	liveSession?: Session;
}

type MetricKey = "fame" | "silver" | "respec" | "might" | "deaths";

interface MetricDef {
	id: MetricKey;
	label: string;
	accent: string;
	field: keyof DailyStat;
	currency: boolean; // true → FixPoint, divide by 10_000
}

// Accents mirror the SessionStrip "Farm hero" glyph colors so the two
// views read as the same metric (Fame violet, Silver pale, Combat Fame
// gold, Might red); Deaths gets its own warm orange.
const METRICS: MetricDef[] = [
	{ id: "fame",   label: "Fame",        accent: "#c08cff", field: "fameTotal",   currency: true },
	{ id: "silver", label: "Silver",      accent: "#d0d4dc", field: "silverTotal", currency: true },
	{ id: "respec", label: "Combat Fame", accent: "#ffd770", field: "respecTotal", currency: true },
	{ id: "might",  label: "Might",       accent: "#ff6464", field: "mightTotal",  currency: true },
	{ id: "deaths", label: "Deaths",      accent: "#ff8a5c", field: "deathsTotal", currency: false },
];

const MAX_BARS = 30;

function localToday(): string {
	const d = new Date();
	const mm = String(d.getMonth() + 1).padStart(2, "0");
	const dd = String(d.getDate()).padStart(2, "0");
	return `${d.getFullYear()}-${mm}-${dd}`;
}

function valueOf(d: DailyStat, m: MetricDef): number {
	const raw = (d[m.field] as number) ?? 0;
	return m.currency ? Math.floor(raw / 10_000) : raw;
}

// ProgressChart is the Sessions-tab header: a per-day bar chart of one
// economy metric at a time (Fame / Silver / Combat Fame / Might / Deaths)
// with totals, a daily average, and CSV / clipboard export. Data comes
// from the agent's daily.json (snapshot.daily); it survives session
// resets, so it answers "how much did I earn each day this week".
export function ProgressChart({ daily, liveSession }: ProgressChartProps): React.ReactElement {
	const [metricId, setMetricId] = useState<MetricKey>("fame");
	const [toast, setToast] = useState<string | null>(null);
	const metric = METRICS.find((m) => m.id === metricId) ?? METRICS[0];

	const days = useMemo<DailyStat[]>(() => {
		if (daily.length > 0) return daily;
		if (liveSession) {
			return [{
				date: localToday(),
				fameTotal: liveSession.fameTotal,
				silverTotal: liveSession.silverTotal,
				respecTotal: liveSession.respecTotal,
				mightTotal: liveSession.mightTotal,
				deathsTotal: liveSession.deathsTotal,
			}];
		}
		return [];
	}, [daily, liveSession]);

	const windowed = useMemo(() => days.slice(-MAX_BARS), [days]);

	const fireToast = (msg: string): void => {
		setToast(msg);
		window.setTimeout(() => setToast(null), 1500);
	};

	const onDownloadCSV = (): void => {
		const header = "date,fame,silver,combatFame,might,deaths";
		const rows = days.map((d) => [
			d.date,
			Math.floor(d.fameTotal / 10_000),
			Math.floor(d.silverTotal / 10_000),
			Math.floor(d.respecTotal / 10_000),
			Math.floor(d.mightTotal / 10_000),
			d.deathsTotal,
		].join(","));
		const csv = [header, ...rows].join("\r\n");
		const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
		const url = URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = "gda-daily-progress.csv";
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
		fireToast("CSV downloaded");
	};

	const onCopyTable = (): void => {
		const header = ["Date", "Fame", "Silver", "Combat Fame", "Might", "Deaths"].join("\t");
		const rows = days.map((d) => [
			d.date,
			Math.floor(d.fameTotal / 10_000),
			Math.floor(d.silverTotal / 10_000),
			Math.floor(d.respecTotal / 10_000),
			Math.floor(d.mightTotal / 10_000),
			d.deathsTotal,
		].join("\t"));
		navigator.clipboard.writeText([header, ...rows].join("\n")).then(
			() => fireToast("Copied"),
			() => fireToast("Copy failed"),
		);
	};

	const total = windowed.reduce((s, d) => s + valueOf(d, metric), 0);
	const avg = windowed.length > 0 ? Math.round(total / windowed.length) : 0;
	const max = windowed.reduce((m, d) => Math.max(m, valueOf(d, metric)), 0);
	const today = localToday();
	const showLabels = windowed.length <= 16;

	return (
		<section
			style={{
				flexShrink: 0,
				padding: "16px 24px 14px",
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
				position: "relative",
			}}
		>
			{/* Head: index + metric segments (left), totals + export (right) */}
			<div className="flex items-center justify-between" style={{ gap: 16, flexWrap: "wrap", marginBottom: 14 }}>
				<div className="flex items-center" style={{ gap: 12 }}>
					<span className="sk-index" style={{ color: "var(--sk-damage-soft)", fontWeight: 700 }}>02</span>
					<span className="sk-mono" style={{ fontSize: 13, fontWeight: 700, color: "var(--sk-fg-0)", letterSpacing: "0.12em", textTransform: "uppercase" }}>
						Progress
					</span>
					<MetricSegment metric={metricId} onChange={setMetricId} />
				</div>
				<div className="flex items-center" style={{ gap: 14 }}>
					<span className="sk-mono" style={{ fontSize: 11.5, color: "var(--sk-fg-2)" }}>
						<span style={{ color: metric.accent, fontWeight: 600 }}>{fmt(total)}</span> total
						<span style={{ color: "var(--sk-fg-3)" }}>{" · "}{fmt(avg)}/day</span>
					</span>
					<div className="flex items-center" style={{ gap: 6 }}>
						<button onClick={onDownloadCSV} className="sk-upper" style={ghostBtn} title="Download daily totals as a CSV (opens in Excel)">Download CSV</button>
						<button onClick={onCopyTable} className="sk-upper" style={ghostBtn} title="Copy a tab-separated table for Discord / a spreadsheet">Copy as table</button>
					</div>
				</div>
			</div>

			{windowed.length === 0 ? (
				<div style={{ padding: "10px 0 4px", fontSize: 12, color: "var(--sk-fg-3)" }}>
					No daily data yet — your Fame, Silver, Combat Fame, Might, and Deaths will tally here per day as you farm. Daily totals persist across New Session resets.
				</div>
			) : (
				<div className="flex items-end" style={{ gap: 5, height: 168 }}>
					{windowed.map((d) => {
						const v = valueOf(d, metric);
						const pct = max > 0 ? (v / max) * 100 : 0;
						const isToday = d.date === today;
						return (
							<div
								key={d.date}
								className="flex flex-col items-center"
								style={{ flex: 1, minWidth: 0, height: "100%" }}
								title={`${d.date} — ${fmt(v)} ${metric.label.toLowerCase()}`}
							>
								<span style={{ height: 14, fontSize: 9, color: "var(--sk-fg-3)", fontVariantNumeric: "tabular-nums", lineHeight: "14px" }}>
									{showLabels && v > 0 ? fmt(v) : ""}
								</span>
								<div className="flex items-end justify-center" style={{ flex: 1, width: "100%" }}>
									<div
										style={{
											width: "72%",
											maxWidth: 30,
											height: v > 0 ? `${Math.max(pct, 1.5)}%` : 0,
											minHeight: v > 0 ? 2 : 0,
											background: v === 0
												? "transparent"
												: isToday
												? `color-mix(in oklab, ${metric.accent} 60%, transparent)`
												: `color-mix(in oklab, ${metric.accent} 26%, transparent)`,
											border: v > 0 ? `1px solid ${metric.accent}` : "none",
											borderBottom: "none",
											borderRadius: "3px 3px 0 0",
											transition: "height 220ms ease",
										}}
									/>
								</div>
								<span
									style={{
										height: 16,
										fontSize: 9,
										lineHeight: "16px",
										color: isToday ? metric.accent : "var(--sk-fg-3)",
										fontWeight: isToday ? 700 : 400,
										fontVariantNumeric: "tabular-nums",
										whiteSpace: "nowrap",
									}}
								>
									{showLabels ? d.date.slice(5) : d.date.slice(8)}
								</span>
							</div>
						);
					})}
				</div>
			)}

			{toast && (
				<div
					className="sk-upper"
					style={{
						position: "absolute", top: 14, right: 24,
						background: "var(--sk-bg-3)", color: "var(--sk-fg-0)",
						border: "1px solid var(--sk-line-2)", borderRadius: 5,
						padding: "5px 10px", fontSize: 10.5, fontWeight: 600,
						boxShadow: "0 4px 14px rgba(0,0,0,0.35)",
					}}
				>
					{toast}
				</div>
			)}
		</section>
	);
}

function MetricSegment({ metric, onChange }: { metric: MetricKey; onChange: (m: MetricKey) => void }): React.ReactElement {
	return (
		<div
			className="inline-flex items-center"
			style={{ padding: 2, gap: 1, background: "var(--sk-bg-2)", border: "1px solid var(--sk-line)", borderRadius: 5 }}
		>
			{METRICS.map((m) => {
				const active = metric === m.id;
				return (
					<button
						key={m.id}
						onClick={() => onChange(m.id)}
						className="sk-upper"
						title={`Show daily ${m.label}`}
						style={{
							appearance: "none", border: 0, padding: "3px 9px", borderRadius: 3,
							fontSize: 9.5, fontWeight: 600, letterSpacing: "0.05em", cursor: "pointer",
							color: active ? "var(--sk-fg-0)" : "var(--sk-fg-2)",
							background: active ? "var(--sk-bg-3)" : "transparent",
							boxShadow: active ? `inset 0 0 0 1px ${m.accent}` : "none",
						}}
					>
						{m.label}
					</button>
				);
			})}
		</div>
	);
}

const ghostBtn: React.CSSProperties = {
	appearance: "none",
	background: "var(--sk-bg-2)",
	border: "1px solid var(--sk-line)",
	borderRadius: 5,
	color: "var(--sk-fg-1)",
	padding: "5px 10px",
	fontSize: 10,
	fontWeight: 600,
	letterSpacing: "0.06em",
	cursor: "pointer",
};
