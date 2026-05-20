import { useEffect, useMemo, useRef, useState } from "react";
import type { Snapshot } from "./types.ts";
import { classAccent, fmtDuration, rankBy, roleKeyOf } from "./format.ts";

interface SessionStripProps {
	snapshot: Snapshot | null;
}

const SERIF = "'Fraunces', Georgia, 'Times New Roman', serif";

// SessionStrip renders the design's C3 "Farm hero": an editorial 1+3
// layout where Fame leads as a giant mono number on the left and the
// three secondary stats (Silver / Combat Fame / Might) stack on the
// right, each with its own sparkline. Mirrors the Skirmish C3
// production candidate. All four metrics live in the Go agent's RAM;
// the per-second sparkline buffer is held in the browser and resets on
// New Session (session.startedAt changes).
export function SessionStrip({ snapshot }: SessionStripProps): React.ReactElement {
	const s = snapshot?.session;
	const startedAt = s?.startedAt ?? "";
	const elapsedSec = (s?.elapsedMs ?? 0) / 1000;
	const hours = Math.max(elapsedSec / 3600, 1 / 3600);

	// All four currency fields are FixPoint internal units in the snapshot
	// (10_000 internal = 1 real). Confirmed against SAT's UpdateFameEvent /
	// UpdateMoneyEvent / UpdateReSpecPointsEvent / MightAndFavorReceivedEvent
	// sources. Divide before formatting.
	const fame   = (s?.fameTotal   ?? 0) / 10_000;
	const silver = (s?.silverTotal ?? 0) / 10_000;
	const respec = (s?.respecTotal ?? 0) / 10_000;
	const might  = (s?.mightTotal  ?? 0) / 10_000;

	const liveValues = { fame, silver, respec, might };
	const sparks = useSparkBuffers(liveValues, startedAt);
	const live = (snapshot?.fight?.inCombat) ?? false;

	// The "carried by" credit goes to the top damage dealer this session.
	// Memoised on generatedAt so it doesn't re-sort every render.
	const carrier = useMemo(() => {
		const players = snapshot?.players ?? [];
		if (players.length === 0) return null;
		return [...players].sort(rankBy((p) => p.overallDamage ?? 0))[0];
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [snapshot?.generatedAt]);

	// Literal hex so the SVG sparkline gradient stop-color resolves
	// reliably (Chrome/Firefox quirks with var() inside <stop>).
	const lead: StatSpec = { label: "Fame", value: kFormat(fame), rate: ratePerHour(fame / hours), spark: sparks.fame, accent: "#c08cff", glyph: "★" };
	const rest: StatSpec[] = [
		{ label: "Silver",      value: kFormat(silver), rate: ratePerHour(silver / hours), spark: sparks.silver, accent: "#d0d4dc", glyph: "◇" },
		{ label: "Combat Fame", value: kFormat(respec), rate: ratePerHour(respec / hours), spark: sparks.respec, accent: "#ffd770", glyph: "⚔" },
		{ label: "Might",       value: kFormat(might),  rate: ratePerHour(might  / hours), spark: sparks.might,  accent: "#ff6464", glyph: "✦" },
	];

	const carrierAccent = carrier ? classAccent(carrier.classCode, roleKeyOf(carrier.role)) : "var(--sk-fg-2)";

	return (
		<section
			style={{
				padding: "20px 28px 18px",
				borderBottom: "1px solid var(--sk-line)",
				background: "var(--sk-bg-1)",
			}}
		>
			{/* Section head — 01 · FARM + "carried by" credit */}
			<div className="flex items-baseline justify-between" style={{ marginBottom: 14, gap: 16, flexWrap: "wrap" }}>
				<div className="flex items-baseline" style={{ gap: 12 }}>
					<span className="sk-index" style={{ color: "var(--sk-damage-soft)", fontWeight: 700 }}>01</span>
					<span className="sk-mono" style={{ fontSize: 13, fontWeight: 700, color: "var(--sk-fg-0)", letterSpacing: "0.12em", textTransform: "uppercase" }}>
						Farm
					</span>
					<span style={{ fontSize: 11.5, color: "var(--sk-fg-2)" }}>
						· {fmtDuration(elapsedSec)} session
					</span>
				</div>
				{carrier && (
					<span style={{ fontSize: 12, color: "var(--sk-fg-1)" }}>
						Carried by{" "}
						<span style={{ fontFamily: SERIF, fontStyle: "italic", fontSize: 20, color: "var(--sk-fg-0)" }}>
							{carrier.name || "—"}
						</span>
						{carrier.roleLabel && (
							<span className="sk-mono" style={{ marginLeft: 8, color: carrierAccent, fontWeight: 700, letterSpacing: "0.1em", textTransform: "uppercase", fontSize: 10.5 }}>
								· {carrier.roleLabel}
							</span>
						)}
					</span>
				)}
			</div>

			{/* 1 + 3 editorial grid */}
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "minmax(0, 1.5fr) minmax(0, 1fr)",
					gap: 36,
					alignItems: "stretch",
				}}
			>
				{/* LEAD — Fame */}
				<div className="flex flex-col" style={{ gap: 10, minWidth: 0 }}>
					<div className="flex items-center" style={{ gap: 11 }}>
						<StatGlyph glyph={lead.glyph} accent={lead.accent} size={34} />
						<span className="sk-upper" style={{ color: "var(--sk-fg-0)", fontSize: 11.5, letterSpacing: "0.16em", fontWeight: 700 }}>{lead.label}</span>
					</div>
					<div className="flex items-end" style={{ gap: 20, minWidth: 0 }}>
						<span style={{
							fontFamily: "var(--sk-font-mono)",
							fontVariantNumeric: "tabular-nums",
							fontWeight: 500,
							color: "var(--sk-fg-0)",
							fontSize: "clamp(64px, 8.5vw, 128px)",
							lineHeight: 0.82,
							letterSpacing: "-0.05em",
						}}>{lead.value}</span>
						<div className="flex flex-col" style={{ gap: 3, paddingBottom: 10 }}>
							<span className="sk-mono" style={{ fontSize: 18, color: lead.accent, fontWeight: 500 }}>{lead.rate}</span>
							<span className="sk-upper" style={{ fontSize: 9, color: "var(--sk-fg-3)" }}>session pace</span>
						</div>
					</div>
					<div style={{ marginTop: 4 }}>
						<Sparkline data={lead.spark} color={lead.accent} live={live} w={320} h={52} />
					</div>
				</div>

				{/* SECONDARY — Silver / Combat Fame / Might */}
				<div className="flex flex-col" style={{ borderLeft: "1px solid var(--sk-line)", paddingLeft: 28, minWidth: 0 }}>
					{rest.map((stat, i) => (
						<div
							key={stat.label}
							className="grid items-center"
							style={{
								gridTemplateColumns: "26px minmax(0, 1fr) 96px",
								gap: 13,
								padding: "14px 0",
								borderBottom: i < rest.length - 1 ? "1px solid var(--sk-line-soft)" : "none",
							}}
						>
							<StatGlyph glyph={stat.glyph} accent={stat.accent} size={24} />
							<div style={{ minWidth: 0 }}>
								<span style={{
									fontFamily: "var(--sk-font-mono)",
									fontVariantNumeric: "tabular-nums",
									fontWeight: 500,
									color: "var(--sk-fg-0)",
									fontSize: 34,
									lineHeight: 0.95,
									letterSpacing: "-0.04em",
								}}>{stat.value}</span>
								<div className="flex items-baseline" style={{ gap: 8, marginTop: 3 }}>
									<span className="sk-upper" style={{ fontSize: 9.5, color: "var(--sk-fg-1)", letterSpacing: "0.14em", fontWeight: 700 }}>{stat.label}</span>
									<span className="sk-mono" style={{ fontSize: 11, color: stat.accent, fontWeight: 600 }}>{stat.rate}</span>
								</div>
							</div>
							<Sparkline data={stat.spark} color={stat.accent} live={live} w={96} h={26} />
						</div>
					))}
				</div>
			</div>
		</section>
	);
}

interface StatSpec {
	label: string;
	value: string;
	rate: string;
	spark: number[];
	accent: string;
	glyph: string;
}

function StatGlyph({ glyph, accent, size }: { glyph: string; accent: string; size: number }): React.ReactElement {
	return (
		<span
			style={{
				width: size, height: size, borderRadius: Math.round(size * 0.22),
				display: "inline-flex", alignItems: "center", justifyContent: "center",
				flexShrink: 0,
				background: `color-mix(in oklab, ${accent} 18%, var(--sk-bg-2))`,
				border: `1px solid color-mix(in oklab, ${accent} 45%, var(--sk-line))`,
				color: accent, fontSize: Math.round(size * 0.5), fontWeight: 700, lineHeight: 1,
			}}
		>{glyph}</span>
	);
}

function Sparkline({ data, color, live, w = 64, h = 18 }: { data: number[]; color: string; live: boolean; w?: number; h?: number }): React.ReactElement {
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
}

const SPARK_LENGTH = 22;

// useSparkBuffers samples each metric once per second and stores the
// PER-SECOND GAIN (current - previous), not the cumulative total. That
// way the sparkline pulses with activity instead of climbing monotonic-
// ally — out-of-combat samples sit near zero, kills spike, and you can
// read the line as "how hard am I farming right now."
//
// Resets when startedAt changes (the agent fired ResetSession).
function useSparkBuffers(values: Record<keyof SparkSet, number>, startedAt: string): SparkSet {
	const [state, setState] = useState<SparkSet>(() => emptySparks());
	const valuesRef = useRef(values);
	valuesRef.current = values;
	const prevRef = useRef<Record<keyof SparkSet, number>>({ fame: 0, silver: 0, respec: 0, might: 0 });
	const baselineRef = useRef<boolean>(false);

	useEffect(() => {
		setState(emptySparks());
		prevRef.current = { fame: 0, silver: 0, respec: 0, might: 0 };
		baselineRef.current = false;
	}, [startedAt]);

	useEffect(() => {
		const t = setInterval(() => {
			const cur = valuesRef.current;
			const prev = prevRef.current;
			// First tick after a reset just seeds the baseline; emit 0
			// so we don't spike from "0 → 95K fame" in one frame.
			const delta: Record<keyof SparkSet, number> = baselineRef.current
				? {
					fame:   Math.max(0, cur.fame   - prev.fame),
					silver: Math.max(0, cur.silver - prev.silver),
					respec: Math.max(0, cur.respec - prev.respec),
					might:  Math.max(0, cur.might  - prev.might),
				}
				: { fame: 0, silver: 0, respec: 0, might: 0 };
			prevRef.current = cur;
			baselineRef.current = true;
			setState((s) => pushSample(s, delta));
		}, 2000); // 2s sample — matches the footer's "spark 2s" label.
		return () => clearInterval(t);
	}, []);

	return state;
}

function emptySparks(): SparkSet {
	return { fame: [], silver: [], respec: [], might: [] };
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
