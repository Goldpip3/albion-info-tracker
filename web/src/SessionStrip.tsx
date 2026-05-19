import { useEffect, useRef, useState } from "react";
import type { Snapshot } from "./types.ts";
import { fmt } from "./format.ts";

interface SessionStripProps {
	snapshot: Snapshot | null;
	onReset: () => void;
}

// SessionStrip renders the design's "FarmStrip": a full-bleed grid of
// large mono cards across the top of the meter. Each card shows the
// session total, per-hour rate, and a sparkline trend pulled from a
// client-side ring buffer that records the last ~22 ticks of values.
//
// All five metrics live in the Go agent's RAM; closing the agent clears
// them, so the sparkline buffer is the only thing held in the browser
// and it resets when "New session" fires (session.startedAt changes).
export function SessionStrip({ snapshot, onReset }: SessionStripProps): React.ReactElement {
	const s = snapshot?.session;
	const startedAt = s?.startedAt ?? "";
	const elapsedSec = (s?.elapsedMs ?? 0) / 1000;
	const hours = Math.max(elapsedSec / 3600, 1 / 3600);

	const fame   = s?.fameTotal ?? 0;
	const silver = (s?.silverTotal ?? 0) / 10_000;
	const respec = (s?.respecTotal ?? 0) / 10_000;
	const might  = (s?.mightTotal  ?? 0) / 10_000;
	const deaths = s?.deathsTotal  ?? 0;

	const liveValues = { fame, silver, respec, might, deaths };
	const sparks = useSparkBuffers(liveValues, startedAt);

	const cards: CardSpec[] = [
		{ key: "fame",   label: "Fame",   value: fmt(fame),                rate: ratePerHour(fame   / hours), spark: sparks.fame,   accent: "var(--sk-card-fame)",   glyph: "★" },
		{ key: "silver", label: "Silver", value: fmt(silver),              rate: ratePerHour(silver / hours), spark: sparks.silver, accent: "var(--sk-card-silver)", glyph: "◇" },
		{ key: "respec", label: "Respec", value: kFormat(respec),          rate: ratePerHour(respec / hours), spark: sparks.respec, accent: "var(--sk-card-respec)", glyph: "↻" },
		{ key: "might",  label: "Might",  value: kFormat(might),           rate: ratePerHour(might  / hours), spark: sparks.might,  accent: "var(--sk-card-might)",  glyph: "✦" },
		{ key: "deaths", label: "Deaths", value: deaths.toString(),        rate: ratePerHour(deaths / hours), spark: sparks.deaths, accent: "var(--sk-card-deaths)", glyph: "✕" },
	];

	const live = (snapshot?.fight?.inCombat) ?? false;

	return (
		<div style={{ position: "relative" }}>
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "repeat(5, 1fr)",
					gap: 1,
					background: "var(--sk-line)",
					borderBottom: "1px solid var(--sk-line)",
				}}
			>
				{cards.map(({ key, ...rest }) => (
					<FarmCard key={key} {...rest} live={live} />
				))}
			</div>
			<button
				onClick={onReset}
				className="sk-upper"
				style={{
					position: "absolute",
					top: 8,
					right: 12,
					appearance: "none",
					border: "1px solid var(--sk-line-2)",
					background: "var(--sk-bg-3)",
					color: "var(--sk-fg-0)",
					padding: "4px 10px",
					borderRadius: 4,
					cursor: "pointer",
					fontSize: 10,
					fontWeight: 600,
					letterSpacing: "0.08em",
					zIndex: 1,
				}}
				title="Reset all session counters"
			>
				New session
			</button>
		</div>
	);
}

interface CardSpec {
	key: string;
	label: string;
	value: string;
	rate: string;
	spark: number[];
	accent: string;
	glyph: string;
}

function FarmCard({ label, accent, value, rate, spark, glyph, live }: Omit<CardSpec, "key"> & { live: boolean }): React.ReactElement {
	return (
		<div
			style={{
				background: "var(--sk-bg-1)",
				padding: "14px 18px 14px",
				display: "flex",
				flexDirection: "column",
				gap: 6,
				position: "relative",
				overflow: "hidden",
			}}
		>
			<div className="flex items-center justify-between" style={{ marginBottom: 2 }}>
				<div className="flex items-center" style={{ gap: 7 }}>
					<span
						style={{
							width: 18, height: 18, borderRadius: 4,
							display: "inline-flex", alignItems: "center", justifyContent: "center",
							background: `color-mix(in oklab, ${accent} 18%, var(--sk-bg-2))`,
							border: `1px solid color-mix(in oklab, ${accent} 45%, var(--sk-line))`,
							color: accent, fontSize: 11, fontWeight: 700, lineHeight: 1,
						}}
					>{glyph}</span>
					<span className="sk-upper" style={{
						color: "var(--sk-fg-1)", fontSize: 10.5, letterSpacing: "0.12em", fontWeight: 600,
					}}>{label}</span>
				</div>
				<Sparkline data={spark} color={accent} live={live} />
			</div>
			<span className="sk-display" style={{
				color: "var(--sk-fg-0)", fontSize: 30, lineHeight: 1,
				letterSpacing: "-0.03em",
			}}>{value}</span>
			<div className="flex items-baseline" style={{ gap: 8 }}>
				<span className="sk-mono" style={{ color: accent, fontSize: 12, fontWeight: 600, letterSpacing: "0.02em" }}>
					{rate}
				</span>
				<span className="sk-upper" style={{ color: "var(--sk-fg-3)", fontSize: 9 }}>session pace</span>
			</div>
			<div style={{
				position: "absolute", left: 0, right: 0, bottom: 0,
				height: 2,
				background: `linear-gradient(90deg, ${accent} 0%, color-mix(in oklab, ${accent} 0%, transparent) 80%)`,
				opacity: live ? 0.9 : 0.35,
			}}/>
		</div>
	);
}

function Sparkline({ data, color, live }: { data: number[]; color: string; live: boolean }): React.ReactElement {
	const w = 64, h = 18;
	if (!data || data.length === 0) {
		return <div style={{ width: w, height: h }} />;
	}
	const max = Math.max(...data, 1);
	const min = Math.min(...data, 0);
	const span = Math.max(max - min, 0.001);
	const points = data.map((v, i) => {
		const x = (i / Math.max(data.length - 1, 1)) * (w - 2) + 1;
		const y = h - 2 - ((v - min) / span) * (h - 4);
		return `${x.toFixed(1)},${y.toFixed(1)}`;
	});
	const lastParts = points[points.length - 1].split(",");
	const last = [Number(lastParts[0]), Number(lastParts[1])];
	const gradientId = `spk-${color.replace(/[^a-z0-9]/gi, "")}`;
	return (
		<svg width={w} height={h} style={{ display: "block" }}>
			<defs>
				<linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
					<stop offset="0%"   stopColor={color} stopOpacity="0.35" />
					<stop offset="100%" stopColor={color} stopOpacity="0" />
				</linearGradient>
			</defs>
			<polygon
				points={`1,${h - 1} ${points.join(" ")} ${w - 1},${h - 1}`}
				fill={`url(#${gradientId})`}
			/>
			<polyline
				points={points.join(" ")}
				fill="none" stroke={color} strokeWidth="1.3"
				strokeLinejoin="round" strokeLinecap="round"
				opacity={live ? 1 : 0.55}
			/>
			<circle cx={last[0]} cy={last[1]} r="1.6" fill={color} />
		</svg>
	);
}

interface SparkSet {
	fame: number[];
	silver: number[];
	respec: number[];
	might: number[];
	deaths: number[];
}

const SPARK_LENGTH = 22;

// useSparkBuffers keeps a rolling window of the last SPARK_LENGTH values
// for each metric, sampled once per second. Resets when startedAt
// changes (i.e. the agent fired ResetSession).
function useSparkBuffers(values: Record<keyof SparkSet, number>, startedAt: string): SparkSet {
	const [state, setState] = useState<SparkSet>(() => emptySparks());
	const valuesRef = useRef(values);
	valuesRef.current = values;

	useEffect(() => {
		setState(emptySparks());
	}, [startedAt]);

	useEffect(() => {
		const t = setInterval(() => {
			setState((prev) => pushSample(prev, valuesRef.current));
		}, 1000);
		return () => clearInterval(t);
	}, []);

	return state;
}

function emptySparks(): SparkSet {
	return { fame: [], silver: [], respec: [], might: [], deaths: [] };
}

function pushSample(prev: SparkSet, v: Record<keyof SparkSet, number>): SparkSet {
	const push = (arr: number[], val: number): number[] => {
		const next = arr.length >= SPARK_LENGTH ? arr.slice(arr.length - SPARK_LENGTH + 1) : arr.slice();
		next.push(val);
		return next;
	};
	return {
		fame:   push(prev.fame,   v.fame),
		silver: push(prev.silver, v.silver),
		respec: push(prev.respec, v.respec),
		might:  push(prev.might,  v.might),
		deaths: push(prev.deaths, v.deaths),
	};
}

function ratePerHour(n: number): string {
	if (n <= 0) return "0/hr";
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M/hr`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K/hr`;
	return `${Math.round(n)}/hr`;
}

function kFormat(n: number): string {
	if (n <= 0) return "0";
	if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
	if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
	return n.toFixed(0);
}
